package render

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var catalogBundleLinkPattern = regexp.MustCompile(`\]\((bundles/[^)]+\.md)\)`)

func inspectRoutingTarget(repoRoot string, routing *RoutingResult) error {
	store, err := openRoutingFileSystem(repoRoot)
	if err != nil {
		return err
	}
	defer store.close()
	return inspectRoutingTargetWithFS(store, routing)
}

func inspectRoutingTargetWithFS(store routingFileSystem, routing *RoutingResult) error {
	if routing == nil {
		return nil
	}
	target := routing.Path
	info, err := store.lstat(target)
	if errors.Is(err, fs.ErrNotExist) {
		routing.targetExists = false
		routing.targetDigest = ""
		routing.Changed = routing.Exists
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect routing target: %w", err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: routing target must be a real directory", ErrUnsafeTarget)
	}
	existing, err := readExistingRoutingTreeWithFS(store, target)
	if err != nil {
		return err
	}
	routing.targetExists = true
	routing.targetDigest = digestExistingRoutingTree(existing)
	if !routing.Exists {
		routing.Changed = true
		return nil
	}
	planned := make(map[string][]byte, len(routing.Files))
	for _, file := range routing.Files {
		relative, err := routingRelativePath(file.Path)
		if err != nil {
			return err
		}
		planned[relative] = file.Content
	}
	routing.Changed = len(existing) != len(planned)
	if !routing.Changed {
		for relative, content := range planned {
			if !bytes.Equal(existing[relative], content) {
				routing.Changed = true
				break
			}
		}
	}
	return nil
}

func readExistingRoutingTreeWithFS(store routingFileSystem, target string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := store.walkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == target {
			return nil
		}
		relative := strings.TrimPrefix(current, target+"/")
		if relative == current || relative == "" {
			return fmt.Errorf("%w: routing walk returned path %s outside %s", ErrUnsafeTarget, current, target)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%w: routing path %s is a symlink", ErrUnsafeTarget, relative)
		}
		if entry.IsDir() {
			if relative != "bundles" {
				return fmt.Errorf("%w: routing directory contains unexpected directory %s", ErrUnsafeTarget, relative)
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: routing path %s is not a regular file", ErrUnsafeTarget, relative)
		}
		if relative != "catalog.md" && !strings.HasPrefix(relative, "bundles/") {
			return fmt.Errorf("%w: routing directory contains unexpected file %s", ErrUnsafeTarget, relative)
		}
		content, err := store.readFile(current)
		if err != nil {
			return err
		}
		if err := verifyRoutingFile(content); err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
		files[relative] = content
		return nil
	})
	if err != nil {
		return nil, err
	}
	catalog, exists := files["catalog.md"]
	if !exists {
		return nil, fmt.Errorf("%w: routing catalog is missing", ErrDrift)
	}
	referenced := make(map[string]struct{})
	for _, match := range catalogBundleLinkPattern.FindAllSubmatch(catalog, -1) {
		referenced[string(match[1])] = struct{}{}
	}
	for relative := range files {
		if relative == "catalog.md" {
			continue
		}
		if _, exists := referenced[relative]; !exists {
			return nil, fmt.Errorf("%w: routing bundle %s is not listed by catalog.md", ErrDrift, relative)
		}
	}
	for relative := range referenced {
		if _, exists := files[relative]; !exists {
			return nil, fmt.Errorf("%w: routing catalog references missing bundle %s", ErrDrift, relative)
		}
	}
	return files, nil
}

func verifyRoutingFile(content []byte) error {
	start := []byte(routingStart + "\n<!-- source-digest: ")
	if !bytes.HasPrefix(content, start) || !bytes.HasSuffix(content, []byte(routingEnd+"\n")) ||
		bytes.Count(content, []byte(routingStart)) != 1 || bytes.Count(content, []byte(routingEnd)) != 1 {
		return fmt.Errorf("%w: generated routing markers are malformed", ErrMarkers)
	}
	rest := content[len(start):]
	sourceEnd := bytes.Index(rest, []byte(" -->\n"))
	if sourceEnd < 0 || !validDigest(string(rest[:sourceEnd])) {
		return fmt.Errorf("%w: generated routing source digest is malformed", ErrMarkers)
	}
	sourceDigest := string(rest[:sourceEnd])
	rest = rest[sourceEnd+len(" -->\n"):]
	contentPrefix := []byte("<!-- content-digest: ")
	if !bytes.HasPrefix(rest, contentPrefix) {
		return fmt.Errorf("%w: generated routing content digest is missing", ErrMarkers)
	}
	rest = rest[len(contentPrefix):]
	contentEnd := bytes.Index(rest, []byte(" -->\n"))
	if contentEnd < 0 || !validDigest(string(rest[:contentEnd])) {
		return fmt.Errorf("%w: generated routing content digest is malformed", ErrMarkers)
	}
	recorded := string(rest[:contentEnd])
	bodyWithEnd := rest[contentEnd+len(" -->\n"):]
	body := bodyWithEnd[:len(bodyWithEnd)-len(routingEnd)-1]
	actual := digest(append([]byte("<!-- source-digest: "+sourceDigest+" -->\n"), body...))
	if recorded != actual {
		return fmt.Errorf("%w: edit canonical artifact sources and manifest.yaml instead of generated routing files", ErrDrift)
	}
	return nil
}

func stageRoutingTarget(repoRoot string, routing *RoutingResult) (string, error) {
	store, err := openRoutingFileSystem(repoRoot)
	if err != nil {
		return "", err
	}
	defer store.close()
	stage, err := stageRoutingTargetWithFS(store, routing)
	if err != nil {
		return "", err
	}
	return filepath.Join(repoRoot, filepath.FromSlash(stage)), nil
}

func stageRoutingTargetWithFS(store routingFileSystem, routing *RoutingResult) (string, error) {
	stage, err := store.makeTempDir(".software-standards", ".ssb-routing-stage-")
	if err != nil {
		return "", fmt.Errorf("create staged routing directory: %w", err)
	}
	failed := true
	defer func() {
		if failed {
			_ = store.removeAll(stage)
		}
	}()
	for _, file := range routing.Files {
		relative, err := routingRelativePath(file.Path)
		if err != nil {
			return "", err
		}
		target := path.Join(stage, relative)
		if err := store.mkdirAll(path.Dir(target), 0o755); err != nil {
			return "", fmt.Errorf("create staged routing parent: %w", err)
		}
		if err := writeSyncedFileWithFS(store, target, file.Content); err != nil {
			return "", err
		}
	}
	failed = false
	return stage, nil
}

func routingRelativePath(filePath string) (string, error) {
	prefix := RoutingDirectory + "/"
	if !strings.HasPrefix(filePath, prefix) {
		return "", fmt.Errorf("%w: routing output %s is outside %s", ErrUnsafeTarget, filePath, RoutingDirectory)
	}
	relative := strings.TrimPrefix(filePath, prefix)
	if relative == "" || strings.HasPrefix(relative, "/") || filepath.IsAbs(filepath.FromSlash(relative)) ||
		filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative))) != relative || strings.HasPrefix(relative, "../") {
		return "", fmt.Errorf("%w: routing output path %s is unsafe", ErrUnsafeTarget, filePath)
	}
	return relative, nil
}

func writeSyncedFileWithFS(store routingFileSystem, target string, content []byte) (returnErr error) {
	file, err := store.createExclusiveFile(target, 0o644)
	if err != nil {
		return fmt.Errorf("create staged routing file: %w", err)
	}
	defer func() {
		if err := file.Close(); returnErr == nil && err != nil {
			returnErr = fmt.Errorf("close staged routing file: %w", err)
		}
	}()
	if written, err := file.Write(content); err != nil {
		return fmt.Errorf("write staged routing file: %w", err)
	} else if written != len(content) {
		return fmt.Errorf("write staged routing file: %w", io.ErrShortWrite)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync staged routing file: %w", err)
	}
	return nil
}

func publishRoutingTarget(repoRoot string, routing *RoutingResult, staged string) (string, error) {
	store, err := openRoutingFileSystem(repoRoot)
	if err != nil {
		return "", err
	}
	defer store.close()
	stagedRelative, err := repositoryRelativePath(repoRoot, staged)
	if err != nil {
		return "", err
	}
	backup, err := publishRoutingTargetWithFS(store, routing, stagedRelative)
	if backup != "" {
		backup = filepath.Join(repoRoot, filepath.FromSlash(backup))
	}
	return backup, err
}

func publishRoutingTargetWithRename(
	repoRoot string,
	routing *RoutingResult,
	staged string,
	rename func(string, string) error,
) (string, error) {
	store, err := openRoutingFileSystem(repoRoot)
	if err != nil {
		return "", err
	}
	defer store.close()
	stagedRelative, err := repositoryRelativePath(repoRoot, staged)
	if err != nil {
		return "", err
	}
	store = &renameRoutingFileSystem{routingFileSystem: store, repoRoot: repoRoot, renamePath: rename}
	backup, err := publishRoutingTargetWithFS(store, routing, stagedRelative)
	if backup != "" {
		backup = filepath.Join(repoRoot, filepath.FromSlash(backup))
	}
	return backup, err
}

type renameRoutingFileSystem struct {
	routingFileSystem
	repoRoot   string
	renamePath func(string, string) error
}

func (fileSystem *renameRoutingFileSystem) rename(oldName, newName string) error {
	return fileSystem.renamePath(
		filepath.Join(fileSystem.repoRoot, filepath.FromSlash(oldName)),
		filepath.Join(fileSystem.repoRoot, filepath.FromSlash(newName)),
	)
}

func repositoryRelativePath(repoRoot, target string) (string, error) {
	if target == "" {
		return "", nil
	}
	if !filepath.IsAbs(target) {
		relative := filepath.ToSlash(target)
		if relative == "." || relative == ".." || strings.HasPrefix(relative, "../") || path.IsAbs(relative) || path.Clean(relative) != relative {
			return "", fmt.Errorf("%w: routing path %s is outside the repository", ErrUnsafeTarget, target)
		}
		return relative, nil
	}
	relative, err := filepath.Rel(repoRoot, target)
	if err != nil {
		return "", fmt.Errorf("resolve routing path within repository: %w", err)
	}
	relative = filepath.ToSlash(relative)
	if relative == "." || relative == ".." || strings.HasPrefix(relative, "../") || path.IsAbs(relative) {
		return "", fmt.Errorf("%w: routing path %s is outside the repository", ErrUnsafeTarget, target)
	}
	return relative, nil
}

func publishRoutingTargetWithFS(
	store routingFileSystem,
	routing *RoutingResult,
	staged string,
) (string, error) {
	if routing == nil || !routing.Changed {
		return "", nil
	}
	const parent = ".software-standards"
	target := routing.Path
	if routing.Exists && staged == "" {
		return "", fmt.Errorf("%w: staged routing directory is missing", ErrUnsafeTarget)
	}
	if err := verifyRoutingPrestateWithFS(store, target, routing); err != nil {
		return "", err
	}
	backup := ""
	if _, err := store.lstat(target); err == nil {
		backup, err = store.makeTempDir(parent, ".ssb-routing-backup-")
		if err != nil {
			return "", fmt.Errorf("reserve routing backup: %w", err)
		}
		if err := store.remove(backup); err != nil {
			return "", fmt.Errorf("prepare routing backup: %w", err)
		}
		if err := store.rename(target, backup); err != nil {
			return "", fmt.Errorf("backup routing directory: %w", err)
		}
		moved, movedErr := readExistingRoutingTreeWithFS(store, backup)
		if movedErr != nil || digestExistingRoutingTree(moved) != routing.targetDigest {
			driftErr := fmt.Errorf("%w: routing directory changed while moving it to the transaction backup", ErrDrift)
			if movedErr != nil {
				driftErr = fmt.Errorf("%w: %v", driftErr, movedErr)
			}
			if restoreErr := restoreRoutingBackupWithFS(store, backup, target); restoreErr != nil {
				return backup, errors.Join(driftErr, fmt.Errorf("restore changed routing directory: %w", restoreErr))
			}
			return "", driftErr
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("inspect routing directory before publish: %w", err)
	}
	if !routing.Exists {
		return backup, nil
	}
	if err := store.rename(staged, target); err != nil {
		if backup != "" {
			if restoreErr := restoreRoutingBackupWithFS(store, backup, target); restoreErr != nil {
				return backup, errors.Join(
					fmt.Errorf("publish routing directory: %w", err),
					fmt.Errorf("restore previous tree: %w", restoreErr),
				)
			}
		}
		return "", fmt.Errorf("publish routing directory: %w", err)
	}
	return backup, nil
}

func restoreRoutingBackupWithFS(store routingFileSystem, backup, target string) error {
	if err := store.rename(backup, target); err == nil {
		return nil
	} else if copyErr := restoreRoutingBackupByCopyWithFS(store, backup, target); copyErr != nil {
		return fmt.Errorf("rename backup: %v; copy recovery: %w; previous tree remains at %s", err, copyErr, backup)
	}
	if err := discardRoutingBackupWithFS(store, backup); err != nil {
		return fmt.Errorf("copy recovery restored the target but backup cleanup failed: %w", err)
	}
	return nil
}

func restoreRoutingBackupByCopyWithFS(store routingFileSystem, backup, target string) error {
	existing, err := readExistingRoutingTreeWithFS(store, backup)
	if err != nil {
		return fmt.Errorf("validate routing backup: %w", err)
	}
	if err := store.mkdir(target, 0o755); err != nil {
		return fmt.Errorf("create routing recovery target: %w", err)
	}
	paths := make([]string, 0, len(existing))
	for relative := range existing {
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		fileTarget := path.Join(target, relative)
		if err := store.mkdirAll(path.Dir(fileTarget), 0o755); err != nil {
			return fmt.Errorf("create routing recovery parent: %w; partial recovery remains at %s", err, target)
		}
		if err := writeSyncedFileWithFS(store, fileTarget, existing[relative]); err != nil {
			return fmt.Errorf("restore routing backup file %s: %w; partial recovery remains at %s", relative, err, target)
		}
	}
	return nil
}

func verifyRoutingPrestateWithFS(store routingFileSystem, target string, routing *RoutingResult) error {
	info, err := store.lstat(target)
	if !routing.targetExists {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect routing directory before publish: %w", err)
		}
		return fmt.Errorf("%w: routing directory appeared while rendering", ErrUnsafeTarget)
	}
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: routing directory disappeared while rendering", ErrDrift)
		}
		return fmt.Errorf("inspect routing directory before publish: %w", err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: routing directory changed while rendering", ErrUnsafeTarget)
	}
	existing, err := readExistingRoutingTreeWithFS(store, target)
	if err != nil {
		return err
	}
	if digestExistingRoutingTree(existing) != routing.targetDigest {
		return fmt.Errorf("%w: routing directory changed while rendering", ErrDrift)
	}
	return nil
}

func digestExistingRoutingTree(existing map[string][]byte) string {
	files := make([]FileResult, 0, len(existing))
	for relative, content := range existing {
		files = append(files, FileResult{
			Path:   RoutingDirectory + "/" + relative,
			SHA256: digest(content),
		})
	}
	return routingTreeDigest(files)
}

func rollbackRoutingTarget(repoRoot string, routing *RoutingResult, backup string) error {
	store, err := openRoutingFileSystem(repoRoot)
	if err != nil {
		return err
	}
	defer store.close()
	backupRelative, err := repositoryRelativePath(repoRoot, backup)
	if err != nil {
		return err
	}
	return rollbackRoutingTargetWithFS(store, routing, backupRelative)
}

func rollbackRoutingTargetWithFS(store routingFileSystem, routing *RoutingResult, backup string) error {
	if routing == nil || !routing.Changed {
		return nil
	}
	target := routing.Path
	if !routing.Exists {
		if _, err := store.lstat(target); err == nil {
			return fmt.Errorf("%w: routing directory changed after publish; previous tree remains at %s", ErrDrift, backup)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("inspect routing directory before rollback: %w", err)
		}
		if backup != "" {
			return restoreRoutingBackupWithFS(store, backup, target)
		}
		return nil
	}
	info, err := store.lstat(target)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: published routing directory disappeared before rollback; previous tree remains at %s", ErrDrift, backup)
		}
		return fmt.Errorf("inspect routing directory before rollback: %w", err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: routing directory changed after publish; previous tree remains at %s", ErrUnsafeTarget, backup)
	}
	quarantine, err := store.makeTempDir(".software-standards", ".ssb-routing-rollback-")
	if err != nil {
		return fmt.Errorf("reserve routing rollback quarantine: %w", err)
	}
	if err := store.remove(quarantine); err != nil {
		return fmt.Errorf("prepare routing rollback quarantine: %w", err)
	}
	if err := store.rename(target, quarantine); err != nil {
		return fmt.Errorf("quarantine published routing directory: %w", err)
	}
	published, readErr := readExistingRoutingTreeWithFS(store, quarantine)
	expectedDigest := routing.TreeDigest
	if expectedDigest == "" {
		expectedDigest = routingTreeDigest(routing.Files)
	}
	if readErr != nil || digestExistingRoutingTree(published) != expectedDigest {
		driftErr := fmt.Errorf("%w: routing directory changed after publish; previous tree remains at %s", ErrDrift, backup)
		if readErr != nil {
			driftErr = fmt.Errorf("%w: %v", driftErr, readErr)
		}
		if restoreErr := restoreRoutingBackupWithFS(store, quarantine, target); restoreErr != nil {
			return errors.Join(driftErr, fmt.Errorf("restore changed routing directory: %w", restoreErr))
		}
		return driftErr
	}
	if backup != "" {
		if err := restoreRoutingBackupWithFS(store, backup, target); err != nil {
			return errors.Join(
				fmt.Errorf("restore previous routing tree: %w", err),
				fmt.Errorf("published routing tree remains at %s", quarantine),
			)
		}
	}
	if err := store.removeAll(quarantine); err != nil {
		return fmt.Errorf("remove rolled-back routing tree: %w", err)
	}
	return nil
}

func discardRoutingBackupWithFS(store routingFileSystem, backup string) error {
	if backup == "" {
		return nil
	}
	if err := store.removeAll(backup); err != nil {
		return fmt.Errorf("remove routing backup: %w", err)
	}
	return nil
}

package render

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var catalogBundleLinkPattern = regexp.MustCompile(`\]\((bundles/[^)]+\.md)\)`)

func inspectRoutingTarget(repoRoot string, routing *RoutingResult) error {
	if routing == nil {
		return nil
	}
	target := filepath.Join(repoRoot, filepath.FromSlash(routing.Path))
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		routing.targetExists = false
		routing.targetDigest = ""
		routing.Changed = routing.Exists
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect routing target: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: routing target must be a real directory", ErrUnsafeTarget)
	}
	existing, err := readExistingRoutingTree(target)
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

func readExistingRoutingTree(target string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := filepath.WalkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == target {
			return nil
		}
		relative, err := filepath.Rel(target, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
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
		content, err := os.ReadFile(current)
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
	parent := filepath.Join(repoRoot, ".software-standards")
	stage, err := os.MkdirTemp(parent, ".ssb-routing-stage-*")
	if err != nil {
		return "", fmt.Errorf("create staged routing directory: %w", err)
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(stage)
		}
	}()
	for _, file := range routing.Files {
		relative, err := routingRelativePath(file.Path)
		if err != nil {
			return "", err
		}
		target := filepath.Join(stage, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", fmt.Errorf("create staged routing parent: %w", err)
		}
		if err := writeSyncedFile(target, file.Content); err != nil {
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

func writeSyncedFile(target string, content []byte) (returnErr error) {
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create staged routing file: %w", err)
	}
	defer func() {
		if err := file.Close(); returnErr == nil && err != nil {
			returnErr = fmt.Errorf("close staged routing file: %w", err)
		}
	}()
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("write staged routing file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync staged routing file: %w", err)
	}
	return nil
}

func publishRoutingTarget(repoRoot string, routing *RoutingResult, staged string) (string, error) {
	return publishRoutingTargetWithRename(repoRoot, routing, staged, os.Rename)
}

func publishRoutingTargetWithRename(
	repoRoot string,
	routing *RoutingResult,
	staged string,
	rename func(string, string) error,
) (string, error) {
	if routing == nil || !routing.Changed {
		return "", nil
	}
	parent := filepath.Join(repoRoot, ".software-standards")
	target := filepath.Join(repoRoot, filepath.FromSlash(routing.Path))
	if err := verifyRoutingPrestate(target, routing); err != nil {
		return "", err
	}
	backup := ""
	if _, err := os.Lstat(target); err == nil {
		backup, err = os.MkdirTemp(parent, ".ssb-routing-backup-*")
		if err != nil {
			return "", fmt.Errorf("reserve routing backup: %w", err)
		}
		if err := os.Remove(backup); err != nil {
			return "", fmt.Errorf("prepare routing backup: %w", err)
		}
		if err := rename(target, backup); err != nil {
			return "", fmt.Errorf("backup routing directory: %w", err)
		}
		// Validate after claiming the target so concurrent changes are preserved.
		if err := verifyRoutingPrestate(backup, routing); err != nil {
			if restoreErr := restoreRoutingBackup(backup, target, rename); restoreErr != nil {
				return "", fmt.Errorf("routing tree changed while claiming it: %w; restore: %v", err, restoreErr)
			}
			return "", fmt.Errorf("routing tree changed while claiming it: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect routing directory before publish: %w", err)
	}
	if !routing.Exists {
		return backup, nil
	}
	if err := rename(staged, target); err != nil {
		if backup != "" {
			if restoreErr := restoreRoutingBackup(backup, target, rename); restoreErr != nil {
				return backup, fmt.Errorf("publish routing directory: %w; restore previous tree: %v", err, restoreErr)
			}
		}
		return "", fmt.Errorf("publish routing directory: %w", err)
	}
	return backup, nil
}

func restoreRoutingBackup(backup, target string, rename func(string, string) error) error {
	if err := rename(backup, target); err == nil {
		return nil
	} else if copyErr := restoreRoutingBackupByCopy(backup, target); copyErr != nil {
		return fmt.Errorf("rename backup: %v; copy recovery: %w; previous tree remains at %s", err, copyErr, backup)
	}
	if err := discardRoutingBackup(backup); err != nil {
		return fmt.Errorf("copy recovery restored the target but backup cleanup failed: %w", err)
	}
	return nil
}

func restoreRoutingBackupByCopy(backup, target string) (returnErr error) {
	existing, err := readExistingRoutingTree(backup)
	if err != nil {
		return fmt.Errorf("validate routing backup: %w", err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		return fmt.Errorf("create routing recovery target: %w", err)
	}
	defer func() {
		if returnErr == nil {
			return
		}
		if err := os.RemoveAll(target); err != nil {
			returnErr = fmt.Errorf("%w; remove partial recovery target: %v", returnErr, err)
		}
	}()
	paths := make([]string, 0, len(existing))
	for relative := range existing {
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		fileTarget := filepath.Join(target, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(fileTarget), 0o755); err != nil {
			return fmt.Errorf("create routing recovery parent: %w", err)
		}
		if err := writeSyncedFile(fileTarget, existing[relative]); err != nil {
			return fmt.Errorf("restore routing backup file %s: %w", relative, err)
		}
	}
	return nil
}

func verifyRoutingPrestate(target string, routing *RoutingResult) error {
	info, err := os.Lstat(target)
	if !routing.targetExists {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect routing directory before publish: %w", err)
		}
		return fmt.Errorf("%w: routing directory appeared while rendering", ErrUnsafeTarget)
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: routing directory disappeared while rendering", ErrDrift)
		}
		return fmt.Errorf("inspect routing directory before publish: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: routing directory changed while rendering", ErrUnsafeTarget)
	}
	existing, err := readExistingRoutingTree(target)
	if err != nil {
		return err
	}
	if digestExistingRoutingTree(existing) != routing.targetDigest {
		return fmt.Errorf("%w: routing directory changed while rendering", ErrDrift)
	}
	return nil
}

func verifyRoutingPoststate(target string, routing *RoutingResult) error {
	info, err := os.Lstat(target)
	if !routing.Exists {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("routing target was recreated")
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("routing target is no longer a directory")
	}
	existing, err := readExistingRoutingTree(target)
	if err != nil {
		return err
	}
	if digestExistingRoutingTree(existing) != routing.TreeDigest {
		return fmt.Errorf("routing target changed after publication")
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
	if routing == nil || !routing.Changed {
		return nil
	}
	target := filepath.Join(repoRoot, filepath.FromSlash(routing.Path))
	if err := verifyRoutingPoststate(target, routing); err != nil {
		return fmt.Errorf("refuse to roll back routing tree: %w", err)
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if backup != "" {
		if err := restoreRoutingBackup(backup, target, os.Rename); err != nil {
			return err
		}
	}
	return nil
}

func discardRoutingBackup(backup string) error {
	if backup == "" {
		return nil
	}
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("remove routing backup: %w", err)
	}
	return nil
}

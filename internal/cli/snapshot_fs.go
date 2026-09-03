package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
)

var errSnapshotDrift = errors.New("rollback target changed after render")

const (
	maxSnapshotFileBytes = int64(8 << 20)
	maxSnapshotTreeBytes = int64(64 << 20)
	maxSnapshotFiles     = 10_000
	maxSnapshotEntries   = 20_000
)

type fileSnapshot struct {
	existed bool
	mode    os.FileMode
	content []byte
}

type directorySnapshot struct {
	existed     bool
	mode        fs.FileMode
	directories map[string]fs.FileMode
	files       map[string]fileSnapshot
}

type snapshotFileSystem interface {
	close() error
	lstat(string) (fs.FileInfo, error)
	open(string) (snapshotFile, error)
	walkDir(string, fs.WalkDirFunc) error
	reserveUnusedName(string, string) (string, error)
	mkdir(string, fs.FileMode) error
	chmod(string, fs.FileMode) error
	createExclusiveFile(string, fs.FileMode) (snapshotFile, error)
	remove(string) error
	removeAll(string) error
	rename(string, string) error
}

type snapshotFile interface {
	io.Reader
	io.Seeker
	Write([]byte) (int, error)
	Stat() (fs.FileInfo, error)
	Sync() error
	Chmod(fs.FileMode) error
	Close() error
}

type rootedSnapshotFileSystem struct {
	root *os.Root
}

var _ snapshotFileSystem = (*rootedSnapshotFileSystem)(nil)

func openSnapshotFileSystem(repoRoot string) (snapshotFileSystem, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("open repository filesystem root: %w", err)
	}
	return &rootedSnapshotFileSystem{root: root}, nil
}

func (fileSystem *rootedSnapshotFileSystem) close() error {
	return fileSystem.root.Close()
}

func (fileSystem *rootedSnapshotFileSystem) lstat(name string) (fs.FileInfo, error) {
	return fileSystem.root.Lstat(filepath.FromSlash(name))
}

func (fileSystem *rootedSnapshotFileSystem) open(name string) (snapshotFile, error) {
	return fileSystem.root.Open(filepath.FromSlash(name))
}

func (fileSystem *rootedSnapshotFileSystem) walkDir(name string, walk fs.WalkDirFunc) error {
	return fs.WalkDir(fileSystem.root.FS(), name, walk)
}

func (fileSystem *rootedSnapshotFileSystem) reserveUnusedName(parent, prefix string) (string, error) {
	for range 100 {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("generate rollback path: %w", err)
		}
		name := path.Join(parent, prefix+hex.EncodeToString(random[:]))
		file, err := fileSystem.root.OpenFile(filepath.FromSlash(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
		if err := fileSystem.root.Remove(filepath.FromSlash(name)); err != nil {
			return "", err
		}
		return name, nil
	}
	return "", fmt.Errorf("reserve rollback path: exhausted collision retries")
}

func (fileSystem *rootedSnapshotFileSystem) mkdir(name string, mode fs.FileMode) error {
	return fileSystem.root.Mkdir(filepath.FromSlash(name), mode)
}

func (fileSystem *rootedSnapshotFileSystem) chmod(name string, mode fs.FileMode) error {
	rootedName := filepath.FromSlash(name)
	before, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return err
	}
	if before.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("refuse to change permissions through snapshot symlink %s", name)
	}
	file, err := fileSystem.root.Open(rootedName)
	if err != nil {
		return err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(before, opened) {
		return fmt.Errorf("snapshot path %s changed before setting permissions", name)
	}
	if err := file.Chmod(mode); err != nil {
		return err
	}
	after, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return err
	}
	if after.Mode()&fs.ModeSymlink != 0 || !os.SameFile(opened, after) {
		return fmt.Errorf("snapshot path %s changed while setting permissions", name)
	}
	return nil
}

func (fileSystem *rootedSnapshotFileSystem) createExclusiveFile(name string, mode fs.FileMode) (snapshotFile, error) {
	return fileSystem.root.OpenFile(filepath.FromSlash(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
}

func (fileSystem *rootedSnapshotFileSystem) remove(name string) error {
	return fileSystem.root.Remove(filepath.FromSlash(name))
}

func (fileSystem *rootedSnapshotFileSystem) removeAll(name string) error {
	return fileSystem.root.RemoveAll(filepath.FromSlash(name))
}

func (fileSystem *rootedSnapshotFileSystem) rename(oldName, newName string) error {
	return fileSystem.root.Rename(filepath.FromSlash(oldName), filepath.FromSlash(newName))
}

func readStableRegularFile(
	fileSystem snapshotFileSystem,
	name string,
) ([]byte, fs.FileInfo, error) {
	before, err := fileSystem.lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if before.Mode()&fs.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("snapshot path %s is not a real regular file", name)
	}
	if before.Size() > maxSnapshotFileBytes {
		return nil, nil, fmt.Errorf("snapshot path %s exceeds the %d-byte file limit", name, maxSnapshotFileBytes)
	}
	file, err := fileSystem.open(name)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !sameSnapshotFileState(before, opened) {
		return nil, nil, fmt.Errorf("snapshot path %s changed while opening", name)
	}
	first, err := readBoundedSnapshot(file, name)
	if err != nil {
		return nil, nil, err
	}
	afterFirst, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, err
	}
	second, err := readBoundedSnapshot(file, name)
	if err != nil {
		return nil, nil, err
	}
	afterSecond, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	after, err := fileSystem.lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(first, second) ||
		!sameSnapshotFileState(opened, afterFirst) ||
		!sameSnapshotFileState(afterFirst, afterSecond) ||
		!sameSnapshotFileState(afterSecond, after) {
		return nil, nil, fmt.Errorf("snapshot path %s changed while reading", name)
	}
	return first, afterSecond, nil
}

func readBoundedSnapshot(file io.Reader, name string) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(file, maxSnapshotFileBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxSnapshotFileBytes {
		return nil, fmt.Errorf("snapshot path %s exceeds the %d-byte file limit", name, maxSnapshotFileBytes)
	}
	return content, nil
}

func sameSnapshotFileState(left, right fs.FileInfo) bool {
	return left != nil && right != nil &&
		os.SameFile(left, right) &&
		left.Mode() == right.Mode() &&
		left.Size() == right.Size() &&
		left.ModTime().Equal(right.ModTime())
}

func sameSnapshotDirectoryState(left, right fs.FileInfo) bool {
	return left != nil && right != nil &&
		os.SameFile(left, right) &&
		left.IsDir() && right.IsDir() &&
		left.Mode()&fs.ModeSymlink == 0 && right.Mode()&fs.ModeSymlink == 0 &&
		left.Mode() == right.Mode() &&
		left.ModTime().Equal(right.ModTime())
}

func captureDirectory(repoRoot, target string) (directorySnapshot, error) {
	fileSystem, err := openSnapshotFileSystem(repoRoot)
	if err != nil {
		return directorySnapshot{}, err
	}
	defer fileSystem.close()
	return captureDirectoryWithFS(fileSystem, target)
}

func captureDirectoryWithFS(
	fileSystem snapshotFileSystem,
	target string,
) (directorySnapshot, error) {
	first, err := captureDirectoryPass(fileSystem, target)
	if err != nil {
		return directorySnapshot{}, err
	}
	second, err := captureDirectoryPass(fileSystem, target)
	if err != nil {
		return directorySnapshot{}, err
	}
	if !equalDirectorySnapshots(first, second) {
		return directorySnapshot{}, fmt.Errorf("snapshot directory %s changed while reading", target)
	}
	return second, nil
}

func captureDirectoryPass(
	fileSystem snapshotFileSystem,
	target string,
) (directorySnapshot, error) {
	if err := validateSnapshotPath(target); err != nil {
		return directorySnapshot{}, err
	}
	info, err := fileSystem.lstat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return emptyDirectorySnapshot(), nil
	}
	if err != nil {
		return directorySnapshot{}, err
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return directorySnapshot{}, fmt.Errorf("snapshot target %s is not a real directory", target)
	}
	snapshot := directorySnapshot{
		existed: true, mode: info.Mode().Perm(),
		directories: make(map[string]fs.FileMode), files: make(map[string]fileSnapshot),
	}
	var totalBytes int64
	entryCount := 0
	err = fileSystem.walkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == target {
			return nil
		}
		entryCount++
		if entryCount > maxSnapshotEntries {
			return fmt.Errorf("snapshot directory %s exceeds the %d-entry limit", target, maxSnapshotEntries)
		}
		relative := strings.TrimPrefix(current, target+"/")
		if relative == current || relative == "" {
			return fmt.Errorf("snapshot walk returned path %s outside %s", current, target)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("snapshot path %s is a symlink", relative)
		}
		if entry.IsDir() {
			directoryInfo, err := fileSystem.lstat(current)
			if err != nil {
				return err
			}
			if directoryInfo.Mode()&fs.ModeSymlink != 0 || !directoryInfo.IsDir() {
				return fmt.Errorf("snapshot path %s changed while reading", relative)
			}
			snapshot.directories[relative] = directoryInfo.Mode().Perm()
			return nil
		}
		if len(snapshot.files) >= maxSnapshotFiles {
			return fmt.Errorf("snapshot directory %s exceeds the %d-file limit", target, maxSnapshotFiles)
		}
		content, opened, err := readStableRegularFile(fileSystem, current)
		if err != nil {
			return err
		}
		totalBytes += int64(len(content))
		if totalBytes > maxSnapshotTreeBytes {
			return fmt.Errorf("snapshot directory %s exceeds the %d-byte tree limit", target, maxSnapshotTreeBytes)
		}
		snapshot.files[relative] = fileSnapshot{
			existed: true, mode: opened.Mode().Perm(), content: content,
		}
		return nil
	})
	if err != nil {
		return directorySnapshot{}, err
	}
	after, err := fileSystem.lstat(target)
	if err != nil || !sameSnapshotDirectoryState(info, after) {
		return directorySnapshot{}, fmt.Errorf("snapshot directory %s changed while reading", target)
	}
	return snapshot, nil
}

type fileRollback struct {
	target          string
	before          fileSnapshot
	expectedCurrent fileSnapshot
}

type directoryRollback struct {
	target          string
	before          directorySnapshot
	expectedCurrent directorySnapshot
}

type quarantinedFile struct {
	target   string
	path     string
	snapshot fileSnapshot
}

type quarantinedDirectory struct {
	target   string
	path     string
	snapshot directorySnapshot
}

func restoreDirectoryIfCurrent(
	repoRoot, target string,
	before, expectedCurrent directorySnapshot,
) error {
	return restoreSnapshotsIfCurrent(
		repoRoot,
		nil,
		&directoryRollback{target: target, before: before, expectedCurrent: expectedCurrent},
	)
}

func restoreSnapshotsIfCurrent(
	repoRoot string,
	filePlan *fileRollback,
	directoryPlan *directoryRollback,
) error {
	fileSystem, err := openSnapshotFileSystem(repoRoot)
	if err != nil {
		return err
	}
	defer fileSystem.close()
	return restoreSnapshotsWithFS(fileSystem, filePlan, directoryPlan)
}

func restoreSnapshotsWithFS(
	fileSystem snapshotFileSystem,
	filePlan *fileRollback,
	directoryPlan *directoryRollback,
) error {
	var err error
	var currentFile quarantinedFile
	if filePlan != nil {
		currentFile, err = quarantineFile(fileSystem, filePlan.target, filePlan.expectedCurrent)
		if err != nil {
			return err
		}
	}
	var currentDirectory quarantinedDirectory
	if directoryPlan != nil {
		currentDirectory, err = quarantineDirectory(
			fileSystem,
			directoryPlan.target,
			directoryPlan.expectedCurrent,
		)
		if err != nil {
			restoreErr := restoreQuarantinedFile(fileSystem, currentFile)
			return errors.Join(err, restoreErr)
		}
	}
	var driftErr error
	if filePlan != nil && !equalFileSnapshots(currentFile.snapshot, filePlan.expectedCurrent) {
		driftErr = errors.Join(driftErr, fmt.Errorf("%w: file %s changed after render", errSnapshotDrift, filePlan.target))
	}
	if directoryPlan != nil && !equalDirectorySnapshots(currentDirectory.snapshot, directoryPlan.expectedCurrent) {
		driftErr = errors.Join(driftErr, fmt.Errorf("%w: directory %s changed after render", errSnapshotDrift, directoryPlan.target))
	}
	if driftErr != nil {
		restoreErr := errors.Join(
			restoreQuarantinedDirectory(fileSystem, currentDirectory),
			restoreQuarantinedFile(fileSystem, currentFile),
		)
		return errors.Join(driftErr, restoreErr)
	}
	if directoryPlan != nil && directoryPlan.before.existed {
		if err := writeDirectorySnapshotExclusive(fileSystem, directoryPlan.target, directoryPlan.before); err != nil {
			restoreErr := errors.Join(
				restoreQuarantinedDirectory(fileSystem, currentDirectory),
				restoreQuarantinedFile(fileSystem, currentFile),
			)
			return errors.Join(fmt.Errorf("restore previous routing directory: %w", err), restoreErr)
		}
	}
	if filePlan != nil && filePlan.before.existed {
		if err := writeFileSnapshotExclusive(fileSystem, filePlan.target, filePlan.before); err != nil {
			return errors.Join(
				fmt.Errorf("restore previous file: %w", err),
				fmt.Errorf("rendered projection remains quarantined at %s and %s", currentFile.path, currentDirectory.path),
			)
		}
	}
	if err := cleanupQuarantinedDirectory(fileSystem, currentDirectory); err != nil {
		return err
	}
	return cleanupQuarantinedFile(fileSystem, currentFile)
}

func quarantineFile(
	fileSystem snapshotFileSystem,
	target string,
	expectedCurrent fileSnapshot,
) (quarantinedFile, error) {
	result := quarantinedFile{target: target, snapshot: fileSnapshot{}}
	if err := validateSnapshotPath(target); err != nil {
		return result, err
	}
	if !expectedCurrent.existed {
		if _, err := fileSystem.lstat(target); err == nil {
			return result, fmt.Errorf("%w: file %s appeared", errSnapshotDrift, target)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return result, err
		}
		return result, nil
	}
	info, err := fileSystem.lstat(target)
	if err != nil || info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return result, fmt.Errorf("%w: file %s is not the rendered regular file", errSnapshotDrift, target)
	}
	result.path, err = fileSystem.reserveUnusedName(path.Dir(target), ".ssb-review-agents-")
	if err != nil {
		return result, fmt.Errorf("reserve file rollback quarantine: %w", err)
	}
	if err := fileSystem.rename(target, result.path); err != nil {
		return result, fmt.Errorf("quarantine file rollback target: %w", err)
	}
	result.snapshot, err = captureFileWithFS(fileSystem, result.path)
	if err != nil {
		validationErr := fmt.Errorf(
			"validate quarantined file: %w; quarantined candidate retained at %s",
			err,
			result.path,
		)
		if restoreErr := writeFileSnapshotExclusive(fileSystem, target, expectedCurrent); restoreErr != nil {
			return result, errors.Join(
				validationErr,
				fmt.Errorf("restore expected file after quarantine validation failure: %w", restoreErr),
			)
		}
		return result, fmt.Errorf("%w; restored expected file at %s", validationErr, target)
	}
	return result, nil
}

func quarantineDirectory(
	fileSystem snapshotFileSystem,
	target string,
	expectedCurrent directorySnapshot,
) (quarantinedDirectory, error) {
	result := quarantinedDirectory{target: target, snapshot: emptyDirectorySnapshot()}
	if err := validateSnapshotPath(target); err != nil {
		return result, err
	}
	if !expectedCurrent.existed {
		if _, err := fileSystem.lstat(target); err == nil {
			return result, fmt.Errorf("%w: directory %s appeared", errSnapshotDrift, target)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return result, err
		}
		return result, nil
	}
	info, err := fileSystem.lstat(target)
	if err != nil || info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return result, fmt.Errorf("%w: directory %s is not the rendered directory", errSnapshotDrift, target)
	}
	result.path, err = fileSystem.reserveUnusedName(path.Dir(target), ".ssb-review-routing-")
	if err != nil {
		return result, fmt.Errorf("reserve routing rollback quarantine: %w", err)
	}
	if err := fileSystem.rename(target, result.path); err != nil {
		return result, fmt.Errorf("quarantine routing rollback target: %w", err)
	}
	result.snapshot, err = captureDirectoryWithFS(fileSystem, result.path)
	if err != nil {
		validationErr := fmt.Errorf(
			"validate quarantined directory: %w; quarantined candidate retained at %s",
			err,
			result.path,
		)
		if restoreErr := writeDirectorySnapshotExclusive(fileSystem, target, expectedCurrent); restoreErr != nil {
			return result, errors.Join(
				validationErr,
				fmt.Errorf("restore expected directory after quarantine validation failure: %w", restoreErr),
			)
		}
		return result, fmt.Errorf("%w; restored expected directory at %s", validationErr, target)
	}
	return result, nil
}

func restoreQuarantinedFile(fileSystem snapshotFileSystem, current quarantinedFile) error {
	if current.path == "" {
		return nil
	}
	if err := writeFileSnapshotExclusive(fileSystem, current.target, current.snapshot); err != nil {
		return fmt.Errorf("restore current file from %s: %w", current.path, err)
	}
	return cleanupQuarantinedFile(fileSystem, current)
}

func restoreQuarantinedDirectory(fileSystem snapshotFileSystem, current quarantinedDirectory) error {
	if current.path == "" {
		return nil
	}
	if err := writeDirectorySnapshotExclusive(fileSystem, current.target, current.snapshot); err != nil {
		return fmt.Errorf("restore current directory from %s: %w", current.path, err)
	}
	return cleanupQuarantinedDirectory(fileSystem, current)
}

func cleanupQuarantinedFile(fileSystem snapshotFileSystem, current quarantinedFile) error {
	if current.path == "" {
		return nil
	}
	if err := fileSystem.remove(current.path); err != nil {
		return fmt.Errorf("remove quarantined file %s: %w", current.path, err)
	}
	return nil
}

func cleanupQuarantinedDirectory(fileSystem snapshotFileSystem, current quarantinedDirectory) error {
	if current.path == "" {
		return nil
	}
	if err := fileSystem.removeAll(current.path); err != nil {
		return fmt.Errorf("remove quarantined directory %s: %w", current.path, err)
	}
	return nil
}

func writeDirectorySnapshotExclusive(
	fileSystem snapshotFileSystem,
	target string,
	snapshot directorySnapshot,
) error {
	if !snapshot.existed {
		return fmt.Errorf("cannot write an absent directory snapshot")
	}
	if err := fileSystem.mkdir(target, 0o700); err != nil {
		return err
	}
	directories := make([]string, 0, len(snapshot.directories))
	for relative := range snapshot.directories {
		directories = append(directories, relative)
	}
	sort.Slice(directories, func(i, j int) bool {
		leftDepth := strings.Count(directories[i], "/")
		rightDepth := strings.Count(directories[j], "/")
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return directories[i] < directories[j]
	})
	for _, relative := range directories {
		directoryTarget := path.Join(target, relative)
		if err := fileSystem.mkdir(directoryTarget, 0o700); err != nil {
			return err
		}
	}
	files := make([]string, 0, len(snapshot.files))
	for relative := range snapshot.files {
		files = append(files, relative)
	}
	sort.Strings(files)
	for _, relative := range files {
		if err := writeFileSnapshotExclusive(fileSystem, path.Join(target, relative), snapshot.files[relative]); err != nil {
			return err
		}
	}
	for index := len(directories) - 1; index >= 0; index-- {
		relative := directories[index]
		if err := fileSystem.chmod(path.Join(target, relative), snapshot.directories[relative]); err != nil {
			return err
		}
	}
	if err := fileSystem.chmod(target, snapshot.mode); err != nil {
		return err
	}
	return nil
}

func captureFile(repoRoot, target string) (fileSnapshot, error) {
	fileSystem, err := openSnapshotFileSystem(repoRoot)
	if err != nil {
		return fileSnapshot{}, err
	}
	defer fileSystem.close()
	return captureFileWithFS(fileSystem, target)
}

func captureFileWithFS(fileSystem snapshotFileSystem, target string) (fileSnapshot, error) {
	if err := validateSnapshotPath(target); err != nil {
		return fileSnapshot{}, err
	}
	content, info, err := readStableRegularFile(fileSystem, target)
	if errors.Is(err, fs.ErrNotExist) {
		return fileSnapshot{}, nil
	}
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{existed: true, mode: info.Mode().Perm(), content: content}, nil
}

func projectedFileSnapshot(before fileSnapshot, content []byte) fileSnapshot {
	if before.existed && bytes.Equal(before.content, content) {
		return before
	}
	if !before.existed && len(content) == 0 {
		return before
	}
	mode := fs.FileMode(0o644)
	if before.existed {
		mode = before.mode
	}
	mode = materializedFileMode(mode)
	return fileSnapshot{existed: true, mode: mode, content: append([]byte(nil), content...)}
}

func projectedRoutingSnapshot(
	before directorySnapshot,
	routing *render.RoutingResult,
) directorySnapshot {
	if routing == nil || !routing.Changed {
		return before
	}
	if !routing.Exists {
		return emptyDirectorySnapshot()
	}
	snapshot := directorySnapshot{
		existed:     true,
		mode:        materializedDirectoryMode(0o700),
		directories: map[string]fs.FileMode{},
		files:       make(map[string]fileSnapshot, len(routing.Files)),
	}
	prefix := render.RoutingDirectory + "/"
	for _, file := range routing.Files {
		relative := strings.TrimPrefix(file.Path, prefix)
		for parent := path.Dir(relative); parent != "."; parent = path.Dir(parent) {
			snapshot.directories[parent] = materializedDirectoryMode(0o755)
		}
		snapshot.files[relative] = fileSnapshot{
			existed: true, mode: materializedFileMode(0o644), content: append([]byte(nil), file.Content...),
		}
	}
	return snapshot
}

func materializedFileMode(mode fs.FileMode) fs.FileMode {
	if runtime.GOOS != "windows" {
		return mode.Perm()
	}
	if mode.Perm()&0o200 == 0 {
		return 0o444
	}
	return 0o666
}

func materializedDirectoryMode(mode fs.FileMode) fs.FileMode {
	if runtime.GOOS != "windows" {
		return mode.Perm()
	}
	if mode.Perm()&0o200 == 0 {
		return 0o555
	}
	return 0o777
}

func restoreFileIfCurrent(repoRoot, target string, before, expectedCurrent fileSnapshot) error {
	return restoreSnapshotsIfCurrent(
		repoRoot,
		&fileRollback{target: target, before: before, expectedCurrent: expectedCurrent},
		nil,
	)
}

func writeFileSnapshotExclusive(
	fileSystem snapshotFileSystem,
	target string,
	snapshot fileSnapshot,
) (returnErr error) {
	if !snapshot.existed {
		return fmt.Errorf("cannot write an absent file snapshot")
	}
	file, err := fileSystem.createExclusiveFile(target, snapshot.mode)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); returnErr == nil && err != nil {
			returnErr = err
		}
	}()
	if err := file.Chmod(snapshot.mode); err != nil {
		return err
	}
	if written, err := file.Write(snapshot.content); err != nil {
		return err
	} else if written != len(snapshot.content) {
		return io.ErrShortWrite
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return nil
}

func equalFileSnapshots(left, right fileSnapshot) bool {
	return left.existed == right.existed &&
		(!left.existed || left.mode == right.mode && bytes.Equal(left.content, right.content))
}

func equalDirectorySnapshots(left, right directorySnapshot) bool {
	if left.existed != right.existed {
		return false
	}
	if !left.existed {
		return true
	}
	if left.mode != right.mode || len(left.directories) != len(right.directories) || len(left.files) != len(right.files) {
		return false
	}
	for relative, mode := range left.directories {
		if otherMode, exists := right.directories[relative]; !exists || mode != otherMode {
			return false
		}
	}
	for relative, file := range left.files {
		if other, exists := right.files[relative]; !exists || !equalFileSnapshots(file, other) {
			return false
		}
	}
	return true
}

func emptyDirectorySnapshot() directorySnapshot {
	return directorySnapshot{directories: make(map[string]fs.FileMode), files: make(map[string]fileSnapshot)}
}

func validateSnapshotPath(target string) error {
	if target == "" || path.IsAbs(target) || path.Clean(target) != target || target == ".." || strings.HasPrefix(target, "../") {
		return fmt.Errorf("snapshot path %q is outside the repository", target)
	}
	return nil
}

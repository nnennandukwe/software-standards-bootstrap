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
	"sort"
	"strings"
)

var errSnapshotDrift = errors.New("rollback target changed after render")

type fileSnapshot struct {
	existed bool
	mode    os.FileMode
	content []byte
}

type directorySnapshot struct {
	existed     bool
	directories map[string]struct{}
	files       map[string]fileSnapshot
}

type rootedSnapshotFileSystem struct {
	root       *os.Root
	beforeOpen func(string)
}

func openSnapshotFileSystem(repoRoot string) (*rootedSnapshotFileSystem, error) {
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

func (fileSystem *rootedSnapshotFileSystem) readRegularFile(name string) ([]byte, fs.FileInfo, error) {
	rootedName := filepath.FromSlash(name)
	before, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return nil, nil, err
	}
	if before.Mode()&fs.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("snapshot path %s is not a real regular file", name)
	}
	if fileSystem.beforeOpen != nil {
		fileSystem.beforeOpen(name)
	}
	file, err := fileSystem.root.Open(rootedName)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, nil, fmt.Errorf("snapshot path %s changed while opening", name)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}
	after, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return nil, nil, err
	}
	if after.Mode()&fs.ModeSymlink != 0 || !after.Mode().IsRegular() || !os.SameFile(opened, after) {
		return nil, nil, fmt.Errorf("snapshot path %s changed while reading", name)
	}
	return content, opened, nil
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

func captureDirectory(repoRoot, target string) (directorySnapshot, error) {
	fileSystem, err := openSnapshotFileSystem(repoRoot)
	if err != nil {
		return directorySnapshot{}, err
	}
	defer fileSystem.close()
	return captureDirectoryWithFS(fileSystem, target)
}

func captureDirectoryWithFS(
	fileSystem *rootedSnapshotFileSystem,
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
		existed: true, directories: make(map[string]struct{}), files: make(map[string]fileSnapshot),
	}
	err = fileSystem.walkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == target {
			return nil
		}
		relative := strings.TrimPrefix(current, target+"/")
		if relative == current || relative == "" {
			return fmt.Errorf("snapshot walk returned path %s outside %s", current, target)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("snapshot path %s is a symlink", relative)
		}
		if entry.IsDir() {
			snapshot.directories[relative] = struct{}{}
			return nil
		}
		content, opened, err := fileSystem.readRegularFile(current)
		if err != nil {
			return err
		}
		snapshot.files[relative] = fileSnapshot{
			existed: true, mode: opened.Mode().Perm(), content: content,
		}
		return nil
	})
	if err != nil {
		return directorySnapshot{}, err
	}
	return snapshot, nil
}

func restoreDirectoryIfCurrent(
	repoRoot, target string,
	before, expectedCurrent directorySnapshot,
) error {
	fileSystem, err := openSnapshotFileSystem(repoRoot)
	if err != nil {
		return err
	}
	defer fileSystem.close()
	if err := validateSnapshotPath(target); err != nil {
		return err
	}
	if !expectedCurrent.existed {
		if _, err := fileSystem.lstat(target); err == nil {
			return fmt.Errorf("%w: directory %s appeared", errSnapshotDrift, target)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if before.existed {
			return writeDirectorySnapshotExclusive(fileSystem, target, before)
		}
		return nil
	}
	info, err := fileSystem.lstat(target)
	if err != nil {
		return fmt.Errorf("%w: inspect directory %s: %v", errSnapshotDrift, target, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: directory %s is not a real directory", errSnapshotDrift, target)
	}
	quarantine, err := fileSystem.reserveUnusedName(path.Dir(target), ".ssb-review-routing-")
	if err != nil {
		return fmt.Errorf("reserve routing rollback quarantine: %w", err)
	}
	if err := fileSystem.root.Rename(filepath.FromSlash(target), filepath.FromSlash(quarantine)); err != nil {
		return fmt.Errorf("quarantine routing rollback target: %w", err)
	}
	moved, captureErr := captureDirectoryWithFS(fileSystem, quarantine)
	if captureErr != nil {
		return fmt.Errorf("%w: validate quarantined directory: %v; current tree remains at %s", errSnapshotDrift, captureErr, quarantine)
	}
	if !equalDirectorySnapshots(moved, expectedCurrent) {
		restoreErr := writeDirectorySnapshotExclusive(fileSystem, target, moved)
		if restoreErr == nil {
			restoreErr = fileSystem.root.RemoveAll(filepath.FromSlash(quarantine))
		}
		if restoreErr != nil {
			return errors.Join(
				fmt.Errorf("%w: directory %s changed after render", errSnapshotDrift, target),
				fmt.Errorf("preserve changed directory from %s: %w", quarantine, restoreErr),
			)
		}
		return fmt.Errorf("%w: directory %s changed after render", errSnapshotDrift, target)
	}
	if before.existed {
		if err := writeDirectorySnapshotExclusive(fileSystem, target, before); err != nil {
			return fmt.Errorf("restore previous directory; rendered tree remains at %s: %w", quarantine, err)
		}
	} else if _, err := fileSystem.lstat(target); err == nil {
		return fmt.Errorf("%w: directory %s appeared during rollback; rendered tree remains at %s", errSnapshotDrift, target, quarantine)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := fileSystem.root.RemoveAll(filepath.FromSlash(quarantine)); err != nil {
		return fmt.Errorf("remove rendered directory after rollback: %w", err)
	}
	return nil
}

func writeDirectorySnapshotExclusive(
	fileSystem *rootedSnapshotFileSystem,
	target string,
	snapshot directorySnapshot,
) error {
	if !snapshot.existed {
		return fmt.Errorf("cannot write an absent directory snapshot")
	}
	if err := fileSystem.root.Mkdir(filepath.FromSlash(target), 0o755); err != nil {
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
		if err := fileSystem.root.Mkdir(filepath.FromSlash(path.Join(target, relative)), 0o755); err != nil {
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

func captureFileWithFS(fileSystem *rootedSnapshotFileSystem, target string) (fileSnapshot, error) {
	if err := validateSnapshotPath(target); err != nil {
		return fileSnapshot{}, err
	}
	content, info, err := fileSystem.readRegularFile(target)
	if errors.Is(err, fs.ErrNotExist) {
		return fileSnapshot{}, nil
	}
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{existed: true, mode: info.Mode().Perm(), content: content}, nil
}

func restoreFileIfCurrent(repoRoot, target string, before, expectedCurrent fileSnapshot) error {
	fileSystem, err := openSnapshotFileSystem(repoRoot)
	if err != nil {
		return err
	}
	defer fileSystem.close()
	if err := validateSnapshotPath(target); err != nil {
		return err
	}
	if !expectedCurrent.existed {
		if _, err := fileSystem.lstat(target); err == nil {
			return fmt.Errorf("%w: file %s appeared", errSnapshotDrift, target)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if before.existed {
			return writeFileSnapshotExclusive(fileSystem, target, before)
		}
		return nil
	}
	info, err := fileSystem.lstat(target)
	if err != nil {
		return fmt.Errorf("%w: inspect file %s: %v", errSnapshotDrift, target, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: file %s is not a real regular file", errSnapshotDrift, target)
	}
	quarantine, err := fileSystem.reserveUnusedName(path.Dir(target), ".ssb-review-agents-")
	if err != nil {
		return fmt.Errorf("reserve file rollback quarantine: %w", err)
	}
	if err := fileSystem.root.Rename(filepath.FromSlash(target), filepath.FromSlash(quarantine)); err != nil {
		return fmt.Errorf("quarantine file rollback target: %w", err)
	}
	moved, captureErr := captureFileWithFS(fileSystem, quarantine)
	if captureErr != nil {
		return fmt.Errorf("%w: validate quarantined file: %v; current file remains at %s", errSnapshotDrift, captureErr, quarantine)
	}
	if !equalFileSnapshots(moved, expectedCurrent) {
		restoreErr := writeFileSnapshotExclusive(fileSystem, target, moved)
		if restoreErr == nil {
			restoreErr = fileSystem.root.Remove(filepath.FromSlash(quarantine))
		}
		if restoreErr != nil {
			return errors.Join(
				fmt.Errorf("%w: file %s changed after render", errSnapshotDrift, target),
				fmt.Errorf("preserve changed file from %s: %w", quarantine, restoreErr),
			)
		}
		return fmt.Errorf("%w: file %s changed after render", errSnapshotDrift, target)
	}
	if before.existed {
		if err := writeFileSnapshotExclusive(fileSystem, target, before); err != nil {
			return fmt.Errorf("restore previous file; rendered file remains at %s: %w", quarantine, err)
		}
	} else if _, err := fileSystem.lstat(target); err == nil {
		return fmt.Errorf("%w: file %s appeared during rollback; rendered file remains at %s", errSnapshotDrift, target, quarantine)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := fileSystem.root.Remove(filepath.FromSlash(quarantine)); err != nil {
		return fmt.Errorf("remove rendered file after rollback: %w", err)
	}
	return nil
}

func writeFileSnapshotExclusive(
	fileSystem *rootedSnapshotFileSystem,
	target string,
	snapshot fileSnapshot,
) (returnErr error) {
	if !snapshot.existed {
		return fmt.Errorf("cannot write an absent file snapshot")
	}
	file, err := fileSystem.root.OpenFile(
		filepath.FromSlash(target),
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		snapshot.mode,
	)
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
	if len(left.directories) != len(right.directories) || len(left.files) != len(right.files) {
		return false
	}
	for relative := range left.directories {
		if _, exists := right.directories[relative]; !exists {
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
	return directorySnapshot{directories: make(map[string]struct{}), files: make(map[string]fileSnapshot)}
}

func validateSnapshotPath(target string) error {
	if target == "" || path.IsAbs(target) || path.Clean(target) != target || target == ".." || strings.HasPrefix(target, "../") {
		return fmt.Errorf("snapshot path %q is outside the repository", target)
	}
	return nil
}

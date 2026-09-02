package render

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type routingFileSystem interface {
	close() error
	lstat(string) (fs.FileInfo, error)
	readFile(string, int64) ([]byte, error)
	walkDir(string, fs.WalkDirFunc) error
	makeTempDir(string, string) (string, error)
	mkdir(string, fs.FileMode) error
	mkdirAll(string, fs.FileMode) error
	chmod(string, fs.FileMode) error
	createExclusiveFile(string, fs.FileMode) (routingFile, error)
	remove(string) error
	removeAll(string) error
	rename(string, string) error
}

type routingFile interface {
	Write([]byte) (int, error)
	Sync() error
	Chmod(fs.FileMode) error
	Close() error
}

type rootedRoutingFileSystem struct {
	root *os.Root
}

var _ routingFileSystem = (*rootedRoutingFileSystem)(nil)

func openRoutingFileSystem(repoRoot string) (routingFileSystem, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("open repository filesystem root: %w", err)
	}
	return &rootedRoutingFileSystem{root: root}, nil
}

func (fileSystem *rootedRoutingFileSystem) close() error {
	return fileSystem.root.Close()
}

func (fileSystem *rootedRoutingFileSystem) lstat(name string) (fs.FileInfo, error) {
	return fileSystem.root.Lstat(filepath.FromSlash(name))
}

func (fileSystem *rootedRoutingFileSystem) readFile(name string, maxBytes int64) ([]byte, error) {
	rootedName := filepath.FromSlash(name)
	before, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return nil, err
	}
	if before.Mode()&fs.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: routing path %s is not a regular file", ErrUnsafeTarget, name)
	}
	if before.Size() > maxBytes {
		return nil, fmt.Errorf("%w: routing path %s exceeds the %d-byte file limit", ErrUnsafeTarget, name, maxBytes)
	}
	file, err := fileSystem.root.Open(rootedName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !sameRoutingFileState(before, opened) {
		return nil, fmt.Errorf("%w: routing path %s changed while opening", ErrUnsafeTarget, name)
	}
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("%w: routing path %s exceeds the %d-byte file limit", ErrUnsafeTarget, name, maxBytes)
	}
	afterFirst, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	second, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(second)) > maxBytes {
		return nil, fmt.Errorf("%w: routing path %s exceeds the %d-byte file limit", ErrUnsafeTarget, name, maxBytes)
	}
	afterSecond, err := file.Stat()
	if err != nil {
		return nil, err
	}
	after, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(content, second) ||
		!sameRoutingFileState(opened, afterFirst) ||
		!sameRoutingFileState(afterFirst, afterSecond) ||
		!sameRoutingFileState(afterSecond, after) {
		return nil, fmt.Errorf("%w: routing path %s changed while reading", ErrUnsafeTarget, name)
	}
	return content, nil
}

func sameRoutingFileState(left, right fs.FileInfo) bool {
	return left != nil && right != nil &&
		os.SameFile(left, right) &&
		left.Mode() == right.Mode() &&
		left.Size() == right.Size() &&
		left.ModTime().Equal(right.ModTime())
}

func (fileSystem *rootedRoutingFileSystem) walkDir(name string, walk fs.WalkDirFunc) error {
	return fs.WalkDir(fileSystem.root.FS(), filepath.ToSlash(name), walk)
}

func (fileSystem *rootedRoutingFileSystem) makeTempDir(parent, prefix string) (string, error) {
	for range 100 {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("generate temporary routing name: %w", err)
		}
		name := filepath.ToSlash(filepath.Join(parent, prefix+hex.EncodeToString(random[:])))
		if err := fileSystem.mkdir(name, 0o700); err == nil {
			if err := fileSystem.chmod(name, 0o700); err != nil {
				return "", err
			}
			return name, nil
		} else if !os.IsExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("reserve temporary routing directory: exhausted collision retries")
}

func (fileSystem *rootedRoutingFileSystem) mkdir(name string, mode fs.FileMode) error {
	return fileSystem.root.Mkdir(filepath.FromSlash(name), mode)
}

func (fileSystem *rootedRoutingFileSystem) mkdirAll(name string, mode fs.FileMode) error {
	return fileSystem.root.MkdirAll(filepath.FromSlash(name), mode)
}

func (fileSystem *rootedRoutingFileSystem) chmod(name string, mode fs.FileMode) error {
	rootedName := filepath.FromSlash(name)
	before, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return err
	}
	if before.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%w: refuse to change routing permissions through symlink %s", ErrUnsafeTarget, name)
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
		return fmt.Errorf("%w: routing path %s changed before setting permissions", ErrUnsafeTarget, name)
	}
	if err := file.Chmod(mode); err != nil {
		return err
	}
	after, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return err
	}
	if after.Mode()&fs.ModeSymlink != 0 || !os.SameFile(opened, after) {
		return fmt.Errorf("%w: routing path %s changed while setting permissions", ErrUnsafeTarget, name)
	}
	return nil
}

func (fileSystem *rootedRoutingFileSystem) createExclusiveFile(name string, mode fs.FileMode) (routingFile, error) {
	return fileSystem.root.OpenFile(filepath.FromSlash(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
}

func (fileSystem *rootedRoutingFileSystem) remove(name string) error {
	return fileSystem.root.Remove(filepath.FromSlash(name))
}

func (fileSystem *rootedRoutingFileSystem) removeAll(name string) error {
	return fileSystem.root.RemoveAll(filepath.FromSlash(name))
}

func (fileSystem *rootedRoutingFileSystem) rename(oldName, newName string) error {
	return fileSystem.root.Rename(filepath.FromSlash(oldName), filepath.FromSlash(newName))
}

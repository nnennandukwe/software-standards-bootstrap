package render

import (
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
	readFile(string) ([]byte, error)
	walkDir(string, fs.WalkDirFunc) error
	makeTempDir(string, string) (string, error)
	mkdir(string, fs.FileMode) error
	mkdirAll(string, fs.FileMode) error
	createExclusiveFile(string, fs.FileMode) (routingFile, error)
	remove(string) error
	removeAll(string) error
	rename(string, string) error
}

type routingFile interface {
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

type rootedRoutingFileSystem struct {
	root *os.Root
}

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

func (fileSystem *rootedRoutingFileSystem) readFile(name string) ([]byte, error) {
	rootedName := filepath.FromSlash(name)
	before, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return nil, err
	}
	if before.Mode()&fs.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: routing path %s is not a regular file", ErrUnsafeTarget, name)
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
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("%w: routing path %s changed while opening", ErrUnsafeTarget, name)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	after, err := fileSystem.root.Lstat(rootedName)
	if err != nil {
		return nil, err
	}
	if after.Mode()&fs.ModeSymlink != 0 || !after.Mode().IsRegular() || !os.SameFile(opened, after) {
		return nil, fmt.Errorf("%w: routing path %s changed while reading", ErrUnsafeTarget, name)
	}
	return content, nil
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

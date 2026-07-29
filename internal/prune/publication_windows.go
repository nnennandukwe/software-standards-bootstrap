//go:build windows

package prune

import (
	"os"

	"golang.org/x/sys/windows"
)

func publishApplicationFileDurably(source, target string) error {
	sourcePath, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	targetPath, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(
		sourcePath,
		targetPath,
		windows.MOVEFILE_WRITE_THROUGH,
	)
}

func removeApplicationPublicationFileDurably(filePath string) error {
	return os.Remove(filePath)
}

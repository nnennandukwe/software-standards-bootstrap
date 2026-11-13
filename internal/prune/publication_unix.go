//go:build !windows

package prune

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type directorySyncOps struct{ sync func(string) error }

func defaultApplicationPublicationOps() applicationPublicationOps {
	durability := directorySyncOps{sync: syncDirectory}
	return applicationPublicationOps{exclusive: defaultExclusiveFileOps(), publish: durability.publish, removePublication: durability.remove}
}

func (durability directorySyncOps) publish(source, target string) error {
	if err := os.Link(source, target); err != nil {
		return err
	}
	if err := durability.sync(filepath.Dir(target)); err != nil {
		return fmt.Errorf("sync application directory after publication: %w", err)
	}
	return nil
}

func (durability directorySyncOps) remove(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		return err
	}
	if err := durability.sync(filepath.Dir(filePath)); err != nil {
		return fmt.Errorf("sync application directory after claim cleanup: %w", err)
	}
	return nil
}

func syncDirectory(directory string) (returnErr error) {
	handle, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, handle.Close())
	}()
	if err := handle.Sync(); err != nil {
		return err
	}
	return nil
}

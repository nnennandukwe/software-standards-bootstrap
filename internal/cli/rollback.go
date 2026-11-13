package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func missingDirectories(root, target string) ([]string, error) {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("directory escapes repository")
	}
	current := root
	missing := make([]string, 0)
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			missing = append(missing, current)
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", current)
		}
	}
	return missing, nil
}

func removeEmptyDirectories(directories []string) error {
	for index := len(directories) - 1; index >= 0; index-- {
		if err := os.Remove(directories[index]); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

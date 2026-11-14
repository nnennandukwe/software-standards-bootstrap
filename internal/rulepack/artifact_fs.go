package rulepack

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
)

func readManifestArtifact(root string, manifest AcceptedArtifact) ([]byte, []Diagnostic, error) {
	return readDigestBoundFile(
		root,
		FileReference{Path: manifest.Path, SHA256: manifest.SHA256},
		manifest.Kind,
		inventory.DefaultLimits().MaxFileBytes,
	)
}

func readDigestBoundFile(
	root string,
	reference FileReference,
	label string,
	maxBytes int64,
) ([]byte, []Diagnostic, error) {
	if component, found, err := findSymlinkComponent(root, reference.Path); err != nil {
		return nil, nil, err
	} else if found {
		return nil, []Diagnostic{diagnostic(
			reference.Path,
			"file",
			reference.Path+" must be a real regular file, not a symlink; contains symlink component "+component,
			"place the digest-bound artifact inside the repository",
		)}, nil
	}
	data, diagnostics, err := readRequiredRegularFileLimited(
		filepath.Join(root, filepath.FromSlash(reference.Path)),
		reference.Path,
		maxBytes,
	)
	if err != nil || len(data) == 0 {
		return data, diagnostics, err
	}
	actual := digest(data)
	if reference.SHA256 != actual {
		diagnostics = append(diagnostics, diagnostic(
			reference.Path,
			"sha256",
			label+" SHA-256 does not match manifest; expected "+actual,
			"restore the reviewed bytes or update the manifest digest through explicit pack review",
		))
	}
	return data, diagnostics, nil
}

func readRequiredRegularFileLimited(absolute, relative string, maxBytes int64) ([]byte, []Diagnostic, error) {
	data, diagnostics, err := readRegularFileLimited(absolute, relative, maxBytes)
	if err != nil || len(diagnostics) != 0 {
		return data, diagnostics, err
	}
	if len(data) == 0 {
		return data, []Diagnostic{diagnostic(
			relative,
			"file",
			relative+" must not be empty",
			"write the required artifact and rerun ssb validate",
		)}, nil
	}
	return data, nil, nil
}

func readRegularFileLimited(absolute, relative string, maxBytes int64) ([]byte, []Diagnostic, error) {
	info, err := os.Lstat(absolute)
	if errors.Is(err, os.ErrNotExist) {
		return nil, []Diagnostic{diagnostic(
			relative,
			"file",
			relative+" does not exist",
			"create the required file and rerun ssb validate",
		)}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("inspect %s: %w", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, []Diagnostic{diagnostic(
			relative,
			"file",
			relative+" must be a real regular file, not a symlink",
			"replace it with a file inside the repository",
		)}, nil
	}
	if info.Size() > maxBytes {
		return nil, []Diagnostic{diagnostic(
			relative,
			"file",
			fmt.Sprintf("%s is larger than %d bytes", relative, maxBytes),
			"reduce the artifact to the documented size ceiling and rerun ssb validate",
		)}, nil
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", relative, err)
	}
	return data, nil, nil
}

func readEmbeddedArtifact(
	root string,
	manifest AcceptedArtifact,
) ([]byte, []Diagnostic, error) {
	if component, found, err := findSymlinkComponent(root, manifest.Path); err != nil {
		return nil, nil, err
	} else if found {
		return nil, []Diagnostic{diagnostic(
			manifest.Path,
			"file",
			manifest.Path+" contains a symlink component "+component,
			"place the accepted artifact inside the repository",
		)}, nil
	}
	data, diagnostics, err := readRequiredRegularFile(
		filepath.Join(root, filepath.FromSlash(manifest.Path)),
		manifest.Path,
	)
	if len(data) == 0 && len(diagnostics) != 0 {
		for index := range diagnostics {
			diagnostics[index].Recovery = "remove its manifest entry or restore the artifact at the canonical path"
		}
	}
	return data, diagnostics, err
}

func findSymlinkComponent(root, relative string) (string, bool, error) {
	current := root
	for _, component := range strings.Split(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		if err != nil {
			return "", false, fmt.Errorf("inspect %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return filepath.ToSlash(current), true, nil
		}
	}
	return "", false, nil
}

func unlistedNativeArtifacts(
	root string,
	listed map[string]AcceptedArtifact,
) ([]string, []Diagnostic, error) {
	directories := []struct {
		relative string
		suffix   string
	}{
		{relative: ".software-standards/rules", suffix: ".md"},
		{relative: ".software-standards/verification", suffix: ".yaml"},
		{relative: ".software-standards/automation", suffix: ".yaml"},
	}
	unlisted := make([]string, 0)
	diagnostics := make([]Diagnostic, 0)
	for _, directory := range directories {
		absolute := filepath.Join(root, filepath.FromSlash(directory.relative))
		info, err := os.Lstat(absolute)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("inspect artifact directory %s: %w", directory.relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			diagnostics = append(diagnostics, diagnostic(
				directory.relative,
				"file",
				"artifact directory "+directory.relative+" must be a real directory, not a symlink",
				"replace it with a directory inside the repository",
			))
			continue
		}
		entries, err := os.ReadDir(absolute)
		if err != nil {
			return nil, nil, fmt.Errorf("read artifact directory %s: %w", directory.relative, err)
		}
		for _, entry := range entries {
			relative := path.Join(directory.relative, entry.Name())
			if entry.IsDir() {
				diagnostics = append(diagnostics, diagnostic(
					relative,
					"file",
					"native artifact directories must be flat",
					"move the artifact to the top-level canonical directory or remove the nested directory",
				))
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return nil, nil, fmt.Errorf("inspect artifact path %s: %w", relative, err)
			}
			if entry.Type()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				diagnostics = append(diagnostics, diagnostic(
					relative,
					"file",
					"native artifact paths must be real regular files",
					"replace the path with a canonical regular artifact file or remove it",
				))
				continue
			}
			if !strings.HasSuffix(entry.Name(), directory.suffix) {
				diagnostics = append(diagnostics, diagnostic(
					relative,
					"file",
					relative+" is not a supported artifact path",
					"use the canonical "+directory.suffix+" extension at the top level or remove the file",
				))
				continue
			}
			if _, exists := listed[relative]; !exists {
				unlisted = append(unlisted, relative)
			}
		}
	}
	sort.Strings(unlisted)
	return unlisted, diagnostics, nil
}

func readRequiredRegularFile(absolute, relative string) ([]byte, []Diagnostic, error) {
	info, err := os.Lstat(absolute)
	if errors.Is(err, os.ErrNotExist) {
		return nil, []Diagnostic{diagnostic(
			relative,
			"file",
			relative+" does not exist",
			"create the required file and rerun ssb validate",
		)}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("inspect %s: %w", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, []Diagnostic{diagnostic(
			relative,
			"file",
			relative+" must be a real regular file, not a symlink",
			"replace it with a file inside the repository",
		)}, nil
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", relative, err)
	}
	if len(data) == 0 {
		return data, []Diagnostic{diagnostic(
			relative,
			"file",
			relative+" must not be empty",
			"write the required artifact and rerun ssb validate",
		)}, nil
	}
	return data, nil, nil
}

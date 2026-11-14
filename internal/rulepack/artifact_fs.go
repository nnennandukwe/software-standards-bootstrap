package rulepack

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
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
	file, err := os.Open(absolute)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", relative, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("inspect opened %s: %w", relative, err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, []Diagnostic{diagnostic(relative, "file", "retained file changed before reading", "preserve edits and retry after the file is stable")}, nil
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", relative, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, []Diagnostic{diagnostic(
			relative, "file", fmt.Sprintf("%s is larger than %d bytes", relative, maxBytes),
			"reduce the artifact to the documented size ceiling and rerun ssb validate",
		)}, nil
	}
	current, err := os.Lstat(absolute)
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(opened, current) {
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("recheck %s: %w", relative, err)
		}
		return nil, []Diagnostic{diagnostic(relative, "file", "retained file changed while reading", "preserve edits and retry after the file is stable")}, nil
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

func unlistedNativeArtifacts(ctx context.Context, root string, listed map[string]AcceptedArtifact) ([]string, []Diagnostic, error) {
	return unlistedNativeArtifactsLimited(ctx, root, listed, inventory.DefaultLimits())
}

func unlistedNativeArtifactsLimited(ctx context.Context, root string, listed map[string]AcceptedArtifact, limits inventory.Limits) ([]string, []Diagnostic, error) {
	anchor, err := os.OpenRoot(root)
	if err != nil {
		return nil, nil, err
	}
	defer anchor.Close()
	directories := []struct{ relative, suffix string }{
		{relative: ".software-standards/rules", suffix: ".md"},
		{relative: ".software-standards/verification", suffix: ".yaml"},
		{relative: ".software-standards/automation", suffix: ".yaml"},
	}
	unlisted := make([]string, 0)
	diagnostics := make([]Diagnostic, 0)
	var entriesSeen int
	var nameBytes int64
	for _, directory := range directories {
		expected, err := anchor.Lstat(filepath.FromSlash(directory.relative))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("inspect artifact directory %s: %w", directory.relative, err)
		}
		if expected.Mode()&os.ModeSymlink != 0 || !expected.IsDir() {
			diagnostics = append(diagnostics, diagnostic(directory.relative, "file", "artifact directory "+directory.relative+" must be a real directory, not a symlink", "replace it with a directory inside the repository"))
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		store, err := anchor.Open(filepath.FromSlash(directory.relative))
		if err != nil {
			return nil, nil, fmt.Errorf("read artifact directory %s: %w", directory.relative, err)
		}
		defer store.Close()
		opened, err := store.Stat()
		if err != nil {
			return nil, nil, err
		}
		if !opened.IsDir() || !os.SameFile(expected, opened) {
			return nil, nil, fmt.Errorf("%w: native artifact directory changed during discovery", workspace.ErrPrecondition)
		}
		diagnosticStart := len(diagnostics)
		for {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
			entries, readErr := store.ReadDir(256)
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				return nil, nil, fmt.Errorf("read artifact directory %s: %w", directory.relative, readErr)
			}
			entriesSeen += len(entries)
			if entriesSeen > limits.MaxCandidateFiles {
				return nil, []Diagnostic{diagnostic(".software-standards", "files", "native discovery exceeds the candidate-file limit", "preserve the pack and review its entry count before continuation")}, nil
			}
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return nil, nil, err
				}
				relative := path.Join(directory.relative, entry.Name())
				nameBytes += int64(len(fmt.Sprintf("%q", relative)))
				if nameBytes > limits.MaxFileBytes {
					return nil, []Diagnostic{diagnostic(".software-standards", "files", "native discovery exceeds the path-name byte limit", "preserve the pack and review its path names before continuation")}, nil
				}
				if entry.IsDir() {
					diagnostics = append(diagnostics, diagnostic(relative, "file", "native artifact directories must be flat", "move the artifact to the top-level canonical directory or remove the nested directory"))
					continue
				}
				info, err := anchor.Lstat(filepath.FromSlash(relative))
				if err != nil {
					return nil, nil, fmt.Errorf("inspect artifact path %s: %w", relative, err)
				}
				if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
					diagnostics = append(diagnostics, diagnostic(relative, "file", "native artifact paths must be real regular files", "replace the path with a canonical regular artifact file or remove it"))
					continue
				}
				if !strings.HasSuffix(entry.Name(), directory.suffix) {
					diagnostics = append(diagnostics, diagnostic(relative, "file", relative+" is not a supported artifact path", "use the canonical "+directory.suffix+" extension at the top level or remove the file"))
					continue
				}
				if _, exists := listed[relative]; !exists {
					unlisted = append(unlisted, relative)
				}
			}
			if errors.Is(readErr, io.EOF) {
				break
			}
		}
		current, err := anchor.Lstat(filepath.FromSlash(directory.relative))
		if err != nil {
			return nil, nil, err
		}
		if !current.IsDir() || !os.SameFile(opened, current) {
			return nil, nil, fmt.Errorf("%w: native artifact directory changed during discovery", workspace.ErrPrecondition)
		}
		part := diagnostics[diagnosticStart:]
		sort.SliceStable(part, func(i, j int) bool {
			if part[i].Path != part[j].Path {
				return part[i].Path < part[j].Path
			}
			return part[i].Message < part[j].Message
		})
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

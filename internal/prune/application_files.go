package prune

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	prestateClaim    = "pre"
	poststateClaim   = "post"
	publicationClaim = "publication"
)

func (publication applicationPublicationOps) executeOperations(repoRoot string, operations []operation) (map[string]struct{}, error) {
	completed := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if err := safeApplicationTarget(repoRoot, operation.Path); err != nil {
			return completed, err
		}
		target, err := resolvePortablePath(repoRoot, operation.Path)
		if err != nil {
			return completed, err
		}
		completed[operation.Path] = struct{}{}
		switch operation.Kind {
		case "remove":
			claim, err := claimExpectedFile(target, operation.ExpectedSHA256, prestateClaim)
			if err != nil {
				return completed, fmt.Errorf("claim %s before removal: %w", operation.Path, err)
			}
			if err := os.Remove(claim); err != nil {
				return completed, fmt.Errorf("remove claimed %s: %w", operation.Path, err)
			}
		case "write":
			var err error
			if operation.ExpectedAbsent {
				err = publication.writeNewExclusive(target, operation.Content, operation.Mode)
			} else {
				var claim string
				claim, err = claimExpectedFile(target, operation.ExpectedSHA256, prestateClaim)
				if err == nil {
					err = publication.writeNewExclusive(target, operation.Content, operation.Mode)
				}
				if err == nil {
					err = os.Remove(claim)
				} else if claim != "" {
					if restoreErr := restoreClaimedFile(claim, target); restoreErr != nil {
						err = fmt.Errorf("%w; preserve claimed source: %v", err, restoreErr)
					}
				}
			}
			if err != nil {
				return completed, fmt.Errorf("write %s: %w", operation.Path, err)
			}
		default:
			return completed, fmt.Errorf("unsupported application operation %s", operation.Kind)
		}
	}
	for _, operation := range operations {
		target, err := resolvePortablePath(repoRoot, operation.Path)
		if err != nil {
			return completed, err
		}
		if operation.Kind == "remove" {
			if _, err := os.Lstat(target); err == nil {
				return completed, fmt.Errorf("target %s reappeared during application", operation.Path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return completed, fmt.Errorf("verify removed target %s: %w", operation.Path, err)
			}
			continue
		}
		if err := requireCurrentDigest(target, operation.Poststate.SHA256); err != nil {
			return completed, fmt.Errorf("verify written target %s: %w", operation.Path, err)
		}
	}
	return completed, nil
}

func claimExpectedFile(target, expectedDigest, phase string) (string, error) {
	claim := claimPathForTarget(target, phase)
	if _, err := os.Lstat(claim); err == nil {
		return "", fmt.Errorf("claim path already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Rename(target, claim); err != nil {
		return "", err
	}
	if err := requireCurrentDigest(claim, expectedDigest); err != nil {
		if restoreErr := restoreClaimedFile(claim, target); restoreErr != nil {
			return claim, fmt.Errorf("%w; restore claimed file: %v", err, restoreErr)
		}
		return "", err
	}
	return claim, nil
}

func claimPathForTarget(target, phase string) string {
	return filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".ssb-prune-"+phase+"-claim")
}

func restoreClaimedFile(claim, target string) error {
	if err := os.Link(claim, target); err != nil {
		return err
	}
	return os.Remove(claim)
}

func requireCurrentDigest(filePath, expected string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if digestBytes(data) != expected {
		return fmt.Errorf("current bytes no longer match the approved source digest")
	}
	return nil
}

func missingApplicationDirectories(repoRoot string, operations []operation) ([]string, error) {
	missing := make(map[string]struct{})
	for _, operation := range operations {
		target, err := resolvePortablePath(repoRoot, operation.Path)
		if err != nil {
			return nil, err
		}
		parent := filepath.Dir(target)
		for parent != repoRoot {
			info, err := os.Lstat(parent)
			if err == nil {
				if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					return nil, fmt.Errorf("application parent %s is unsafe", parent)
				}
				break
			}
			if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			relative, err := filepath.Rel(repoRoot, parent)
			if err != nil {
				return nil, err
			}
			missing[filepath.ToSlash(relative)] = struct{}{}
			parent = filepath.Dir(parent)
		}
	}
	result := make([]string, 0, len(missing))
	for relative := range missing {
		result = append(result, relative)
	}
	sort.Strings(result)
	return result, nil
}

func removeApplicationDirectories(repoRoot string, directories []string) error {
	ordered := append([]string(nil), directories...)
	sort.Slice(ordered, func(i, j int) bool {
		return strings.Count(ordered[i], "/") > strings.Count(ordered[j], "/")
	})
	for _, relative := range ordered {
		target, err := resolvePortablePath(repoRoot, relative)
		if err != nil {
			return fmt.Errorf("unsafe application directory %s: %w", relative, err)
		}
		if err := os.Remove(target); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func fileDigest(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func (publication applicationPublicationOps) writeNewExclusive(
	target string,
	content []byte,
	mode os.FileMode,
) (returnErr error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	publicationPath := claimPathForTarget(target, publicationClaim)
	file, err := publication.exclusive.open(
		publicationPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		mode,
	)
	if err != nil {
		return fmt.Errorf("create application publication claim: %w", err)
	}
	defer func() {
		if err := publication.removePublication(publicationPath); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("remove application publication claim %s: %w", publicationPath, err),
			)
		}
	}()
	if err := writeDurableExclusiveWithCleanup(
		publicationPath,
		file,
		content,
		mode,
		func() error { return publication.exclusive.remove(publicationPath) },
	); err != nil {
		return err
	}
	if err := publication.publish(publicationPath, target); err != nil {
		return fmt.Errorf("publish application file without overwrite: %w", err)
	}
	return nil
}

func safeApplicationTarget(repoRoot, relative string) error {
	if !validApplicationPath(relative) {
		return fmt.Errorf("application target %s is not a canonical actionable artifact or report path", relative)
	}
	target, err := resolvePortablePath(repoRoot, relative)
	if err != nil {
		return err
	}
	current := repoRoot
	parent := filepath.Dir(target)
	relativeParent, err := filepath.Rel(repoRoot, parent)
	if err != nil {
		return fmt.Errorf("resolve application target parent %s: %w", relative, err)
	}
	for _, component := range strings.Split(filepath.ToSlash(relativeParent), "/") {
		if component == "." {
			continue
		}
		current = filepath.Join(current, filepath.FromSlash(component))
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect application target %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("application target %s has an unsafe parent component", relative)
		}
	}
	return nil
}

func validApplicationPath(relative string) bool {
	if relative == actionableReportPath || relative == actionableManifestPath {
		return true
	}
	if _, _, ok := artifactIdentity(relative); ok {
		return true
	}
	if !safeRelativePath(relative) || !strings.HasPrefix(relative, ".agents/skills/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(relative, ".agents/skills/"), "/")
	return len(parts) >= 2 && stableIDPattern.MatchString(parts[0])
}

type exclusiveFileOps struct {
	open   func(string, int, os.FileMode) (durableExclusiveFile, error)
	remove func(string) error
}

func defaultExclusiveFileOps() exclusiveFileOps {
	return exclusiveFileOps{
		open: func(name string, flags int, mode os.FileMode) (durableExclusiveFile, error) {
			return os.OpenFile(name, flags, mode)
		},
		remove: os.Remove,
	}
}

// applicationPublicationOps follows one publication and its rollback/recovery.
type applicationPublicationOps struct {
	exclusive         exclusiveFileOps
	publish           func(string, string) error
	removePublication func(string) error
}

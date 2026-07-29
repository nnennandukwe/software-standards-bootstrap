package prune

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// safeRelativePath accepts only one portable slash-separated path spelling.
// Backslashes and colons are rejected even on POSIX so data validated there
// cannot acquire Windows drive, UNC, or traversal semantics later.
func safeRelativePath(value string) bool {
	if value == "" ||
		strings.ContainsAny(value, "\\:\x00") ||
		path.IsAbs(value) ||
		path.Clean(value) != value ||
		value == "." ||
		value == ".." ||
		strings.HasPrefix(value, "../") {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if !portablePathComponent(component) {
			return false
		}
	}
	return true
}

func portablePathComponent(component string) bool {
	if component == "" ||
		strings.HasSuffix(component, ".") ||
		strings.HasSuffix(component, " ") ||
		strings.ContainsAny(component, `<>:"\|?*`) {
		return false
	}
	for _, character := range component {
		if character < 32 {
			return false
		}
	}
	base := component
	if dot := strings.IndexByte(base, '.'); dot >= 0 {
		base = base[:dot]
	}
	switch strings.ToUpper(base) {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"COM¹", "COM²", "COM³",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		"LPT¹", "LPT²", "LPT³":
		return false
	default:
		return true
	}
}

func validateGovernedTreePaths(paths []string) error {
	seen := make(map[string]string, len(paths))
	skillEntrypoints := make(map[string]bool)
	skillsWithFiles := make(map[string]bool)
	for _, itemPath := range paths {
		if !safeRelativePath(itemPath) {
			return fmt.Errorf("non-portable governed path %q", itemPath)
		}
		key := portablePathKey(itemPath)
		if prior, exists := seen[key]; exists {
			return fmt.Errorf(
				"governed paths %q and %q collide on a case-insensitive filesystem",
				prior,
				itemPath,
			)
		}
		seen[key] = itemPath
		switch {
		case strings.HasPrefix(itemPath, ".software-standards/rules/"):
			if kind, _, ok := artifactIdentity(itemPath); !ok || kind != ArtifactRule {
				return fmt.Errorf("non-canonical governed rule path %q", itemPath)
			}
		case strings.HasPrefix(itemPath, ".agents/skills/"):
			remainder := strings.TrimPrefix(itemPath, ".agents/skills/")
			parts := strings.Split(remainder, "/")
			if len(parts) < 2 || !stableIDPattern.MatchString(parts[0]) {
				return fmt.Errorf("non-canonical governed skill path %q", itemPath)
			}
			skillsWithFiles[parts[0]] = true
			if len(parts) == 2 && parts[1] == "SKILL.md" {
				skillEntrypoints[parts[0]] = true
			}
		default:
			return fmt.Errorf("path %q is outside the governed roots", itemPath)
		}
	}
	for skillID := range skillsWithFiles {
		if !skillEntrypoints[skillID] {
			return fmt.Errorf("governed skill %s has no canonical SKILL.md", skillID)
		}
	}
	return nil
}

// resolvePortablePath converts a validated portable path beneath root and
// verifies the native result remains contained before any filesystem access.
func resolvePortablePath(root, relative string) (string, error) {
	if !safeRelativePath(relative) {
		return "", fmt.Errorf("unsafe portable relative path %q", relative)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve path root: %w", err)
	}
	target := filepath.Join(absoluteRoot, filepath.FromSlash(relative))
	contained, err := filepath.Rel(absoluteRoot, target)
	if err != nil {
		return "", fmt.Errorf("resolve path %q beneath root: %w", relative, err)
	}
	if contained == "." || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("portable path %q escapes its root", relative)
	}
	return target, nil
}

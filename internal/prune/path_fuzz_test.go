package prune

import (
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzPortableGovernedPath(f *testing.F) {
	for _, seed := range []string{
		".software-standards/rules/keep-rule.md", ".agents/skills/example/SKILL.md",
		".agents/skills/example/references/guide.md", "", "../escape", "/absolute",
		`C:\escape`, `\\server\share`, "a/../b", "a//b", "a/NUL.txt", "a/COM¹", "a/trailing.", "a/space ", "a/\x00b",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		accepted := safeRelativePath(value)
		if accepted != safeRelativePath(value) {
			t.Fatal("path validation isn't deterministic")
		}
		if !accepted {
			return
		}
		if value == "" || value == "." || value == ".." || path.IsAbs(value) || path.Clean(value) != value || strings.HasPrefix(value, "../") || strings.ContainsAny(value, "\\:\x00") {
			t.Fatalf("accepted unsafe path %q", value)
		}
		for _, component := range strings.Split(value, "/") {
			if !portablePathComponent(component) {
				t.Fatalf("accepted non-portable component %q", component)
			}
		}
		// Use a fixed root to check native containment without per-input I/O.
		root, err := filepath.Abs(filepath.Join(string(filepath.Separator), "ssb-fuzz-root"))
		if err != nil {
			t.Fatal(err)
		}
		target, err := resolvePortablePath(root, value)
		if err != nil {
			t.Fatalf("accepted path can't resolve: %v", err)
		}
		relative, err := filepath.Rel(root, target)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			t.Fatalf("accepted path escaped root: %q, %v", target, err)
		}
		if safeRelativePath("../"+value) || safeRelativePath(value+"/../escape") || safeRelativePath(value+"\\escape") {
			t.Fatal("unsafe mutation of accepted path was allowed")
		}
	})
}

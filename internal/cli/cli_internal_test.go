package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRoutingSnapshotRestoresExistingAndAbsentTrees(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, ".software-standards", "routing")
		writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "before catalog\n")
		writeSnapshotTestFile(t, filepath.Join(target, "bundles", "route-one.md"), "before bundle\n")
		snapshot, err := captureDirectory(target)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(target); err != nil {
			t.Fatal(err)
		}
		writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "after\n")
		if err := restoreDirectory(target, snapshot); err != nil {
			t.Fatal(err)
		}
		assertSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "before catalog\n")
		assertSnapshotTestFile(t, filepath.Join(target, "bundles", "route-one.md"), "before bundle\n")
	})

	t.Run("absent", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), ".software-standards", "routing")
		snapshot, err := captureDirectory(target)
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "generated\n")
		if err := restoreDirectory(target, snapshot); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("absent snapshot left directory behind: %v", err)
		}
	})
}

func writeSnapshotTestFile(t *testing.T, target, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertSnapshotTestFile(t *testing.T, target, want string) {
	t.Helper()
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("%s = %q, want %q", target, content, want)
	}
}

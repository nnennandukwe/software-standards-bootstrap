package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestADRRollbackRemovesDirectoriesCreatedByFailedTransition(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "docs", "adr")
	missing, err := missingDirectories(root, targetDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(targetDir, "0001-agentic-rules.md")
	if err := os.WriteFile(record, []byte("partial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(record); err != nil {
		t.Fatal(err)
	}
	if err := removeEmptyDirectories(missing); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs")); !os.IsNotExist(err) {
		t.Fatalf("rollback left created directory tree: %v", err)
	}
}

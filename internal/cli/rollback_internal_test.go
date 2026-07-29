package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
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

func TestCommittedTransitionNeverRollsBackItsArtifact(t *testing.T) {
	rolledBack := false
	err := reconcileTransitionCompletion(
		prune.Event{EventDigest: "sha256:committed"},
		errors.New("injected lock cleanup failure"),
		"render",
		"AGENTS.md was restored",
		func() error {
			rolledBack = true
			return nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "lock cleanup failure") {
		t.Fatalf("error = %v, want committed cleanup failure", err)
	}
	if rolledBack {
		t.Fatal("committed transition rolled back its artifact")
	}
}

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/adr"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
)

func TestWriteADRErrorClassifiesNoAdoptableArtifactsAsRecoverable(t *testing.T) {
	var stderr bytes.Buffer
	if code := writeADRError(&stderr, adr.ErrNoAdoptableArtifacts); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "retain a semantic rule, verification recipe, or Agent Skill") {
		t.Fatalf("missing recovery guidance: %q", stderr.String())
	}
}

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
	record := filepath.Join(targetDir, "0001-actionable-standards.md")
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

func TestCommittedTransitionCleanupUsesRecoverableExitCode(t *testing.T) {
	completionErr := fmt.Errorf(
		"%w: transition event was recorded; run ssb prune recover --review review-one --clear-stale-lock",
		prune.ErrPrecondition,
	)
	err := reconcileTransitionCompletion(
		prune.Event{EventDigest: "sha256:committed"},
		completionErr,
		"render",
		"AGENTS.md was restored",
		func() error {
			t.Fatal("committed transition attempted rollback")
			return nil
		},
	)
	var stderr bytes.Buffer
	if exitCode := writePruneError(&stderr, err); exitCode != 2 {
		t.Fatalf("exit code = %d, want recoverable exit code 2", exitCode)
	}
	if output := stderr.String(); !strings.Contains(output, "--clear-stale-lock") {
		t.Fatalf("stderr = %q, want exact recovery guidance", output)
	}
}

package adr_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/adr"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestCreatePreservesConventionAndNeverOverwrites(t *testing.T) {
	repo := committedRepository(t)
	writeFile(t, filepath.Join(repo, "docs", "adrs", "0007-existing.md"), "# Existing\n")
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := testPack(ws.Baseline())

	dryRun, err := adr.Create(context.Background(), ws, pack, adr.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.Path != "docs/adrs/0008-actionable-standards.md" {
		t.Fatalf("dry-run path = %q", dryRun.Path)
	}
	if _, err := os.Lstat(filepath.Join(repo, dryRun.Path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run wrote an ADR: %v", err)
	}

	first, err := adr.Create(context.Background(), ws, pack, adr.Options{})
	if err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(repo, filepath.FromSlash(first.Path))
	firstBytes, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adr.Create(context.Background(), ws, pack, adr.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Path != "docs/adrs/0009-actionable-standards.md" {
		t.Fatalf("second path = %q", second.Path)
	}
	after, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(firstBytes) {
		t.Fatal("second ADR creation overwrote the first")
	}
}

func TestCreateRequiresDirectoryChoiceWhenAmbiguous(t *testing.T) {
	repo := committedRepository(t)
	if err := os.MkdirAll(filepath.Join(repo, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "docs", "adrs"), 0o755); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adr.Create(context.Background(), ws, testPack(ws.Baseline()), adr.Options{})
	if !errors.Is(err, adr.ErrAmbiguousDirectory) {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
	matches, globErr := filepath.Glob(filepath.Join(repo, "docs", "*", "*-actionable-standards.md"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("ambiguous ADR selection wrote files: %#v", matches)
	}
}

func TestCreateRejectsTraversalAndSymlinkEscapes(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := testPack(ws.Baseline())

	_, err = adr.Create(context.Background(), ws, pack, adr.Options{Directory: "../outside"})
	if !errors.Is(err, adr.ErrUnsafeTarget) {
		t.Fatalf("expected traversal rejection, got %v", err)
	}

	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "docs", "adr")); err != nil {
		t.Fatal(err)
	}
	_, err = adr.Create(context.Background(), ws, pack, adr.Options{})
	if !errors.Is(err, adr.ErrUnsafeTarget) {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
	entries, readErr := os.ReadDir(outside)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatal("ADR creation followed directory symlink")
	}
}

func TestCreateRejectsADRDirectoryInsideSubmodule(t *testing.T) {
	child := committedRepository(t)
	repo := committedRepository(t)
	git(t, repo, "-c", "protocol.file.allow=always", "submodule", "add", child, "docs")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "docs submodule")

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adr.Create(context.Background(), ws, testPack(ws.Baseline()), adr.Options{Directory: "docs/adr"})
	if !errors.Is(err, adr.ErrUnsafeTarget) || !strings.Contains(err.Error(), "submodule") {
		t.Fatalf("expected submodule target rejection, got %v", err)
	}
}

func TestCreateWriteFailureLeavesNoPartialADR(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows directory permission semantics do not provide this failure injection")
	}
	repo := committedRepository(t)
	directory := filepath.Join(repo, "docs", "adr")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directory, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(directory, 0o755); err != nil {
			t.Errorf("restore permissions: %v", err)
		}
	}()

	_, err = adr.Create(context.Background(), ws, testPack(ws.Baseline()), adr.Options{})
	if err == nil {
		t.Fatal("expected ADR write failure")
	}
	matches, globErr := filepath.Glob(filepath.Join(directory, "*-actionable-standards.md"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("failed ADR write left partial output: %#v", matches)
	}
}

func testPack(baseline string) rulepack.Pack {
	const id = "keep-rule"
	const sourcePath = ".software-standards/rules/keep-rule.md"
	return rulepack.Pack{
		BaselineCommit: baseline,
		ReportPath:     ".software-standards/report.md",
		Report: rulepack.Report{
			Schema:         rulepack.ReportSchema,
			BaselineCommit: baseline,
			Artifacts: []rulepack.AcceptedArtifact{{
				ID: id, Kind: "rule", Path: sourcePath, Confidence: "high",
				Utility: rulepack.Utility{Method: rulepack.UtilityMethod, Total: 70},
			}},
		},
		Rules: []rulepack.Rule{{
			Schema:     rulepack.RuleSchema,
			ID:         id,
			Title:      "Keep rule",
			Category:   "correctness",
			Lenses:     []rulepack.Lens{{Kind: "base"}},
			Directive:  "always",
			Scopes:     []string{"src/**"},
			Derivation: "extracted",
			Evidence: []rulepack.Evidence{{
				Role: "declares", Path: "README.md", Lines: "1-1",
			}},
			SourcePath: sourcePath,
			Body:       "Keep this exact body.\n",
		}},
	}
}

func committedRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	writeFile(t, filepath.Join(dir, "README.md"), "fixture\n")
	git(t, dir, "add", "README.md")
	git(t, dir, "commit", "-m", "baseline")
	return dir
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := append([]string{"-c", "user.name=SSB Test", "-c", "user.email=ssb@example.invalid", "-C", dir}, args...)
	cmd := exec.Command("git", command...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

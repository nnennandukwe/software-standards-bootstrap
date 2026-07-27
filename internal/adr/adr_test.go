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

func TestCreateDefaultsToConciseProposedADRWithOnlySurvivingArtifacts(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := testPack(ws.Baseline(), "keep-rule", "Keep this exact body.", "keep-skill")

	dryRun, err := adr.Create(context.Background(), ws, pack, adr.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.Path != "docs/adr/0001-agentic-rules.md" ||
		!strings.Contains(string(dryRun.Content), "Status: Proposed") ||
		!strings.Contains(string(dryRun.Content), "Keep this exact body.") ||
		!strings.Contains(string(dryRun.Content), "- Primary topic: `correctness`") ||
		!strings.Contains(string(dryRun.Content), "  - Primary topic: `compatibility`") ||
		!strings.Contains(string(dryRun.Content), "keep-skill") ||
		strings.Contains(string(dryRun.Content), "deleted-rule") {
		t.Fatalf("unexpected ADR:\n%s", dryRun.Content)
	}
	if _, err := os.Lstat(filepath.Join(repo, "docs")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created directories: %v", err)
	}

	created, err := adr.Create(context.Background(), ws, pack, adr.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Created {
		t.Fatal("expected ADR creation")
	}
	firstPath := filepath.Join(repo, filepath.FromSlash(created.Path))
	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}

	second, err := adr.Create(context.Background(), ws, pack, adr.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Path != "docs/adr/0002-agentic-rules.md" {
		t.Fatalf("second ADR path = %q", second.Path)
	}
	after, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(first) {
		t.Fatal("second ADR creation overwrote the existing ADR")
	}
}

func TestCreateRecordsRuleV2ActivationDirectiveAndProofCoverage(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := rulepack.Pack{
		BaselineCommit: ws.Baseline(),
		AssessmentPath: ".software-standards/assessment.md",
		Rules: []rulepack.Rule{{
			Schema:         rulepack.SchemaVersionV2,
			ID:             "review-cobra-command",
			Title:          "Review Cobra commands",
			Topic:          "maintainability",
			Lenses:         []rulepack.Lens{{Kind: "language", Value: "go"}, {Kind: "framework", Value: "cobra"}, {Kind: "task", Value: "review"}},
			Directive:      "prefer",
			Scopes:         []string{"cmd/**"},
			Classification: "guidance",
			Importance:     "medium",
			Score:          rulepack.Score{Method: rulepack.ScoreMethod, Total: 50},
			Confidence:     "medium",
			SourcePath:     ".software-standards/rules/review-cobra-command.md",
			Body:           "Review the command family together.\n",
			Verification: rulepack.Verification{
				Command:  "go test ./...",
				Coverage: "partial",
				Proves:   "The retained command assertions when the command passes.",
			},
		}},
	}

	result, err := adr.Create(context.Background(), ws, pack, adr.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, required := range []string{
		"- Lenses: `language:go`, `framework:cobra`, `task:review`",
		"- Directive: `prefer`",
		"- Verification coverage: `partial`",
		"- Proves when the mapped command passes: The retained command assertions when the command passes.",
		"- Existing verification: `go test ./...` (mapped, not executed)",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("ADR missing %q:\n%s", required, content)
		}
	}
}

func TestCreatePreservesRuleV1MappedProofSemantics(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := testPack(ws.Baseline(), "verify-before-merge", "Run the retained verification command.", "")
	pack.Rules[0].Classification = "deterministic"
	pack.Rules[0].Verification = rulepack.Verification{Command: "go test ./..."}

	result, err := adr.Create(context.Background(), ws, pack, adr.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	if !strings.Contains(content, "- Existing verification: `go test ./...` (mapped, not executed)") ||
		strings.Contains(content, "- Lenses:") ||
		strings.Contains(content, "- Directive:") ||
		strings.Contains(content, "- Verification coverage:") ||
		strings.Contains(content, "- Proves when the mapped command passes:") {
		t.Fatalf("v1 ADR changed legacy proof semantics:\n%s", content)
	}
}

func TestCreateTreatsWhitespaceVerificationCommandAsAbsent(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := testPack(ws.Baseline(), "review-change", "Review the change semantically.", "")
	pack.Rules[0].Schema = rulepack.SchemaVersionV2
	pack.Rules[0].Lenses = []rulepack.Lens{{Kind: "task", Value: "review"}}
	pack.Rules[0].Directive = "prefer"
	pack.Rules[0].Verification = rulepack.Verification{
		Command:  " \t ",
		ProofGap: "No repository check proves semantic review quality.",
	}

	result, err := adr.Create(context.Background(), ws, pack, adr.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	if !strings.Contains(content, "- Proof gap: No repository check proves semantic review quality.") ||
		strings.Contains(content, "- Existing verification:") ||
		strings.Contains(content, "- Verification coverage:") ||
		strings.Contains(content, "- Proves when the mapped command passes:") {
		t.Fatalf("ADR treated whitespace-only command as mapped proof:\n%s", content)
	}
}

func TestCreatePreservesExistingConventionAndRequiresChoiceWhenAmbiguous(t *testing.T) {
	t.Run("existing convention", func(t *testing.T) {
		repo := committedRepository(t)
		writeFile(t, filepath.Join(repo, "docs", "adrs", "0007-existing.md"), "# Existing\n")
		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		result, err := adr.Create(context.Background(), ws, testPack(ws.Baseline(), "rule", "Body.", ""), adr.Options{DryRun: true})
		if err != nil {
			t.Fatal(err)
		}
		if result.Path != "docs/adrs/0008-agentic-rules.md" {
			t.Fatalf("path = %q", result.Path)
		}
	})

	t.Run("ambiguous", func(t *testing.T) {
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
		_, err = adr.Create(context.Background(), ws, testPack(ws.Baseline(), "rule", "Body.", ""), adr.Options{})
		if !errors.Is(err, adr.ErrAmbiguousDirectory) {
			t.Fatalf("expected ambiguity error, got %v", err)
		}
		matches, globErr := filepath.Glob(filepath.Join(repo, "docs", "*", "*-agentic-rules.md"))
		if globErr != nil {
			t.Fatal(globErr)
		}
		if len(matches) != 0 {
			t.Fatalf("ambiguous ADR selection wrote files: %#v", matches)
		}
	})
}

func TestCreateRejectsTraversalAndSymlinkEscapes(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := testPack(ws.Baseline(), "rule", "Body.", "")

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
	_, err = adr.Create(context.Background(), ws, testPack(ws.Baseline(), "rule", "Body.", ""), adr.Options{Directory: "docs/adr"})
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

	_, err = adr.Create(context.Background(), ws, testPack(ws.Baseline(), "rule", "Body.", ""), adr.Options{})
	if err == nil {
		t.Fatal("expected ADR write failure")
	}
	matches, globErr := filepath.Glob(filepath.Join(directory, "*-agentic-rules.md"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("failed ADR write left partial output: %#v", matches)
	}
}

func testPack(baseline, ruleID, body, skillID string) rulepack.Pack {
	rule := rulepack.Rule{
		Schema:         rulepack.SchemaVersionV1,
		ID:             ruleID,
		Title:          "Retained rule",
		Topic:          "correctness",
		Scopes:         []string{"src/**"},
		Classification: "guidance",
		Importance:     "high",
		Score: rulepack.Score{
			Method: "ssb-score-v1",
			Total:  70,
		},
		Confidence:     "high",
		BaselineCommit: baseline,
		SourcePath:     ".software-standards/rules/" + ruleID + ".md",
		Body:           body + "\n",
		Evidence: []rulepack.Evidence{{
			Path:          "README.md",
			Lines:         "1-1",
			ExcerptSHA256: "sha256:" + strings.Repeat("0", 64),
		}},
		Verification: rulepack.Verification{ProofGap: "No existing checker."},
	}
	pack := rulepack.Pack{
		BaselineCommit: baseline,
		AssessmentPath: ".software-standards/assessment.md",
		Assessment:     "# Assessment\n",
		Rules:          []rulepack.Rule{rule},
	}
	if skillID != "" {
		pack.Rules[0].RelatedSkillIDs = []string{skillID}
		pack.Skills = []rulepack.Skill{{
			ID:          skillID,
			Description: "A retained procedural workflow.",
			Topic:       "compatibility",
			SourcePath:  ".agents/skills/" + skillID + "/SKILL.md",
			Body:        "# Skill\n",
		}}
	}
	return pack
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

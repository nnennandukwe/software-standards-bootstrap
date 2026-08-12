package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/cli"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"go.yaml.in/yaml/v4"
)

func TestInspectJSONIsReadOnlyAndMachineReadable(t *testing.T) {
	repo := committedRepository(t)
	writeFile(t, filepath.Join(repo, "untracked notes.txt"), "must remain untracked\n")
	before := git(t, repo, "status", "--porcelain=v1", "-z")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"inspect", "--repo", repo, "--format", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	var response struct {
		SchemaVersion  int    `json:"schema_version"`
		BaselineCommit string `json:"baseline_commit"`
		Files          []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
	}
	if response.SchemaVersion != 2 || response.BaselineCommit == "" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if len(response.Files) != 1 || response.Files[0].Path != "README.md" {
		t.Fatalf("unexpected files: %#v", response.Files)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	after := git(t, repo, "status", "--porcelain=v1", "-z")
	if after != before {
		t.Fatalf("inspect changed the repository:\nbefore %q\nafter  %q", before, after)
	}
}

func TestInspectTextGuidesManifestLayoutGeneration(t *testing.T) {
	repo := committedRepository(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"inspect", "--repo", repo, "--format", "text"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("inspect failed: exit=%d stderr=%q", code, stderr.String())
	}
	for _, required := range []string{"inventory.json", "human-facing artifacts", "manifest.yaml"} {
		if !strings.Contains(stdout.String(), required) {
			t.Fatalf("manifest-layout generation guidance missing %q:\n%s", required, stdout.String())
		}
	}
}

func TestInspectFailsClosedOnPartialCoverageUnlessExplicitlyAllowed(t *testing.T) {
	repo := committedRepository(t)
	writeFile(t, filepath.Join(repo, "SECOND.md"), "second\n")
	git(t, repo, "add", "SECOND.md")
	git(t, repo, "commit", "-m", "second candidate")

	var blockedOut bytes.Buffer
	var blockedErr bytes.Buffer
	code := cli.Run([]string{
		"inspect",
		"--repo", repo,
		"--format", "json",
		"--max-candidate-files", "1",
	}, &blockedOut, &blockedErr)
	if code != 4 {
		t.Fatalf("exit = %d, want 4; stdout=%q stderr=%q", code, blockedOut.String(), blockedErr.String())
	}
	if !strings.Contains(blockedErr.String(), "inventory coverage incomplete") ||
		!strings.Contains(blockedErr.String(), "--max-candidate-files") ||
		!strings.Contains(blockedErr.String(), "--allow-partial") {
		t.Fatalf("missing partial-coverage recovery: %q", blockedErr.String())
	}
	var blocked struct {
		Truncated               bool `json:"truncated"`
		ScannedFiles            int  `json:"scanned_files"`
		RemainingCandidateFiles int  `json:"remaining_candidate_files"`
	}
	if err := json.Unmarshal(blockedOut.Bytes(), &blocked); err != nil {
		t.Fatalf("invalid partial JSON %q: %v", blockedOut.String(), err)
	}
	if !blocked.Truncated || blocked.ScannedFiles != 1 || blocked.RemainingCandidateFiles != 1 {
		t.Fatalf("unexpected partial report: %#v", blocked)
	}

	var allowedOut bytes.Buffer
	var allowedErr bytes.Buffer
	code = cli.Run([]string{
		"inspect",
		"--repo", repo,
		"--format", "json",
		"--max-candidate-files", "1",
		"--allow-partial",
	}, &allowedOut, &allowedErr)
	if code != 0 {
		t.Fatalf("allowed partial exit = %d, want 0; stderr=%q", code, allowedErr.String())
	}
	if allowedOut.String() != blockedOut.String() {
		t.Fatalf("--allow-partial changed report:\nblocked %q\nallowed %q", blockedOut.String(), allowedOut.String())
	}
	if allowedErr.Len() != 0 {
		t.Fatalf("allowed partial wrote stderr: %q", allowedErr.String())
	}

	var textOut bytes.Buffer
	var textErr bytes.Buffer
	code = cli.Run([]string{
		"inspect",
		"--repo", repo,
		"--max-candidate-files", "1",
	}, &textOut, &textErr)
	if code != 4 {
		t.Fatalf("text partial exit = %d, want 4; stderr=%q", code, textErr.String())
	}
	if !strings.Contains(textOut.String(), "Coverage: TRUNCATED") {
		t.Fatalf("text report does not disclose truncation:\n%s", textOut.String())
	}
	if !strings.Contains(textErr.String(), "raise --max-candidate-files or --max-candidate-bytes") {
		t.Fatalf("text failure lacks blocked recovery:\n%s", textErr.String())
	}
	if strings.Contains(textOut.String(), "perform targeted semantic reads") {
		t.Fatalf("partial report advertised unsafe progression:\n%s", textOut.String())
	}
}

func TestInspectRejectsInvalidCandidateLimitsBeforeOpeningRepository(t *testing.T) {
	for _, test := range []struct {
		name  string
		flag  string
		value string
	}{
		{name: "zero files", flag: "--max-candidate-files", value: "0"},
		{name: "negative files", flag: "--max-candidate-files", value: "-1"},
		{name: "zero bytes", flag: "--max-candidate-bytes", value: "0"},
		{name: "negative bytes", flag: "--max-candidate-bytes", value: "-1"},
		{name: "overflow files", flag: "--max-candidate-files", value: "9223372036854775808"},
		{name: "overflow bytes", flag: "--max-candidate-bytes", value: "9223372036854775808"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := cli.Run([]string{
				"inspect",
				"--repo", filepath.Join(t.TempDir(), "does-not-exist"),
				test.flag, test.value,
			}, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("exit = %d, want 2; stderr=%q", code, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("unexpected stdout: %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), test.flag) {
				t.Fatalf("error does not name %s: %q", test.flag, stderr.String())
			}
			if strings.Contains(stderr.String(), "not a Git worktree") {
				t.Fatalf("repository was opened before flag validation: %q", stderr.String())
			}
		})
	}
}

func TestInspectReportsDirtyRecoveryAndUsageErrors(t *testing.T) {
	repo := committedRepository(t)
	writeFile(t, filepath.Join(repo, "README.md"), "changed\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"inspect", "--repo", repo}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "commit, stash, or restore tracked changes") ||
		!strings.Contains(stderr.String(), "ssb inspect --repo") {
		t.Fatalf("error lacks executable recovery guidance: %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"inspect", "--format", "xml"}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "--format must be text or json") {
		t.Fatalf("invalid format response: exit=%d stderr=%q", code, stderr.String())
	}
}

func TestInspectNeverExecutesTrackedRepositoryCode(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	marker := filepath.Join(repo, "executed.txt")
	script := filepath.Join(repo, "danger.sh")
	writeFile(t, script, "#!/bin/sh\nprintf executed > executed.txt\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "danger.sh")
	git(t, repo, "commit", "-m", "executable fixture")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"inspect", "--repo", repo}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect failed: exit=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Lstat(marker); !os.IsNotExist(err) {
		t.Fatalf("inspect executed repository code: %v", err)
	}
}

func TestHelpDocumentsOnlyTheCanonicalCommandForms(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"--help"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("help failed: exit=%d stderr=%q", code, stderr.String())
	}
	for _, form := range []string{
		"ssb inspect  [--repo PATH] [--format text|json]",
		"ssb validate [--repo PATH] [--format text|json]",
		"ssb render   [--repo PATH] [--review ID] [--dry-run]",
		"ssb adr      [--repo PATH] [--review ID] [--adr-dir PATH] [--dry-run]",
		"ssb prune    <inspect|validate|approve|apply|recover|status|verify> [options]",
	} {
		if !strings.Contains(stdout.String(), form) {
			t.Fatalf("help missing %q:\n%s", form, stdout.String())
		}
	}
	for _, forbidden := range []string{"commit", "push", "pull-request", "sync", "model"} {
		if strings.Contains(strings.ToLower(stdout.String()), forbidden) {
			t.Fatalf("help advertises forbidden surface %q:\n%s", forbidden, stdout.String())
		}
	}
	if !strings.Contains(stdout.String(), "4  inventory coverage incomplete") {
		t.Fatalf("help missing incomplete-coverage exit code:\n%s", stdout.String())
	}
}

func TestEachSubcommandProvidesSuccessfulFocusedHelp(t *testing.T) {
	for _, command := range []string{"inspect", "validate", "render", "adr"} {
		t.Run(command, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := cli.Run([]string{command, "--help"}, &stdout, &stderr)
			if code != 0 || stderr.Len() != 0 {
				t.Fatalf("%s help failed: exit=%d stderr=%q", command, code, stderr.String())
			}
			if !strings.Contains(stdout.String(), "Usage: ssb "+command) ||
				!strings.Contains(stdout.String(), "--repo PATH") {
				t.Fatalf("%s help is not actionable:\n%s", command, stdout.String())
			}
			if command == "inspect" {
				for _, required := range []string{
					"--max-candidate-files",
					"--max-candidate-bytes",
					"--allow-partial",
				} {
					if !strings.Contains(stdout.String(), required) {
						t.Errorf("inspect help missing %q:\n%s", required, stdout.String())
					}
				}
			}
		})
	}
}

func TestEachPruneSubcommandProvidesFocusedHelp(t *testing.T) {
	for _, command := range []string{"inspect", "validate", "approve", "apply", "recover", "status", "verify"} {
		t.Run(command, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := cli.Run([]string{"prune", command, "--help"}, &stdout, &stderr)
			if code != 0 || stderr.Len() != 0 {
				t.Fatalf("%s help failed: exit=%d stderr=%q", command, code, stderr.String())
			}
			if !strings.Contains(stdout.String(), "Usage: ssb prune "+command) ||
				!strings.Contains(stdout.String(), "--review ID") {
				t.Fatalf("%s help is not actionable:\n%s", command, stdout.String())
			}
		})
	}
}

func TestReviewAwareRenderAndADRValidateTransitionBeforeWriting(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)
	for _, test := range []struct {
		command []string
		target  string
	}{
		{
			command: []string{"render", "--repo", repo, "--review", "missing-review"},
			target:  filepath.Join(repo, "AGENTS.md"),
		},
		{
			command: []string{"adr", "--repo", repo, "--review", "missing-review"},
			target:  filepath.Join(repo, "docs", "adr", "0001-actionable-standards.md"),
		},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.Run(test.command, &stdout, &stderr)
		if code != 2 {
			t.Fatalf("%v exit = %d, stderr=%q", test.command, code, stderr.String())
		}
		if _, err := os.Stat(test.target); !os.IsNotExist(err) {
			t.Fatalf("%v wrote %s before validating review: %v", test.command, test.target, err)
		}
	}
}

func TestReviewAwareRenderAndADRValidateRetainedHistoricalRules(t *testing.T) {
	repo, evidenceBaseline := evidenceRepository(t)
	writeValidPack(t, repo, evidenceBaseline)
	updatedRulePath := filepath.Join(
		repo,
		".software-standards",
		"rules",
		"verify-before-merge.md",
	)
	updatedRule, err := os.ReadFile(updatedRulePath)
	if err != nil {
		t.Fatal(err)
	}
	retainedRule := strings.ReplaceAll(string(updatedRule), "verify-before-merge", "retain-history")
	writeFile(t, filepath.Join(
		repo,
		".software-standards",
		"rules",
		"retain-history.md",
	), retainedRule)
	reportPath := filepath.Join(repo, ".software-standards", "report.md")
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	reportData = []byte(strings.Replace(
		string(reportData),
		"---\n# Software standards report",
		`  - id: retain-history
    kind: rule
    path: .software-standards/rules/retain-history.md
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 70
      factors:
        marginal_value: 20
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 10
---
# Software standards report`,
		1,
	))
	if err := os.WriteFile(reportPath, reportData, 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".software-standards")
	git(t, repo, "commit", "-m", "adopt two rules")

	profileDir := t.TempDir()
	capabilityEvidence := filepath.Join(profileDir, "host-run.json")
	writeFile(t, capabilityEvidence, "{\"supported\":true}\n")
	profile := filepath.Join(profileDir, "capabilities.yaml")
	writeFile(t, profile, `schema: ssb.dev/capability-profile/v1
id: retained-rule-fixture
host: {name: codex, version: 1.2.3}
model: {provider: openai, id: gpt-example}
observed_at: 2026-07-27T18:00:00Z
evidence:
  - id: host-run
    kind: conformance
    path: host-run.json
    sha256: `+fileSHA256(t, capabilityEvidence)+`
capabilities:
  - id: repository-instructions
    status: supported
    evidence_ids: [host-run]
`)
	provenance := filepath.Join(profileDir, "provenance.yaml")
	writeFile(t, provenance, `schema: ssb.dev/artifact-provenance/v1
artifacts:
  - path: .software-standards/rules/retain-history.md
    sha256: `+fileSHA256(t, filepath.Join(repo, ".software-standards", "rules", "retain-history.md"))+`
    origin: generated
    declaration: Generated and adopted through the repository standards workflow.
  - path: .software-standards/rules/verify-before-merge.md
    sha256: `+fileSHA256(t, updatedRulePath)+`
    origin: generated
    declaration: Generated and adopted through the repository standards workflow.
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runCLI := func(arguments ...string) string {
		t.Helper()
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run(arguments, &stdout, &stderr); code != 0 {
			t.Fatalf("%v exit=%d stdout=%q stderr=%q", arguments, code, stdout.String(), stderr.String())
		}
		return stdout.String()
	}
	runCLI(
		"prune", "inspect",
		"--repo", repo,
		"--review", "retained-rule-review",
		"--capabilities", profile,
		"--provenance", provenance,
		"--format", "json",
	)

	reviewRoot := filepath.Join(
		repo,
		".software-standards",
		"reviews",
		"retained-rule-review",
	)
	contextData, err := os.ReadFile(filepath.Join(reviewRoot, "context.json"))
	if err != nil {
		t.Fatal(err)
	}
	var reviewContext prune.Context
	if err := json.Unmarshal(contextData, &reviewContext); err != nil {
		t.Fatal(err)
	}
	artifacts := make(map[string]prune.Artifact)
	for _, artifact := range reviewContext.Artifacts {
		artifacts[artifact.ID] = artifact
	}
	retainedArtifact, retainedExists := artifacts["retain-history"]
	updatedArtifact, updatedExists := artifacts["verify-before-merge"]
	if !retainedExists || !updatedExists {
		t.Fatalf("context artifacts = %#v, want both rules", reviewContext.Artifacts)
	}
	var repositoryEvidence prune.EvidenceRef
	for _, file := range reviewContext.Inventory.Files {
		if file.Path == "main.go" {
			repositoryEvidence = prune.EvidenceRef{
				Path: "main.go", Lines: "1-1", SHA256: file.SHA256,
			}
		}
	}
	if repositoryEvidence.SHA256 == "" {
		t.Fatal("main.go evidence is absent from the review context")
	}

	candidateRelative := "candidates/update-rule/verify-before-merge.md"
	candidatePath := filepath.Join(reviewRoot, filepath.FromSlash(candidateRelative))
	candidate := strings.Replace(
		string(updatedRule),
		"Run the repository verification command before merging.",
		"Run the repository verification command before merging any changed rule.",
		1,
	)
	writeFile(t, candidatePath, candidate)

	action := func(id, disposition string, artifact prune.Artifact) prune.Action {
		return prune.Action{
			ID:          id,
			Disposition: disposition,
			Sources: []prune.ArtifactRef{{
				Kind: artifact.Kind, ID: artifact.ID, Path: artifact.Path, SHA256: artifact.SHA256,
			}},
			Rationale:          "Pinned repository and capability evidence support this disposition.",
			Confidence:         prune.ConfidenceHigh,
			RepositoryEvidence: []prune.EvidenceRef{repositoryEvidence},
			CapabilityRefs:     []string{"repository-instructions"},
			RequiredVerification: []prune.CheckRequirement{{
				ID: "review-check", Command: "ssb validate --repo .",
			}},
		}
	}
	keep := action("keep-rule", prune.DispositionKeep, retainedArtifact)
	update := action("update-rule", prune.DispositionUpdate, updatedArtifact)
	update.Target = &prune.CandidateRef{
		Kind:       prune.ArtifactRule,
		ID:         "verify-before-merge",
		TargetPath: updatedArtifact.Path,
		SourcePath: candidateRelative,
		SHA256:     fileSHA256(t, candidatePath),
		Mode:       "100644",
	}
	proposalData, err := yaml.Marshal(prune.Proposal{
		Schema:        prune.ProposalSchema,
		ReviewID:      "retained-rule-review",
		ContextDigest: reviewContext.ContextDigest,
		Actions:       []prune.Action{keep, update},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reviewRoot, "proposal.yaml"), proposalData, 0o644); err != nil {
		t.Fatal(err)
	}

	runCLI("prune", "validate", "--repo", repo, "--review", "retained-rule-review")
	runCLI(
		"prune", "approve",
		"--repo", repo,
		"--review", "retained-rule-review",
		"--approve", "update-rule",
		"--reject", "keep-rule",
	)
	var dryRun prune.ApplyResult
	if err := json.Unmarshal([]byte(runCLI(
		"prune", "apply",
		"--repo", repo,
		"--review", "retained-rule-review",
		"--format", "json",
	)), &dryRun); err != nil {
		t.Fatal(err)
	}
	var applied prune.ApplyResult
	if err := json.Unmarshal([]byte(runCLI(
		"prune", "apply",
		"--repo", repo,
		"--review", "retained-rule-review",
		"--format", "json",
		"--write",
	)), &applied); err != nil {
		t.Fatal(err)
	}
	if applied.PlanDigest == "" || applied.PlanDigest != dryRun.PlanDigest {
		t.Fatalf("dry-run plan %q != applied plan %q", dryRun.PlanDigest, applied.PlanDigest)
	}

	runCLI("render", "--repo", repo, "--review", "retained-rule-review")
	runCLI("adr", "--repo", repo, "--review", "retained-rule-review")
	review, diagnostics, err := prune.LoadReview(repo, "retained-rule-review")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("review diagnostics = %#v", diagnostics)
	}
	var applicationEventDigest string
	var renderEventDigest string
	for _, event := range review.Events {
		switch event.Kind {
		case prune.EventApplied:
			applicationEventDigest = event.EventDigest
		case prune.EventRendered:
			renderEventDigest = event.EventDigest
		}
	}
	if applicationEventDigest == "" || renderEventDigest == "" {
		t.Fatalf("review events = %#v, want applied and rendered", review.Events)
	}

	receipts := t.TempDir()
	receiptEvidence := filepath.Join(receipts, "logs", "ssb-validate.txt")
	writeFile(t, receiptEvidence, "PASS\n")
	writeFile(t, filepath.Join(receipts, "review-check.yaml"), `schema: ssb.dev/prune-check-receipt/v1
review_id: retained-rule-review
proposal_digest: `+review.ProposalDigest+`
application_event_digest: `+applicationEventDigest+`
plan_digest: `+applied.PlanDigest+`
render_event_digest: `+renderEventDigest+`
check_id: review-check
command: ssb validate --repo .
status: passed
observed_at: 2099-07-27T18:02:00Z
evidence:
  - path: logs/ssb-validate.txt
    sha256: `+fileSHA256(t, receiptEvidence)+`
`)
	agentsPath := filepath.Join(repo, "AGENTS.md")
	renderedAgents, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	assertVerifyFails := func(want string) {
		t.Helper()
		stdout.Reset()
		stderr.Reset()
		code := cli.Run([]string{
			"prune", "verify",
			"--repo", repo,
			"--review", "retained-rule-review",
			"--receipts", receipts,
		}, &stdout, &stderr)
		if code != 2 || !strings.Contains(stderr.String(), want) {
			t.Fatalf("verify exit=%d stdout=%q stderr=%q, want %q", code, stdout.String(), stderr.String(), want)
		}
		status := runCLI(
			"prune", "status",
			"--repo", repo,
			"--review", "retained-rule-review",
			"--format", "json",
		)
		if strings.Contains(status, `"verified": true`) {
			t.Fatalf("failed verification recorded verified status: %s", status)
		}
	}

	writeFile(t, agentsPath, "human edit outside the managed section\n"+string(renderedAgents))
	assertVerifyFails("rendered poststate drift")
	if err := os.WriteFile(agentsPath, renderedAgents, 0o644); err != nil {
		t.Fatal(err)
	}

	externalAgents := filepath.Join(t.TempDir(), "AGENTS.md")
	writeFile(t, externalAgents, string(renderedAgents))
	if err := os.Remove(agentsPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalAgents, agentsPath); err == nil {
		assertVerifyFails("safe regular repository file")
		if err := os.Remove(agentsPath); err != nil {
			t.Fatal(err)
		}
	} else if !os.IsNotExist(err) {
		t.Logf("symlink unavailable; whole-file drift path still exercised: %v", err)
	}
	if err := os.WriteFile(agentsPath, renderedAgents, 0o644); err != nil {
		t.Fatal(err)
	}

	runCLI(
		"prune", "verify",
		"--repo", repo,
		"--review", "retained-rule-review",
		"--receipts", receipts,
	)
	status := runCLI(
		"prune", "status",
		"--repo", repo,
		"--review", "retained-rule-review",
		"--format", "json",
	)
	if !strings.Contains(status, `"verified": true`) {
		t.Fatalf("status = %s, want verified review", status)
	}
}

func TestPruneCLIWalksExplicitStatesWithoutImplicitVerification(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)
	git(t, repo, "add", ".software-standards")
	git(t, repo, "commit", "-m", "adopt standards")

	profileDir := t.TempDir()
	evidence := filepath.Join(profileDir, "host-run.json")
	writeFile(t, evidence, "{\"supported\":true}\n")
	profile := filepath.Join(profileDir, "capabilities.yaml")
	writeFile(t, profile, `schema: ssb.dev/capability-profile/v1
id: cli-fixture
host: {name: codex, version: 1.2.3}
model: {provider: openai, id: gpt-example}
observed_at: 2026-07-27T18:00:00Z
evidence:
  - id: host-run
    kind: conformance
    path: host-run.json
    sha256: `+fileSHA256(t, evidence)+`
capabilities:
  - id: repository-instructions
    status: supported
    evidence_ids: [host-run]
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{
		"prune", "inspect", "--repo", repo, "--review", "cli-review",
		"--capabilities", profile, "--format", "json",
	}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("inspect exit=%d stderr=%q", code, stderr.String())
	}
	contextPath := filepath.Join(repo, ".software-standards", "reviews", "cli-review", "context.json")
	contextData, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatal(err)
	}
	var reviewContext struct {
		ContextDigest string `json:"context_digest"`
		Artifacts     []struct {
			Kind, ID, Path, SHA256 string
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(contextData, &reviewContext); err != nil {
		t.Fatal(err)
	}
	if len(reviewContext.Artifacts) != 1 {
		t.Fatalf("artifacts = %#v", reviewContext.Artifacts)
	}
	artifact := reviewContext.Artifacts[0]
	proposalPath := filepath.Join(filepath.Dir(contextPath), "proposal.yaml")
	writeFile(t, proposalPath, "schema: [malformed\n")
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{
		"prune", "validate", "--repo", repo, "--review", "cli-review",
	}, &stdout, &stderr); code != 1 {
		t.Fatalf("malformed proposal exit=%d, want 1; stderr=%q", code, stderr.String())
	}
	proposal := fmt.Sprintf(`schema: ssb.dev/prune-proposal/v1
review_id: cli-review
context_digest: %s
actions:
  - id: review-keep-rule
    disposition: unable-to-determine
    sources:
      - kind: %s
        id: %s
        path: %s
        sha256: %s
    rationale: Provenance was not declared.
    confidence: low
    evidence_gaps:
      - kind: provenance
        artifact_path: %s
        detail: No provenance declaration matches the inventoried bytes.
    unresolved_questions:
      - Who authored and adopted this rule?
`, reviewContext.ContextDigest, artifact.Kind, artifact.ID, artifact.Path, artifact.SHA256, artifact.Path)
	writeFile(t, proposalPath, proposal)

	for _, command := range [][]string{
		{"prune", "validate", "--repo", repo, "--review", "cli-review"},
		{"prune", "approve", "--repo", repo, "--review", "cli-review", "--reject", "review-keep-rule"},
		{"prune", "apply", "--repo", repo, "--review", "cli-review"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run(command, &stdout, &stderr); code != 0 {
			t.Fatalf("%v exit=%d stdout=%q stderr=%q", command, code, stdout.String(), stderr.String())
		}
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{
		"prune", "apply", "--repo", repo, "--review", "cli-review", "--write",
	}, &stdout, &stderr); code != 0 ||
		!strings.Contains(stdout.String(), "No changes were approved") ||
		!strings.Contains(stdout.String(), "Plan: sha256:") ||
		!strings.Contains(stdout.String(), "without application or verification") {
		t.Fatalf("no-change apply exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"prune", "status", "--repo", repo, "--review", "cli-review", "--format", "json",
	}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"applied": false`) ||
		!strings.Contains(stdout.String(), `"no_changes_approved": true`) ||
		!strings.Contains(stdout.String(), `"verified": false`) {
		t.Fatalf("status exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"prune", "status", "--repo", repo, "--review", "cli-review",
	}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "no-changes-approved=true") {
		t.Fatalf("text status exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{
		"prune", "verify", "--repo", repo, "--review", "cli-review", "--receipts", t.TempDir(),
	}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "no changes were approved") {
		t.Fatalf("no-change verify exit=%d, want 2; stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{
		"prune", "recover", "--repo", repo, "--review", "cli-review",
	}, &stdout, &stderr); code != 2 {
		t.Fatalf("journal-free recover exit=%d, want 2; stderr=%q", code, stderr.String())
	}
}

func TestValidateUsesExitOneForActionablePackFailuresAndNeverWrites(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)
	before := git(t, repo, "status", "--porcelain=v1", "-z")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"validate", "--repo", repo, "--format", "json"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("valid pack failed: exit=%d stderr=%q", code, stderr.String())
	}
	var validResponse struct {
		SchemaVersion int  `json:"schema_version"`
		Valid         bool `json:"valid"`
		RuleCount     int  `json:"rule_count"`
		Pack          *struct {
			BaselineCommit string `json:"baseline_commit"`
			Rules          []struct {
				Schema     string `json:"schema"`
				ID         string `json:"id"`
				SourcePath string `json:"source_path"`
				Body       string `json:"body"`
			} `json:"rules"`
		} `json:"pack"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &validResponse); err != nil {
		t.Fatal(err)
	}
	if validResponse.SchemaVersion != 3 || !validResponse.Valid || validResponse.RuleCount != 1 ||
		validResponse.Pack == nil || validResponse.Pack.BaselineCommit != baseline ||
		len(validResponse.Pack.Rules) != 1 ||
		validResponse.Pack.Rules[0].Schema != rulepack.RuleSchema ||
		validResponse.Pack.Rules[0].ID != "verify-before-merge" ||
		validResponse.Pack.Rules[0].SourcePath == "" ||
		!strings.Contains(validResponse.Pack.Rules[0].Body, "Run the repository verification command") {
		t.Fatalf("unexpected valid response: %#v", validResponse)
	}
	if after := git(t, repo, "status", "--porcelain=v1", "-z"); after != before {
		t.Fatalf("validate changed repository: before=%q after=%q", before, after)
	}

	rulePath := filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md")
	rule, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, rulePath, strings.Replace(string(rule), excerptHash("package main\n"), "sha256:"+strings.Repeat("0", 64), 1))
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"validate", "--repo", repo, "--format", "json"}, &stdout, &stderr)
	if code != 1 || stderr.Len() != 0 {
		t.Fatalf("invalid pack response: exit=%d stderr=%q", code, stderr.String())
	}
	var invalidResponse struct {
		Valid       bool            `json:"valid"`
		Pack        json.RawMessage `json:"pack"`
		Diagnostics []struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &invalidResponse); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
	}
	if invalidResponse.Valid || len(invalidResponse.Diagnostics) == 0 ||
		len(invalidResponse.Pack) != 0 ||
		!strings.Contains(invalidResponse.Diagnostics[0].Message, "excerpt hash does not match") {
		t.Fatalf("unexpected invalid response: %#v", invalidResponse)
	}
}

func TestValidateJSONExportsActionableInterchangeFields(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"validate", "--repo", repo, "--format", "json"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("valid embedded-layout pack failed: exit=%d stderr=%q", code, stderr.String())
	}
	var response struct {
		SchemaVersion int  `json:"schema_version"`
		Valid         bool `json:"valid"`
		Pack          *struct {
			Layout   rulepack.Layout `json:"layout"`
			Manifest struct {
				Schema    string `json:"schema"`
				Artifacts []struct {
					Confidence string           `json:"confidence"`
					Utility    rulepack.Utility `json:"utility"`
				} `json:"artifacts"`
			} `json:"manifest"`
			Inventory struct {
				BaselineCommit string `json:"baseline_commit"`
			} `json:"inventory"`
			Report struct {
				Body string `json:"body"`
			} `json:"report"`
			Rules []struct {
				Schema     string              `json:"schema"`
				Category   string              `json:"category"`
				Lenses     []rulepack.Lens     `json:"lenses"`
				Directive  string              `json:"directive"`
				Derivation string              `json:"derivation"`
				Evidence   []rulepack.Evidence `json:"evidence"`
			} `json:"rules"`
		} `json:"pack"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != 3 || !response.Valid || response.Pack == nil ||
		response.Pack.Layout != rulepack.LayoutEmbedded ||
		response.Pack.Manifest.Schema != rulepack.ReportSchema ||
		len(response.Pack.Manifest.Artifacts) != 1 ||
		response.Pack.Manifest.Artifacts[0].Confidence != "high" ||
		response.Pack.Manifest.Artifacts[0].Utility.Method != rulepack.UtilityMethod ||
		response.Pack.Inventory.BaselineCommit != baseline ||
		!strings.HasPrefix(response.Pack.Report.Body, "# Software standards report") ||
		len(response.Pack.Rules) != 1 ||
		response.Pack.Rules[0].Schema != rulepack.RuleSchema ||
		response.Pack.Rules[0].Category != "correctness" ||
		len(response.Pack.Rules[0].Lenses) != 1 ||
		response.Pack.Rules[0].Lenses[0] != (rulepack.Lens{Kind: "base"}) ||
		response.Pack.Rules[0].Directive != "always" ||
		response.Pack.Rules[0].Derivation != "extracted" ||
		len(response.Pack.Rules[0].Evidence) != 1 ||
		response.Pack.Rules[0].Evidence[0].Role != "declares" {
		t.Fatalf("unexpected actionable interchange response: %#v", response)
	}
	if strings.Contains(stdout.String(), `"manifest_path"`) || strings.Contains(stdout.String(), `"inventory_path"`) {
		t.Fatalf("embedded schema 3 response exposed nonexistent manifest paths:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"format"`) ||
		strings.Contains(stdout.String(), "split"+"-v1") ||
		strings.Contains(stdout.String(), "legacy"+"-v1") {
		t.Fatalf("embedded schema 3 response retained obsolete layout terminology:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"topic"`) ||
		strings.Contains(stdout.String(), `"classification"`) ||
		strings.Contains(stdout.String(), `"verification":`) ||
		strings.Contains(stdout.String(), `"score":`) {
		t.Fatalf("normalized JSON retained removed rule fields:\n%s", stdout.String())
	}
	for _, required := range []string{`"rules": [`, `"verification_recipes": []`, `"skills": [`, `"automation_proposals": []`} {
		if !strings.Contains(stdout.String(), required) {
			t.Fatalf("normalized JSON omitted artifact array %s:\n%s", required, stdout.String())
		}
	}
}

func TestValidateJSONSchemaThreeIdentifiesManifestLayout(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"validate", "--repo", repo, "--format", "json"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("valid manifest-layout pack failed: exit=%d stderr=%q", code, stderr.String())
	}
	var response struct {
		SchemaVersion int  `json:"schema_version"`
		Valid         bool `json:"valid"`
		Pack          *struct {
			Layout        rulepack.Layout `json:"layout"`
			ManifestPath  string          `json:"manifest_path"`
			InventoryPath string          `json:"inventory_path"`
			ReportPath    string          `json:"report_path"`
			Manifest      struct {
				Schema    string                      `json:"schema"`
				Inventory rulepack.FileReference      `json:"inventory"`
				Artifacts []rulepack.AcceptedArtifact `json:"artifacts"`
			} `json:"manifest"`
			Inventory struct {
				BaselineCommit string `json:"baseline_commit"`
				IndexedFiles   int    `json:"indexed_files"`
			} `json:"inventory"`
			Report struct {
				Body string `json:"body"`
			} `json:"report"`
			Rules []rulepack.Rule `json:"rules"`
		} `json:"pack"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid schema 3 JSON %q: %v", stdout.String(), err)
	}
	if response.SchemaVersion != 3 || !response.Valid || response.Pack == nil ||
		response.Pack.Layout != rulepack.LayoutManifest ||
		response.Pack.ManifestPath != ".software-standards/manifest.yaml" ||
		response.Pack.InventoryPath != ".software-standards/inventory.json" ||
		response.Pack.ReportPath != ".software-standards/report.md" ||
		response.Pack.Manifest.Schema != rulepack.ManifestSchema ||
		response.Pack.Manifest.Inventory.Path != response.Pack.InventoryPath ||
		len(response.Pack.Manifest.Artifacts) != 1 ||
		response.Pack.Inventory.BaselineCommit != baseline || response.Pack.Inventory.IndexedFiles != 2 ||
		!strings.HasPrefix(response.Pack.Report.Body, "# Software standards report") ||
		len(response.Pack.Rules) != 1 || response.Pack.Rules[0].Title != "Verify before merge" {
		t.Fatalf("unexpected schema 3 manifest-layout response: %#v\n%s", response, stdout.String())
	}
	if strings.Contains(stdout.String(), `"report": {\n      "schema":`) {
		t.Fatalf("schema 3 report repeated machine frontmatter metadata:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"format"`) ||
		strings.Contains(stdout.String(), "split"+"-v1") ||
		strings.Contains(stdout.String(), "legacy"+"-v1") {
		t.Fatalf("manifest schema 3 response retained obsolete layout terminology:\n%s", stdout.String())
	}
}

func TestRenderDryRunAndValidationFailureHaveNoFilesystemEffects(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"render", "--repo", repo, "--dry-run"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("dry-run failed: exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "AGENTS.md") ||
		!strings.Contains(stdout.String(), "Run the repository verification command before merging.") {
		t.Fatalf("dry-run did not disclose projection:\n%s", stdout.String())
	}
	if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created AGENTS.md: %v", err)
	}

	reportPath := filepath.Join(repo, ".software-standards", "report.md")
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, reportPath, strings.Replace(string(report), "total: 70", "total: 71", 1))
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "utility total 71 does not equal factor sum 70") {
		t.Fatalf("invalid render response: exit=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("invalid render created AGENTS.md: %v", err)
	}
}

func TestRenderDryRunReportsNoWriteForZeroArtifactPack(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)
	removeAllArtifactsFromPack(t, repo)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"render", "--repo", repo, "--dry-run"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("zero-artifact dry run failed: exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "AGENTS.md would not be changed") ||
		!strings.Contains(stdout.String(), "no active semantic rule, verification recipe, or Agent Skill") ||
		strings.Contains(stdout.String(), "proposed AGENTS.md") {
		t.Fatalf("zero-artifact dry run response is unclear:\n%s", stdout.String())
	}
	if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("zero-artifact dry run created AGENTS.md: %v", err)
	}
}

func TestRenderExplainsZeroArtifactProjectionRemoval(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr); code != 0 {
		t.Fatalf("initial render failed: exit=%d stderr=%q", code, stderr.String())
	}
	agentsPath := filepath.Join(repo, "AGENTS.md")
	before, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	removeAllArtifactsFromPack(t, repo)

	stdout.Reset()
	stderr.Reset()
	code := cli.Run([]string{"render", "--repo", repo, "--dry-run"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("removal dry run failed: exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "would remove its managed Software Standards Bootstrap section") {
		t.Fatalf("removal dry run is unclear:\n%s", stdout.String())
	}
	afterDryRun, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterDryRun, before) {
		t.Fatal("removal dry run changed AGENTS.md")
	}

	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("projection removal failed: exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Removed the managed Software Standards Bootstrap section") {
		t.Fatalf("projection removal response is unclear:\n%s", stdout.String())
	}
	afterRemoval, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(afterRemoval), render.StartMarker) {
		t.Fatalf("managed section remains:\n%s", afterRemoval)
	}
}

func TestRenderDryRunReportsAlreadyCurrentForActivePack(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr); code != 0 {
		t.Fatalf("initial render failed: exit=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := cli.Run([]string{"render", "--repo", repo, "--dry-run"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("stable dry run failed: exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "AGENTS.md is already current") ||
		strings.Contains(stdout.String(), "no active semantic rule") {
		t.Fatalf("stable active-pack response is misleading:\n%s", stdout.String())
	}
}

func TestRenderManifestLayoutGuidanceNamesManifestDigestUpdate(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("render failed: exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "update manifest.yaml SHA-256 values") ||
		strings.Contains(stdout.String(), "report manifest together") {
		t.Fatalf("manifest-layout render guidance is stale:\n%s", stdout.String())
	}

	agentsPath := filepath.Join(repo, "AGENTS.md")
	rendered, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, agentsPath, strings.Replace(string(rendered), "Run the repository verification command", "Direct section edit", 1))
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "update manifest.yaml SHA-256 values") ||
		strings.Contains(stderr.String(), "sources and report.md") {
		t.Fatalf("manifest-layout drift recovery is stale: exit=%d stderr=%q", code, stderr.String())
	}
}

func TestRenderWritesOnlyAgentsAndReportsDriftAsPrecondition(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("render failed: exit=%d stderr=%q", code, stderr.String())
	}
	agentsPath := filepath.Join(repo, "AGENTS.md")
	rendered, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	drifted := strings.Replace(string(rendered), "Run the repository verification command", "Direct section edit", 1)
	writeFile(t, agentsPath, drifted)

	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "edit canonical artifact sources") {
		t.Fatalf("drift response: exit=%d stderr=%q", code, stderr.String())
	}
	after, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != drifted {
		t.Fatal("drift failure modified AGENTS.md")
	}
}

func TestADRValidatesThenCreatesExactlyOneNewProposedRecord(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"adr", "--repo", repo, "--dry-run"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("ADR dry-run failed: exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "docs/adr/0001-actionable-standards.md") ||
		!strings.Contains(stdout.String(), "Status: Proposed") {
		t.Fatalf("ADR dry-run output is incomplete:\n%s", stdout.String())
	}
	if _, err := os.Lstat(filepath.Join(repo, "docs")); !os.IsNotExist(err) {
		t.Fatalf("ADR dry-run created directories: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"adr", "--repo", repo}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("ADR creation failed: exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Created docs/adr/0001-actionable-standards.md") {
		t.Fatalf("ADR output did not disclose path: %q", stdout.String())
	}
	record, err := os.ReadFile(filepath.Join(repo, "docs", "adr", "0001-actionable-standards.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(record), "Status: Proposed") {
		t.Fatalf("ADR status is not Proposed:\n%s", record)
	}
}

func TestADRDirectoryAmbiguityIsARecoverablePrecondition(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPack(t, repo, baseline)
	if err := os.MkdirAll(filepath.Join(repo, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "docs", "adrs"), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"adr", "--repo", repo}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "--adr-dir PATH") {
		t.Fatalf("ambiguous ADR response: exit=%d stderr=%q", code, stderr.String())
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

func evidenceRepository(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(repo, "Makefile"), "verify:\n\tgo test ./...\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "baseline")
	return repo, strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
}

func writeValidPack(t *testing.T, repo, baseline string) {
	t.Helper()
	makefileOID := strings.TrimSpace(git(t, repo, "rev-parse", baseline+":Makefile"))
	mainOID := strings.TrimSpace(git(t, repo, "rev-parse", baseline+":main.go"))
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), fmt.Sprintf(`---
schema: ssb.dev/report/v1
baseline_commit: %s
inventory:
  schema_version: 2
  inventory_version: ssb-inventory-v2
  baseline_commit: %s
  limits:
    max_candidate_files: 40000
    max_candidate_bytes: 134217728
    max_file_bytes: 1048576
  candidate_files: 2
  candidate_bytes: 52
  scanned_files: 2
  scanned_bytes: 52
  indexed_files: 2
  indexed_bytes: 52
  files:
    - path: Makefile
      blob_oid: "%s"
      bytes: 23
      lines: 2
      sha256: %s
    - path: main.go
      blob_oid: "%s"
      bytes: 29
      lines: 3
      language: Go
      sha256: %s
  excluded:
    binary: 0
    generated: 0
    oversized: 0
    secret_like: 0
    symlink: 0
    submodule: 0
    vendor_or_generated_tree: 0
    non_regular: 0
  truncated: false
  remaining_candidate_files: 0
  remaining_candidate_bytes: 0
artifacts:
  - id: verify-before-merge
    kind: rule
    path: .software-standards/rules/verify-before-merge.md
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 70
      factors:
        marginal_value: 20
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 10
---
# Software standards report

Inventory coverage was complete and the retained artifact is listed above.
`,
		baseline,
		baseline,
		makefileOID,
		excerptHash("verify:\n\tgo test ./...\n"),
		mainOID,
		excerptHash("package main\n\nfunc main() {}\n"),
	))
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: verify-before-merge
title: Verify before merge
category: correctness
lenses:
  - kind: base
directive: always
scopes:
  - "**/*.go"
derivation: extracted
evidence:
  - role: declares
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
---
Run the repository verification command before merging.
`, excerptHash("package main\n")))
}

func writeValidManifestLayoutPack(t *testing.T, repo, baseline string) {
	t.Helper()
	var inventoryOut bytes.Buffer
	var inventoryErr bytes.Buffer
	if code := cli.Run([]string{"inspect", "--repo", repo, "--format", "json"}, &inventoryOut, &inventoryErr); code != 0 {
		t.Fatalf("inspect manifest-layout fixture: exit=%d stderr=%q", code, inventoryErr.String())
	}
	report := []byte(`# Software standards report

Inventory coverage was complete. Review the [manifest](manifest.yaml) and [inventory](inventory.json) for machine metadata.
`)
	rule := []byte(`# Verify before merge

Run the repository verification command before merging.
`)
	manifest := fmt.Sprintf(`schema: ssb.dev/manifest/v1
baseline_commit: %s
inventory:
  path: .software-standards/inventory.json
  sha256: %s
report:
  path: .software-standards/report.md
  sha256: %s
artifacts:
  - id: verify-before-merge
    kind: rule
    path: .software-standards/rules/verify-before-merge.md
    sha256: %s
    category: correctness
    lenses:
      - kind: base
    directive: always
    scopes:
      - "**/*.go"
    derivation: extracted
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 70
      factors:
        marginal_value: 20
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 10
`,
		baseline,
		excerptHash(inventoryOut.String()),
		excerptHash(string(report)),
		excerptHash(string(rule)),
		excerptHash("package main\n"),
	)
	writeFile(t, filepath.Join(repo, ".software-standards", "manifest.yaml"), manifest)
	writeFile(t, filepath.Join(repo, ".software-standards", "inventory.json"), inventoryOut.String())
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), string(report))
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), string(rule))
}

func removeAllArtifactsFromPack(t *testing.T, repo string) {
	t.Helper()
	reportPath := filepath.Join(repo, ".software-standards", "report.md")
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	artifactsStart := strings.Index(string(report), "artifacts:\n")
	bodyStart := strings.Index(string(report), "---\n# Software standards report")
	if artifactsStart < 0 || bodyStart < 0 || bodyStart <= artifactsStart {
		t.Fatalf("unexpected report fixture:\n%s", report)
	}
	emptyReport := string(report[:artifactsStart]) + "artifacts: []\n" + string(report[bodyStart:])
	writeFile(t, reportPath, emptyReport)
	if err := os.Remove(filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md")); err != nil {
		t.Fatal(err)
	}
}

func excerptHash(excerpt string) string {
	sum := sha256.Sum256([]byte(excerpt))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fileSHA256(t *testing.T, filePath string) string {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
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

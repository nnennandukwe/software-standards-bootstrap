package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/cli"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
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
			target:  filepath.Join(repo, "docs", "adr", "0001-agentic-rules.md"),
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
    unresolved_questions:
      - Who authored and adopted this rule?
`, reviewContext.ContextDigest, artifact.Kind, artifact.ID, artifact.Path, artifact.SHA256)
	writeFile(t, proposalPath, proposal)

	for _, command := range [][]string{
		{"prune", "validate", "--repo", repo, "--review", "cli-review"},
		{"prune", "approve", "--repo", repo, "--review", "cli-review", "--reject", "review-keep-rule"},
		{"prune", "apply", "--repo", repo, "--review", "cli-review"},
		{"prune", "apply", "--repo", repo, "--review", "cli-review", "--write"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := cli.Run(command, &stdout, &stderr); code != 0 {
			t.Fatalf("%v exit=%d stdout=%q stderr=%q", command, code, stdout.String(), stderr.String())
		}
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"prune", "status", "--repo", repo, "--review", "cli-review", "--format", "json",
	}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"applied": true`) ||
		!strings.Contains(stdout.String(), `"verified": false`) {
		t.Fatalf("status exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{
		"prune", "verify", "--repo", repo, "--review", "cli-review", "--receipts", t.TempDir(),
	}, &stdout, &stderr); code != 1 {
		t.Fatalf("evidence-free verify exit=%d, want 1; stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli.Run([]string{
		"prune", "recover", "--repo", repo, "--review", "cli-review",
	}, &stdout, &stderr); code != 2 {
		t.Fatalf("journal-free recover exit=%d, want 2; stderr=%q", code, stderr.String())
	}
}

func TestValidateUsesExitOneForRulePackFailuresAndNeverWrites(t *testing.T) {
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
	if validResponse.SchemaVersion != 2 || !validResponse.Valid || validResponse.RuleCount != 1 ||
		validResponse.Pack == nil || validResponse.Pack.BaselineCommit != baseline ||
		len(validResponse.Pack.Rules) != 1 ||
		validResponse.Pack.Rules[0].Schema != rulepack.SchemaVersionV1 ||
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

func TestValidateJSONExportsRuleV2InterchangeFields(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidPackV2(t, repo, baseline)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"validate", "--repo", repo, "--format", "json"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("valid v2 pack failed: exit=%d stderr=%q", code, stderr.String())
	}
	var response struct {
		SchemaVersion int  `json:"schema_version"`
		Valid         bool `json:"valid"`
		Pack          *struct {
			Rules []struct {
				Schema       string          `json:"schema"`
				Lenses       []rulepack.Lens `json:"lenses"`
				Directive    string          `json:"directive"`
				Verification struct {
					Coverage string `json:"coverage"`
					Proves   string `json:"proves"`
				} `json:"verification"`
			} `json:"rules"`
		} `json:"pack"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != 2 || !response.Valid || response.Pack == nil ||
		len(response.Pack.Rules) != 1 ||
		response.Pack.Rules[0].Schema != rulepack.SchemaVersionV2 ||
		len(response.Pack.Rules[0].Lenses) != 2 ||
		response.Pack.Rules[0].Lenses[0] != (rulepack.Lens{Kind: "language", Value: "go"}) ||
		response.Pack.Rules[0].Lenses[1] != (rulepack.Lens{Kind: "task", Value: "review"}) ||
		response.Pack.Rules[0].Directive != "always" ||
		response.Pack.Rules[0].Verification.Coverage != "full" ||
		response.Pack.Rules[0].Verification.Proves != "The retained Go assertions when the command passes." {
		t.Fatalf("unexpected v2 interchange response: %#v", response)
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

	rulePath := filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md")
	rule, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, rulePath, strings.Replace(string(rule), "total: 70", "total: 71", 1))
	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{"render", "--repo", repo}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "score total 71 does not equal factor sum 70") {
		t.Fatalf("invalid render response: exit=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("invalid render created AGENTS.md: %v", err)
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
	if code != 2 || !strings.Contains(stderr.String(), "edit or delete rule source files") {
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
	if !strings.Contains(stdout.String(), "docs/adr/0001-agentic-rules.md") ||
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
	if !strings.Contains(stdout.String(), "Created docs/adr/0001-agentic-rules.md") {
		t.Fatalf("ADR output did not disclose path: %q", stdout.String())
	}
	record, err := os.ReadFile(filepath.Join(repo, "docs", "adr", "0001-agentic-rules.md"))
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
	writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n")
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v1
id: verify-before-merge
title: Verify before merge
topic: correctness
scopes:
  - "**/*.go"
classification: deterministic
importance: high
score:
  method: ssb-score-v1
  total: 70
  factors:
    prevalence: 15
    consistency: 15
    authority: 15
    risk: 15
    applicability: 10
confidence: high
baseline_commit: %s
evidence:
  - path: main.go
    lines: 1-1
    excerpt_sha256: %s
    authoritative: true
verification:
  command: go test ./...
  source:
    path: Makefile
    lines: 1-2
    excerpt_sha256: %s
---
Run the repository verification command before merging.
`, baseline, excerptHash("package main\n"), excerptHash("verify:\n\tgo test ./...\n")))
}

func writeValidPackV2(t *testing.T, repo, baseline string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n")
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: verify-before-merge
title: Verify before merge
topic: correctness
lenses:
  - kind: language
    value: go
  - kind: task
    value: review
directive: always
scopes:
  - "**/*.go"
classification: deterministic
importance: high
score:
  method: ssb-score-v1
  total: 70
  factors:
    prevalence: 15
    consistency: 15
    authority: 15
    risk: 15
    applicability: 10
confidence: high
baseline_commit: %s
evidence:
  - path: main.go
    lines: 1-1
    excerpt_sha256: %s
    authoritative: true
verification:
  command: go test ./...
  source:
    path: Makefile
    lines: 1-2
    excerpt_sha256: %s
  coverage: full
  proves: The retained Go assertions when the command passes.
---
Run the repository verification command before merging.
`, baseline, excerptHash("package main\n"), excerptHash("verify:\n\tgo test ./...\n")))
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

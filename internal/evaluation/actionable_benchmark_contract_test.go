package evaluation_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestBenchmarkAcceptanceCountsEveryActionableArtifact(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "docs", "benchmarks.md"))
	if err != nil {
		t.Fatal(err)
	}
	benchmark := strings.Join(strings.Fields(string(content)), " ")
	for _, required := range []string{
		"counts for every emitted rule, verification recipe, Agent Skill, and automation proposal",
		"Orientation is repository context, not an actionable artifact",
		"does not enter the artifact denominator or ADR eligibility",
		"100% of emitted artifacts have resolvable evidence",
		"Every accepted candidate has exactly one actionable artifact kind",
		"At least 70% of final artifacts are judged “keep” or “edit and keep” in each pinned repository independently",
		"A pooled cross-repository average is not acceptance",
		"denominator is every final artifact emitted for that fixture before developer review",
		"deferred and rejected artifacts remain in the denominator",
		"exact fraction must meet the threshold without rounding",
		"At least one conforming agent host finishes the workflow",
		"fresh blind pass over all four pins",
		"proposal generation",
		"developer retention",
		"rendering",
		"ADR creation",
		"release evidence",
	} {
		if !strings.Contains(benchmark, required) {
			t.Errorf("benchmark acceptance missing %q", required)
		}
	}
}

func TestFreshActionableBenchmarkLedgerDoesNotPromoteInventoryToAcceptance(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(
		repositoryRoot(t),
		"docs",
		"benchmarks",
		"results",
		"2026-07-29-actionable",
		"README.md",
	))
	if err != nil {
		t.Fatal(err)
	}
	ledger := strings.Join(strings.Fields(string(content)), " ")
	for _, required := range []string{
		"Cobra | 66 | 705,271 | 66 | 705,271 | 65 | 631,792 | 0 | 0",
		"Flask | 235 | 1,814,782 | 235 | 1,814,782 | 230 | 1,474,850 | 0 | 0",
		"Django | 7,001 | 45,506,636 | 7,001 | 45,506,636 | 5,619 | 36,820,618 | 0 | 0",
		"Next.js | 29,073 | 111,110,455 | 29,073 | 111,110,455 | 28,403 | 88,643,646 | 0 | 0",
		"Cobra | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0",
		"Flask | 5 | 0 | 0 | 1 | 0 | 0 | 0 | 0",
		"Django | 1,382 | 0 | 0 | 0 | 4 | 0 | 72 | 0",
		"Next.js | 652 | 18 | 21 | 113 | 29 | 0 | 1,060 | 0",
		"SSB and repository-tool network use: none",
		"Claude Code 2.1.220 with `claude-sonnet-5`; model-service transport was used",
		"Tree-level exclusions (`oversized`, `secret-like`, `symlink`, `submodule`, `vendor/generated tree`, and `non-regular`) are removed before candidate accounting",
		"Scan-level `binary` and `generated` exclusions explain the candidate-to-indexed delta",
		"Proposal generation | Not complete",
		"Developer retention | Not performed",
		"Actionable-artifact acceptance remains open",
	} {
		if !strings.Contains(ledger, required) {
			t.Errorf("fresh benchmark ledger missing %q", required)
		}
	}
}

func TestHoopAgentsContractRecordBindsProjectionAndRouting(t *testing.T) {
	repoRoot := repositoryRoot(t)
	benchmarkRelativeRoot := filepath.Join(
		"docs", "benchmarks", "results", "2026-08-12-hoop-agents-contract",
	)
	root := filepath.Join(repoRoot, benchmarkRelativeRoot)
	runBytes, err := os.ReadFile(filepath.Join(root, "run.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var run struct {
		Schema string `yaml:"schema"`
		Target struct {
			Repository string `yaml:"repository"`
			URL        string `yaml:"url"`
			Commit     string `yaml:"commit"`
		} `yaml:"target"`
		SSB struct {
			SourceCommit string `yaml:"source_commit"`
			BinarySHA256 string `yaml:"binary_sha256"`
			SkillVersion string `yaml:"skill_version"`
		} `yaml:"ssb"`
		Host struct {
			Name         string `yaml:"name"`
			Version      string `yaml:"version"`
			Model        string `yaml:"model"`
			OS           string `yaml:"os"`
			Architecture string `yaml:"architecture"`
		} `yaml:"host"`
		Inventory struct {
			SchemaVersion  int  `yaml:"schema_version"`
			Complete       bool `yaml:"complete"`
			Truncated      bool `yaml:"truncated"`
			CandidateFiles int  `yaml:"candidate_files"`
			CandidateBytes int  `yaml:"candidate_bytes"`
			IndexedFiles   int  `yaml:"indexed_files"`
			IndexedBytes   int  `yaml:"indexed_bytes"`
		} `yaml:"inventory"`
		Proposal struct {
			ArtifactCount        int    `yaml:"artifact_count"`
			SemanticRules        int    `yaml:"semantic_rules"`
			VerificationRecipes  int    `yaml:"verification_recipes"`
			AgentSkills          int    `yaml:"agent_skills"`
			AutomationProposals  int    `yaml:"automation_proposals"`
			Orientation          bool   `yaml:"orientation"`
			OrientationArtifacts int    `yaml:"orientation_artifacts"`
			SourceDigest         string `yaml:"source_digest"`
			ContentDigest        string `yaml:"content_digest"`
			OutputDigest         string `yaml:"output_digest"`
			RetainedPath         string `yaml:"retained_path"`
		} `yaml:"proposal"`
		Verification struct {
			Validate              string `yaml:"validate"`
			DryRun                string `yaml:"dry_run"`
			Render                string `yaml:"render"`
			Replay                string `yaml:"canonical_replay"`
			CommandsDisplayedOnly bool   `yaml:"commands_displayed_only"`
		} `yaml:"verification"`
		Routing []struct {
			Lens              string   `yaml:"lens"`
			Status            string   `yaml:"status"`
			Path              string   `yaml:"path"`
			ActiveArtifactIDs []string `yaml:"active_artifact_ids"`
		} `yaml:"routing"`
		ExecutionBoundaries struct {
			TargetCommandsExecuted         []string `yaml:"target_commands_executed"`
			RepositoryCodeExecuted         bool     `yaml:"repository_code_executed"`
			EvaluatorCheckoutBranchCreated bool     `yaml:"evaluator_checkout_branch_created"`
			SSBOrHostGitMutation           bool     `yaml:"ssb_or_host_git_mutation"`
			GeneratedADR                   bool     `yaml:"generated_adr"`
			PullRequestOpened              bool     `yaml:"pull_request_opened"`
			AdoptionClaimed                bool     `yaml:"adoption_claimed"`
		} `yaml:"execution_boundaries"`
		Files []struct {
			Path   string `yaml:"path"`
			SHA256 string `yaml:"sha256"`
		} `yaml:"files"`
	}
	if err := yaml.Load(runBytes, &run, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		t.Fatal(err)
	}
	if run.Schema != "ssb.dev/benchmark-run/v1" ||
		run.Target.Repository != "hoophq/hoop" ||
		run.Target.Commit != "3e0091b89fdaa3912f4f8e7b33fbb3104e47d71e" ||
		run.SSB.SourceCommit != "a1b7a9317bb8dbbc5a001d2fa2f535d2bdef6a06" {
		t.Fatalf("unexpected pinned run identity: %#v", run)
	}
	if run.Host.Name != "Codex CLI" || run.Host.Version != "0.145.0" || run.Host.Model != "gpt-5.6-sol" {
		t.Fatalf("host identity is not exact: %#v", run.Host)
	}
	if run.Proposal.ArtifactCount != 11 || !run.Proposal.Orientation ||
		run.Proposal.OrientationArtifacts != 0 ||
		run.Proposal.RetainedPath != "proposal/AGENTS.proposed.md" {
		t.Fatalf("proposal identity is incomplete: %#v", run.Proposal)
	}
	if run.Verification.Validate != "pass" || run.Verification.DryRun != "pass" ||
		run.Verification.Render != "pass" || run.Verification.Replay != "byte-identical" ||
		!run.Verification.CommandsDisplayedOnly {
		t.Fatalf("verification record is incomplete: %#v", run.Verification)
	}
	wantLenses := []string{"implementation", "planning", "verification"}
	gotLenses := make([]string, 0, len(run.Routing))
	for _, record := range run.Routing {
		if record.Status != "pass" {
			t.Fatalf("routing record is not passing: %#v", record)
		}
		gotLenses = append(gotLenses, record.Lens)
	}
	sort.Strings(gotLenses)
	if !reflect.DeepEqual(gotLenses, wantLenses) {
		t.Fatalf("routing lenses = %#v, want %#v", gotLenses, wantLenses)
	}
	if !run.ExecutionBoundaries.EvaluatorCheckoutBranchCreated ||
		len(run.ExecutionBoundaries.TargetCommandsExecuted) != 0 ||
		run.ExecutionBoundaries.RepositoryCodeExecuted || run.ExecutionBoundaries.GeneratedADR ||
		run.ExecutionBoundaries.SSBOrHostGitMutation || run.ExecutionBoundaries.PullRequestOpened ||
		run.ExecutionBoundaries.AdoptionClaimed {
		t.Fatalf("inert run recorded a forbidden action: %#v", run.ExecutionBoundaries)
	}

	bindings := make(map[string]string, len(run.Files))
	for _, binding := range run.Files {
		if !safeBenchmarkBindingPath(binding.Path) || binding.SHA256 == "" {
			t.Fatalf("unsafe or incomplete binding: %#v", binding)
		}
		if _, duplicate := bindings[binding.Path]; duplicate {
			t.Fatalf("duplicate file binding %q", binding.Path)
		}
		bindings[binding.Path] = binding.SHA256
	}
	gotBindingPaths := make([]string, 0, len(bindings))
	for bindingPath := range bindings {
		gotBindingPaths = append(gotBindingPaths, bindingPath)
	}
	sort.Strings(gotBindingPaths)
	wantBindingPaths := []string{
		"proposal/AGENTS.proposed.md",
		"routing/implementation.md",
		"routing/planning.md",
		"routing/verification.md",
	}
	if !reflect.DeepEqual(gotBindingPaths, wantBindingPaths) {
		t.Fatalf("retained evidence paths = %#v, want %#v", gotBindingPaths, wantBindingPaths)
	}
	observed := make(map[string]string)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path == filepath.Join(root, "run.yaml") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		blobPath := filepath.ToSlash(filepath.Join(benchmarkRelativeRoot, relative))
		data, err := readGitBlob(repoRoot, "HEAD:"+blobPath)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		observed[filepath.ToSlash(relative)] = "sha256:" + fmt.Sprintf("%x", sum)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(bindings, observed) {
		t.Fatalf("run bindings do not cover exact retained bytes:\nbindings=%#v\nobserved=%#v", bindings, observed)
	}
	agents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(run.Proposal.RetainedPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "<!-- source-digest: "+run.Proposal.SourceDigest+" -->") ||
		!strings.Contains(string(agents), "<!-- content-digest: "+run.Proposal.ContentDigest+" -->") ||
		bindings[run.Proposal.RetainedPath] != run.Proposal.OutputDigest {
		t.Fatal("retained projection does not match its source, content, and output digest bindings")
	}
}

func TestBenchmarkBindingPathSafetyIsPortable(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "proposal", path: "proposal/AGENTS.proposed.md", want: true},
		{name: "nested skill", path: "proposal/.agents/skills/add-feature-flag/SKILL.md", want: true},
		{name: "empty", path: "", want: false},
		{name: "current directory", path: ".", want: false},
		{name: "parent directory", path: "..", want: false},
		{name: "parent traversal", path: "../escape", want: false},
		{name: "embedded traversal", path: "inside/../escape", want: false},
		{name: "absolute", path: "/escape", want: false},
		{name: "network path", path: "//server/share", want: false},
		{name: "Windows volume", path: "C:/escape", want: false},
		{name: "Windows separators", path: `C:\escape`, want: false},
		{name: "alternate separator", path: `inside\file`, want: false},
		{name: "empty segment", path: "inside//file", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := safeBenchmarkBindingPath(test.path); got != test.want {
				t.Fatalf("safeBenchmarkBindingPath(%q) = %t, want %t", test.path, got, test.want)
			}
		})
	}
}

func safeBenchmarkBindingPath(value string) bool {
	return value != "" &&
		value != "." &&
		value != ".." &&
		!path.IsAbs(value) &&
		path.Clean(value) == value &&
		!strings.HasPrefix(value, "../") &&
		!strings.ContainsAny(value, `\:`)
}

func readGitBlob(root, object string) ([]byte, error) {
	command := exec.Command("git", "-C", root, "cat-file", "blob", object)
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, fmt.Errorf(
				"git cat-file blob %s: %s",
				object,
				strings.TrimSpace(string(exitError.Stderr)),
			)
		}
		return nil, fmt.Errorf("git cat-file blob %s: %w", object, err)
	}
	return output, nil
}

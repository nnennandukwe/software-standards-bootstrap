package evaluation_test

import (
	"bytes"
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

type actionableBenchmarkTool struct {
	SourceCommit string `yaml:"source_commit"`
	BinarySHA256 string `yaml:"binary_sha256"`
}

type actionableBenchmarkExclusions struct {
	Binary                int64 `yaml:"binary"`
	Generated             int64 `yaml:"generated"`
	Oversized             int64 `yaml:"oversized"`
	SecretLike            int64 `yaml:"secret_like"`
	Symlink               int64 `yaml:"symlink"`
	Submodule             int64 `yaml:"submodule"`
	VendorOrGeneratedTree int64 `yaml:"vendor_or_generated_tree"`
	NonRegular            int64 `yaml:"non_regular"`
}

type actionableBenchmarkInventory struct {
	SchemaVersion           int                           `yaml:"schema_version"`
	Complete                bool                          `yaml:"complete"`
	Truncated               bool                          `yaml:"truncated"`
	CandidateFiles          int64                         `yaml:"candidate_files"`
	CandidateBytes          int64                         `yaml:"candidate_bytes"`
	ScannedFiles            int64                         `yaml:"scanned_files"`
	ScannedBytes            int64                         `yaml:"scanned_bytes"`
	IndexedFiles            int64                         `yaml:"indexed_files"`
	IndexedBytes            int64                         `yaml:"indexed_bytes"`
	RemainingCandidateFiles int64                         `yaml:"remaining_candidate_files"`
	RemainingCandidateBytes int64                         `yaml:"remaining_candidate_bytes"`
	Excluded                actionableBenchmarkExclusions `yaml:"excluded"`
}

type actionableBenchmarkRetention struct {
	Keep          int    `yaml:"keep"`
	EditAndKeep   int    `yaml:"edit_and_keep"`
	Defer         int    `yaml:"defer"`
	Reject        int    `yaml:"reject"`
	Retained      int    `yaml:"retained"`
	Denominator   int    `yaml:"denominator"`
	ExactFraction string `yaml:"exact_fraction"`
	Result        string `yaml:"result"`
}

type actionableBenchmarkProposal struct {
	ArtifactCount            int                          `yaml:"artifact_count"`
	SemanticRules            int                          `yaml:"semantic_rules"`
	VerificationRecipes      int                          `yaml:"verification_recipes"`
	AgentSkills              int                          `yaml:"agent_skills"`
	AutomationProposals      int                          `yaml:"automation_proposals"`
	OrientationPresent       bool                         `yaml:"orientation_present"`
	OrientationArtifactCount int                          `yaml:"orientation_artifact_count"`
	EvidenceResolvable       int                          `yaml:"evidence_resolvable"`
	Retention                actionableBenchmarkRetention `yaml:"retention"`
}

type actionableBenchmarkProjection struct {
	SourceDigest  string `yaml:"source_digest"`
	ContentDigest string `yaml:"content_digest"`
	OutputSHA256  string `yaml:"output_sha256"`
}

type actionableBenchmarkFinalProjection struct {
	SourceDigest  string `yaml:"source_digest"`
	ContentDigest string `yaml:"content_digest"`
	OutputSHA256  string `yaml:"output_sha256"`
	RetainedPath  string `yaml:"retained_path"`
}

type actionableBenchmarkSourceBindings struct {
	InventorySHA256   string `yaml:"inventory_sha256"`
	ManifestSHA256    string `yaml:"manifest_sha256"`
	OrientationSHA256 string `yaml:"orientation_sha256"`
	ReportSHA256      string `yaml:"report_sha256"`
}

type actionableBenchmarkDecision struct {
	ID              string `yaml:"id"`
	Kind            string `yaml:"kind"`
	CanonicalPath   string `yaml:"canonical_path"`
	Decision        string `yaml:"decision"`
	PreReviewSHA256 string `yaml:"pre_review_sha256"`
	FinalSHA256     string `yaml:"final_sha256"`
	EditSummary     string `yaml:"edit_summary"`
}

type actionableBenchmarkFileBinding struct {
	Path        string `yaml:"path"`
	SHA256      string `yaml:"sha256"`
	GitBlobSHA1 string `yaml:"git_blob_sha1"`
}

type actionableBenchmarkRun struct {
	Schema string `yaml:"schema"`
	Target struct {
		Fixture        string `yaml:"fixture"`
		Repository     string `yaml:"repository"`
		URL            string `yaml:"url"`
		Commit         string `yaml:"commit"`
		AttachedBranch string `yaml:"attached_branch"`
	} `yaml:"target"`
	Dates struct {
		Generated string `yaml:"generated"`
		Reviewed  string `yaml:"reviewed"`
		Approved  string `yaml:"approved"`
		Retained  string `yaml:"retained"`
	} `yaml:"dates"`
	Tools struct {
		Generation       actionableBenchmarkTool `yaml:"generation"`
		ProjectionAndADR actionableBenchmarkTool `yaml:"projection_and_adr"`
	} `yaml:"tools"`
	Host struct {
		Name           string `yaml:"name"`
		Version        string `yaml:"version"`
		Model          string `yaml:"model"`
		Reasoning      string `yaml:"reasoning"`
		ApprovalPolicy string `yaml:"approval_policy"`
		SessionID      string `yaml:"session_id"`
		LocalRunID     string `yaml:"local_run_id"`
		OS             string `yaml:"os"`
		Architecture   string `yaml:"architecture"`
	} `yaml:"host"`
	Inventory               actionableBenchmarkInventory       `yaml:"inventory"`
	Proposal                actionableBenchmarkProposal        `yaml:"proposal"`
	PreReviewProjection     actionableBenchmarkProjection      `yaml:"pre_review_projection"`
	FinalProjection         actionableBenchmarkFinalProjection `yaml:"final_projection"`
	CanonicalSourceBindings actionableBenchmarkSourceBindings  `yaml:"canonical_source_bindings"`
	Verification            struct {
		RawInventoryMatchesFixtureFile bool   `yaml:"raw_inventory_matches_fixture_file"`
		NormalizedInventoryReplay      string `yaml:"normalized_inventory_replay"`
		Schema3Validate                string `yaml:"schema_3_validate"`
		DiagnosticsEmpty               bool   `yaml:"diagnostics_empty"`
		RenderDryRunPayload            string `yaml:"render_dry_run_payload"`
		RenderWrite                    string `yaml:"render_write"`
		CanonicalReplay                string `yaml:"canonical_replay"`
		CommandsDisplayedOnly          bool   `yaml:"commands_displayed_only"`
		RepositoryCodeExecuted         bool   `yaml:"repository_code_executed"`
		VerificationRecipesExecuted    bool   `yaml:"verification_recipes_executed"`
		GitMutationByHost              bool   `yaml:"git_mutation_by_host"`
		NetworkUsedBySSBOrRepoTools    bool   `yaml:"network_used_by_ssb_or_repository_tools"`
	} `yaml:"verification"`
	ADR struct {
		Status       string   `yaml:"status"`
		RetainedPath string   `yaml:"retained_path"`
		OutputSHA256 string   `yaml:"output_sha256"`
		ArtifactIDs  []string `yaml:"artifact_ids"`
	} `yaml:"adr"`
	Decisions []actionableBenchmarkDecision `yaml:"decisions"`
	Lifecycle struct {
		HumanApproval          string `yaml:"human_approval"`
		GeneratedGuidanceState string `yaml:"generated_guidance_state"`
		ADRState               string `yaml:"adr_state"`
		ReleaseClaimed         bool   `yaml:"release_claimed"`
		AdoptionClaimed        bool   `yaml:"adoption_claimed"`
		PublicationClaimed     bool   `yaml:"publication_claimed"`
	} `yaml:"lifecycle"`
	Files []actionableBenchmarkFileBinding `yaml:"files"`
}

type actionableBenchmarkDecisionExpectation struct {
	Kind         string
	Decision     string
	PreReview    string
	Final        string
	ManifestOnly bool
}

type actionableBenchmarkFixtureExpectation struct {
	Repository       string
	Commit           string
	SessionID        string
	Inventory        actionableBenchmarkInventory
	Rules            int
	Recipes          int
	Skills           int
	Keep             int
	EditAndKeep      int
	PreReview        actionableBenchmarkProjection
	Final            actionableBenchmarkProjection
	Sources          actionableBenchmarkSourceBindings
	ADRSHA256        string
	ProposalBlobSHA1 string
	ADRBlobSHA1      string
	Decisions        map[string]actionableBenchmarkDecisionExpectation
}

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

func TestV020ActionableBenchmarkRecord(t *testing.T) {
	repoRoot := repositoryRoot(t)
	relativeRoot := filepath.Join(
		"docs", "benchmarks", "results", "2026-08-17-v0.2.0-actionable",
	)
	root := filepath.Join(repoRoot, relativeRoot)
	fixtures := []string{"cobra", "django", "flask", "nextjs"}
	wantFiles := []string{"README.md"}
	wantDirectories := make([]string, 0, len(fixtures)*3)
	for _, fixture := range fixtures {
		wantDirectories = append(
			wantDirectories,
			fixture,
			fixture+"/adr",
			fixture+"/proposal",
		)
		wantFiles = append(
			wantFiles,
			fixture+"/adr/0001-actionable-standards.md",
			fixture+"/proposal/AGENTS.proposed.md",
			fixture+"/run.yaml",
		)
	}
	sort.Strings(wantDirectories)
	sort.Strings(wantFiles)

	var gotDirectories []string
	var gotFiles []string
	err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == root {
			return nil
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			gotDirectories = append(gotDirectories, relative)
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("benchmark record contains non-regular path %q", relative)
		}
		gotFiles = append(gotFiles, relative)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(gotDirectories)
	sort.Strings(gotFiles)
	if !reflect.DeepEqual(gotDirectories, wantDirectories) {
		t.Fatalf("benchmark directories = %#v, want %#v", gotDirectories, wantDirectories)
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("benchmark files = %#v, want exact 13-file tree %#v", gotFiles, wantFiles)
	}

	expectations := v020ActionableBenchmarkExpectations()
	const generationCommit = "a51431ef0faa417c6b9160d0bf9793b0f0db5538"
	const generationBinary = "sha256:78318c6ac51bb12a42776522a47011e1e0a752d8ad1606167373a4eb10ee8981"
	const projectionCommit = "908f990ab63d775ca31c0be3fc1632acde9b843f"
	const projectionBinary = "sha256:7ff82f5429ce565da648ad2124870e680c35542b5886256897704fd61acb2783"

	totalArtifacts := 0
	totalKeep := 0
	totalEditAndKeep := 0
	manifestOnlyEdits := 0
	for _, fixture := range fixtures {
		expected := expectations[fixture]
		fixtureRoot := filepath.Join(root, fixture)
		runBytes, err := os.ReadFile(filepath.Join(fixtureRoot, "run.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		var run actionableBenchmarkRun
		if err := yaml.Load(runBytes, &run, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
			t.Fatalf("parse %s run: %v", fixture, err)
		}

		if run.Schema != "ssb.dev/actionable-benchmark-run/v1" ||
			run.Target.Fixture != fixture ||
			run.Target.Repository != expected.Repository ||
			run.Target.URL != "https://github.com/"+expected.Repository ||
			run.Target.Commit != expected.Commit ||
			run.Target.AttachedBranch != "ssb-evaluation" {
			t.Fatalf("%s run identity is not exact: %#v", fixture, run.Target)
		}
		if run.Dates.Generated != "2026-08-14" || run.Dates.Reviewed != "2026-08-14" ||
			run.Dates.Approved != "2026-08-17" || run.Dates.Retained != "2026-08-17" {
			t.Fatalf("%s lifecycle dates are not exact: %#v", fixture, run.Dates)
		}
		if run.Tools.Generation.SourceCommit != generationCommit ||
			run.Tools.Generation.BinarySHA256 != generationBinary ||
			run.Tools.ProjectionAndADR.SourceCommit != projectionCommit ||
			run.Tools.ProjectionAndADR.BinarySHA256 != projectionBinary {
			t.Fatalf("%s tool bindings are not exact: %#v", fixture, run.Tools)
		}
		if run.Host.Name != "Codex CLI" || run.Host.Version != "0.145.0" ||
			run.Host.Model != "gpt-5.6-sol" || run.Host.Reasoning != "xhigh" ||
			run.Host.ApprovalPolicy != "never" || run.Host.SessionID != expected.SessionID ||
			run.Host.LocalRunID != "ssb-v020-acceptance-"+fixture+"-a51431e" ||
			run.Host.OS != "macOS 15.7.3 (24G419)" || run.Host.Architecture != "arm64" {
			t.Fatalf("%s host identity is not exact: %#v", fixture, run.Host)
		}
		if !reflect.DeepEqual(run.Inventory, expected.Inventory) {
			t.Fatalf("%s inventory = %#v, want %#v", fixture, run.Inventory, expected.Inventory)
		}
		if run.Proposal.ArtifactCount != len(expected.Decisions) ||
			run.Proposal.SemanticRules != expected.Rules ||
			run.Proposal.VerificationRecipes != expected.Recipes ||
			run.Proposal.AgentSkills != expected.Skills ||
			run.Proposal.AutomationProposals != 0 || !run.Proposal.OrientationPresent ||
			run.Proposal.OrientationArtifactCount != 0 ||
			run.Proposal.EvidenceResolvable != len(expected.Decisions) {
			t.Fatalf("%s proposal counts are not exact: %#v", fixture, run.Proposal)
		}
		retention := run.Proposal.Retention
		if retention.Keep != expected.Keep || retention.EditAndKeep != expected.EditAndKeep ||
			retention.Defer != 0 || retention.Reject != 0 ||
			retention.Retained != len(expected.Decisions) ||
			retention.Denominator != len(expected.Decisions) ||
			retention.ExactFraction != fmt.Sprintf("%d/%d", len(expected.Decisions), len(expected.Decisions)) ||
			retention.Result != "pass" {
			t.Fatalf("%s retention is not exact: %#v", fixture, retention)
		}
		if !reflect.DeepEqual(run.PreReviewProjection, expected.PreReview) ||
			run.FinalProjection.SourceDigest != expected.Final.SourceDigest ||
			run.FinalProjection.ContentDigest != expected.Final.ContentDigest ||
			run.FinalProjection.OutputSHA256 != expected.Final.OutputSHA256 ||
			run.FinalProjection.RetainedPath != "proposal/AGENTS.proposed.md" {
			t.Fatalf("%s projection digests are not exact", fixture)
		}
		if !reflect.DeepEqual(run.CanonicalSourceBindings, expected.Sources) {
			t.Fatalf("%s canonical source bindings are not exact", fixture)
		}
		if !run.Verification.RawInventoryMatchesFixtureFile ||
			run.Verification.NormalizedInventoryReplay != "byte-identical" ||
			run.Verification.Schema3Validate != "pass" || !run.Verification.DiagnosticsEmpty ||
			run.Verification.RenderDryRunPayload != "byte-identical" ||
			run.Verification.RenderWrite != "pass" ||
			run.Verification.CanonicalReplay != "no-write-current" ||
			!run.Verification.CommandsDisplayedOnly || run.Verification.RepositoryCodeExecuted ||
			run.Verification.VerificationRecipesExecuted || run.Verification.GitMutationByHost ||
			run.Verification.NetworkUsedBySSBOrRepoTools {
			t.Fatalf("%s verification boundary is incomplete: %#v", fixture, run.Verification)
		}

		decisionIDs := make([]string, 0, len(run.Decisions))
		seenDecisions := make(map[string]struct{}, len(run.Decisions))
		kindCounts := make(map[string]int)
		decisionCounts := make(map[string]int)
		for _, decision := range run.Decisions {
			want, ok := expected.Decisions[decision.ID]
			if !ok {
				t.Fatalf("%s has unexpected decision %q", fixture, decision.ID)
			}
			if _, duplicate := seenDecisions[decision.ID]; duplicate {
				t.Fatalf("%s has duplicate decision %q", fixture, decision.ID)
			}
			seenDecisions[decision.ID] = struct{}{}
			decisionIDs = append(decisionIDs, decision.ID)
			if decision.Kind != want.Kind || decision.Decision != want.Decision ||
				decision.PreReviewSHA256 != want.PreReview || decision.FinalSHA256 != want.Final {
				t.Fatalf("%s decision %q is not exact: %#v", fixture, decision.ID, decision)
			}
			if decision.CanonicalPath == "" || !safeBenchmarkBindingPath(decision.CanonicalPath) ||
				decision.EditSummary == "" {
				t.Fatalf("%s decision %q lacks a safe path or edit summary", fixture, decision.ID)
			}
			if decision.Kind == "automation" || strings.Contains(decision.ID, "orientation") {
				t.Fatalf("%s incorrectly treats orientation or automation as an approved artifact: %#v", fixture, decision)
			}
			switch decision.Kind {
			case "rule", "verification", "skill":
				kindCounts[decision.Kind]++
			default:
				t.Fatalf("%s decision %q has unknown kind %q", fixture, decision.ID, decision.Kind)
			}
			switch decision.Decision {
			case "keep", "edit-and-keep", "defer", "reject":
				decisionCounts[decision.Decision]++
			default:
				t.Fatalf("%s decision %q has unknown outcome %q", fixture, decision.ID, decision.Decision)
			}
			if decision.Decision == "keep" && decision.PreReviewSHA256 != decision.FinalSHA256 {
				t.Fatalf("%s keep decision %q changed canonical bytes", fixture, decision.ID)
			}
			if decision.Decision == "edit-and-keep" {
				canonicalBytesUnchanged := decision.PreReviewSHA256 == decision.FinalSHA256
				if canonicalBytesUnchanged != want.ManifestOnly {
					t.Fatalf("%s decision %q has an incorrect manifest-only classification", fixture, decision.ID)
				}
				if want.ManifestOnly {
					if !strings.Contains(strings.ToLower(decision.EditSummary), "manifest") {
						t.Fatalf("%s manifest-only decision %q omits its manifest edit", fixture, decision.ID)
					}
					manifestOnlyEdits++
				}
			}
		}
		if len(seenDecisions) != len(expected.Decisions) {
			t.Fatalf("%s decision count = %d, want %d", fixture, len(seenDecisions), len(expected.Decisions))
		}
		if kindCounts["rule"] != run.Proposal.SemanticRules ||
			kindCounts["verification"] != run.Proposal.VerificationRecipes ||
			kindCounts["skill"] != run.Proposal.AgentSkills ||
			run.Proposal.AutomationProposals != 0 {
			t.Fatalf("%s decision kinds do not reconcile with proposal counts", fixture)
		}
		if decisionCounts["keep"] != retention.Keep ||
			decisionCounts["edit-and-keep"] != retention.EditAndKeep ||
			decisionCounts["defer"] != retention.Defer ||
			decisionCounts["reject"] != retention.Reject ||
			len(run.Decisions) != retention.Denominator ||
			retention.Retained != decisionCounts["keep"]+decisionCounts["edit-and-keep"] {
			t.Fatalf("%s decisions do not reconcile with retention arithmetic", fixture)
		}
		sort.Strings(decisionIDs)
		adrIDs := append([]string(nil), run.ADR.ArtifactIDs...)
		sort.Strings(adrIDs)
		if !reflect.DeepEqual(adrIDs, decisionIDs) || run.ADR.Status != "Proposed" ||
			run.ADR.RetainedPath != "adr/0001-actionable-standards.md" ||
			run.ADR.OutputSHA256 != expected.ADRSHA256 {
			t.Fatalf("%s ADR metadata is not exact: %#v", fixture, run.ADR)
		}

		bindings := make(map[string]actionableBenchmarkFileBinding, len(run.Files))
		for _, binding := range run.Files {
			if !safeBenchmarkBindingPath(binding.Path) || binding.SHA256 == "" || binding.GitBlobSHA1 == "" {
				t.Fatalf("%s has unsafe or incomplete retained binding: %#v", fixture, binding)
			}
			if _, duplicate := bindings[binding.Path]; duplicate {
				t.Fatalf("%s has duplicate retained binding %q", fixture, binding.Path)
			}
			blobPath := filepath.ToSlash(filepath.Join(relativeRoot, fixture, binding.Path))
			committed, err := readGitBlob(repoRoot, "HEAD:"+blobPath)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(committed)
			if got := "sha256:" + fmt.Sprintf("%x", sum); got != binding.SHA256 {
				t.Fatalf("%s %s SHA-256 = %s, want %s", fixture, binding.Path, got, binding.SHA256)
			}
			blobSHA1, err := readGitObjectID(repoRoot, "HEAD:"+blobPath)
			if err != nil {
				t.Fatal(err)
			}
			if blobSHA1 != binding.GitBlobSHA1 {
				t.Fatalf("%s %s Git blob = %s, want %s", fixture, binding.Path, blobSHA1, binding.GitBlobSHA1)
			}
			bindings[binding.Path] = binding
		}
		wantBindingPaths := []string{
			"adr/0001-actionable-standards.md",
			"proposal/AGENTS.proposed.md",
		}
		gotBindingPaths := make([]string, 0, len(bindings))
		for bindingPath := range bindings {
			gotBindingPaths = append(gotBindingPaths, bindingPath)
		}
		sort.Strings(gotBindingPaths)
		if !reflect.DeepEqual(gotBindingPaths, wantBindingPaths) {
			t.Fatalf("%s retained bindings = %#v, want %#v", fixture, gotBindingPaths, wantBindingPaths)
		}
		if bindings[run.FinalProjection.RetainedPath].SHA256 != run.FinalProjection.OutputSHA256 ||
			bindings[run.FinalProjection.RetainedPath].GitBlobSHA1 != expected.ProposalBlobSHA1 ||
			bindings[run.ADR.RetainedPath].SHA256 != run.ADR.OutputSHA256 ||
			bindings[run.ADR.RetainedPath].GitBlobSHA1 != expected.ADRBlobSHA1 {
			t.Fatalf("%s retained bindings do not match projection and ADR metadata", fixture)
		}

		proposal, err := os.ReadFile(filepath.Join(fixtureRoot, filepath.FromSlash(run.FinalProjection.RetainedPath)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(proposal, []byte("<!-- source-digest: "+run.FinalProjection.SourceDigest+" -->")) ||
			!bytes.Contains(proposal, []byte("<!-- content-digest: "+run.FinalProjection.ContentDigest+" -->")) {
			t.Fatalf("%s retained projection does not carry its final source/content digests", fixture)
		}
		adr, err := os.ReadFile(filepath.Join(fixtureRoot, filepath.FromSlash(run.ADR.RetainedPath)))
		if err != nil {
			t.Fatal(err)
		}
		adrText := string(adr)
		if !strings.Contains(adrText, "- Status: Proposed") {
			t.Fatalf("%s retained ADR is not Proposed", fixture)
		}
		for _, id := range decisionIDs {
			if !strings.Contains(adrText, "(`"+id+"`)") {
				t.Errorf("%s retained ADR is missing artifact ID %q", fixture, id)
			}
		}
		lowerADR := strings.ToLower(adrText)
		if strings.Contains(lowerADR, "orientation") || strings.Contains(lowerADR, "automation") {
			t.Fatalf("%s retained ADR contains excluded orientation or automation content", fixture)
		}
		if run.Lifecycle.HumanApproval != "recorded" ||
			run.Lifecycle.GeneratedGuidanceState != "retained-proposal" ||
			run.Lifecycle.ADRState != "proposed" || run.Lifecycle.ReleaseClaimed ||
			run.Lifecycle.AdoptionClaimed || run.Lifecycle.PublicationClaimed {
			t.Fatalf("%s lifecycle boundaries are not exact: %#v", fixture, run.Lifecycle)
		}

		totalArtifacts += run.Proposal.ArtifactCount
		totalKeep += retention.Keep
		totalEditAndKeep += retention.EditAndKeep
	}
	if totalArtifacts != 21 || totalKeep != 10 || totalEditAndKeep != 11 || manifestOnlyEdits != 3 {
		t.Fatalf(
			"aggregate decisions = artifacts %d, keep %d, edit-and-keep %d, manifest-only edits %d",
			totalArtifacts,
			totalKeep,
			totalEditAndKeep,
			manifestOnlyEdits,
		)
	}

	readmeBytes, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme := strings.Join(strings.Fields(string(readmeBytes)), " ")
	for _, required := range []string{
		"approved all 21 recommended keep/edit-and-keep decisions",
		"does not claim that the fixture repositories adopted the proposals",
		"exactly 13 files",
		"no raw inventory, manifest, report, orientation, canonical artifact, transcript, command log, or replay bundle",
		"SSB-PROPAGATION-EDIT-20260817",
		"validated 4 artifacts (3 rules, 1 skill, 0 recipes, 0 automation)",
		"90c2b7d74fffbc4b3d1669624464ac352336bf189f70dc77fc0c8bef0291b498",
		"d9987b87d2a87ade019a113059e90a674e022ac9b4eb1b29ca0c1a9489390304",
		"Release evidence | Not complete",
	} {
		if !strings.Contains(readme, required) {
			t.Errorf("v0.2.0 benchmark README missing %q", required)
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

func v020ActionableBenchmarkExpectations() map[string]actionableBenchmarkFixtureExpectation {
	return map[string]actionableBenchmarkFixtureExpectation{
		"cobra": {
			Repository: "spf13/cobra",
			Commit:     "adbc8813901bba65827259daa8e22ff94ec1f30e",
			SessionID:  "01a001ed-1402-7ac2-983d-f79d2eb8cd6d",
			Inventory: actionableBenchmarkInventory{
				SchemaVersion: 2, Complete: true,
				CandidateFiles: 66, CandidateBytes: 705271,
				ScannedFiles: 66, ScannedBytes: 705271,
				IndexedFiles: 65, IndexedBytes: 631792,
				Excluded: actionableBenchmarkExclusions{Binary: 1},
			},
			Rules: 3, Skills: 2, Keep: 3, EditAndKeep: 2,
			PreReview: actionableBenchmarkProjection{
				SourceDigest:  "sha256:1430a8917b00835474072bc04477e9f98c1eb3f80fd99e49aa9e55cf1bb5bfc5",
				ContentDigest: "sha256:b3348b42ee32f961e22060826d360a4565857e7a97e542b95b399c137d921576",
				OutputSHA256:  "sha256:7e8c0f70d634b06ec91fedde901a24b62746dcfc050e9f08c50e8d6db9c2959d",
			},
			Final: actionableBenchmarkProjection{
				SourceDigest:  "sha256:9031b05ab3cccbc2d9feff8ad16d17d563e600e488036a6f3230e5fdf4bde8de",
				ContentDigest: "sha256:68d7ad02a1fbc9d60962c92f8dc426a5b5644ba6c2f6b815b7e4b157cb1e9f5c",
				OutputSHA256:  "sha256:3c5a23aa45a4cfaf445459a536a6f863a55e253b0df4d360de50e067d2b3c612",
			},
			Sources: actionableBenchmarkSourceBindings{
				InventorySHA256:   "sha256:98e65eea07a092693e3de5a5cc945996bd13b8a970a5370dbc631760ac43ddf3",
				ManifestSHA256:    "sha256:41194f22e2367205657d6a9b8cd132167f318d95561a683591ab2df63651efdf",
				OrientationSHA256: "sha256:7764c47bd376259b9d4e6f12a81bff3eb9d3204ec1f36164e31dbc9d158fa317",
				ReportSHA256:      "sha256:2a3e5c6d336a1875684a556449dfb5090181eb638aef727cf3fbb56dea31703f",
			},
			ADRSHA256:        "sha256:e915c6b56ab3bc0e29e5468bf4e6b1cd37f0de434ebfe5cdae1a66403898fec9",
			ProposalBlobSHA1: "4c2134f5cd376d8741e3b76b106615bbfdb6b94b",
			ADRBlobSHA1:      "e83ab8cc9593e26efdaf696a81d688fa4c596040",
			Decisions: map[string]actionableBenchmarkDecisionExpectation{
				"cover-go-code-changes-with-tests": {
					Kind:      "rule",
					Decision:  "edit-and-keep",
					PreReview: "sha256:1ebd242e55e47184f82f0d756ea411c64adeb852b7e52888577d22938193f853",
					Final:     "sha256:bf7375d626434d34c6fe4b97dc7351cef04866491ec0827753029319a57accb6",
				},
				"preserve-compatibility-apis-until-cobra-v2": {
					Kind:      "rule",
					Decision:  "keep",
					PreReview: "sha256:f1c7695b6d045381977d186b20a6f235f95384f2d534e385f9aebba64d978d28",
					Final:     "sha256:f1c7695b6d045381977d186b20a6f235f95384f2d534e385f9aebba64d978d28",
				},
				"keep-build-constraints-compatible-with-go-1-15": {
					Kind:      "rule",
					Decision:  "keep",
					PreReview: "sha256:064bbd3de8837ea86bfe7fccd5b0b7dd0415e4650579f8c99fda907f39095a33",
					Final:     "sha256:064bbd3de8837ea86bfe7fccd5b0b7dd0415e4650579f8c99fda907f39095a33",
				},
				"change-shell-completion-behavior": {
					Kind:      "skill",
					Decision:  "edit-and-keep",
					PreReview: "sha256:81c493d64710209aa8dad84caef584451eead903b1d9a624e9be6f59b98aa61f",
					Final:     "sha256:48714e636d31d17db00b0cb0206dcbde92c2156ed6d0c4658f739557df6b843b",
				},
				"change-document-generation": {
					Kind:      "skill",
					Decision:  "keep",
					PreReview: "sha256:788195ec559ab2b634eea75742cdd6e172e30f442d2db68c45a24218302be120",
					Final:     "sha256:788195ec559ab2b634eea75742cdd6e172e30f442d2db68c45a24218302be120",
				},
			},
		},
		"flask": {
			Repository: "pallets/flask",
			Commit:     "36e4a824f340fdee7ed50937ba8e7f6bc7d17f81",
			SessionID:  "01a001ed-1402-7aa3-83a5-53439c9fcdb5",
			Inventory: actionableBenchmarkInventory{
				SchemaVersion: 2, Complete: true,
				CandidateFiles: 235, CandidateBytes: 1814782,
				ScannedFiles: 235, ScannedBytes: 1814782,
				IndexedFiles: 230, IndexedBytes: 1474850,
				Excluded: actionableBenchmarkExclusions{Binary: 5, SecretLike: 1},
			},
			Rules: 3, Keep: 2, EditAndKeep: 1,
			PreReview: actionableBenchmarkProjection{
				SourceDigest:  "sha256:83224f59b37cf91ce6b6adcd9ad33ea082972ea4a49eecd542acf2ffb19cda1e",
				ContentDigest: "sha256:034f2733c8a7ff9b35fc38cc6f9982ea526ebc4d4cc546658fde34d2b1473949",
				OutputSHA256:  "sha256:d289c5d2db51a92c50449abe208f6a3a42b8fe341e8182424756e864e62d31bb",
			},
			Final: actionableBenchmarkProjection{
				SourceDigest:  "sha256:af86995fa2dbddfde27f460f2ad24b7a38df5f83817680ed76779d7ec4789fe7",
				ContentDigest: "sha256:8d18558410b8dc61f77b58d60a6673e56d7b499a7da2d63669a65a7e67652bf7",
				OutputSHA256:  "sha256:25419cf7ab9b4d4566a7e8a11e82b20a7fc78e966836fa09c9e778fafc71090f",
			},
			Sources: actionableBenchmarkSourceBindings{
				InventorySHA256:   "sha256:ed8013e208958ef189a329644ea1ddfcc4d73657c17c9a27f25576d4cf7e84ef",
				ManifestSHA256:    "sha256:f8b563509dd0f69dc986829ca08560685d8675ec86564b790b3b869dfb51e40e",
				OrientationSHA256: "sha256:64346f12999afe4f22dc48ab55964aa4d1bc3c975ad262fff9ddf4503d7fd645",
				ReportSHA256:      "sha256:3988be67bcbf6af373640784985535232a8e1d50ba495aafcb7caff6c317ec45",
			},
			ADRSHA256:        "sha256:0a68f2e52d028e22733534e701e2d0c1027bbae08f47edbe1b716edf4e49638f",
			ProposalBlobSHA1: "78a71200a95305b82f970c2dbd4073542d397201",
			ADRBlobSHA1:      "9b703d235c237647de1171d17c71ea0906be9eee",
			Decisions: map[string]actionableBenchmarkDecisionExpectation{
				"keep-sansio-free-of-io-and-globals": {
					Kind:      "rule",
					Decision:  "keep",
					PreReview: "sha256:48e8f2e3543b8bfc522961d52323a55f32f6e989c3eaf1dccbb825109b1270a5",
					Final:     "sha256:48e8f2e3543b8bfc522961d52323a55f32f6e989c3eaf1dccbb825109b1270a5",
				},
				"keep-change-evidence-synchronized": {
					Kind:      "rule",
					Decision:  "keep",
					PreReview: "sha256:5235913b2acfff2e7e590f5eed2050471d53f02b40adb5ca9b3787f3e15b9aed",
					Final:     "sha256:5235913b2acfff2e7e590f5eed2050471d53f02b40adb5ca9b3787f3e15b9aed",
				},
				"deprecate-public-apis-before-removal": {
					Kind:      "rule",
					Decision:  "edit-and-keep",
					PreReview: "sha256:108a3a20c1253a222b372f5ef962ca8f1eebc3152145284c307fa600b7ff30c2",
					Final:     "sha256:541988a1d471693c440bc131d14275a72ab4a9ec9825d208797f7b71146aee17",
				},
			},
		},
		"django": {
			Repository: "django/django",
			Commit:     "50c2b7c83661a61da48f78dd0130fc3cbf8ed39f",
			SessionID:  "01a001ed-1402-7ce3-812b-c555405b0e14",
			Inventory: actionableBenchmarkInventory{
				SchemaVersion: 2, Complete: true,
				CandidateFiles: 7001, CandidateBytes: 45506636,
				ScannedFiles: 7001, ScannedBytes: 45506636,
				IndexedFiles: 5619, IndexedBytes: 36820618,
				Excluded: actionableBenchmarkExclusions{Binary: 1382, Symlink: 4, VendorOrGeneratedTree: 72},
			},
			Rules: 3, Recipes: 2, Skills: 1, Keep: 3, EditAndKeep: 3,
			PreReview: actionableBenchmarkProjection{
				SourceDigest:  "sha256:ca275421a671428b78ab8da95ed2d591f46d5ce0f0e866545d332a30dd51ae5a",
				ContentDigest: "sha256:83fc5bd03984e43bf7c22967b1bee1184127c267cb4fd4b8df5198d22a070c0e",
				OutputSHA256:  "sha256:f516585aaf953677ead0d58ec27f6da04a87aae3c4de83087933dc944d6736ea",
			},
			Final: actionableBenchmarkProjection{
				SourceDigest:  "sha256:4b26e1cd9a90669832c351290e9cb6b943689ce91a3e8990064bf29ce45c6c7b",
				ContentDigest: "sha256:f4a4f2bc62a59ff77948935716b68ad1e083950117a7325a22db4fc36487b09e",
				OutputSHA256:  "sha256:7f2f1a954170dcf97b92928d001383d1db88d9bdec2b12e19481af26ff2f6878",
			},
			Sources: actionableBenchmarkSourceBindings{
				InventorySHA256:   "sha256:7dfbdc561cf24d8d2bef2807e4e1bf4b81266740deff444f36da9dee230d585c",
				ManifestSHA256:    "sha256:613a50858fc33960cdc89e7fbdc153555b32802d8e7fea48b21cec6fdbc34232",
				OrientationSHA256: "sha256:cca9e978c48fed075b258aab5c0872c5e9db3f73e8fa4d3cca651b3354247bd6",
				ReportSHA256:      "sha256:f457ad1d3ed3694172e20e2f40d3e4b732dbfd243d055561f6648a2bcb664b95",
			},
			ADRSHA256:        "sha256:732890fc9f253e158b3548c290c1d78c8fa7ca5410664d6b4892754cceb05b5c",
			ProposalBlobSHA1: "8b7de42218e83e4b29e65634adf4a918c7e545d9",
			ADRBlobSHA1:      "dc0674e287a02dce9b73500a12109c019bccdca2",
			Decisions: map[string]actionableBenchmarkDecisionExpectation{
				"cover-behavior-changes-with-tests": {
					Kind:      "rule",
					Decision:  "edit-and-keep",
					PreReview: "sha256:f644735aaba7d7cbef5ca1584c39d737d90a6ee9c512d064ebdca0f672d63005",
					Final:     "sha256:e66a5c2dde1fd40f2a63cd92fb58dada4bbb4760cac4810b3a111ee64c75053e",
				},
				"document-user-visible-behavior-changes": {
					Kind:      "rule",
					Decision:  "keep",
					PreReview: "sha256:c2a3fea6a219a11baec9bb1761e50097859fd9e9a685123fdd5c4a963b5b2640",
					Final:     "sha256:c2a3fea6a219a11baec9bb1761e50097859fd9e9a685123fdd5c4a963b5b2640",
				},
				"gate-database-tests-by-feature": {
					Kind:         "rule",
					Decision:     "edit-and-keep",
					PreReview:    "sha256:a11d395dd402da0615089cc41df81c14d40be2cb5e5ebc963fecf8d07ecd4b09",
					Final:        "sha256:a11d395dd402da0615089cc41df81c14d40be2cb5e5ebc963fecf8d07ecd4b09",
					ManifestOnly: true,
				},
				"run-django-tests": {
					Kind:      "verification",
					Decision:  "keep",
					PreReview: "sha256:2d56b7f4a3dabe9a6851272d2ffae91a7954341daf415a85d502025c0cf93daf",
					Final:     "sha256:2d56b7f4a3dabe9a6851272d2ffae91a7954341daf415a85d502025c0cf93daf",
				},
				"check-documentation": {
					Kind:      "verification",
					Decision:  "keep",
					PreReview: "sha256:2232bf85dc5bdc8018756b5652fbf02d7db4f1fefa3d122f8016f4e0795ee645",
					Final:     "sha256:2232bf85dc5bdc8018756b5652fbf02d7db4f1fefa3d122f8016f4e0795ee645",
				},
				"deprecate-django-feature": {
					Kind:      "skill",
					Decision:  "edit-and-keep",
					PreReview: "sha256:47a6b94e3932a77c66038cc8d1f54ebda9ab893010b72758bbfa8cbc2c53d8ae",
					Final:     "sha256:e38af415eddf949a352bba461581181ef1fa7654cc3b1a3c00296c75cb8dbd78",
				},
			},
		},
		"nextjs": {
			Repository: "vercel/next.js",
			Commit:     "6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd",
			SessionID:  "01a001ed-1402-7cb2-b63c-2a727ed402ee",
			Inventory: actionableBenchmarkInventory{
				SchemaVersion: 2, Complete: true,
				CandidateFiles: 29073, CandidateBytes: 111110455,
				ScannedFiles: 29073, ScannedBytes: 111110455,
				IndexedFiles: 28403, IndexedBytes: 88643646,
				Excluded: actionableBenchmarkExclusions{
					Binary: 652, Generated: 18, Oversized: 21,
					SecretLike: 113, Symlink: 29, VendorOrGeneratedTree: 1060,
				},
			},
			Rules: 6, Recipes: 1, Keep: 2, EditAndKeep: 5,
			PreReview: actionableBenchmarkProjection{
				SourceDigest:  "sha256:2501fc2478dda081c8742a9ba5a71c3059b9966a2541adf038189af905e6576f",
				ContentDigest: "sha256:0b0250942c9b67d8b287504e5ecb3b00d8866ec85e495c1cdf48c71fc25af92e",
				OutputSHA256:  "sha256:c0a44f6a0907c8004c4064ad314dab91f4eaf0f19d9c47a9e3bf8b1f87f30703",
			},
			Final: actionableBenchmarkProjection{
				SourceDigest:  "sha256:c021daf472ad2b93c8b067c2dd2be080b0da869a575873705c4285c2835ab6fd",
				ContentDigest: "sha256:7e3799840105bed29f2a1a9dd14bb1f8f2955b363ce46a1fe0e4d5d82d16b20d",
				OutputSHA256:  "sha256:dd8f32cfd759835cba10b8a657393f57b2c6b0e239e1264e12095fbf316eafb1",
			},
			Sources: actionableBenchmarkSourceBindings{
				InventorySHA256:   "sha256:791e4b443dad3b80eed3cd85e194f0c09fee2a2e6f222ec24456a8a18addc180",
				ManifestSHA256:    "sha256:6475d9ca1329d71984bbc423dc80c9bab40eb49e8b45405c15b78d64d0262cea",
				OrientationSHA256: "sha256:d7bb23d9dc4ce563d375f03e0be3fe066eaa17925f0b5326abb7638599dbce24",
				ReportSHA256:      "sha256:5c3ca8970b65da592eb04a777a132222dd4eaeec409312fca4af36b000cefb33",
			},
			ADRSHA256:        "sha256:0a5bf3268cd50593201aa2b4f8baeb3b77c2abdbed2b8721077d224147fdcf12",
			ProposalBlobSHA1: "1e28ec98a341eaa982609727fb4b88c1c3a75ecf",
			ADRBlobSHA1:      "6d188d376ae30e7ce1ad03c1cbc873ce5359a816",
			Decisions: map[string]actionableBenchmarkDecisionExpectation{
				"read-path-readmes-before-editing": {
					Kind:      "rule",
					Decision:  "keep",
					PreReview: "sha256:be048e7435c2d7b80738ad4858d84f98cf3d1e1bbc5ac9de9186113e47dfa047",
					Final:     "sha256:be048e7435c2d7b80738ad4858d84f98cf3d1e1bbc5ac9de9186113e47dfa047",
				},
				"keep-pnpm-security-settings-synchronized": {
					Kind:         "rule",
					Decision:     "edit-and-keep",
					PreReview:    "sha256:3eee29c50d2a4fcbde9c06c880980d7746624fdbc2ccc7855b19d6c92823ab47",
					Final:        "sha256:3eee29c50d2a4fcbde9c06c880980d7746624fdbc2ccc7855b19d6c92823ab47",
					ManifestOnly: true,
				},
				"wire-next-runtime-flags-across-all-consumers": {
					Kind:      "rule",
					Decision:  "keep",
					PreReview: "sha256:37abee50d0ad00e19528b2cdfd6039673eadc92dcf4f0b8bdd64689f6488f22b",
					Final:     "sha256:37abee50d0ad00e19528b2cdfd6039673eadc92dcf4f0b8bdd64689f6488f22b",
				},
				"treat-unfiltered-internal-request-headers-as-forgeable": {
					Kind:         "rule",
					Decision:     "edit-and-keep",
					PreReview:    "sha256:2e64fe11fcafc4b913382b673e1879beb39df959d4b50f9bf50f8b61e77cb564",
					Final:        "sha256:2e64fe11fcafc4b913382b673e1879beb39df959d4b50f9bf50f8b61e77cb564",
					ManifestOnly: true,
				},
				"use-next-test-fixtures-and-polling-conventions": {
					Kind:      "rule",
					Decision:  "edit-and-keep",
					PreReview: "sha256:8bcbc0ce0293c1589f59fdefdeabbefcceb5c3aec82d42c86a93ee4333aae745",
					Final:     "sha256:94aa0075d1e0f7b719aa0ff70b6fe6e4233e3403eff96c3ac6c8d754b98a2ef2",
				},
				"match-next-test-mode-and-bundler": {
					Kind:      "rule",
					Decision:  "edit-and-keep",
					PreReview: "sha256:60a93f3a1ee5e593bc1558328ed45007f64185f3c56a06ad01cf329cd235234a",
					Final:     "sha256:e4736371248c4cbd0c260c8a7d9cd0a9ef5670dd77cf6a2c77bc410bf707f5b9",
				},
				"verify-next-package-types": {
					Kind:      "verification",
					Decision:  "edit-and-keep",
					PreReview: "sha256:a117d93e4602245dd99639d4a0321604771fb63e86b33e026452d732b0c86134",
					Final:     "sha256:502ec3b0e3db2d8b8354f9198dcecb583ef1952468a6afe95e5098c5d759500c",
				},
			},
		},
	}
}

func readGitObjectID(root, object string) (string, error) {
	command := exec.Command("git", "-C", root, "rev-parse", "--verify", object)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"git rev-parse --verify %s: %s",
			object,
			strings.TrimSpace(string(output)),
		)
	}
	return strings.TrimSpace(string(output)), nil
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

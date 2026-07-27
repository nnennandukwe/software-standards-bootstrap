package prune_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

func TestCreateReviewWritesImmutableContext(t *testing.T) {
	root := lifecycleRepository(t)
	ws, err := workspace.OpenForPrune(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := prune.CreateReview(context.Background(), ws, prune.ContextOptions{
		ReviewID:        "review-one",
		Capabilities:    capabilityProfile(t),
		InventoryLimits: inventory.DefaultLimits(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Context.ContextDigest == "" ||
		result.ContextPath != ".software-standards/reviews/review-one/context.json" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, result.ContextPath)); err != nil {
		t.Fatal(err)
	}
	for _, snapshot := range []string{
		result.Context.CapabilityProfilePath,
		"inputs/capability/host-run.json",
	} {
		if _, err := os.Stat(filepath.Join(root, ".software-standards", "reviews", "review-one", snapshot)); err != nil {
			t.Fatalf("missing durable capability snapshot %s: %v", snapshot, err)
		}
	}
	if _, err := prune.CreateReview(context.Background(), ws, prune.ContextOptions{
		ReviewID:        "review-one",
		Capabilities:    capabilityProfile(t),
		InventoryLimits: inventory.DefaultLimits(),
	}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want immutable review collision", err)
	}
}

func TestCreateReviewWriteFailureLeavesNoPartialReview(t *testing.T) {
	root := lifecycleRepository(t)
	writeFile(t, filepath.Join(root, ".software-standards", "reviews"), "blocks directory\n")
	ws, err := workspace.OpenForPrune(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = prune.CreateReview(context.Background(), ws, prune.ContextOptions{
		ReviewID:        "review-one",
		Capabilities:    capabilityProfile(t),
		InventoryLimits: inventory.DefaultLimits(),
	})
	if err == nil {
		t.Fatal("expected review directory write failure")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".software-standards", "reviews", "review-one")); statErr == nil {
		t.Fatalf("partial review remains: %v", statErr)
	}
}

func TestApproveBindsExactProposalAndRequiresEveryDecision(t *testing.T) {
	root := lifecycleRepository(t)
	review := createReviewWithProposal(t, root, false)
	_, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule"},
		Now:      func() time.Time { return time.Unix(1, 0).UTC() },
	})
	if err == nil || !strings.Contains(err.Error(), "orphan-skill") {
		t.Fatalf("error = %v, want missing decision", err)
	}

	event, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule"},
		Rejected: []string{"orphan-skill"},
		Now:      func() time.Time { return time.Unix(1, 0).UTC() },
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != prune.EventApproved || event.ProposalDigest != review.ProposalDigest {
		t.Fatalf("unexpected approval: %#v", event)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule"},
		Rejected: []string{"orphan-skill"},
	}); err == nil || !strings.Contains(err.Error(), "already") {
		t.Fatalf("error = %v, want single approval event", err)
	}
}

func TestApproveNeverApprovesUnableToDetermine(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, true)
	_, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	})
	if err == nil || !strings.Contains(err.Error(), "unable-to-determine") {
		t.Fatalf("error = %v, want UTD approval rejection", err)
	}
}

func TestValidateRequiresVerificationForEachActionableDisposition(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	review.Proposal.Actions[1].Disposition = prune.DispositionRemove
	review.Proposal.Actions[1].RequiredVerification = nil
	data, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, diagnostics, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) == 0 ||
		!strings.Contains(diagnostics[0].Message, "every actionable disposition") {
		t.Fatalf("diagnostics = %#v, want per-action verification requirement", diagnostics)
	}
}

func TestApplyDefaultsToDryRunThenAppliesApprovedRemoval(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	review.Proposal.Actions[1].Disposition = prune.DispositionRemove
	data, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}

	skillPath := filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md")
	dryRun, err := prune.Apply(context.Background(), root, prune.ApplyOptions{ReviewID: "review-one"})
	if err != nil {
		t.Fatal(err)
	}
	if !dryRun.DryRun || len(dryRun.Changes) != 1 {
		t.Fatalf("unexpected dry run: %#v", dryRun)
	}
	for _, lockPath := range []string{
		filepath.Join(review.Root, ".transition.lock"),
		filepath.Join(root, ".software-standards", ".prune-mutation.lock"),
	} {
		if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
			t.Fatalf("dry run created persistent lock %s: %v", lockPath, err)
		}
	}
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("dry run changed skill: %v", err)
	}

	applied, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied.DryRun || len(applied.Changes) != 1 {
		t.Fatalf("unexpected application: %#v", applied)
	}
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatalf("removed skill still exists: %v", err)
	}
}

func TestApplyRemovesTheCompleteTrackedSkillBundle(t *testing.T) {
	root := lifecycleRepository(t)
	supportingPaths := []string{
		".agents/skills/orphan-skill/references/checklist.md",
		".agents/skills/orphan-skill/scripts/preflight.sh",
	}
	for _, relative := range supportingPaths {
		writeFile(t, filepath.Join(root, filepath.FromSlash(relative)), relative+"\n")
	}
	git(t, root, "add", ".agents/skills/orphan-skill")
	git(t, root, "commit", "-m", "add skill supporting files")
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(review.Context.Artifacts[0].SupportingFiles); got != len(supportingPaths) {
		t.Fatalf("supporting file count = %d, want %d", got, len(supportingPaths))
	}
	review.Proposal.Actions[1].Disposition = prune.DispositionRemove
	data, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	dryRun, err := prune.Apply(context.Background(), root, prune.ApplyOptions{ReviewID: "review-one"})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(dryRun.Changes); got != 1+len(supportingPaths) {
		t.Fatalf("dry-run change count = %d, want %d: %#v", got, 1+len(supportingPaths), dryRun)
	}
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	}); err != nil {
		t.Fatal(err)
	}
	for _, relative := range append([]string{".agents/skills/orphan-skill/SKILL.md"}, supportingPaths...) {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Fatalf("removed skill bundle file %s still exists: %v", relative, err)
		}
	}
}

func TestApplyReplacesTheCompleteTrackedSkillBundle(t *testing.T) {
	root := lifecycleRepository(t)
	staleRelative := ".agents/skills/orphan-skill/references/stale.md"
	writeFile(t, filepath.Join(root, filepath.FromSlash(staleRelative)), "stale\n")
	git(t, root, "add", ".agents/skills/orphan-skill")
	git(t, root, "commit", "-m", "add stale skill support")
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	entrySource := "candidates/orphan-skill/SKILL.md"
	entryPath := filepath.Join(review.Root, filepath.FromSlash(entrySource))
	writeFile(t, entryPath, `---
name: orphan-skill
description: A fully replaced repository skill.
metadata:
  topic: maintainability
---
Use the replacement workflow.
`)
	supportSource := "candidates/orphan-skill/references/new.md"
	supportPath := filepath.Join(review.Root, filepath.FromSlash(supportSource))
	writeFile(t, supportPath, "new support\n")
	review.Proposal.Actions[1].Disposition = prune.DispositionUpdate
	review.Proposal.Actions[1].Target = &prune.CandidateRef{
		Kind:       prune.ArtifactSkill,
		ID:         "orphan-skill",
		TargetPath: ".agents/skills/orphan-skill/SKILL.md",
		SourcePath: entrySource,
		SHA256:     fileDigest(t, entryPath),
		Mode:       "100755",
		SupportingFiles: []prune.CandidateFileRef{{
			TargetPath: ".agents/skills/orphan-skill/references/new.md",
			SourcePath: supportSource,
			SHA256:     fileDigest(t, supportPath),
			Mode:       "100644",
		}},
	}
	data, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	result, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changes) != 3 {
		t.Fatalf("changes = %#v, want entrypoint replacement, stale removal, and support write", result.Changes)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(staleRelative))); !os.IsNotExist(err) {
		t.Fatalf("stale support remains: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".agents", "skills", "orphan-skill", "references", "new.md"))
	if err != nil || string(got) != "new support\n" {
		t.Fatalf("replacement support = %q, %v", got, err)
	}
	entryInfo, err := os.Stat(filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if entryInfo.Mode().Perm() != 0o755 {
		t.Fatalf("replacement entrypoint mode = %v; want 0755", entryInfo.Mode().Perm())
	}
}

func TestApplyFailsClosedWhenTrackedSourceDrifts(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".software-standards", "rules", "keep-rule.md"), "drift\n")
	_, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	})
	if err == nil || !strings.Contains(err.Error(), "tracked") {
		t.Fatalf("error = %v, want tracked-drift block", err)
	}
}

func TestApplyRejectsDanglingRuleToSkillGraph(t *testing.T) {
	root := lifecycleRepository(t)
	rulePath := filepath.Join(root, ".software-standards", "rules", "keep-rule.md")
	ruleData, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, rulePath, strings.Replace(string(ruleData), "id: keep-rule\n", "id: keep-rule\nrelated_skills: [orphan-skill]\n", 1))
	git(t, root, "add", rulePath)
	git(t, root, "commit", "-m", "link skill")
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	review.Proposal.Actions[1].Disposition = prune.DispositionRemove
	data, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
	}); err == nil || !strings.Contains(err.Error(), "references missing skill") {
		t.Fatalf("error = %v, want resulting-graph block", err)
	}
}

func TestApplyRejectsMalformedCompleteCandidateBeforeWriting(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	candidateRelative := "candidates/keep-rule/replacement.md"
	candidatePath := filepath.Join(review.Root, filepath.FromSlash(candidateRelative))
	writeFile(t, candidatePath, "---\nid: replacement\n---\ninvalid\n")
	review.Proposal.Actions[0].Disposition = prune.DispositionUpdate
	review.Proposal.Actions[0].Target = &prune.CandidateRef{
		Kind:       prune.ArtifactRule,
		ID:         "replacement",
		TargetPath: ".software-standards/rules/replacement.md",
		SourcePath: candidateRelative,
		SHA256:     fileDigest(t, candidatePath),
		Mode:       "100644",
	}
	data, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
	}); err == nil || !strings.Contains(err.Error(), "rule contract") {
		t.Fatalf("error = %v, want candidate-contract block", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".software-standards", "rules", "replacement.md")); !os.IsNotExist(err) {
		t.Fatalf("invalid candidate was written: %v", err)
	}
}

func TestVerifyRequiresExactExternalReceiptAndKeepsStatesSeparate(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	review.Proposal.Actions[1].Disposition = prune.DispositionRemove
	review.Proposal.Actions[0].RequiredVerification = []prune.CheckRequirement{{
		ID: "repository-tests", Command: "go test ./...",
	}}
	data, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one", Write: true,
	}); err != nil {
		t.Fatal(err)
	}
	status, _, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied || status.Rendered || status.ADRRecorded || status.Verified {
		t.Fatalf("states collapsed after application: %#v", status)
	}

	receipts := t.TempDir()
	evidence := filepath.Join(receipts, "logs", "go-test.txt")
	writeFile(t, evidence, "PASS\n")
	writeFile(t, filepath.Join(receipts, "repository-tests.yaml"), `schema: ssb.dev/prune-check-receipt/v1
review_id: review-one
proposal_digest: `+reviewDigest(t, root)+`
check_id: repository-tests
command: go test ./wrong
status: passed
observed_at: 2026-07-27T18:00:00Z
evidence:
  - path: logs/go-test.txt
    sha256: `+fileDigest(t, evidence)+`
`)
	if _, err := prune.Verify(context.Background(), root, "review-one", receipts, nil); err == nil ||
		!strings.Contains(err.Error(), "does not match") {
		t.Fatalf("error = %v, want exact receipt mismatch", err)
	}
}

func TestTransitionsCannotSkipApplication(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.RecordTransition(root, "review-one", prune.EventRendered, map[string]string{
		"path": "AGENTS.md",
	}, nil); err == nil || !strings.Contains(err.Error(), "application") {
		t.Fatalf("error = %v, want skipped-state block", err)
	}
}

func TestRecoverRestoresJournaledBytesWithoutInferringApplication(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	review.Proposal.Actions[1].Disposition = prune.DispositionRemove
	proposalData, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), proposalData, 0o644); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md")
	original, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(skillPath); err != nil {
		t.Fatal(err)
	}
	journalPath := filepath.Join(root, ".software-standards", "reviews", "review-one", "application-journal.json")
	writeFile(t, journalPath, `{
  "schema": "ssb.dev/prune-application-journal/v1",
  "review_id": "review-one",
  "proposal_digest": "sha256:`+strings.Repeat("1", 64)+`",
  "entries": [{
    "path": ".agents/skills/orphan-skill/SKILL.md",
    "existed": true,
    "mode": 420,
    "content": "`+base64.StdEncoding.EncodeToString(original)+`"
  }]
}
`)
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	review, _, err = prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	journalData, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	journalData = []byte(strings.Replace(string(journalData), "sha256:"+strings.Repeat("1", 64), review.ProposalDigest, 1))
	if err := os.WriteFile(journalPath, journalData, 0o644); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(review.Root, ".transition.lock"), "crashed owner\n")
	if err := prune.Recover(context.Background(), root, "review-one", true); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(original) {
		t.Fatalf("recovery did not restore exact bytes:\n%s", restored)
	}
	if _, err := os.Stat(journalPath); !os.IsNotExist(err) {
		t.Fatalf("recovery journal remains: %v", err)
	}
	status, _, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	if status.Applied {
		t.Fatal("recovery inferred an application event")
	}
}

func TestRecoverRejectsJournalPathEscape(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	journalPath := filepath.Join(root, ".software-standards", "reviews", "review-one", "application-journal.json")
	writeFile(t, journalPath, `{
  "schema": "ssb.dev/prune-application-journal/v1",
  "review_id": "review-one",
  "proposal_digest": "`+reviewDigest(t, root)+`",
  "entries": [{"path": "../outside", "existed": false}]
}
`)
	if err := prune.Recover(context.Background(), root, "review-one", false); err == nil ||
		!strings.Contains(err.Error(), "non-canonical") {
		t.Fatalf("error = %v, want path-escape block", err)
	}
}

func TestRecoverRejectsCanonicalPathOutsideApprovedPlan(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(root, ".software-standards", "rules", "keep-rule.md")
	original, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	journalPath := filepath.Join(root, ".software-standards", "reviews", "review-one", "application-journal.json")
	writeFile(t, journalPath, `{
  "schema": "ssb.dev/prune-application-journal/v1",
  "review_id": "review-one",
  "proposal_digest": "`+reviewDigest(t, root)+`",
  "entries": [{
    "path": ".software-standards/rules/keep-rule.md",
    "existed": true,
    "mode": 420,
    "content": "`+base64.StdEncoding.EncodeToString(original)+`"
  }]
}
`)
	if err := prune.Recover(context.Background(), root, "review-one", false); err == nil ||
		!strings.Contains(err.Error(), "approved operation") {
		t.Fatalf("error = %v, want approved-plan binding block", err)
	}
}

func TestRecoverRejectsAndPreservesPostCrashHumanEdit(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	review.Proposal.Actions[1].Disposition = prune.DispositionRemove
	proposalData, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), proposalData, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md")
	original, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	humanEdit := "human edit after crash\n"
	writeFile(t, skillPath, humanEdit)
	journalPath := filepath.Join(review.Root, "application-journal.json")
	writeFile(t, journalPath, `{
  "schema": "ssb.dev/prune-application-journal/v1",
  "review_id": "review-one",
  "proposal_digest": "`+reviewDigest(t, root)+`",
  "entries": [{
    "path": ".agents/skills/orphan-skill/SKILL.md",
    "existed": true,
    "mode": 420,
    "content": "`+base64.StdEncoding.EncodeToString(original)+`"
  }]
}
`)
	if err := prune.Recover(context.Background(), root, "review-one", false); err == nil ||
		!strings.Contains(err.Error(), "neither approved prestate nor poststate") {
		t.Fatalf("error = %v, want third-state recovery block", err)
	}
	after, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != humanEdit {
		t.Fatalf("recovery overwrote human edit: %q", after)
	}
}

type preparedReview struct {
	ProposalDigest string
}

func reviewDigest(t *testing.T, root string) string {
	t.Helper()
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	return review.ProposalDigest
}

func createReviewWithProposal(t *testing.T, root string, unknownSkill bool) preparedReview {
	t.Helper()
	ws, err := workspace.OpenForPrune(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	provenance := filepath.Join(t.TempDir(), "provenance.yaml")
	skillOrigin := prune.OriginUserAuthored
	if unknownSkill {
		skillOrigin = prune.OriginUnknown
	}
	writeFile(t, provenance, `schema: ssb.dev/artifact-provenance/v1
artifacts:
  - path: .software-standards/rules/keep-rule.md
    sha256: `+fileDigest(t, filepath.Join(root, ".software-standards", "rules", "keep-rule.md"))+`
    origin: generated
    declaration: Generated.
  - path: .agents/skills/orphan-skill/SKILL.md
    sha256: `+fileDigest(t, filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md"))+`
    origin: `+skillOrigin+`
    declaration: Declared.
`)
	created, err := prune.CreateReview(context.Background(), ws, prune.ContextOptions{
		ReviewID:        "review-one",
		Capabilities:    capabilityProfile(t),
		Provenance:      provenance,
		InventoryLimits: inventory.DefaultLimits(),
	})
	if err != nil {
		t.Fatal(err)
	}
	disposition := prune.DispositionKeep
	if unknownSkill {
		disposition = prune.DispositionUnableToDetermine
	}
	proposal := prune.Proposal{
		Schema:        prune.ProposalSchema,
		ReviewID:      "review-one",
		ContextDigest: created.Context.ContextDigest,
		Actions: []prune.Action{
			validAction("keep-rule", prune.DispositionKeep, created.Context.Artifacts[1], contextFileDigest(created.Context, "README.md")),
			validAction("orphan-skill", disposition, created.Context.Artifacts[0], contextFileDigest(created.Context, "README.md")),
		},
	}
	if disposition == prune.DispositionUnableToDetermine {
		proposal.Actions[1].RepositoryEvidence = nil
		proposal.Actions[1].CapabilityRefs = nil
		proposal.Actions[1].UnresolvedQuestions = []string{"Who authored and adopted this skill?"}
	}
	data, err := yaml.Marshal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".software-standards", "reviews", "review-one", "proposal.yaml"), string(data))
	loaded, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	return preparedReview{ProposalDigest: loaded.ProposalDigest}
}

func validAction(id, disposition string, artifact prune.Artifact, evidenceDigest string) prune.Action {
	return prune.Action{
		ID:          id,
		Disposition: disposition,
		Sources:     []prune.ArtifactRef{prune.Reference(artifact)},
		Rationale:   "Current repository and pinned host evidence support this disposition.",
		Confidence:  prune.ConfidenceHigh,
		RepositoryEvidence: []prune.EvidenceRef{{
			Path:   "README.md",
			Lines:  "1-1",
			SHA256: evidenceDigest,
		}},
		CapabilityRefs: []string{"agent-skills-discovery"},
		RequiredVerification: []prune.CheckRequirement{{
			ID: "review-check", Command: "ssb validate --repo .",
		}},
	}
}

func contextFileDigest(context prune.Context, path string) string {
	for _, file := range context.Inventory.Files {
		if file.Path == path {
			return file.SHA256
		}
	}
	panic("fixture evidence missing from context")
}

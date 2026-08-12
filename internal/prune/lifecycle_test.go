package prune_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
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

func TestLoadReviewRejectsSymlinkedSnapshotDirectories(t *testing.T) {
	for _, snapshotKind := range []string{"capability", "provenance"} {
		t.Run(snapshotKind, func(t *testing.T) {
			root := lifecycleRepository(t)
			createReviewWithProposal(t, root, false)
			review, _, err := prune.LoadReview(root, "review-one")
			if err != nil {
				t.Fatal(err)
			}
			snapshotDir := filepath.Join(review.Root, "inputs", snapshotKind)
			externalDir := filepath.Join(t.TempDir(), snapshotKind)
			if err := os.Rename(snapshotDir, externalDir); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(externalDir, snapshotDir); err != nil {
				t.Skipf("symlinks are unavailable: %v", err)
			}
			if _, _, err := prune.LoadReview(root, "review-one"); err == nil ||
				!strings.Contains(err.Error(), "symlink") {
				t.Fatalf("error = %v, want symlinked %s snapshot block", err, snapshotKind)
			}
		})
	}
}

func TestReviewLifecycleRejectsSymlinkedReviewRootBeforeExternalWrites(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	externalReview := filepath.Join(t.TempDir(), "review-one")
	if err := os.Rename(review.Root, externalReview); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalReview, review.Root); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	assertSymlinkBlock := func(label string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "review path") ||
			!strings.Contains(err.Error(), "symlink") {
			t.Fatalf("%s error = %v, want review-root symlink block", label, err)
		}
	}
	_, _, err = prune.LoadReview(root, "review-one")
	assertSymlinkBlock("load", err)
	_, err = prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	})
	assertSymlinkBlock("approve", err)
	_, err = prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	})
	assertSymlinkBlock("apply", err)
	err = prune.Recover(context.Background(), root, "review-one", true)
	assertSymlinkBlock("recover", err)
	_, _, err = prune.ReviewStatus(root, "review-one")
	assertSymlinkBlock("status", err)

	for _, relative := range []string{".transition.lock", "events.jsonl", "application-journal.json"} {
		if _, err := os.Lstat(filepath.Join(externalReview, relative)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("external review path %s was created or modified: %v", relative, err)
		}
	}
}

func TestTransitionRevalidatesReviewRootBeforeEventAndLockCleanup(t *testing.T) {
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
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	}); err != nil {
		t.Fatal(err)
	}
	transition, err := prune.BeginTransition(root, "review-one", prune.EventADR, nil)
	if err != nil {
		t.Fatal(err)
	}

	eventPath := filepath.Join(review.Root, "events.jsonl")
	lockPath := filepath.Join(review.Root, ".transition.lock")
	eventsBefore, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatal(err)
	}
	lockBefore, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	externalReview := filepath.Join(t.TempDir(), "review-one")
	if err := os.Rename(review.Root, externalReview); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalReview, review.Root); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	event, err := transition.Complete(struct {
		Path    string `json:"path"`
		Created bool   `json:"created"`
		DryRun  bool   `json:"dry_run"`
	}{
		Path:    "docs/adr/0001-prune.md",
		Created: true,
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") || event.EventDigest != "" {
		t.Fatalf("event = %#v, error = %v, want pre-event symlink block", event, err)
	}
	if err := transition.Cancel(); err == nil ||
		(!strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "escapes")) {
		t.Fatalf("cancel error = %v, want anchored cleanup block", err)
	}
	eventsAfter, err := os.ReadFile(filepath.Join(externalReview, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lockAfter, err := os.ReadFile(filepath.Join(externalReview, ".transition.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(eventsAfter) != string(eventsBefore) {
		t.Fatal("external event log changed after review-root swap")
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("external transition lock changed after review-root swap")
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

func TestApproveAllowsZeroRulePoststateWithAtomicReportUpdate(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	review.Proposal.Actions[0].Disposition = prune.DispositionRemove
	data, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule"},
		Rejected: []string{"orphan-skill"},
	}); err != nil {
		t.Fatalf("approve zero-rule poststate: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(review.Root, "events.jsonl")); err != nil {
		t.Fatalf("approval event was not recorded: %v", err)
	}
}

func TestReviewStatusAcceptsApprovedZeroRulePoststate(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	review.Proposal.Actions[0].Disposition = prune.DispositionRemove
	proposalData, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(review.Root, "proposal.yaml"), proposalData, 0o644); err != nil {
		t.Fatal(err)
	}
	review, diagnostics, err := prune.LoadReview(root, "review-one")
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("load review: diagnostics=%#v, error=%v", diagnostics, err)
	}
	payload, err := json.Marshal(prune.ApprovalPayload{
		Approved: []string{"keep-rule"},
		Rejected: []string{"orphan-skill"},
	})
	if err != nil {
		t.Fatal(err)
	}
	event := prune.Event{
		Schema:         prune.EventSchema,
		ID:             "approved-001",
		ReviewID:       "review-one",
		Kind:           prune.EventApproved,
		RecordedAt:     "2026-07-27T18:00:00Z",
		BaselineCommit: review.Context.BaselineCommit,
		ContextDigest:  review.Context.ContextDigest,
		ProposalDigest: review.ProposalDigest,
		Payload:        payload,
	}
	event.EventDigest = testCanonicalDigest(t, event)
	eventData, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(review.Root, "events.jsonl"),
		append(eventData, '\n'),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	status, diagnostics, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatalf("status failed on a well-formed legacy bundle: %v", err)
	}
	if !status.Approved || len(diagnostics) != 0 {
		t.Fatalf("status=%#v diagnostics=%#v, want valid zero-rule approval", status, diagnostics)
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
	if !dryRun.DryRun || len(dryRun.Changes) != 2 {
		t.Fatalf("unexpected dry run: %#v", dryRun)
	}
	if dryRun.Changes[0].Path != ".agents/skills/orphan-skill/SKILL.md" ||
		dryRun.Changes[1].Path != ".software-standards/report.md" {
		t.Fatalf("dry run did not bind the artifact and report atomically: %#v", dryRun.Changes)
	}
	dryRunJSON, err := json.Marshal(dryRun)
	if err != nil {
		t.Fatal(err)
	}
	var dryRunPayload struct {
		PlanDigest string `json:"plan_digest"`
		Changes    []struct {
			Prestate struct {
				Exists bool `json:"exists"`
			} `json:"prestate"`
			Poststate struct {
				Exists bool `json:"exists"`
			} `json:"poststate"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(dryRunJSON, &dryRunPayload); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dryRunPayload.PlanDigest, "sha256:") ||
		len(dryRunPayload.Changes) != 2 ||
		!dryRunPayload.Changes[0].Prestate.Exists ||
		dryRunPayload.Changes[0].Poststate.Exists ||
		!dryRunPayload.Changes[1].Prestate.Exists ||
		!dryRunPayload.Changes[1].Poststate.Exists {
		t.Fatalf("dry run lacks canonical plan identity and states: %s", dryRunJSON)
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
	if applied.DryRun || len(applied.Changes) != 2 {
		t.Fatalf("unexpected application: %#v", applied)
	}
	appliedJSON, err := json.Marshal(applied)
	if err != nil {
		t.Fatal(err)
	}
	var appliedPayload struct {
		PlanDigest string `json:"plan_digest"`
	}
	if err := json.Unmarshal(appliedJSON, &appliedPayload); err != nil {
		t.Fatal(err)
	}
	if appliedPayload.PlanDigest != dryRunPayload.PlanDigest {
		t.Fatalf("write plan %q differs from dry-run plan %q", appliedPayload.PlanDigest, dryRunPayload.PlanDigest)
	}
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatalf("removed skill still exists: %v", err)
	}
	report, err := os.ReadFile(filepath.Join(root, ".software-standards", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(report), "id: orphan-skill") {
		t.Fatalf("report retained removed skill entry:\n%s", report)
	}
	repo, err := workspace.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if _, diagnostics, err := rulepack.ValidateRetainedPack(context.Background(), repo); err != nil {
		t.Fatal(err)
	} else if len(diagnostics) != 0 {
		t.Fatalf("atomically updated pack is invalid: %#v", diagnostics)
	}
}

func TestApplySplitRemovalUpdatesManifestAndCleansRelationshipsAtomically(t *testing.T) {
	root := splitLifecycleRepository(t)
	manifestPath := filepath.Join(root, ".software-standards", "manifest.yaml")
	reportPath := filepath.Join(root, ".software-standards", "report.md")
	inventoryPath := filepath.Join(root, ".software-standards", "inventory.json")
	reportBefore, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	inventoryBefore, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatal(err)
	}
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
	dryRun, err := prune.Apply(context.Background(), root, prune.ApplyOptions{ReviewID: "review-one"})
	if err != nil {
		t.Fatal(err)
	}
	if len(dryRun.Changes) != 2 ||
		dryRun.Changes[0].Path != ".agents/skills/orphan-skill/SKILL.md" ||
		dryRun.Changes[1].Path != ".software-standards/manifest.yaml" {
		t.Fatalf("split removal plan did not bind artifact and manifest: %#v", dryRun.Changes)
	}
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	}); err != nil {
		t.Fatal(err)
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest rulepack.Manifest
	if err := yaml.Load(manifestData, &manifest, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Artifacts) != 1 || manifest.Artifacts[0].ID != "keep-rule" ||
		len(manifest.Artifacts[0].RelatedArtifactIDs) != 0 {
		t.Fatalf("split manifest retained removal or dangling relationship: %#v", manifest)
	}
	if reportAfter, err := os.ReadFile(reportPath); err != nil || !bytes.Equal(reportAfter, reportBefore) {
		t.Fatalf("split prune changed human report: error=%v\nbefore=%s\nafter=%s", err, reportBefore, reportAfter)
	}
	if inventoryAfter, err := os.ReadFile(inventoryPath); err != nil || !bytes.Equal(inventoryAfter, inventoryBefore) {
		t.Fatalf("split prune changed inventory: error=%v", err)
	}
	repo, err := workspace.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if _, diagnostics, err := rulepack.ValidateRetainedPack(context.Background(), repo); err != nil {
		t.Fatal(err)
	} else if len(diagnostics) != 0 {
		t.Fatalf("split pack invalid after atomic removal: %#v", diagnostics)
	}
}

func TestApplySplitRuleUpdateRefreshesOnlyPrimaryDigest(t *testing.T) {
	root := splitLifecycleRepository(t)
	configureRuleUpdateCandidate(t, root)
	manifestPath := filepath.Join(root, ".software-standards", "manifest.yaml")
	manifestBeforeData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifestBefore rulepack.Manifest
	if err := yaml.Load(manifestBeforeData, &manifestBefore, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
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
	if len(dryRun.Changes) != 2 ||
		dryRun.Changes[0].Path != ".software-standards/manifest.yaml" ||
		dryRun.Changes[1].Path != ".software-standards/rules/keep-rule.md" {
		t.Fatalf("split update plan did not bind rule and manifest: %#v", dryRun.Changes)
	}
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	}); err != nil {
		t.Fatal(err)
	}
	manifestAfterData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifestAfter rulepack.Manifest
	if err := yaml.Load(manifestAfterData, &manifestAfter, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		t.Fatal(err)
	}
	before := manifestBefore.Artifacts[0]
	after := manifestAfter.Artifacts[0]
	rulePath := filepath.Join(root, ".software-standards", "rules", "keep-rule.md")
	if after.SHA256 != fileDigest(t, rulePath) || after.SHA256 == before.SHA256 ||
		after.Category != before.Category || after.Directive != before.Directive ||
		!reflect.DeepEqual(after.Lenses, before.Lenses) ||
		!reflect.DeepEqual(after.Scopes, before.Scopes) ||
		after.Derivation != before.Derivation || !reflect.DeepEqual(after.Evidence, before.Evidence) ||
		manifestAfter.Inventory != manifestBefore.Inventory || manifestAfter.Report != manifestBefore.Report {
		t.Fatalf("split update changed immutable metadata or missed digest: before=%#v after=%#v", manifestBefore, manifestAfter)
	}
	repo, err := workspace.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if _, diagnostics, err := rulepack.ValidateRetainedPack(context.Background(), repo); err != nil {
		t.Fatal(err)
	} else if len(diagnostics) != 0 {
		t.Fatalf("split pack invalid after atomic update: %#v", diagnostics)
	}
}

func TestAppliedReviewRemainsAuditableAfterCommittingChanges(t *testing.T) {
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
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	}); err != nil {
		t.Fatal(err)
	}

	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "apply governed removal")

	status, diagnostics, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatalf("durable review became unreadable after commit: %v", err)
	}
	if len(diagnostics) != 0 || !status.Applied {
		t.Fatalf("status=%#v diagnostics=%#v, want durable applied review", status, diagnostics)
	}
}

func TestZeroArtifactRemovalClearsProjectionAndRecordsReplayableRender(t *testing.T) {
	root := lifecycleRepository(t)
	repo, err := workspace.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	pack := rulepack.Pack{
		BaselineCommit: repo.Baseline(),
		Report: rulepack.Report{
			Artifacts: []rulepack.ManifestArtifact{{
				ID:   "keep-rule",
				Kind: "rule",
				Path: ".software-standards/rules/keep-rule.md",
			}},
		},
		Rules: []rulepack.Rule{{
			Schema:     rulepack.RuleSchema,
			ID:         "keep-rule",
			Title:      "Keep the rule",
			Category:   "maintainability",
			Lenses:     []rulepack.Lens{{Kind: "base"}},
			Directive:  "always",
			Scopes:     []string{"**/*"},
			Derivation: "extracted",
			Evidence: []rulepack.Evidence{{
				Role:  "declares",
				Path:  "README.md",
				Lines: "1-1",
			}},
			SourcePath: ".software-standards/rules/keep-rule.md",
			Body:       "Keep the rule.\n",
		}},
	}
	if _, err := render.Apply(repo, pack, false); err != nil {
		t.Fatalf("render initial pack: %v", err)
	}

	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	for index := range review.Proposal.Actions {
		review.Proposal.Actions[index].Disposition = prune.DispositionRemove
	}
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
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	}); err != nil {
		t.Fatal(err)
	}

	repo, err = workspace.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	pack, packDiagnostics, err := rulepack.ValidateRetainedPack(context.Background(), repo)
	if err != nil || len(packDiagnostics) != 0 {
		t.Fatalf("validate zero-artifact pack: diagnostics=%#v error=%v", packDiagnostics, err)
	}
	transition, err := prune.BeginTransition(root, "review-one", prune.EventRendered, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := render.Apply(repo, pack, false)
	if err != nil {
		_ = transition.Cancel()
		t.Fatal(err)
	}
	if !result.Changed {
		_ = transition.Cancel()
		t.Fatalf("zero-artifact render did not remove the stale managed section: %#v", result)
	}
	if _, err := transition.Complete(result); err != nil {
		t.Fatal(err)
	}

	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(agents), render.StartMarker) ||
		strings.Contains(string(agents), "Keep the rule.") {
		t.Fatalf("zero-artifact render retained stale guidance:\n%s", agents)
	}
	status, diagnostics, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatalf("zero-artifact render event is not replayable: %v", err)
	}
	if len(diagnostics) != 0 || !status.Rendered {
		t.Fatalf("status=%#v diagnostics=%#v, want replayable render state", status, diagnostics)
	}
}

func TestZeroArtifactRenderWithoutTargetRecordsReplayableNoOp(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	for index := range review.Proposal.Actions {
		review.Proposal.Actions[index].Disposition = prune.DispositionRemove
	}
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
	if _, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	}); err != nil {
		t.Fatal(err)
	}

	repo, err := workspace.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.ValidateRetainedPack(context.Background(), repo)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("validate zero-artifact pack: diagnostics=%#v error=%v", diagnostics, err)
	}
	transition, err := prune.BeginTransition(root, "review-one", prune.EventRendered, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := render.Apply(repo, pack, false)
	if err != nil {
		_ = transition.Cancel()
		t.Fatal(err)
	}
	if result.Changed || result.Exists ||
		result.SourceDigest == "" || result.ContentDigest == "" || result.OutputDigest == "" {
		_ = transition.Cancel()
		t.Fatalf("unexpected zero-artifact no-op render: %#v", result)
	}
	if _, err := transition.Complete(result); err != nil {
		t.Fatal(err)
	}
	status, statusDiagnostics, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatalf("zero-artifact no-op render event is not replayable: %v", err)
	}
	if len(statusDiagnostics) != 0 || !status.Rendered {
		t.Fatalf("status=%#v diagnostics=%#v, want replayable render state", status, statusDiagnostics)
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
	if got := len(dryRun.Changes); got != 2+len(supportingPaths) {
		t.Fatalf("dry-run change count = %d, want %d: %#v", got, 2+len(supportingPaths), dryRun)
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
  category: maintainability
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
		Mode:       "100644",
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
}

func TestApproveRejectsSkillCategoryDriftFromReport(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	source := "candidates/orphan-skill/SKILL.md"
	candidatePath := filepath.Join(review.Root, filepath.FromSlash(source))
	writeFile(t, candidatePath, `---
name: orphan-skill
description: A replacement that changes the manifest-owned category.
metadata:
  category: correctness
---
Use the replacement workflow.
`)
	review.Proposal.Actions[1].Disposition = prune.DispositionUpdate
	review.Proposal.Actions[1].Target = &prune.CandidateRef{
		Kind:       prune.ArtifactSkill,
		ID:         "orphan-skill",
		TargetPath: ".agents/skills/orphan-skill/SKILL.md",
		SourcePath: source,
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
	}); err == nil ||
		!strings.Contains(err.Error(), "changes metadata.category from maintainability to correctness") {
		t.Fatalf("error = %v, want report category drift block", err)
	}
	if _, err := os.Lstat(filepath.Join(review.Root, "events.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("category drift recorded approval event: %v", err)
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

func TestApprovalRejectsRetainedRuleWithUnreachableBaselineBeforeMutation(t *testing.T) {
	root := lifecycleRepository(t)
	reportPath := filepath.Join(root, ".software-standards", "report.md")
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	const marker = "baseline_commit: "
	start := strings.Index(string(reportData), marker)
	if start < 0 {
		t.Fatal("fixture report has no baseline_commit")
	}
	valueStart := start + len(marker)
	valueEnd := valueStart + strings.Index(string(reportData[valueStart:]), "\n")
	recordedBaseline := string(reportData[valueStart:valueEnd])
	corrupted := strings.ReplaceAll(string(reportData), recordedBaseline, strings.Repeat("0", 40))
	if err := os.WriteFile(reportPath, []byte(corrupted), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", reportPath)
	git(t, root, "commit", "-m", "record unreachable report baseline")

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
	before, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err == nil ||
		!strings.Contains(err.Error(), "reachable ancestor") ||
		!strings.Contains(err.Error(), "evidence cannot be verified") {
		t.Fatalf("error = %v, want pre-approval retained-baseline block", err)
	}
	after, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("skill was removed before retained-rule validation: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("skill changed before retained-rule validation completed")
	}
	status, _, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	if status.Approved || status.Applied {
		t.Fatal("unreachable retained baseline created a lifecycle event")
	}
}

func TestApprovalRejectsLegacyRuleOwnedRelationship(t *testing.T) {
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
	}); err == nil || !strings.Contains(err.Error(), "field related_skills not found") {
		t.Fatalf("error = %v, want unsupported rule-owned relationship block", err)
	}
}

func TestValidateAndApprovalRejectMalformedCandidateBeforeEventOrWrite(t *testing.T) {
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
		ID:         "keep-rule",
		TargetPath: ".software-standards/rules/keep-rule.md",
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
	repo, err := workspace.Open(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if _, diagnostics, err := prune.ValidateReview(context.Background(), repo, "review-one"); err != nil {
		t.Fatal(err)
	} else if len(diagnostics) == 0 ||
		!strings.Contains(diagnostics[len(diagnostics)-1].Message, "rule contract") {
		t.Fatalf("diagnostics = %#v, want candidate-contract failure", diagnostics)
	}
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err == nil || !errors.Is(err, prune.ErrValidation) ||
		!strings.Contains(err.Error(), "rule contract") {
		t.Fatalf("error = %v, want pre-approval candidate-contract block", err)
	}
	if _, err := os.Lstat(filepath.Join(review.Root, "events.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("malformed candidate recorded an event: %v", err)
	}
	originalRule, err := os.ReadFile(filepath.Join(root, ".software-standards", "rules", "keep-rule.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(originalRule) == "---\nid: replacement\n---\ninvalid\n" {
		t.Fatal("invalid candidate was written")
	}
}

func TestValidateApproveAndApplyShareCandidateDigestCheck(t *testing.T) {
	for _, stage := range []string{"validate", "approve", "apply"} {
		t.Run(stage, func(t *testing.T) {
			root := lifecycleRepository(t)
			candidatePath := configureRuleUpdateCandidate(t, root)
			if stage == "apply" {
				if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
					ReviewID: "review-one",
					Approved: []string{"keep-rule", "orphan-skill"},
				}); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(t, candidatePath, "candidate changed after proposal\n")

			var message string
			switch stage {
			case "validate":
				repo, err := workspace.Open(context.Background(), root)
				if err != nil {
					t.Fatal(err)
				}
				_, diagnostics, err := prune.ValidateReview(
					context.Background(),
					repo,
					"review-one",
				)
				if err != nil {
					t.Fatal(err)
				}
				if len(diagnostics) == 0 {
					t.Fatal("validate accepted candidate digest drift")
				}
				message = diagnostics[len(diagnostics)-1].Message
			case "approve":
				_, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
					ReviewID: "review-one",
					Approved: []string{"keep-rule", "orphan-skill"},
				})
				if err == nil || !errors.Is(err, prune.ErrValidation) {
					t.Fatalf("approve error = %v, want validation failure", err)
				}
				message = err.Error()
			case "apply":
				_, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
					ReviewID: "review-one",
				})
				if err == nil || !errors.Is(err, prune.ErrValidation) {
					t.Fatalf("apply error = %v, want validation failure", err)
				}
				message = err.Error()
			}
			if !strings.Contains(message, "candidate digest does not match") {
				t.Fatalf("%s message = %q, want shared candidate-digest rejection", stage, message)
			}
		})
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

func TestVerifyBindsReceiptToAppliedPoststate(t *testing.T) {
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
		Now:      func() time.Time { return time.Date(2026, 7, 27, 18, 0, 0, 0, time.UTC) },
	}); err != nil {
		t.Fatal(err)
	}
	applied, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
		Now:      func() time.Time { return time.Date(2026, 7, 27, 18, 1, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	review, _, err = prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	var applicationEventDigest string
	for _, event := range review.Events {
		if event.Kind == prune.EventApplied {
			applicationEventDigest = event.EventDigest
		}
	}
	if applicationEventDigest == "" {
		t.Fatal("application event digest is missing")
	}

	receipts := t.TempDir()
	evidence := filepath.Join(receipts, "logs", "ssb-validate.txt")
	writeFile(t, evidence, "PASS\n")
	receiptPath := filepath.Join(receipts, "review-check.yaml")
	writeFile(t, receiptPath, `schema: ssb.dev/prune-check-receipt/v1
review_id: review-one
proposal_digest: `+review.ProposalDigest+`
application_event_digest: `+applicationEventDigest+`
plan_digest: `+applied.PlanDigest+`
check_id: review-check
command: ssb validate --repo .
status: passed
observed_at: 2026-07-27T18:02:00Z
evidence:
  - path: logs/ssb-validate.txt
    sha256: `+fileDigest(t, evidence)+`
`)

	rulePath := filepath.Join(root, ".software-standards", "rules", "keep-rule.md")
	originalRule, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, rulePath, "human edit after application\n")
	if _, err := prune.Verify(context.Background(), root, "review-one", receipts, nil); err == nil ||
		!strings.Contains(err.Error(), "poststate") {
		t.Fatalf("error = %v, want poststate drift block", err)
	}
	status, _, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	if status.Verified {
		t.Fatal("poststate drift created a verification event")
	}

	if err := os.WriteFile(rulePath, originalRule, 0o644); err != nil {
		t.Fatal(err)
	}
	receiptData, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, receiptPath, strings.Replace(string(receiptData), "18:02:00Z", "18:00:00Z", 1))
	if _, err := prune.Verify(context.Background(), root, "review-one", receipts, nil); err == nil ||
		!strings.Contains(err.Error(), "predates") {
		t.Fatalf("error = %v, want pre-application receipt block", err)
	}
	if err := os.WriteFile(receiptPath, receiptData, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Dir(receipts))
	result, err := prune.Verify(
		context.Background(),
		root,
		"review-one",
		filepath.Base(receipts),
		func() time.Time { return time.Date(2026, 7, 27, 18, 3, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Receipts) != 1 {
		t.Fatalf("verification result = %#v", result)
	}
	if _, err := prune.BeginTransition(root, "review-one", prune.EventRendered, nil); err == nil ||
		!errors.Is(err, prune.ErrPrecondition) ||
		!strings.Contains(err.Error(), "before verification") {
		t.Fatalf("post-verification rerender error = %v, want ordering precondition", err)
	}
}

func TestVerifyBindsOptionalRenderForSkillOnlyApplication(t *testing.T) {
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
	applied, err := prune.Apply(context.Background(), root, prune.ApplyOptions{
		ReviewID: "review-one",
		Write:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	writeFile(t, agentsPath, "optional byte-stable projection\n")
	transition, err := prune.BeginTransition(root, "review-one", prune.EventRendered, nil)
	if err != nil {
		t.Fatal(err)
	}
	renderEvent, err := transition.Complete(struct {
		Path          string `json:"path"`
		Changed       bool   `json:"changed"`
		DryRun        bool   `json:"dry_run"`
		Exists        bool   `json:"exists"`
		SourceDigest  string `json:"source_digest"`
		ContentDigest string `json:"content_digest"`
		OutputDigest  string `json:"output_digest"`
	}{
		Path:          "AGENTS.md",
		Changed:       true,
		Exists:        true,
		SourceDigest:  "sha256:" + strings.Repeat("a", 64),
		ContentDigest: "sha256:" + strings.Repeat("b", 64),
		OutputDigest:  fileDigest(t, agentsPath),
	})
	if err != nil {
		t.Fatal(err)
	}
	review, _, err = prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	var applicationEventDigest string
	for _, event := range review.Events {
		if event.Kind == prune.EventApplied {
			applicationEventDigest = event.EventDigest
		}
	}
	if applicationEventDigest == "" {
		t.Fatal("application event digest is missing")
	}

	receipts := t.TempDir()
	evidence := filepath.Join(receipts, "logs", "ssb-validate.txt")
	writeFile(t, evidence, "PASS\n")
	writeFile(t, filepath.Join(receipts, "review-check.yaml"), `schema: ssb.dev/prune-check-receipt/v1
review_id: review-one
proposal_digest: `+review.ProposalDigest+`
application_event_digest: `+applicationEventDigest+`
plan_digest: `+applied.PlanDigest+`
render_event_digest: `+renderEvent.EventDigest+`
check_id: review-check
command: ssb validate --repo .
status: passed
observed_at: 2099-07-27T18:02:00Z
evidence:
  - path: logs/ssb-validate.txt
    sha256: `+fileDigest(t, evidence)+`
`)
	result, err := prune.Verify(context.Background(), root, "review-one", receipts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.RenderEventDigest != renderEvent.EventDigest {
		t.Fatalf("verification render digest = %q, want %q", result.RenderEventDigest, renderEvent.EventDigest)
	}
	if _, _, err := prune.LoadReview(root, "review-one"); err != nil {
		t.Fatalf("verified event log cannot be reloaded: %v", err)
	}
	status, _, err := prune.ReviewStatus(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Verified {
		t.Fatalf("status = %#v, want verified", status)
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
	if _, err := prune.BeginTransition(root, "review-one", prune.EventRendered, nil); err == nil ||
		!strings.Contains(err.Error(), "application") {
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
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	planDigest := reviewPlanDigest(t, root)
	skillPath := filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md")
	original, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, ".software-standards", "report.md")
	originalReport, err := os.ReadFile(reportPath)
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
  "proposal_digest": "`+reviewDigest(t, root)+`",
  "plan_digest": "`+planDigest+`",
  "entries": [{
    "path": ".agents/skills/orphan-skill/SKILL.md",
    "existed": true,
    "mode": 420,
    "content": "`+base64.StdEncoding.EncodeToString(original)+`"
  }, {
    "path": ".software-standards/report.md",
    "existed": true,
    "mode": 420,
    "content": "`+base64.StdEncoding.EncodeToString(originalReport)+`"
  }]
}
`)
	review, _, err = prune.LoadReview(root, "review-one")
	if err != nil {
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
  "plan_digest": "`+reviewPlanDigest(t, root)+`",
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
  "plan_digest": "`+reviewPlanDigest(t, root)+`",
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
	planDigest := reviewPlanDigest(t, root)
	skillPath := filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md")
	original, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, ".software-standards", "report.md")
	originalReport, err := os.ReadFile(reportPath)
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
  "plan_digest": "`+planDigest+`",
  "entries": [{
    "path": ".agents/skills/orphan-skill/SKILL.md",
    "existed": true,
    "mode": 420,
    "content": "`+base64.StdEncoding.EncodeToString(original)+`"
  }, {
    "path": ".software-standards/report.md",
    "existed": true,
    "mode": 420,
    "content": "`+base64.StdEncoding.EncodeToString(originalReport)+`"
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

func TestRecoverClearsReviewOwnedStaleLocksWithoutJournal(t *testing.T) {
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	if _, err := prune.Approve(context.Background(), root, prune.ApprovalOptions{
		ReviewID: "review-one",
		Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
	reviewRoot := filepath.Join(root, ".software-standards", "reviews", "review-one")
	transitionLock := filepath.Join(reviewRoot, ".transition.lock")
	mutationLock := filepath.Join(root, ".software-standards", ".prune-mutation.lock")
	writeFile(t, transitionLock, "review-one\n")
	writeFile(t, mutationLock, "review-one\n")

	if err := prune.Recover(context.Background(), root, "review-one", true); err != nil {
		t.Fatal(err)
	}
	for _, lockPath := range []string{transitionLock, mutationLock} {
		if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
			t.Fatalf("stale lock remains at %s: %v", lockPath, err)
		}
	}
}

type preparedReview struct {
	ProposalDigest string
}

func configureRuleUpdateCandidate(t *testing.T, root string) string {
	t.Helper()
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	sourceRulePath := filepath.Join(root, ".software-standards", "rules", "keep-rule.md")
	sourceRule, err := os.ReadFile(sourceRulePath)
	if err != nil {
		t.Fatal(err)
	}
	candidateContent := strings.Replace(
		string(sourceRule),
		"Keep the rule.\n",
		"Keep the updated rule.\n",
		1,
	)
	candidateRelative := "candidates/keep-rule/replacement.md"
	candidatePath := filepath.Join(review.Root, filepath.FromSlash(candidateRelative))
	writeFile(t, candidatePath, candidateContent)
	review.Proposal.Actions[0].Disposition = prune.DispositionUpdate
	review.Proposal.Actions[0].Target = &prune.CandidateRef{
		Kind:       prune.ArtifactRule,
		ID:         "keep-rule",
		TargetPath: ".software-standards/rules/keep-rule.md",
		SourcePath: candidateRelative,
		SHA256:     fileDigest(t, candidatePath),
		Mode:       "100644",
	}
	proposalData, err := yaml.Marshal(review.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(review.Root, "proposal.yaml"),
		proposalData,
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	return candidatePath
}

func reviewPlanDigest(t *testing.T, root string) string {
	t.Helper()
	result, err := prune.Apply(context.Background(), root, prune.ApplyOptions{ReviewID: "review-one"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.PlanDigest, "sha256:") {
		t.Fatalf("invalid plan digest %q", result.PlanDigest)
	}
	return result.PlanDigest
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
	provenanceContent := `schema: ssb.dev/artifact-provenance/v1
artifacts:
  - path: .software-standards/rules/keep-rule.md
    sha256: ` + fileDigest(t, filepath.Join(root, ".software-standards", "rules", "keep-rule.md")) + `
    origin: generated
    declaration: Generated.
  - path: .agents/skills/orphan-skill/SKILL.md
    sha256: ` + fileDigest(t, filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md")) + `
    origin: ` + skillOrigin + `
    declaration: Declared.
`
	skillRoot := filepath.Join(root, ".agents", "skills", "orphan-skill")
	if err := filepath.WalkDir(skillRoot, func(itemPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || itemPath == filepath.Join(skillRoot, "SKILL.md") {
			return nil
		}
		relative, err := filepath.Rel(root, itemPath)
		if err != nil {
			return err
		}
		provenanceContent += `  - path: ` + filepath.ToSlash(relative) + `
    sha256: ` + fileDigest(t, itemPath) + `
    origin: ` + skillOrigin + `
    declaration: Declared as part of the complete skill bundle.
`
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, provenance, provenanceContent)
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
		proposal.Actions[1].EvidenceGaps = []prune.EvidenceGap{{
			Kind:         prune.EvidenceGapProvenance,
			ArtifactPath: created.Context.Artifacts[0].Path,
			Detail:       "No provenance declaration establishes who authored and adopted these bytes.",
		}}
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
		Sources:     []prune.ArtifactRef{artifactReference(artifact)},
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

func testCanonicalDigest(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

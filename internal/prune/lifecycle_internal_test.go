package prune

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReviewEventsRejectsEmptyRerenderPayload(t *testing.T) {
	proposal := Proposal{Actions: []Action{{
		ID:          "keep-rule",
		Disposition: DispositionKeep,
	}}}
	approval, _ := json.Marshal(ApprovalPayload{Approved: []string{"keep-rule"}})
	applied, _ := json.Marshal(ApplyResult{DryRun: false, Changes: []Change{}})
	review := Review{
		Context: Context{
			ReviewID:       "review-one",
			BaselineCommit: "baseline",
			ContextDigest:  "context",
		},
		Proposal:       proposal,
		ProposalDigest: "proposal",
		Events: []Event{
			{ID: "approved-001", ReviewID: "review-one", Kind: EventApproved, RecordedAt: "2026-07-27T18:00:00Z", BaselineCommit: "baseline", ContextDigest: "context", ProposalDigest: "proposal", Payload: approval},
			{ID: "applied-002", ReviewID: "review-one", Kind: EventApplied, RecordedAt: "2026-07-27T18:01:00Z", BaselineCommit: "baseline", ContextDigest: "context", ProposalDigest: "proposal", Payload: applied},
			{ID: "rendered-003", ReviewID: "review-one", Kind: EventRendered, RecordedAt: "2026-07-27T18:02:00Z", BaselineCommit: "baseline", ContextDigest: "context", ProposalDigest: "proposal", Payload: json.RawMessage(`{}`)},
		},
	}
	if err := validateReviewEvents(review); err == nil || !strings.Contains(err.Error(), "rerender payload") {
		t.Fatalf("error = %v, want invalid rerender payload", err)
	}
}

func TestAppendEventFailurePreservesCompletePriorLog(t *testing.T) {
	root := t.TempDir()
	eventPath := filepath.Join(root, "events.jsonl")
	prior := []byte("{\"complete\":true}\n")
	if err := os.WriteFile(eventPath, prior, 0o644); err != nil {
		t.Fatal(err)
	}
	originalWriter := writeEventLogAtomically
	writeEventLogAtomically = func(string, []byte, os.FileMode) error {
		return errors.New("injected replacement failure")
	}
	t.Cleanup(func() { writeEventLogAtomically = originalWriter })

	err := appendEvent(Review{Root: root}, Event{Schema: EventSchema, ID: "approved-001"})
	if err == nil {
		t.Fatal("expected injected append failure")
	}
	after, readErr := os.ReadFile(eventPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(prior) {
		t.Fatalf("event append left partial output: %q", after)
	}
}

func TestApplicationRollbackRemovesCreatedDirectories(t *testing.T) {
	root := t.TempDir()
	relative := ".agents/skills/new-skill/references/check.md"
	content := []byte("candidate\n")
	operations := []operation{{
		Change:         Change{Path: relative, Kind: "write", SHA256: digestBytes(content)},
		Content:        content,
		ExpectedAbsent: true,
	}}
	directories, err := missingApplicationDirectories(root, operations)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := writeNewExclusive(target, content, 0o644); err != nil {
		t.Fatal(err)
	}
	journal := applicationJournal{
		Entries:            []journalEntry{{Path: relative}},
		CreatedDirectories: directories,
	}
	if err := restoreJournalPaths(
		root,
		journal,
		map[string]struct{}{relative: {}},
		map[string]string{relative: digestBytes(content)},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("rollback left created directory tree: %v", err)
	}
}

func TestClaimedMutationPreservesConcurrentReplacement(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.md")
	original := []byte("approved original\n")
	concurrent := []byte("concurrent edit\n")
	if err := os.WriteFile(target, original, 0o644); err != nil {
		t.Fatal(err)
	}
	claim, err := claimExpectedFile(target, digestBytes(original), prestateClaim)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, concurrent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := restoreClaimedFile(claim, target); err == nil {
		t.Fatal("expected exclusive restoration to reject concurrent target")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(concurrent) {
		t.Fatalf("concurrent replacement overwritten: %q", got)
	}
	claimed, err := os.ReadFile(claim)
	if err != nil || string(claimed) != string(original) {
		t.Fatalf("approved original was not preserved in claim: %q, %v", claimed, err)
	}
}

func TestApplicationJournalCleanupFailureNamesRecoveryCommand(t *testing.T) {
	original := removeApplicationJournal
	removeApplicationJournal = func(string) error { return errors.New("injected cleanup failure") }
	t.Cleanup(func() { removeApplicationJournal = original })

	err := cleanupApplicationJournal("application-journal.json", "review-one")
	if err == nil || !strings.Contains(err.Error(), "ssb prune recover --review review-one") {
		t.Fatalf("error = %v, want exact recovery command", err)
	}
}

func TestRecoveryResumesFromPoststateClaim(t *testing.T) {
	for name, prestateAlreadyWritten := range map[string]bool{
		"after poststate claim":     false,
		"after prestate recreation": true,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, "target.md")
			prestate := []byte("before\n")
			poststate := []byte("after\n")
			if err := os.WriteFile(target, poststate, 0o644); err != nil {
				t.Fatal(err)
			}
			postClaim, err := claimExpectedFile(target, digestBytes(poststate), poststateClaim)
			if err != nil {
				t.Fatal(err)
			}
			if prestateAlreadyWritten {
				if err := writeNewExclusive(target, prestate, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			entry := journalEntry{Path: "target.md", Existed: true, Mode: 0o644, Content: prestate}
			if err := restoreJournalEntry(target, entry, digestBytes(poststate)); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(target)
			if err != nil || string(got) != string(prestate) {
				t.Fatalf("restored bytes = %q, %v", got, err)
			}
			if _, err := os.Stat(postClaim); !os.IsNotExist(err) {
				t.Fatalf("poststate claim remains: %v", err)
			}
		})
	}
}

func TestRecoveryReconcilesDualClaimsAfterSecondCrash(t *testing.T) {
	for name, prestateAlreadyWritten := range map[string]bool{
		"target absent":           false,
		"prestate target written": true,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, "target.md")
			prestate := []byte("before\n")
			poststate := []byte("after\n")
			preClaim := claimPathForTarget(target, prestateClaim)
			postClaim := claimPathForTarget(target, poststateClaim)
			if err := os.WriteFile(preClaim, prestate, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(postClaim, poststate, 0o644); err != nil {
				t.Fatal(err)
			}
			if prestateAlreadyWritten {
				if err := os.WriteFile(target, prestate, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			entry := journalEntry{Path: "target.md", Existed: true, Mode: 0o644, Content: prestate}
			if err := restoreJournalEntry(target, entry, digestBytes(poststate)); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(target)
			if err != nil || string(got) != string(prestate) {
				t.Fatalf("restored bytes = %q, %v", got, err)
			}
			for _, claim := range []string{preClaim, postClaim} {
				if _, err := os.Stat(claim); !os.IsNotExist(err) {
					t.Fatalf("claim remains after reentry: %s: %v", claim, err)
				}
			}
		})
	}
}

func TestRepositoryMutationLockSerializesDistinctReviews(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".software-standards"), 0o755); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireMutationLock(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := acquireMutationLock(root, "review-two"); err == nil ||
		!strings.Contains(err.Error(), "mutation lock") {
		t.Fatalf("error = %v, want cross-review serialization", err)
	}
}

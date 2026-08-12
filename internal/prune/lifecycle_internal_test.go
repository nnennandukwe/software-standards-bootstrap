package prune

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
)

func TestValidateReviewEventsRejectsEmptyRerenderPayload(t *testing.T) {
	proposal := Proposal{Actions: []Action{
		{
			ID:          "keep-rule",
			Disposition: DispositionKeep,
		},
		{
			ID:          "remove-skill",
			Disposition: DispositionRemove,
			Sources: []ArtifactRef{{
				Kind: ArtifactSkill,
				ID:   "old-skill",
				Path: ".agents/skills/old-skill/SKILL.md",
			}},
		},
	}}
	approved := ApprovalPayload{Approved: []string{"keep-rule", "remove-skill"}}
	approval, _ := json.Marshal(approved)
	review := Review{
		Context: Context{
			ReviewID:       "review-one",
			BaselineCommit: "baseline",
			ContextDigest:  "context",
			Artifacts: []Artifact{
				{
					Kind: ArtifactRule, ID: "keep-rule",
					Path:   ".software-standards/rules/keep-rule.md",
					SHA256: "sha256:" + strings.Repeat("a", 64),
					Mode:   "100644",
				},
				{
					Kind: ArtifactSkill, ID: "old-skill",
					Path:   ".agents/skills/old-skill/SKILL.md",
					SHA256: "sha256:" + strings.Repeat("c", 64),
					Mode:   "100644",
				},
			},
		},
		Proposal:       proposal,
		ProposalDigest: "proposal",
		Events: []Event{
			{ID: "approved-001", ReviewID: "review-one", Kind: EventApproved, RecordedAt: "2026-07-27T18:00:00Z", BaselineCommit: "baseline", ContextDigest: "context", ProposalDigest: "proposal", Payload: approval, EventDigest: "sha256:" + strings.Repeat("b", 64)},
		},
	}
	plan, err := canonicalApplicationPlan(review, approved)
	if err != nil {
		t.Fatal(err)
	}
	applied, _ := json.Marshal(ApplyResult{DryRun: false, PlanDigest: plan.PlanDigest, Changes: plan.Changes})
	review.Events = append(review.Events,
		Event{ID: "applied-002", ReviewID: "review-one", Kind: EventApplied, RecordedAt: "2026-07-27T18:01:00Z", BaselineCommit: "baseline", ContextDigest: "context", ProposalDigest: "proposal", Payload: applied},
		Event{ID: "rendered-003", ReviewID: "review-one", Kind: EventRendered, RecordedAt: "2026-07-27T18:02:00Z", BaselineCommit: "baseline", ContextDigest: "context", ProposalDigest: "proposal", Payload: json.RawMessage(`{}`)},
	)
	if err := validateReviewEvents(review); err == nil || !strings.Contains(err.Error(), "rerender payload") {
		t.Fatalf("error = %v, want invalid rerender payload", err)
	}
}

func TestAppendEventFailurePreservesCompletePriorLog(t *testing.T) {
	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	eventPath := filepath.Join(root, "events.jsonl")
	prior := []byte("{\"complete\":true}\n")
	if err := os.WriteFile(eventPath, prior, 0o644); err != nil {
		t.Fatal(err)
	}
	originalWriter := writeEventLogAtomically
	writeEventLogAtomically = func(*reviewStore, []byte, os.FileMode) error {
		return errors.New("injected replacement failure")
	}
	t.Cleanup(func() { writeEventLogAtomically = originalWriter })

	err := appendEvent(Review{
		RepoRoot: repoRoot,
		Root:     root,
		Context:  Context{ReviewID: "review-one"},
	}, Event{Schema: EventSchema, ID: "approved-001"})
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

func TestAtomicReviewWriteReportsRenameAndTemporaryCleanupFailures(t *testing.T) {
	repoRoot := t.TempDir()
	reviewRoot := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
	if err := os.MkdirAll(reviewRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	eventPath := filepath.Join(reviewRoot, "events.jsonl")
	prior := []byte("{\"complete\":true}\n")
	if err := os.WriteFile(eventPath, prior, 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := openReviewStore(repoRoot, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	renameFailure := errors.New("injected rename failure")
	removeFailure := errors.New("injected temporary cleanup failure")
	originalRename := renameAtomicReviewFile
	originalRemove := removeAtomicReviewFile
	renameAtomicReviewFile = func(*os.Root, string, string) error {
		return renameFailure
	}
	removeAtomicReviewFile = func(root *os.Root, name string) error {
		if strings.HasPrefix(filepath.Base(name), ".ssb-prune-") {
			return removeFailure
		}
		return originalRemove(root, name)
	}
	t.Cleanup(func() {
		renameAtomicReviewFile = originalRename
		removeAtomicReviewFile = originalRemove
	})

	err = store.AtomicWrite("events.jsonl", []byte("{\"next\":true}\n"), 0o644)
	if !errors.Is(err, renameFailure) || !errors.Is(err, removeFailure) {
		t.Fatalf("error = %v, want joined rename and cleanup failures", err)
	}
	if !strings.Contains(err.Error(), ".ssb-prune-") ||
		!strings.Contains(err.Error(), "inspect and remove this residual file") {
		t.Fatalf("error = %v, want exact residual-path recovery guidance", err)
	}
	after, readErr := os.ReadFile(eventPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(prior) {
		t.Fatalf("failed replacement changed prior event log: %q", after)
	}
	entries, readDirErr := os.ReadDir(reviewRoot)
	if readDirErr != nil {
		t.Fatal(readDirErr)
	}
	foundResidual := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".ssb-prune-") {
			foundResidual = true
		}
	}
	if !foundResidual {
		t.Fatal("injected cleanup failure did not preserve the reported residual file")
	}
}

func TestAppendEventCannotFollowReviewRootSwapInsideWriter(t *testing.T) {
	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	eventPath := filepath.Join(root, "events.jsonl")
	prior := []byte("{\"complete\":true}\n")
	if err := os.WriteFile(eventPath, prior, 0o644); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "review-one")
	originalWriter := writeEventLogAtomically
	writeEventLogAtomically = func(store *reviewStore, data []byte, mode os.FileMode) error {
		if err := os.Rename(root, external); err != nil {
			return err
		}
		if err := os.Symlink(external, root); err != nil {
			return err
		}
		return originalWriter(store, data, mode)
	}
	t.Cleanup(func() { writeEventLogAtomically = originalWriter })

	err := appendEvent(Review{
		RepoRoot: repoRoot,
		Root:     root,
		Context:  Context{ReviewID: "review-one"},
	}, Event{Schema: EventSchema, ID: "approved-001"})
	if err == nil {
		t.Fatal("review-root swap inside event writer unexpectedly succeeded")
	}
	after, readErr := os.ReadFile(filepath.Join(external, "events.jsonl"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(prior) {
		t.Fatalf("external event log changed after swap: %q", after)
	}
}

func TestReviewLockCleanupCannotFollowReviewRootSwapInsideRemover(t *testing.T) {
	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	_, unlock, err := acquireReviewLock(repoRoot, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "review-one")
	originalRemover := removeReviewTransitionLock
	removeReviewTransitionLock = func(store *reviewStore) error {
		if err := os.Rename(root, external); err != nil {
			return err
		}
		if err := os.Symlink(external, root); err != nil {
			return err
		}
		return originalRemover(store)
	}
	t.Cleanup(func() { removeReviewTransitionLock = originalRemover })

	if err := unlock(); err == nil {
		t.Fatal("review-root swap inside lock remover unexpectedly succeeded")
	}
	lockData, readErr := os.ReadFile(filepath.Join(external, ".transition.lock"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(lockData) != "review-one\n" {
		t.Fatalf("external review lock changed after swap: %q", lockData)
	}
}

func TestApplicationJournalOperationsCannotFollowReviewRootSwap(t *testing.T) {
	for name, exercise := range map[string]func(*reviewStore) error{
		"create": func(store *reviewStore) error {
			return store.WriteExclusive("application-journal.json", []byte("new journal\n"), 0o600)
		},
		"remove": func(store *reviewStore) error {
			return store.Remove("application-journal.json")
		},
	} {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			reviewRoot := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
			if err := os.MkdirAll(reviewRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			journalPath := filepath.Join(reviewRoot, "application-journal.json")
			if name == "remove" {
				if err := os.WriteFile(journalPath, []byte("approved journal\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			store, err := openReviewStore(repoRoot, "review-one")
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			external := filepath.Join(t.TempDir(), "review-one")
			if err := os.Rename(reviewRoot, external); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(external, reviewRoot); err != nil {
				t.Skipf("symlinks are unavailable: %v", err)
			}

			if err := exercise(store); err == nil {
				t.Fatalf("%s unexpectedly followed the swapped review root", name)
			}
			externalJournal := filepath.Join(external, "application-journal.json")
			data, readErr := os.ReadFile(externalJournal)
			if name == "create" {
				if !errors.Is(readErr, os.ErrNotExist) {
					t.Fatalf("external journal was created: %q, %v", data, readErr)
				}
			} else if readErr != nil || string(data) != "approved journal\n" {
				t.Fatalf("external journal changed: %q, %v", data, readErr)
			}
		})
	}
}

func TestReviewStoreCannotFollowSiblingSymlinkInsideRepository(t *testing.T) {
	for name, exercise := range map[string]func(*reviewStore) error{
		"remove lock": func(store *reviewStore) error {
			return store.Remove(".transition.lock")
		},
		"create journal": func(store *reviewStore) error {
			return store.WriteExclusive("application-journal.json", []byte("new journal\n"), 0o600)
		},
		"replace events": func(store *reviewStore) error {
			return store.AtomicWrite("events.jsonl", []byte("new events\n"), 0o644)
		},
	} {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			reviewRoot := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
			victim := filepath.Join(repoRoot, "victim")
			if err := os.MkdirAll(reviewRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(victim, 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile := func(name, content string, mode os.FileMode) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(victim, name), []byte(content), mode); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(".transition.lock", "victim lock\n", 0o600)
			writeFile("events.jsonl", "victim events\n", 0o644)
			store, err := openReviewStore(repoRoot, "review-one")
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			original := filepath.Join(repoRoot, ".software-standards", "reviews", ".review-one-original")
			if err := os.Rename(reviewRoot, original); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join("..", "..", "victim"), reviewRoot); err != nil {
				t.Skipf("symlinks are unavailable: %v", err)
			}

			if err := exercise(store); err == nil {
				t.Fatalf("%s followed an in-repository sibling symlink", name)
			}
			lockData, lockErr := os.ReadFile(filepath.Join(victim, ".transition.lock"))
			eventData, eventErr := os.ReadFile(filepath.Join(victim, "events.jsonl"))
			if lockErr != nil || string(lockData) != "victim lock\n" {
				t.Fatalf("victim lock changed: %q, %v", lockData, lockErr)
			}
			if eventErr != nil || string(eventData) != "victim events\n" {
				t.Fatalf("victim events changed: %q, %v", eventData, eventErr)
			}
			if _, err := os.Lstat(filepath.Join(victim, "application-journal.json")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("victim journal was created: %v", err)
			}
		})
	}
}

func TestLockedReviewSessionRejectsRealDirectoryReplacement(t *testing.T) {
	repoRoot := t.TempDir()
	reviewRoot := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
	replacement := filepath.Join(repoRoot, ".software-standards", "reviews", "replacement")
	if err := os.MkdirAll(reviewRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(replacement, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reviewRoot, "context.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(replacement, "context.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, unlock, err := acquireReviewLock(repoRoot, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(repoRoot, ".software-standards", "reviews", ".review-one-original")
	if err := os.Rename(reviewRoot, original); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, reviewRoot); err != nil {
		t.Fatal(err)
	}

	if _, _, err := loadReviewFromStore(store); err == nil ||
		!strings.Contains(err.Error(), "changed") {
		t.Fatalf("load error = %v, want locked review identity rejection", err)
	}
	if err := appendEvent(Review{
		RepoRoot: repoRoot,
		Root:     reviewRoot,
		Context:  Context{ReviewID: "review-one"},
		store:    store,
	}, Event{Schema: EventSchema, ID: "approved-001"}); err == nil ||
		!strings.Contains(err.Error(), "changed") {
		t.Fatalf("append error = %v, want locked review identity rejection", err)
	}
	if _, err := os.Lstat(filepath.Join(reviewRoot, "events.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replacement review received an event: %v", err)
	}
	if err := unlock(); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("unlock error = %v, want locked review identity rejection", err)
	}
}

func TestReviewPublicationCannotFollowPackRootSwap(t *testing.T) {
	repoRoot := t.TempDir()
	packRoot := filepath.Join(repoRoot, ".software-standards")
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	staging, target, err := createReviewStagingStore(repoRoot, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	defer staging.Close()
	if err := staging.WriteExclusive("context.json", []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "software-standards")
	if err := os.Rename(packRoot, external); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, packRoot); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	if err := staging.Publish(target); err == nil {
		t.Fatal("review publication followed the swapped pack root")
	}
	if _, err := os.Lstat(filepath.Join(external, "reviews", "review-one")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("external review was published: %v", err)
	}
}

func TestApplicationRollbackRemovesCreatedDirectories(t *testing.T) {
	root := t.TempDir()
	relative := ".agents/skills/new-skill/references/check.md"
	content := []byte("candidate\n")
	operations := []operation{{
		Change: Change{
			Path: relative,
			Kind: "write",
			Poststate: FileState{
				Exists: true,
				SHA256: digestBytes(content),
				Mode:   "100644",
			},
		},
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

func TestManifestPublicationRollback(t *testing.T) {
	root := t.TempDir()
	rulePath := filepath.Join(root, ".software-standards", "rules", "keep-rule.md")
	skillPath := filepath.Join(root, ".agents", "skills", "orphan-skill", "SKILL.md")
	manifestPath := filepath.Join(root, filepath.FromSlash(actionableManifestPath))
	ruleBefore := []byte("# Keep the rule\n\nBefore.\n")
	ruleAfter := []byte("# Keep the rule\n\nAfter.\n")
	skillBefore := []byte("skill before\n")
	manifestBefore := []byte("schema: ssb.dev/manifest/v1\nstate: before\n")
	manifestAfter := []byte("schema: ssb.dev/manifest/v1\nstate: after\n")
	for itemPath, content := range map[string][]byte{
		rulePath: ruleBefore, skillPath: skillBefore, manifestPath: manifestBefore,
	} {
		if err := os.MkdirAll(filepath.Dir(itemPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(itemPath, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	operations := []operation{
		{
			Change:  Change{Path: ".software-standards/rules/keep-rule.md", Kind: "write", Prestate: FileState{Exists: true, SHA256: digestBytes(ruleBefore), Mode: "100644"}, Poststate: FileState{Exists: true, SHA256: digestBytes(ruleAfter), Mode: "100644"}},
			Content: ruleAfter, ExpectedSHA256: digestBytes(ruleBefore), Mode: 0o644,
		},
		{
			Change:         Change{Path: ".agents/skills/orphan-skill/SKILL.md", Kind: "remove", Prestate: FileState{Exists: true, SHA256: digestBytes(skillBefore), Mode: "100644"}, Poststate: FileState{}},
			ExpectedSHA256: digestBytes(skillBefore),
		},
		{
			Change:  Change{Path: actionableManifestPath, Kind: "write", Prestate: FileState{Exists: true, SHA256: digestBytes(manifestBefore), Mode: "100644"}, Poststate: FileState{Exists: true, SHA256: digestBytes(manifestAfter), Mode: "100644"}},
			Content: manifestAfter, ExpectedSHA256: digestBytes(manifestBefore), Mode: 0o644,
		},
	}
	journal := applicationJournal{Entries: []journalEntry{
		{Path: ".software-standards/rules/keep-rule.md", Existed: true, Mode: 0o644, Content: ruleBefore},
		{Path: ".agents/skills/orphan-skill/SKILL.md", Existed: true, Mode: 0o644, Content: skillBefore},
		{Path: actionableManifestPath, Existed: true, Mode: 0o644, Content: manifestBefore},
	}}
	originalPublish := publishApplicationFile
	publishApplicationFile = func(staging, target string) error {
		if strings.HasSuffix(filepath.ToSlash(target), actionableManifestPath) {
			return errors.New("injected manifest publication failure")
		}
		return originalPublish(staging, target)
	}
	t.Cleanup(func() { publishApplicationFile = originalPublish })

	completed, executeErr := executeOperations(root, operations)
	if executeErr == nil || !strings.Contains(executeErr.Error(), "injected manifest publication failure") {
		t.Fatalf("execute error = %v, want injected manifest failure", executeErr)
	}
	if err := restoreJournalPaths(root, journal, completed, operationPoststates(operations)); err != nil {
		t.Fatalf("rollback manifest-layout pack: %v", err)
	}
	for itemPath, want := range map[string][]byte{
		rulePath: ruleBefore, skillPath: skillBefore, manifestPath: manifestBefore,
	} {
		got, err := os.ReadFile(itemPath)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("restored %s = %q, %v; want %q", itemPath, got, err, want)
		}
	}
}

func TestWriteNewExclusiveDoesNotExposePartialTargetAfterProcessCrash(t *testing.T) {
	if target := os.Getenv("SSB_PRUNE_CRASH_WRITE_TARGET"); target != "" {
		if os.Getenv("SSB_PRUNE_CRASH_WRITE_EXISTED") == "1" {
			if _, err := claimExpectedFile(
				target,
				digestBytes([]byte("approved prestate\n")),
				prestateClaim,
			); err != nil {
				os.Exit(96)
			}
		}
		openExclusiveFile = func(
			name string,
			flag int,
			perm os.FileMode,
		) (durableExclusiveFile, error) {
			file, err := os.OpenFile(name, flag, perm)
			if err != nil {
				return nil, err
			}
			return &crashDuringWriteFile{File: file}, nil
		}
		_ = writeNewExclusive(target, []byte("complete governed content\n"), 0o644)
		os.Exit(98)
	}

	for _, testCase := range []struct {
		name    string
		existed bool
	}{
		{name: "new target"},
		{name: "existing target", existed: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "governed.md")
			prestate := []byte("approved prestate\n")
			poststate := []byte("complete governed content\n")
			if testCase.existed {
				if err := os.WriteFile(target, prestate, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			command := exec.Command(
				os.Args[0],
				"-test.run=^TestWriteNewExclusiveDoesNotExposePartialTargetAfterProcessCrash$",
			)
			command.Env = append(
				os.Environ(),
				"SSB_PRUNE_CRASH_WRITE_TARGET="+target,
			)
			if testCase.existed {
				command.Env = append(command.Env, "SSB_PRUNE_CRASH_WRITE_EXISTED=1")
			}
			err := command.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 97 {
				t.Fatalf("helper exit = %v, want injected process crash", err)
			}
			if data, err := os.ReadFile(target); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("partial governed target became visible: %q, %v", data, err)
			}
			entry := journalEntry{
				Path:    "governed.md",
				Existed: testCase.existed,
				Mode:    0o644,
				Content: prestate,
			}
			if err := restoreJournalEntry(target, entry, digestBytes(poststate)); err != nil {
				t.Fatalf("recover after process crash: %v", err)
			}
			if testCase.existed {
				got, err := os.ReadFile(target)
				if err != nil || string(got) != string(prestate) {
					t.Fatalf("recovered content = %q, %v", got, err)
				}
			} else if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("new target remains after recovery: %v", err)
			}
			if _, err := os.Lstat(
				claimPathForTarget(target, publicationClaim),
			); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("publication claim remains after recovery: %v", err)
			}
		})
	}
}

func TestInterruptedApplicationPublicationKeepsPrestateClaimRecoverable(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "governed.md")
	prestate := []byte("approved prestate\n")
	poststate := []byte("approved poststate\n")
	if err := os.WriteFile(target, prestate, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := claimExpectedFile(target, digestBytes(prestate), prestateClaim); err != nil {
		t.Fatal(err)
	}

	originalPublish := publishApplicationFile
	publishApplicationFile = func(string, string) error {
		return errors.New("injected interruption before publication")
	}
	t.Cleanup(func() { publishApplicationFile = originalPublish })

	if err := writeNewExclusive(target, poststate, 0o644); err == nil ||
		!strings.Contains(err.Error(), "injected interruption") {
		t.Fatalf("error = %v, want interrupted publication", err)
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("target exists before recovery: %v", err)
	}
	entry := journalEntry{
		Path: "governed.md", Existed: true, Mode: 0o644, Content: prestate,
	}
	if err := restoreJournalEntry(target, entry, digestBytes(poststate)); err != nil {
		t.Fatalf("recover prestate claim: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != string(prestate) {
		t.Fatalf("recovered content = %q, %v", got, err)
	}
}

func TestWriteNewExclusiveNeverOverwritesExistingTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "governed.md")
	existing := []byte("concurrent content\n")
	if err := os.WriteFile(target, existing, 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeNewExclusive(target, []byte("approved content\n"), 0o644)
	if err == nil {
		t.Fatal("exclusive publication overwrote an existing target")
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil || string(got) != string(existing) {
		t.Fatalf("existing target = %q, %v, want unchanged bytes", got, readErr)
	}
	if _, claimErr := os.Lstat(
		claimPathForTarget(target, publicationClaim),
	); !errors.Is(claimErr, os.ErrNotExist) {
		t.Fatalf("publication claim remains after collision: %v", claimErr)
	}
}

func TestRecoveryReconcilesCompleteApplicationPublicationClaims(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		existed   bool
		published bool
	}{
		{name: "new target before publication"},
		{name: "new target after publication", published: true},
		{name: "existing target before publication", existed: true},
		{name: "existing target after publication", existed: true, published: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, "governed.md")
			prestate := []byte("approved prestate\n")
			poststate := []byte("approved poststate\n")
			if testCase.existed {
				if err := os.WriteFile(target, prestate, 0o644); err != nil {
					t.Fatal(err)
				}
				if _, err := claimExpectedFile(
					target,
					digestBytes(prestate),
					prestateClaim,
				); err != nil {
					t.Fatal(err)
				}
			}
			publication := claimPathForTarget(target, publicationClaim)
			if err := os.WriteFile(publication, poststate, 0o644); err != nil {
				t.Fatal(err)
			}
			if testCase.published {
				if err := os.Link(publication, target); err != nil {
					t.Fatal(err)
				}
			}

			entry := journalEntry{
				Path:    "governed.md",
				Existed: testCase.existed,
				Mode:    0o644,
				Content: prestate,
			}
			if err := restoreJournalEntry(target, entry, digestBytes(poststate)); err != nil {
				t.Fatal(err)
			}
			if testCase.existed {
				got, err := os.ReadFile(target)
				if err != nil || string(got) != string(prestate) {
					t.Fatalf("recovered content = %q, %v", got, err)
				}
			} else if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("new target remains after recovery: %v", err)
			}
			for _, claim := range []string{
				publication,
				claimPathForTarget(target, prestateClaim),
				claimPathForTarget(target, poststateClaim),
			} {
				if _, err := os.Lstat(claim); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("claim remains after recovery: %s: %v", claim, err)
				}
			}
		})
	}
}

func TestRecoveryFailsClosedWhenIncompletePublicationClaimMeetsHumanEdit(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "governed.md")
	publication := claimPathForTarget(target, publicationClaim)
	if err := os.WriteFile(publication, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	humanEdit := []byte("post-crash human edit\n")
	if err := os.WriteFile(target, humanEdit, 0o644); err != nil {
		t.Fatal(err)
	}
	entry := journalEntry{
		Path: "governed.md", Existed: true, Content: []byte("approved prestate\n"),
	}
	err := restoreJournalEntry(target, entry, digestBytes([]byte("approved poststate\n")))
	if err == nil ||
		!strings.Contains(err.Error(), "publication claim") ||
		!strings.Contains(err.Error(), "unapproved bytes") {
		t.Fatalf("error = %v, want human-edit recovery block", err)
	}
	if got, readErr := os.ReadFile(publication); readErr != nil || string(got) != "partial" {
		t.Fatalf("incomplete publication claim changed: %q, %v", got, readErr)
	}
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != string(humanEdit) {
		t.Fatalf("human-edited target changed: %q, %v", got, readErr)
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
	removeApplicationJournal = func(*reviewStore) error { return errors.New("injected cleanup failure") }
	t.Cleanup(func() { removeApplicationJournal = original })

	err := cleanupApplicationJournal(nil, "review-one")
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

func TestLockReleaseFailuresAreReported(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".software-standards"), 0o755); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireMutationLock(root, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	original := removePruneMutationLock
	removePruneMutationLock = func(*os.Root, os.FileInfo) error {
		return errors.New("injected lock cleanup failure")
	}
	t.Cleanup(func() {
		removePruneMutationLock = original
		_ = os.Remove(filepath.Join(root, ".software-standards", ".prune-mutation.lock"))
	})
	if err := unlock(); err == nil || !strings.Contains(err.Error(), "injected lock cleanup failure") {
		t.Fatalf("error = %v, want lock cleanup failure", err)
	}
}

func TestMutationLockCleanupCannotFollowPackRootSwapInsideRemover(t *testing.T) {
	repoRoot := t.TempDir()
	packRoot := filepath.Join(repoRoot, ".software-standards")
	if err := os.MkdirAll(packRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireMutationLock(repoRoot, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "software-standards")
	original := removePruneMutationLock
	removePruneMutationLock = func(root *os.Root, identity os.FileInfo) error {
		if err := os.Rename(packRoot, external); err != nil {
			return err
		}
		if err := os.Symlink(external, packRoot); err != nil {
			return err
		}
		return original(root, identity)
	}
	t.Cleanup(func() { removePruneMutationLock = original })

	if err := unlock(); err == nil {
		t.Fatal("pack-root swap inside mutation-lock remover unexpectedly succeeded")
	}
	lockData, readErr := os.ReadFile(filepath.Join(external, ".prune-mutation.lock"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(lockData) != "review-one\n" {
		t.Fatalf("external mutation lock changed after swap: %q", lockData)
	}
}

func TestTransitionCompleteReportsCommittedEventWhenLockCleanupFails(t *testing.T) {
	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	_, unlock, err := acquireReviewLock(repoRoot, "review-one")
	if err != nil {
		t.Fatal(err)
	}
	original := removeReviewTransitionLock
	removeReviewTransitionLock = func(*reviewStore) error {
		return errors.New("injected review lock cleanup failure")
	}
	t.Cleanup(func() {
		removeReviewTransitionLock = original
		_ = os.Remove(filepath.Join(root, ".transition.lock"))
	})
	transition := &Transition{
		review: Review{
			RepoRoot: repoRoot,
			Root:     root,
			Context: Context{
				ReviewID:       "review-one",
				BaselineCommit: "baseline",
				ContextDigest:  "context",
			},
			ProposalDigest: "proposal",
		},
		kind:   EventADR,
		unlock: unlock,
	}
	event, err := transition.Complete(adrEventPayload{
		Path: "docs/adr/0001-prune.md", Created: true,
	})
	if err == nil ||
		event.EventDigest == "" ||
		!strings.Contains(err.Error(), "recover --review review-one --clear-stale-lock") {
		t.Fatalf("event = %#v, error = %v, want committed event and exact recovery", event, err)
	}
	if !errors.Is(err, ErrPrecondition) {
		t.Fatalf("error = %v, want recoverable precondition classification", err)
	}
	data, readErr := os.ReadFile(filepath.Join(root, "events.jsonl"))
	if readErr != nil || !strings.Contains(string(data), event.EventDigest) {
		t.Fatalf("event log = %q, %v, want committed transition", data, readErr)
	}
}

func TestWriteExclusiveRemovesPartialFinalFile(t *testing.T) {
	original := openExclusiveFile
	openExclusiveFile = func(name string, flag int, perm os.FileMode) (durableExclusiveFile, error) {
		file, err := os.OpenFile(name, flag, perm)
		if err != nil {
			return nil, err
		}
		return &shortWriteExclusiveFile{File: file}, nil
	}
	t.Cleanup(func() { openExclusiveFile = original })

	for name, write := range map[string]func(string, []byte, os.FileMode) error{
		"review artifact": writeExclusive,
		"governed file":   writeNewExclusive,
	} {
		t.Run(name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "target")
			if err := write(target, []byte("complete content\n"), 0o600); err == nil {
				t.Fatal("expected injected short write")
			}
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatalf("partial final file remains: %v", err)
			}
		})
	}
}

func TestWriteExclusiveReportsPartialCleanupFailure(t *testing.T) {
	originalOpen := openExclusiveFile
	originalRemove := removeIncompleteExclusiveFile
	openExclusiveFile = func(name string, flag int, perm os.FileMode) (durableExclusiveFile, error) {
		file, err := os.OpenFile(name, flag, perm)
		if err != nil {
			return nil, err
		}
		return &shortWriteExclusiveFile{File: file}, nil
	}
	removeIncompleteExclusiveFile = func(string) error {
		return errors.New("injected incomplete cleanup failure")
	}
	t.Cleanup(func() {
		openExclusiveFile = originalOpen
		removeIncompleteExclusiveFile = originalRemove
	})

	for name, write := range map[string]func(string, []byte, os.FileMode) error{
		"review artifact": writeExclusive,
		"governed file":   writeNewExclusive,
	} {
		t.Run(name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "target")
			t.Cleanup(func() { _ = os.Remove(target) })
			err := write(target, []byte("complete content\n"), 0o600)
			if err == nil ||
				!strings.Contains(err.Error(), "short write") ||
				!strings.Contains(err.Error(), "injected incomplete cleanup failure") {
				t.Fatalf("error = %v, want write and cleanup causes", err)
			}
		})
	}
}

type shortWriteExclusiveFile struct {
	*os.File
}

func (file *shortWriteExclusiveFile) Write(data []byte) (int, error) {
	return file.File.Write(data[:1])
}

type crashDuringWriteFile struct {
	*os.File
}

func (file *crashDuringWriteFile) Write(data []byte) (int, error) {
	if len(data) > 0 {
		_, _ = file.File.Write(data[:1])
		_ = file.File.Sync()
	}
	os.Exit(97)
	return 0, nil
}

func TestCaptureJournalRejectsDisappearedApprovedPrestate(t *testing.T) {
	root := t.TempDir()
	review := Review{
		Context:        Context{ReviewID: "review-one"},
		ProposalDigest: "sha256:" + strings.Repeat("a", 64),
	}
	plan := applicationPlan{PlanDigest: "sha256:" + strings.Repeat("b", 64)}
	operations := []operation{{
		Change: Change{
			Path: ".agents/skills/old-skill/SKILL.md",
			Kind: "remove",
			Prestate: FileState{
				Exists: true,
				SHA256: "sha256:" + strings.Repeat("c", 64),
				Mode:   "100644",
			},
			Poststate: FileState{Exists: false},
		},
		ExpectedSHA256: "sha256:" + strings.Repeat("c", 64),
	}}
	if _, err := captureJournal(root, review, plan, operations); err == nil ||
		!strings.Contains(err.Error(), "disappeared after preflight") {
		t.Fatalf("error = %v, want disappeared-prestate block", err)
	}
}

func TestCaptureJournalRejectsChangedApprovedPrestate(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".agents", "skills", "old-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("changed after preflight\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	review := Review{
		Context:        Context{ReviewID: "review-one"},
		ProposalDigest: "sha256:" + strings.Repeat("a", 64),
	}
	plan := applicationPlan{PlanDigest: "sha256:" + strings.Repeat("b", 64)}
	operations := []operation{{
		Change: Change{
			Path: ".agents/skills/old-skill/SKILL.md",
			Kind: "remove",
			Prestate: FileState{
				Exists: true,
				SHA256: "sha256:" + strings.Repeat("c", 64),
				Mode:   "100644",
			},
			Poststate: FileState{Exists: false},
		},
		ExpectedSHA256: "sha256:" + strings.Repeat("c", 64),
	}}
	if _, err := captureJournal(root, review, plan, operations); err == nil ||
		!strings.Contains(err.Error(), "changed after preflight") {
		t.Fatalf("error = %v, want changed-prestate block", err)
	}
}

func TestReadCandidateFileRejectsSymlinkAfterValidation(t *testing.T) {
	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, ".software-standards", "reviews", "review-one")
	outside := filepath.Join(t.TempDir(), "candidate.md")
	if err := os.WriteFile(outside, []byte("approved bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(root, "candidates", "update-rule", "candidate.md")
	if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, candidate); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	review := Review{
		RepoRoot: repoRoot,
		Root:     root,
		Context: Context{
			ReviewID: "review-one",
			Inventory: inventory.Report{Limits: inventory.Limits{
				MaxFileBytes: 1 << 20,
			}},
		},
	}
	if _, err := readCandidateFile(review, "candidates/update-rule/candidate.md"); err == nil ||
		!strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error = %v, want candidate symlink block", err)
	}
}

func TestCandidateModePortabilityContract(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		goos    string
		wantErr string
	}{
		{name: "portable regular file on Windows", mode: "100644", goos: "windows"},
		{name: "executable file on POSIX", mode: "100755", goos: "linux"},
		{
			name:    "executable file on Windows",
			mode:    "100755",
			goos:    "windows",
			wantErr: "cannot materialize Git executable mode 100755 on Windows without changing the index",
		},
		{name: "unsupported mode", mode: "100600", goos: "linux", wantErr: "must be 100644 or 100755"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCandidateMode(test.mode, test.goos)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("validateCandidateMode(%q, %q) = %v", test.mode, test.goos, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateCandidateMode(%q, %q) = %v, want %q", test.mode, test.goos, err, test.wantErr)
			}
			if test.goos == "windows" && test.mode == "100755" {
				recovery := candidateModeRecovery(test.mode, test.goos)
				if !strings.Contains(recovery, "POSIX host") ||
					!strings.Contains(recovery, "ssb will not stage") {
					t.Fatalf("recovery = %q, want POSIX and no-staging guidance", recovery)
				}
			}
		})
	}
}

func TestRequireRegularBundleFileAcceptsRelativeRoot(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "evidence.json")
	if err := os.WriteFile(target, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if err := requireRegularBundleFile(".", target); err != nil {
		t.Fatalf("relative bundle root rejected: %v", err)
	}
}

func TestApplicationPlanModePortabilityContract(t *testing.T) {
	executable := FileState{
		Exists: true,
		SHA256: "sha256:" + strings.Repeat("a", 64),
		Mode:   "100755",
	}
	regular := FileState{
		Exists: true,
		SHA256: "sha256:" + strings.Repeat("b", 64),
		Mode:   "100644",
	}
	tests := []struct {
		name    string
		goos    string
		change  Change
		wantErr bool
	}{
		{
			name: "Windows rejects executable prestate removal",
			goos: "windows",
			change: Change{
				Path:      ".agents/skills/example/scripts/check.sh",
				Kind:      "remove",
				Prestate:  executable,
				Poststate: FileState{Exists: false},
			},
			wantErr: true,
		},
		{
			name: "Windows rejects executable poststate update",
			goos: "windows",
			change: Change{
				Path:      ".agents/skills/example/scripts/check.sh",
				Kind:      "write",
				Prestate:  regular,
				Poststate: executable,
			},
			wantErr: true,
		},
		{
			name: "Windows accepts regular transition",
			goos: "windows",
			change: Change{
				Path:      ".software-standards/rules/example.md",
				Kind:      "write",
				Prestate:  regular,
				Poststate: regular,
			},
		},
		{
			name: "POSIX accepts executable transition",
			goos: "linux",
			change: Change{
				Path:      ".agents/skills/example/scripts/check.sh",
				Kind:      "write",
				Prestate:  executable,
				Poststate: executable,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateApplicationPlanModes(applicationPlan{
				Changes: []Change{test.change},
			}, test.goos)
			if test.wantErr {
				if err == nil ||
					!strings.Contains(err.Error(), "POSIX host") ||
					!strings.Contains(err.Error(), "will not stage") {
					t.Fatalf("error = %v, want POSIX/no-staging guidance", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestApplicationPlanPreservesWindowsModePreconditionClassification(t *testing.T) {
	executable := FileState{
		Exists: true,
		SHA256: "sha256:" + strings.Repeat("a", 64),
		Mode:   "100755",
	}
	planErr := validateApplicationPlanModes(applicationPlan{
		Changes: []Change{{
			Path:      ".agents/skills/example/scripts/check.sh",
			Kind:      "remove",
			Prestate:  executable,
			Poststate: FileState{Exists: false},
		}},
	}, "windows")
	err := classifyApplicationPlanError(planErr)
	if !errors.Is(err, ErrPrecondition) {
		t.Fatalf("error = %v, want ErrPrecondition", err)
	}
	if errors.Is(err, ErrValidation) {
		t.Fatalf("error = %v, must not be reclassified as ErrValidation", err)
	}
}

func TestWindowsValidationBlocksExecutableEntrypointAndSupportBeforeMutation(t *testing.T) {
	action := Action{
		ID:          "update-skill",
		Disposition: DispositionUpdate,
		Sources: []ArtifactRef{{
			Kind: ArtifactSkill,
			ID:   "example-skill",
			Path: ".agents/skills/example-skill/SKILL.md",
		}},
		Target: &CandidateRef{
			Kind:       ArtifactSkill,
			ID:         "example-skill",
			TargetPath: ".agents/skills/example-skill/SKILL.md",
			SourcePath: "candidates/update-skill/SKILL.md",
			Mode:       "100755",
			SupportingFiles: []CandidateFileRef{{
				TargetPath: ".agents/skills/example-skill/scripts/check.sh",
				SourcePath: "candidates/update-skill/scripts/check.sh",
				Mode:       "100755",
			}},
		},
	}
	var diagnostics []Diagnostic
	validateActionShape(
		action,
		nil,
		1<<20,
		false,
		"windows",
		func(actionID, field, message, recovery string) {
			diagnostics = append(diagnostics, Diagnostic{
				Path: actionID, Field: field, Message: message, Recovery: recovery,
			})
		},
	)
	modeDiagnostics := 0
	for _, diagnostic := range diagnostics {
		if strings.HasSuffix(diagnostic.Field, "mode") {
			modeDiagnostics++
			if !strings.Contains(diagnostic.Message, "cannot materialize Git executable mode") ||
				!strings.Contains(diagnostic.Recovery, "POSIX host") {
				t.Fatalf("non-actionable mode diagnostic: %#v", diagnostic)
			}
		}
	}
	if modeDiagnostics != 2 {
		t.Fatalf("diagnostics = %#v, want root and supporting mode blocks", diagnostics)
	}
}

func TestCandidateValidationRejectsCaseFoldedTargetCollision(t *testing.T) {
	action := Action{
		ID:          "update-skill",
		Disposition: DispositionUpdate,
		Sources: []ArtifactRef{{
			Kind: ArtifactSkill,
			ID:   "example-skill",
			Path: ".agents/skills/example-skill/SKILL.md",
		}},
		Target: &CandidateRef{
			Kind:       ArtifactSkill,
			ID:         "example-skill",
			TargetPath: ".agents/skills/example-skill/SKILL.md",
			SourcePath: "candidates/update-skill/SKILL.md",
			Mode:       "100644",
			SupportingFiles: []CandidateFileRef{{
				TargetPath: ".agents/skills/example-skill/skill.md",
				SourcePath: "candidates/update-skill/skill.md",
				Mode:       "100644",
			}},
		},
	}
	var diagnostics []Diagnostic
	validateActionShape(
		action,
		nil,
		1<<20,
		false,
		"linux",
		func(actionID, field, message, recovery string) {
			diagnostics = append(diagnostics, Diagnostic{
				Path: actionID, Field: field, Message: message, Recovery: recovery,
			})
		},
	)
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "case-insensitive filesystem") {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want case-insensitive target collision", diagnostics)
}

func TestSafeRelativePathRejectsNonPortableForms(t *testing.T) {
	for _, candidate := range []string{
		`..\outside`,
		`.agents\skills\example-skill\SKILL.md`,
		`C:\outside`,
		`\\server\share`,
		`./inside`,
		`inside//file`,
		`candidates/update/CON`,
		`candidates/update/aux.txt`,
		`candidates/update/CONIN$`,
		`candidates/update/conout$.txt`,
		`candidates/update/COM¹`,
		`candidates/update/com².txt`,
		`candidates/update/COM³`,
		`candidates/update/LPT¹`,
		`candidates/update/lpt².txt`,
		`candidates/update/LPT³`,
		`candidates/update/name.`,
		`candidates/update/name `,
		`candidates/update/bad?.md`,
		`inside/../outside`,
	} {
		t.Run(candidate, func(t *testing.T) {
			if safeRelativePath(candidate) {
				t.Fatalf("safeRelativePath(%q) = true, want false", candidate)
			}
		})
	}

	for _, candidate := range []string{
		"README.md",
		".software-standards/rules/example.md",
		".agents/skills/example-skill/SKILL.md",
	} {
		t.Run(candidate, func(t *testing.T) {
			if !safeRelativePath(candidate) {
				t.Fatalf("safeRelativePath(%q) = false, want true", candidate)
			}
		})
	}
}

func TestValidateGovernedTreePathsRejectsNonPortableOrCollidingBaseline(t *testing.T) {
	for name, paths := range map[string][]string{
		"reserved device": {
			".agents/skills/example/SKILL.md",
			".agents/skills/example/CON",
		},
		"console input device": {
			".agents/skills/example/SKILL.md",
			".agents/skills/example/CONIN$",
		},
		"console output device": {
			".agents/skills/example/SKILL.md",
			".agents/skills/example/conout$.txt",
		},
		"superscript com device": {
			".agents/skills/example/SKILL.md",
			".agents/skills/example/COM¹",
		},
		"superscript lpt device": {
			".agents/skills/example/SKILL.md",
			".agents/skills/example/lpt³.txt",
		},
		"trailing dot": {
			".agents/skills/example/SKILL.md",
			".agents/skills/example/name.",
		},
		"backslash": {
			".agents/skills/example/SKILL.md",
			`.agents/skills/example/references\guide.md`,
		},
		"case folded collision": {
			".agents/skills/example/SKILL.md",
			".agents/skills/example/references/Guide.md",
			".agents/skills/example/references/guide.md",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateGovernedTreePaths(paths); err == nil {
				t.Fatalf("paths = %#v, want portable baseline block", paths)
			}
		})
	}
}

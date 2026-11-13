package prune_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

func TestCancellationDuringPlanDerivation(t *testing.T) {
	for _, operation := range []string{"validate", "approve", "apply", "status", "load", "verify"} {
		t.Run(operation, func(t *testing.T) {
			root := cancellationRemovalReview(t)
			if operation != "validate" && operation != "approve" {
				approveCancellationReview(t, root)
			}
			if operation == "load" || operation == "verify" {
				if _, err := prune.Apply(t.Context(), root, prune.ApplyOptions{ReviewID: "review-one", Write: true}); err != nil {
					t.Fatal(err)
				}
			}
			reviewRoot := filepath.Join(root, ".software-standards", "reviews", "review-one")
			priorEvents, priorErr := os.ReadFile(filepath.Join(reviewRoot, "events.jsonl"))
			repo, err := workspace.Open(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			ctx, finish := cancelAtPlanGit(t, false)
			switch operation {
			case "validate":
				_, _, err = prune.ValidateReview(ctx, repo, "review-one")
			case "approve":
				_, err = prune.Approve(ctx, root, prune.ApprovalOptions{ReviewID: "review-one", Approved: []string{"keep-rule", "orphan-skill"}})
			case "apply":
				_, err = prune.Apply(ctx, root, prune.ApplyOptions{ReviewID: "review-one", Write: true})
			case "status":
				_, _, err = prune.ReviewStatus(ctx, root, "review-one")
			case "load":
				_, _, err = prune.LoadReview(ctx, root, "review-one")
			case "verify":
				_, err = prune.Verify(ctx, root, prune.VerificationOptions{ReviewID: "review-one", ReceiptsDir: t.TempDir()})
			}
			finish()
			if !errors.Is(err, context.Canceled) || errors.Is(err, prune.ErrValidation) || errors.Is(err, prune.ErrPrecondition) {
				t.Fatalf("expected preserved cancellation, got %v", err)
			}
			events, readErr := os.ReadFile(filepath.Join(reviewRoot, "events.jsonl"))
			if string(events) != string(priorEvents) || errors.Is(readErr, os.ErrNotExist) != errors.Is(priorErr, os.ErrNotExist) {
				t.Fatalf("cancellation changed events: %q, %v", events, readErr)
			}
			for _, target := range []string{
				filepath.Join(reviewRoot, "application-journal.json"), filepath.Join(reviewRoot, ".transition.lock"),
				filepath.Join(root, ".software-standards", ".prune-mutation.lock"), filepath.Join(root, "AGENTS.md"),
				filepath.Join(root, "docs", "adr"),
			} {
				if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("cancellation left %s: %v", target, err)
				}
			}
		})
	}
}

func TestApprovalCancellationBeforeEvent(t *testing.T) {
	root := cancellationRemovalReview(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, err := prune.Approve(ctx, root, prune.ApprovalOptions{
		ReviewID: "review-one", Approved: []string{"keep-rule", "orphan-skill"},
		Now: func() time.Time { cancel(); return time.Now() },
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("approval error = %v", err)
	}
	status, _, err := prune.ReviewStatus(t.Context(), root, "review-one")
	if err != nil || status.Approved {
		t.Fatalf("approval advanced after cancellation: %+v, %v", status, err)
	}
}

func TestApplicationFinishesAfterJournalCancellation(t *testing.T) {
	root := cancellationRemovalReview(t)
	approveCancellationReview(t, root)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result, err := prune.Apply(ctx, root, prune.ApplyOptions{
		ReviewID: "review-one", Write: true,
		Now: func() time.Time { cancel(); return time.Now() },
	})
	if err != nil || len(result.Changes) == 0 || ctx.Err() != context.Canceled {
		t.Fatalf("bounded application = %+v, %v; caller = %v", result, err, ctx.Err())
	}
	status, diagnostics, err := prune.ReviewStatus(t.Context(), root, "review-one")
	if err != nil || len(diagnostics) != 0 || !status.Applied {
		t.Fatalf("application consumer = %+v, %v, %v", status, diagnostics, err)
	}
	for _, name := range []string{"application-journal.json", ".transition.lock"} {
		if _, err := os.Lstat(filepath.Join(root, ".software-standards", "reviews", "review-one", name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("bounded application left %s: %v", name, err)
		}
	}
}

func cancellationRemovalReview(t *testing.T) string {
	t.Helper()
	root := lifecycleRepository(t)
	createReviewWithProposal(t, root, false)
	review, _, err := prune.LoadReview(t.Context(), root, "review-one")
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
	return root
}

func approveCancellationReview(t *testing.T, root string) {
	t.Helper()
	if _, err := prune.Approve(t.Context(), root, prune.ApprovalOptions{
		ReviewID: "review-one", Approved: []string{"keep-rule", "orphan-skill"},
	}); err != nil {
		t.Fatal(err)
	}
}

func cancelAtPlanGit(t *testing.T, release bool) (context.Context, func()) {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	toolsDir := t.TempDir()
	gitName := "git"
	if runtime.GOOS == "windows" {
		gitName += ".exe"
	}
	build := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(toolsDir, gitName), "../../testdata/git-helper")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Git shim: %v\n%s", err, output)
	}
	ready := filepath.Join(toolsDir, "ready")
	releasePath := ""
	if release {
		releasePath = filepath.Join(toolsDir, "release")
	}
	t.Setenv("PATH", toolsDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SSB_REAL_GIT", realGit)
	t.Setenv("SSB_GIT_BLOCK_MATCH", "cat-file --batch-check")
	t.Setenv("SSB_GIT_READY", ready)
	t.Setenv("SSB_GIT_RELEASE", releasePath)
	deadline, stop := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(stop)
	ctx, cancel := context.WithCancel(deadline)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			if _, err := os.Stat(ready); err == nil {
				cancel()
				if release {
					done <- os.WriteFile(releasePath, []byte("resume\n"), 0o600)
				} else {
					done <- nil
				}
				return
			}
			select {
			case <-deadline.Done():
				done <- errors.New("plan Git handshake timed out")
				return
			case <-ticker.C:
			}
		}
	}()
	return ctx, func() {
		t.Helper()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

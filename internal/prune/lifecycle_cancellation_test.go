package prune_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/adr"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
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

func TestLoadReviewCancellationAfterEntry(t *testing.T) {
	for _, invalidCandidate := range []bool{false, true} {
		t.Run(strconv.FormatBool(invalidCandidate), func(t *testing.T) {
			root := lifecycleRepository(t)
			candidate := configureRuleUpdateCandidate(t, root)
			if invalidCandidate {
				if err := os.WriteFile(candidate, []byte("changed candidate\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			parent, cancel := context.WithCancel(t.Context())
			defer cancel()
			ctx := cancelAfterLoadEntryContext{Context: parent, cancel: cancel}
			review, diagnostics, err := prune.LoadReview(&ctx, root, "review-one")
			if parent.Err() != context.Canceled || !errors.Is(err, context.Canceled) {
				t.Fatalf("caller=%v, loaded=%s, diagnostics=%v, error=%v", parent.Err(), review.Context.ReviewID, diagnostics, err)
			}
			if review.Context.ReviewID != "" || len(diagnostics) != 0 {
				t.Fatalf("canceled load exposed a review or diagnostics: %+v, %v", review, diagnostics)
			}
			for _, name := range []string{"events.jsonl", "application-journal.json", ".transition.lock"} {
				if _, err := os.Lstat(filepath.Join(root, ".software-standards", "reviews", "review-one", name)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("canceled load left %s: %v", name, err)
				}
			}
		})
	}
}

// Cancel just after the loader's entry check observes a live caller. This
// reproduces cancellation during the subsequent filesystem validation window.
type cancelAfterLoadEntryContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (ctx *cancelAfterLoadEntryContext) Err() error {
	err := ctx.Context.Err()
	ctx.checks++
	if ctx.checks == 2 {
		ctx.cancel()
	}
	return err
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
	return cancelAtGit(t, gitCancellationOptions{Match: "cat-file --batch-check", Release: release})
}

type gitCancellationOptions struct {
	Match              string
	Release            bool
	PackValidationRoot string
}

func cancelAtGit(t *testing.T, options gitCancellationOptions) (context.Context, func()) {
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
	if options.Release {
		releasePath = filepath.Join(toolsDir, "release")
	}
	t.Setenv("PATH", toolsDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SSB_REAL_GIT", realGit)
	t.Setenv("SSB_GIT_BLOCK_MATCH", options.Match)
	t.Setenv("SSB_GIT_BLOCK_SKIP", "0")
	t.Setenv("SSB_GIT_RECORD_ONLY", "0")
	t.Setenv("SSB_GIT_READY", ready)
	t.Setenv("SSB_GIT_RELEASE", releasePath)
	if options.PackValidationRoot != "" {
		t.Setenv("SSB_GIT_RECORD_ONLY", "1")
		repo, err := workspace.Open(t.Context(), options.PackValidationRoot)
		if err != nil {
			t.Fatal(err)
		}
		if _, diagnostics, err := rulepack.ValidateRetainedPack(t.Context(), repo); err != nil || len(diagnostics) != 0 {
			t.Fatalf("calibrate pack validation: %v, %v", diagnostics, err)
		}
		data, err := os.ReadFile(ready + ".count")
		if err != nil {
			t.Fatal(err)
		}
		count, err := strconv.Atoi(string(data))
		if err != nil || count < 1 {
			t.Fatalf("pack evidence count = %q, %v", data, err)
		}
		if err := os.WriteFile(ready+".count", []byte("0"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SSB_GIT_BLOCK_SKIP", strconv.Itoa(count))
		t.Setenv("SSB_GIT_RECORD_ONLY", "0")
	}
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
				if options.Release {
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

func TestCancellationDuringCandidateAndResultingRuleEvidence(t *testing.T) {
	for _, candidateUpdate := range []bool{true, false} {
		name := "retained rule evidence"
		if candidateUpdate {
			name = "candidate evidence"
		}
		t.Run(name, func(t *testing.T) {
			root := lifecycleRepository(t)
			if candidateUpdate {
				configureRuleUpdateCandidate(t, root)
			} else {
				createReviewWithProposal(t, root, false)
			}
			for _, operation := range []string{"validate", "approve", "apply"} {
				t.Run(operation, func(t *testing.T) {
					if operation == "apply" {
						approveCancellationReview(t, root)
					}
					eventPath := filepath.Join(root, ".software-standards", "reviews", "review-one", "events.jsonl")
					priorEvents, priorEventErr := os.ReadFile(eventPath)
					oid := strings.TrimSpace(git(t, root, "rev-parse", "HEAD:README.md"))
					ctx, finish := cancelAtGit(t, gitCancellationOptions{
						Match:              "cat-file blob " + oid,
						PackValidationRoot: root,
					})
					var err error
					if operation == "validate" {
						repo, openErr := workspace.Open(ctx, root)
						if openErr != nil {
							t.Fatal(openErr)
						}
						_, diagnostics, validateErr := prune.ValidateReview(ctx, repo, "review-one")
						err = validateErr
						if len(diagnostics) != 0 {
							t.Logf("diagnostics: %v", diagnostics)
						}
					} else if operation == "apply" {
						_, err = prune.Apply(ctx, root, prune.ApplyOptions{ReviewID: "review-one", Write: true})
					} else {
						_, err = prune.Approve(ctx, root, prune.ApprovalOptions{
							ReviewID: "review-one", Approved: []string{"keep-rule", "orphan-skill"},
						})
					}
					finish()
					if !errors.Is(err, context.Canceled) || errors.Is(err, prune.ErrValidation) {
						t.Fatalf("evidence cancellation lost its cause: %v", err)
					}
					events, readErr := os.ReadFile(eventPath)
					if string(events) != string(priorEvents) || errors.Is(readErr, os.ErrNotExist) != errors.Is(priorEventErr, os.ErrNotExist) {
						t.Fatalf("canceled evidence validation changed events: %q, %v", events, readErr)
					}
					for _, name := range []string{"application-journal.json", ".transition.lock"} {
						if _, err := os.Stat(filepath.Join(root, ".software-standards", "reviews", "review-one", name)); !errors.Is(err, os.ErrNotExist) {
							t.Fatalf("canceled evidence validation left %s: %v", name, err)
						}
					}
				})
			}
		})
	}
}

func TestCanceledNoOpRenderDoesNotConsumeTransition(t *testing.T) {
	root := cancellationRemovalReview(t)
	approveCancellationReview(t, root)
	if _, err := prune.Apply(t.Context(), root, prune.ApplyOptions{ReviewID: "review-one", Write: true}); err != nil {
		t.Fatal(err)
	}
	repo, err := workspace.Open(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.ValidateRetainedPack(t.Context(), repo)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("retained pack = %v, %v", diagnostics, err)
	}
	if _, err := render.Apply(t.Context(), repo, pack, false); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	transition, err := prune.BeginTransition(ctx, root, prune.TransitionOptions{
		ReviewID: "review-one", Kind: prune.EventRendered,
		Now: func() time.Time { cancel(); return time.Now() },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = transition.Cancel() })
	result, err := render.Apply(ctx, repo, pack, false)
	if err != nil || result.Changed {
		t.Fatalf("expected no-op render: %+v, %v", result, err)
	}
	_, completionErr := transition.Complete(ctx, result)
	if err := transition.Cancel(); err != nil {
		t.Fatal(err)
	}
	status, _, err := prune.ReviewStatus(t.Context(), root, "review-one")
	if !errors.Is(completionErr, context.Canceled) || err != nil || status.Rendered {
		t.Fatalf("canceled no-op advanced review: completion=%v status=%+v read=%v", completionErr, status, err)
	}
	retry, err := prune.BeginTransition(t.Context(), root, prune.TransitionOptions{ReviewID: "review-one", Kind: prune.EventRendered})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = retry.Cancel() })
	if _, err := retry.Complete(t.Context(), result); err != nil {
		t.Fatal(err)
	}
	status, _, err = prune.ReviewStatus(t.Context(), root, "review-one")
	if err != nil || !status.Rendered {
		t.Fatalf("no-op retry didn't complete: %+v, %v", status, err)
	}
}

func TestPublishedRenderCompletesEventAfterCancellation(t *testing.T) {
	root := cancellationRemovalReview(t)
	approveCancellationReview(t, root)
	if _, err := prune.Apply(t.Context(), root, prune.ApplyOptions{ReviewID: "review-one", Write: true}); err != nil {
		t.Fatal(err)
	}
	repo, err := workspace.Open(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.ValidateRetainedPack(t.Context(), repo)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("retained pack = %v, %v", diagnostics, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	transition, err := prune.BeginTransition(ctx, root, prune.TransitionOptions{
		ReviewID: "review-one", Kind: prune.EventRendered,
		Now: func() time.Time { cancel(); return time.Now() },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = transition.Cancel() })
	result, err := render.Apply(ctx, repo, pack, false)
	if err != nil || !result.Changed {
		t.Fatalf("expected publication: %+v, %v", result, err)
	}
	event, err := transition.Complete(context.WithoutCancel(ctx), result)
	if err != nil || event.EventDigest == "" || ctx.Err() != context.Canceled {
		t.Fatalf("bounded completion: event=%+v error=%v caller=%v", event, err, ctx.Err())
	}
	status, _, err := prune.ReviewStatus(t.Context(), root, "review-one")
	if err != nil || !status.Rendered {
		t.Fatalf("published render event missing: %+v, %v", status, err)
	}
	content, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil || string(content) != string(result.Content) {
		t.Fatalf("published render changed: %v", err)
	}
}

func TestPublishedADRCompletesEventAfterCancellation(t *testing.T) {
	root := cancellationRemovalReview(t)
	approveCancellationReview(t, root)
	if _, err := prune.Apply(t.Context(), root, prune.ApplyOptions{ReviewID: "review-one", Write: true}); err != nil {
		t.Fatal(err)
	}
	repo, err := workspace.Open(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.ValidateRetainedPack(t.Context(), repo)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("retained pack = %v, %v", diagnostics, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	transition, err := prune.BeginTransition(ctx, root, prune.TransitionOptions{
		ReviewID: "review-one", Kind: prune.EventADR,
		Now: func() time.Time { cancel(); return time.Now() },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = transition.Cancel() })
	result, err := adr.Create(ctx, repo, pack, adr.Options{})
	if err != nil || !result.Created {
		t.Fatalf("expected ADR publication: %+v, %v", result, err)
	}
	event, err := transition.Complete(context.WithoutCancel(ctx), result)
	if err != nil || event.EventDigest == "" || ctx.Err() != context.Canceled {
		t.Fatalf("bounded completion: event=%+v error=%v caller=%v", event, err, ctx.Err())
	}
	status, _, err := prune.ReviewStatus(t.Context(), root, "review-one")
	if err != nil || !status.ADRRecorded {
		t.Fatalf("published ADR event missing: %+v, %v", status, err)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.Path)))
	if err != nil || string(content) != string(result.Content) {
		t.Fatalf("published ADR changed: %v", err)
	}
}

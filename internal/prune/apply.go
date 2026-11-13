package prune

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

// ApplyOptions selects a review and requires Write for mutation.
type ApplyOptions struct {
	ReviewID string
	Write    bool
	Now      func() time.Time
}

// Change is one complete-file operation in an application plan.
type Change struct {
	ActionID  string    `json:"action_id"`
	Path      string    `json:"path"`
	Kind      string    `json:"kind"`
	Prestate  FileState `json:"prestate"`
	Poststate FileState `json:"poststate"`
}

// ApplyResult reports the bounded plan or completed application.
type ApplyResult struct {
	DryRun            bool     `json:"dry_run"`
	NoChangesApproved bool     `json:"no_changes_approved"`
	PlanDigest        string   `json:"plan_digest"`
	Changes           []Change `json:"changes"`
}

type operation struct {
	Change
	Content        []byte
	ExpectedAbsent bool
	ExpectedSHA256 string
	Mode           os.FileMode
}

type candidateInput struct {
	sourcePath string
	target     *CandidateRef
	mode       string
}

// Apply computes a dry run by default and mutates only when Write is true.
func Apply(
	ctx context.Context,
	repoPath string,
	options ApplyOptions,
) (returned ApplyResult, returnErr error) {
	publication := defaultApplicationPublicationOps()
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return ApplyResult{}, err
	}
	var lockedReviewStore *reviewStore
	var unlockReview func() error
	var unlockMutation func() error
	defer func() {
		if unlockMutation != nil {
			returnErr = errors.Join(returnErr, unlockMutation())
		}
		if unlockReview != nil {
			returnErr = errors.Join(returnErr, unlockReview())
		}
	}()
	if options.Write {
		lockedReviewStore, unlockReview, err = acquireReviewLock(ctx, repo.Root(), options.ReviewID)
		if err != nil {
			return ApplyResult{}, err
		}
		unlockMutation, err = defaultMutationLockOps().acquireMutationLock(ctx, repo.Root(), options.ReviewID)
		if err != nil {
			return ApplyResult{}, err
		}
	}
	var review Review
	var diagnostics []Diagnostic
	if lockedReviewStore != nil {
		review, diagnostics, err = loadReviewFromStore(ctx, lockedReviewStore)
	} else {
		review, diagnostics, err = LoadReview(ctx, repo.Root(), options.ReviewID)
	}
	if err != nil {
		return ApplyResult{}, err
	}
	if len(diagnostics) != 0 {
		return ApplyResult{}, validationError(
			"proposal is invalid: %s; run ssb prune validate",
			diagnostics[0].Message,
		)
	}
	approval, err := approvalFor(review)
	if err != nil {
		return ApplyResult{}, preconditionError("%v", err)
	}
	for _, event := range review.Events {
		if event.Kind == EventApplied {
			return ApplyResult{}, preconditionError("review already has an application event")
		}
	}
	if repo.Baseline() != review.Context.BaselineCommit {
		return ApplyResult{}, preconditionError("review baseline is stale: expected %s, found %s", review.Context.BaselineCommit, repo.Baseline())
	}
	status, err := repo.Git(
		ctx,
		"status",
		"--porcelain=v1",
		"-z",
		"--untracked-files=no",
		"--ignore-submodules=untracked",
	)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("read tracked worktree state: %w", err)
	}
	if len(status) != 0 {
		return ApplyResult{}, preconditionError("tracked or staged changes block application; commit, stash, or restore them")
	}
	if err := repo.RejectUntrackedConfigurations(ctx); err != nil {
		return ApplyResult{}, err
	}
	plan, operations, err := planOperations(
		ctx,
		repo,
		review,
		approval,
	)
	if err != nil {
		return ApplyResult{}, classifyApplicationPlanError(err)
	}
	result := ApplyResult{
		DryRun:     !options.Write,
		PlanDigest: plan.PlanDigest,
		Changes:    make([]Change, len(plan.Changes)),
	}
	copy(result.Changes, plan.Changes)
	if len(result.Changes) == 0 {
		result.NoChangesApproved = true
		return result, nil
	}
	if !options.Write {
		return result, nil
	}
	journal, err := captureJournal(
		repo.Root(),
		review,
		plan,
		operations,
	)
	if err != nil {
		return ApplyResult{}, err
	}
	journalStore := lockedReviewStore
	journalData, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return ApplyResult{}, fmt.Errorf("encode application journal: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return ApplyResult{}, err
	}
	// From journal creation through event publication, complete or roll back.
	ctx = context.WithoutCancel(ctx)
	if err := journalStore.WriteExclusive("application-journal.json", append(journalData, '\n'), 0o600); err != nil {
		return ApplyResult{}, fmt.Errorf("application recovery journal already exists or cannot be created: %w", err)
	}
	completed, executeErr := publication.executeOperations(repo.Root(), operations)
	if executeErr != nil {
		rollbackErr := publication.restoreJournalPaths(
			repo.Root(),
			journal,
			completed,
			operationPoststates(operations),
		)
		if rollbackErr == nil {
			if removeErr := removeApplicationJournal(journalStore); removeErr != nil {
				return ApplyResult{}, fmt.Errorf(
					"apply review: %w; changed files were restored but recovery journal cleanup failed: %v; run ssb prune recover --review %s",
					executeErr,
					removeErr,
					options.ReviewID,
				)
			}
			return ApplyResult{}, fmt.Errorf("apply review: %w; all changed files were restored", executeErr)
		}
		return ApplyResult{}, fmt.Errorf(
			"apply review: %w; rollback also failed: %v; run ssb prune recover --review %s",
			executeErr,
			rollbackErr,
			options.ReviewID,
		)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		rollbackErr := publication.restoreJournal(repo.Root(), journal, operationPoststates(operations))
		if rollbackErr == nil {
			if removeErr := removeApplicationJournal(journalStore); removeErr != nil {
				return ApplyResult{}, fmt.Errorf(
					"encode application event: %w; changed files were restored but recovery journal cleanup failed: %v; run ssb prune recover --review %s",
					err,
					removeErr,
					options.ReviewID,
				)
			}
			return ApplyResult{}, fmt.Errorf("encode application event: %w; all changed files were restored", err)
		}
		return ApplyResult{}, fmt.Errorf(
			"encode application event: %w; rollback failed: %v; run ssb prune recover --review %s",
			err,
			rollbackErr,
			options.ReviewID,
		)
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	event := newEvent(
		review,
		EventApplied,
		payload,
		now().UTC(),
	)
	if err := appendEvent(ctx, review, event); err != nil {
		rollbackErr := publication.restoreJournal(repo.Root(), journal, operationPoststates(operations))
		if rollbackErr == nil {
			if removeErr := removeApplicationJournal(journalStore); removeErr != nil {
				return ApplyResult{}, fmt.Errorf(
					"record application event: %w; changed files were restored but recovery journal cleanup failed: %v; run ssb prune recover --review %s",
					err,
					removeErr,
					options.ReviewID,
				)
			}
			return ApplyResult{}, fmt.Errorf("record application event: %w; all changed files were restored", err)
		}
		return ApplyResult{}, fmt.Errorf(
			"record application event: %w; rollback failed: %v; run ssb prune recover --review %s",
			err,
			rollbackErr,
			options.ReviewID,
		)
	}
	if err := unlockMutation(); err != nil {
		return ApplyResult{}, fmt.Errorf(
			"application succeeded but lock cleanup failed: %w; run ssb prune recover --review %s --clear-stale-lock",
			err,
			options.ReviewID,
		)
	}
	unlockMutation = nil
	if err := lockedReviewStore.events.removeLock(lockedReviewStore); err != nil {
		return ApplyResult{}, fmt.Errorf(
			"application succeeded but lock cleanup failed: %w; run ssb prune recover --review %s --clear-stale-lock",
			err,
			options.ReviewID,
		)
	}
	// The transition lock is gone, but the identity-pinned descriptor remains
	// open until the journal is removed. This avoids reopening a replaced path.
	unlockReview = lockedReviewStore.Close
	if err := cleanupApplicationJournal(journalStore, options.ReviewID); err != nil {
		return ApplyResult{}, err
	}
	closeReviewStore := unlockReview
	unlockReview = nil
	if err := closeReviewStore(); err != nil {
		return ApplyResult{}, fmt.Errorf(
			"application succeeded but review descriptor cleanup failed: %w",
			err,
		)
	}
	return result, nil
}

func classifyApplicationPlanError(err error) error {
	if errors.Is(err, ErrPrecondition) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, workspace.ErrGitOperation) {
		return err
	}
	return validationError("%v", err)
}

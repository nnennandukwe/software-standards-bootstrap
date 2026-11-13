package prune

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

// Recover restores an interrupted application journal, or finalizes journal
// cleanup when the application event was already recorded.
func Recover(
	ctx context.Context,
	repoPath, reviewID string,
	clearStaleLock bool,
) (returnErr error) {
	publication := defaultApplicationPublicationOps()
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return err
	}
	if clearStaleLock {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Clearing crash-left locks begins this bounded recovery sequence.
		ctx = context.WithoutCancel(ctx)
		store, storeErr := openReviewStore(repo.Root(), reviewID)
		if storeErr != nil {
			return storeErr
		}
		if _, readErr := store.Lstat(".transition.lock"); readErr == nil {
			if removeErr := store.Remove(".transition.lock"); removeErr != nil &&
				!errors.Is(removeErr, os.ErrNotExist) {
				_ = store.Close()
				return fmt.Errorf("clear stale review transition lock: %w", removeErr)
			}
		} else if !errors.Is(readErr, os.ErrNotExist) {
			_ = store.Close()
			return fmt.Errorf("read stale review transition lock: %w", readErr)
		}
		if err := clearMutationLock(repo.Root(), reviewID); err != nil {
			_ = store.Close()
			return err
		}
		_, journalErr := store.Lstat("application-journal.json")
		closeErr := store.Close()
		if errors.Is(journalErr, os.ErrNotExist) {
			if closeErr != nil {
				return closeErr
			}
			return nil
		} else if journalErr != nil {
			return fmt.Errorf("inspect application recovery journal: %w", journalErr)
		} else if closeErr != nil {
			return closeErr
		}
	}
	lockedReviewStore, unlock, err := acquireReviewLock(ctx, repo.Root(), reviewID)
	if err != nil {
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, unlock())
	}()
	unlockMutation, err := defaultMutationLockOps().acquireMutationLock(ctx, repo.Root(), reviewID)
	if err != nil {
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, unlockMutation())
	}()
	review, diagnostics, err := loadReviewFromStore(ctx, lockedReviewStore)
	if err != nil {
		return err
	}
	if len(diagnostics) != 0 {
		return validationError("proposal has %d error(s)", len(diagnostics))
	}
	if repo.Baseline() != review.Context.BaselineCommit {
		return preconditionError("recovery baseline is stale: expected %s, found %s", review.Context.BaselineCommit, repo.Baseline())
	}
	if _, err := approvalFor(review); err != nil {
		return preconditionError("%v", err)
	}
	journalStore := lockedReviewStore
	data, _, err := journalStore.ReadRegular("application-journal.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return preconditionError("review has no application recovery journal")
		}
		return fmt.Errorf("read application journal: %w", err)
	}
	var journal applicationJournal
	if err := decodeStrictJSON(data, &journal); err != nil {
		return fmt.Errorf("parse application journal: %w", err)
	}
	if journal.Schema != journalSchema || journal.ReviewID != reviewID {
		return fmt.Errorf("application journal identity is invalid")
	}
	if !validDigest(journal.ProposalDigest) || !validDigest(journal.PlanDigest) {
		return fmt.Errorf("application journal proposal or plan digest is invalid")
	}
	if journal.ProposalDigest != review.ProposalDigest {
		return fmt.Errorf("application journal is bound to a different proposal digest")
	}
	seen := make(map[string]struct{}, len(journal.Entries))
	for _, entry := range journal.Entries {
		if !validApplicationPath(entry.Path) {
			return fmt.Errorf("application journal contains non-canonical target %s", entry.Path)
		}
		if _, duplicate := seen[entry.Path]; duplicate {
			return fmt.Errorf("application journal repeats target %s", entry.Path)
		}
		seen[entry.Path] = struct{}{}
	}
	if err := validateJournalPlan(
		ctx,
		repo,
		review,
		journal,
	); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return validationError("%v", err)
	}
	for _, event := range review.Events {
		if event.Kind == EventApplied && event.ProposalDigest == journal.ProposalDigest {
			var applied ApplyResult
			if err := decodeStrictJSON(event.Payload, &applied); err != nil {
				return validationError("application event payload is invalid: %v", err)
			}
			if applied.PlanDigest != journal.PlanDigest {
				return validationError("application journal is bound to a different application plan")
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			return removeApplicationJournal(journalStore)
		}
	}
	restorePaths, poststates, err := recoverableJournalPaths(
		ctx,
		repo.Root(),
		review,
		journal,
	)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return validationError("%v", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Restoration and journal cleanup are bounded once the first claim changes.
	if err := publication.restoreJournalPaths(
		repo.Root(),
		journal,
		restorePaths,
		poststates,
	); err != nil {
		return fmt.Errorf("restore application journal: %w", err)
	}
	return removeApplicationJournal(journalStore)
}

func recoverableJournalPaths(
	ctx context.Context,
	repoRoot string,
	review Review,
	journal applicationJournal,
) (map[string]struct{}, map[string]string, error) {
	approval, err := approvalFor(review)
	if err != nil {
		return nil, nil, err
	}
	plan, err := canonicalApplicationPlan(ctx, review, approval)
	if err != nil {
		return nil, nil, err
	}
	if journal.PlanDigest != plan.PlanDigest {
		return nil, nil, fmt.Errorf("application journal is bound to a different application plan")
	}
	poststate := planOperationPoststates(plan)
	restore := make(map[string]struct{})
	for _, entry := range journal.Entries {
		target, err := resolvePortablePath(repoRoot, entry.Path)
		if err != nil {
			return nil, nil, err
		}
		data, readErr := os.ReadFile(target)
		absent := errors.Is(readErr, os.ErrNotExist)
		if readErr != nil && !absent {
			return nil, nil, fmt.Errorf("inspect recovery state for %s: %w", entry.Path, readErr)
		}
		currentDigest := ""
		if !absent {
			currentDigest = digestBytes(data)
		}
		prestateMatches := entry.Existed && !absent && currentDigest == digestBytes(entry.Content)
		if !entry.Existed && absent {
			prestateMatches = true
		}
		expectedPoststate, exists := poststate[entry.Path]
		if !exists {
			return nil, nil, fmt.Errorf("journal target %s has no approved poststate", entry.Path)
		}
		poststateMatches := expectedPoststate == "" && absent
		if expectedPoststate != "" && !absent && currentDigest == expectedPoststate {
			poststateMatches = true
		}
		_, preClaimErr := os.Lstat(claimPathForTarget(target, prestateClaim))
		_, postClaimErr := os.Lstat(claimPathForTarget(target, poststateClaim))
		if preClaimErr == nil || postClaimErr == nil {
			restore[entry.Path] = struct{}{}
			continue
		} else if !errors.Is(preClaimErr, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("inspect recovery prestate claim for %s: %w", entry.Path, preClaimErr)
		} else if !errors.Is(postClaimErr, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("inspect recovery poststate claim for %s: %w", entry.Path, postClaimErr)
		}
		switch {
		case prestateMatches:
			continue
		case poststateMatches:
			restore[entry.Path] = struct{}{}
		default:
			return nil, nil, fmt.Errorf("recovery target %s contains neither approved prestate nor poststate bytes", entry.Path)
		}
	}
	return restore, poststate, nil
}

func validateJournalPlan(
	ctx context.Context,
	repo *workspace.Repository,
	review Review,
	journal applicationJournal,
) error {
	approval, err := approvalFor(review)
	if err != nil {
		return err
	}
	plan, err := canonicalApplicationPlan(ctx, review, approval)
	if err != nil {
		return err
	}
	if journal.PlanDigest != plan.PlanDigest {
		return fmt.Errorf("application journal is bound to a different application plan")
	}
	expected := make(map[string]Change, len(plan.Changes))
	for _, change := range plan.Changes {
		expected[change.Path] = change
	}
	if len(journal.Entries) != len(expected) {
		return fmt.Errorf("application journal does not match the approved operation count")
	}
	seenDirectories := make(map[string]struct{}, len(journal.CreatedDirectories))
	for _, directory := range journal.CreatedDirectories {
		if !safeRelativePath(directory) {
			return fmt.Errorf("application journal contains unsafe created directory %s", directory)
		}
		if _, duplicate := seenDirectories[directory]; duplicate {
			return fmt.Errorf("application journal repeats created directory %s", directory)
		}
		seenDirectories[directory] = struct{}{}
		ownsDirectory := false
		for itemPath := range expected {
			ownsDirectory = ownsDirectory || strings.HasPrefix(itemPath, directory+"/")
		}
		if !ownsDirectory {
			return fmt.Errorf("application journal directory %s is outside the approved operation plan", directory)
		}
	}
	for _, entry := range journal.Entries {
		change, ok := expected[entry.Path]
		if !ok {
			return fmt.Errorf("application journal target %s is not in the approved operation plan", entry.Path)
		}
		if !change.Prestate.Exists {
			if entry.Existed || len(entry.Content) != 0 || entry.Mode != 0 {
				return fmt.Errorf("application journal invents prestate for new target %s", entry.Path)
			}
			continue
		}
		if !entry.Existed || digestBytes(entry.Content) != change.Prestate.SHA256 {
			return fmt.Errorf("application journal prestate does not match %s", entry.Path)
		}
		baselineEntry, exists, err := repo.EntryAtBaseline(ctx, entry.Path)
		if err != nil || !exists {
			return fmt.Errorf("read baseline mode for %s", entry.Path)
		}
		if entry.Mode > 0o777 ||
			materializedMode(os.FileMode(entry.Mode)) != baselineEntry.Mode {
			return fmt.Errorf("application journal mode does not match %s", entry.Path)
		}
	}
	return nil
}

func planOperationPoststates(plan applicationPlan) map[string]string {
	result := make(map[string]string, len(plan.Changes))
	for _, change := range plan.Changes {
		if change.Poststate.Exists {
			result[change.Path] = change.Poststate.SHA256
		} else {
			result[change.Path] = ""
		}
	}
	return result
}

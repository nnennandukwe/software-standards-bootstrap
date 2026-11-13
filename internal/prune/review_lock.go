package prune

import (
	"context"
	"errors"
	"fmt"
	"os"
)

func acquireReviewLock(ctx context.Context, repoRoot, reviewID string) (*reviewStore, func() error, error) {
	store, err := openReviewStore(repoRoot, reviewID)
	if err != nil {
		return nil, nil, err
	}
	lockPath := store.display(".transition.lock")
	if err := ctx.Err(); err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	err = store.WriteExclusive(".transition.lock", []byte(reviewID+"\n"), 0o600)
	if err != nil {
		_ = store.Close()
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, preconditionError("review %s does not exist", reviewID)
		}
		if errors.Is(err, os.ErrExist) {
			return nil, nil, preconditionError("another review transition is active; if it crashed, confirm no process is active and remove %s", lockPath)
		}
		return nil, nil, fmt.Errorf("create review transition lock: %w", err)
	}
	return store, func() error {
		removeErr := store.events.removeLock(store)
		closeErr := store.Close()
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(
				fmt.Errorf("remove review transition lock %s: %w", lockPath, removeErr),
				closeErr,
			)
		}
		return closeErr
	}, nil
}

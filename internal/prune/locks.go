package prune

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (operations *mutationLockOps) acquireMutationLock(ctx context.Context, repoRoot, reviewID string) (func() error, error) {
	lockPath := filepath.Join(repoRoot, ".software-standards", ".prune-mutation.lock")
	root, packIdentity, err := openPruneRepositoryRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	pack, err := openPinnedPack(root, packIdentity)
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("pin .software-standards for mutation lock: %w", err)
	}
	if err := ctx.Err(); err != nil {
		_ = pack.Close()
		_ = root.Close()
		return nil, err
	}
	file, err := pack.OpenFile(".prune-mutation.lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		err = writeDurableExclusiveWithCleanup(
			lockPath,
			file,
			[]byte(reviewID+"\n"),
			0o600,
			func() error { return pack.Remove(".prune-mutation.lock") },
		)
	}
	closePackErr := pack.Close()
	if err != nil {
		_ = root.Close()
		if errors.Is(err, os.ErrExist) {
			return nil, preconditionError("another prune application or recovery owns the repository mutation lock")
		}
		return nil, fmt.Errorf("create repository prune mutation lock: %w", err)
	}
	if closePackErr != nil {
		_ = root.Close()
		return nil, fmt.Errorf("close pinned standards pack after mutation lock creation: %w", closePackErr)
	}
	return func() error {
		removeErr := operations.remove(root, packIdentity)
		closeErr := root.Close()
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(
				fmt.Errorf("remove repository prune mutation lock %s: %w", lockPath, removeErr),
				closeErr,
			)
		}
		return closeErr
	}, nil
}

func clearMutationLock(repoRoot, reviewID string) error {
	root, packIdentity, err := openPruneRepositoryRoot(repoRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	pack, err := openPinnedPack(root, packIdentity)
	if err != nil {
		return fmt.Errorf("pin .software-standards to clear mutation lock: %w", err)
	}
	defer pack.Close()
	data, err := pack.ReadFile(".prune-mutation.lock")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read repository prune mutation lock: %w", err)
	}
	if strings.TrimSpace(string(data)) != reviewID {
		return preconditionError("repository mutation lock belongs to another review and cannot be cleared")
	}
	if err := pack.Remove(".prune-mutation.lock"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear repository prune mutation lock: %w", err)
	}
	return nil
}

type mutationLockOps struct {
	remove func(*os.Root, os.FileInfo) error
}

func defaultMutationLockOps() *mutationLockOps {
	return &mutationLockOps{remove: func(root *os.Root, identity os.FileInfo) error {
		pack, err := openPinnedPack(root, identity)
		if err != nil {
			return err
		}
		defer pack.Close()
		return pack.Remove(".prune-mutation.lock")
	}}
}

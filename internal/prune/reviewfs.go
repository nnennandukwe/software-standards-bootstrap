package prune

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	renameAtomicReviewFile = (*os.Root).Rename
	removeAtomicReviewFile = (*os.Root).Remove
)

// reviewStore keeps lifecycle operations anchored to an already-open
// repository root. A concurrent rename or symlink replacement of the review
// path therefore cannot redirect an operation outside the repository.
type reviewStore struct {
	root     *os.Root
	repoRoot string
	reviewID string
	relative string
	absolute string
	identity os.FileInfo

	parentRelative string
	parentIdentity os.FileInfo
}

func createReviewStagingStore(repoRoot, reviewID string) (*reviewStore, string, error) {
	_, err := reviewRoot(repoRoot, reviewID)
	if err != nil {
		return nil, "", err
	}
	root, _, err := openPruneRepositoryRoot(repoRoot)
	if err != nil {
		return nil, "", err
	}
	parent := filepath.Join(".software-standards", "reviews")
	if err := root.MkdirAll(parent, 0o755); err != nil {
		_ = root.Close()
		return nil, "", fmt.Errorf("create reviews directory: %w", err)
	}
	parentInfo, err := root.Lstat(parent)
	if err != nil || parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() {
		_ = root.Close()
		if err != nil {
			return nil, "", fmt.Errorf("inspect reviews directory: %w", err)
		}
		return nil, "", fmt.Errorf("reviews path is not a real directory")
	}
	target := filepath.Join(parent, reviewID)
	if _, err := root.Lstat(target); err == nil {
		_ = root.Close()
		return nil, "", preconditionError("review %q already exists", reviewID)
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = root.Close()
		return nil, "", fmt.Errorf("inspect review path: %w", err)
	}
	for attempt := 0; attempt < 32; attempt++ {
		suffix, err := randomReviewSuffix()
		if err != nil {
			_ = root.Close()
			return nil, "", err
		}
		staging := filepath.Join(parent, "."+reviewID+"-"+suffix)
		if err := root.Mkdir(staging, 0o700); errors.Is(err, os.ErrExist) {
			continue
		} else if err != nil {
			_ = root.Close()
			return nil, "", fmt.Errorf("create review staging directory: %w", err)
		}
		identity, err := root.Lstat(staging)
		if err != nil {
			_ = root.RemoveAll(staging)
			_ = root.Close()
			return nil, "", fmt.Errorf("inspect review staging directory: %w", err)
		}
		return &reviewStore{
			root:           root,
			repoRoot:       repoRoot,
			reviewID:       reviewID,
			relative:       staging,
			absolute:       filepath.Join(repoRoot, staging),
			identity:       identity,
			parentRelative: parent,
			parentIdentity: parentInfo,
		}, target, nil
	}
	_ = root.Close()
	return nil, "", fmt.Errorf("create unique review staging directory: too many collisions")
}

func openReviewStore(repoRoot, reviewID string) (*reviewStore, error) {
	absolute, err := reviewRoot(repoRoot, reviewID)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("open repository root for review %s: %w", reviewID, err)
	}
	store := &reviewStore{
		root:     root,
		repoRoot: repoRoot,
		reviewID: reviewID,
		relative: filepath.Join(".software-standards", "reviews", reviewID),
		absolute: absolute,
	}
	identity, err := root.Lstat(store.relative)
	if err != nil {
		_ = root.Close()
		if errors.Is(err, os.ErrNotExist) {
			return nil, preconditionError("review %s does not exist", reviewID)
		}
		return nil, err
	}
	if identity.Mode()&os.ModeSymlink != 0 || !identity.IsDir() {
		_ = root.Close()
		return nil, fmt.Errorf("review path %s is not a real directory", store.absolute)
	}
	store.identity = identity
	if err := store.verifyIdentity(); err != nil {
		_ = root.Close()
		return nil, err
	}
	return store, nil
}

func (s *reviewStore) Close() error {
	return s.root.Close()
}

func (s *reviewStore) name(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("review-relative path is empty")
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(clean) || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("review-relative path %q escapes review %s", name, s.reviewID)
	}
	if clean == "." {
		return ".", nil
	}
	return clean, nil
}

func (s *reviewStore) display(name string) string {
	if name == "." {
		return s.absolute
	}
	return filepath.Join(s.absolute, filepath.FromSlash(name))
}

func (s *reviewStore) verifyIdentity() error {
	if err := s.rejectSymlinkComponents(s.relative); err != nil {
		return err
	}
	info, err := s.root.Lstat(s.relative)
	if err != nil {
		return fmt.Errorf("inspect review root %s: %w", s.absolute, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("review root %s is not a real directory", s.absolute)
	}
	if s.identity == nil || !os.SameFile(info, s.identity) {
		return fmt.Errorf("review root %s changed after it was opened", s.absolute)
	}
	return nil
}

func (s *reviewStore) openPinned() (*os.Root, error) {
	if err := s.verifyIdentity(); err != nil {
		return nil, err
	}
	pinned, err := s.root.OpenRoot(s.relative)
	if err != nil {
		return nil, fmt.Errorf("open pinned review root %s: %w", s.absolute, err)
	}
	info, err := pinned.Stat(".")
	if err != nil {
		_ = pinned.Close()
		return nil, fmt.Errorf("inspect pinned review root %s: %w", s.absolute, err)
	}
	if !os.SameFile(info, s.identity) {
		_ = pinned.Close()
		return nil, fmt.Errorf("review root %s changed while it was being opened", s.absolute)
	}
	if err := s.verifyIdentity(); err != nil {
		_ = pinned.Close()
		return nil, err
	}
	return pinned, nil
}

func (s *reviewStore) openPinnedParent() (*os.Root, error) {
	if s.parentIdentity == nil {
		return nil, fmt.Errorf("review staging parent identity is unavailable")
	}
	current, err := s.root.Lstat(s.parentRelative)
	if err != nil {
		return nil, err
	}
	if current.Mode()&os.ModeSymlink != 0 || !current.IsDir() ||
		!os.SameFile(current, s.parentIdentity) {
		return nil, fmt.Errorf("review staging parent changed after it was opened")
	}
	pinned, err := s.root.OpenRoot(s.parentRelative)
	if err != nil {
		return nil, err
	}
	info, err := pinned.Stat(".")
	if err != nil || !os.SameFile(info, s.parentIdentity) {
		_ = pinned.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("review staging parent changed while it was being opened")
	}
	current, err = s.root.Lstat(s.parentRelative)
	if err != nil || current.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(current, s.parentIdentity) {
		_ = pinned.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("review staging parent changed after it was pinned")
	}
	return pinned, nil
}

func (s *reviewStore) Publish(target string) error {
	if filepath.Dir(target) != s.parentRelative {
		return fmt.Errorf("review publication target has a different parent")
	}
	parent, err := s.openPinnedParent()
	if err != nil {
		return err
	}
	defer parent.Close()
	stagingName := filepath.Base(s.relative)
	info, err := parent.Lstat(stagingName)
	if err != nil {
		return err
	}
	if !os.SameFile(info, s.identity) {
		return fmt.Errorf("review staging directory changed before publication")
	}
	return parent.Rename(stagingName, filepath.Base(target))
}

func (s *reviewStore) RemoveStaging() error {
	parent, err := s.openPinnedParent()
	if err != nil {
		return err
	}
	defer parent.Close()
	stagingName := filepath.Base(s.relative)
	info, err := parent.Lstat(stagingName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !os.SameFile(info, s.identity) {
		return fmt.Errorf("review staging directory changed before cleanup")
	}
	return parent.RemoveAll(stagingName)
}

func rejectPinnedSymlinkComponents(root *os.Root, name string) error {
	current := ""
	for _, component := range strings.Split(filepath.Clean(name), string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("review path %s contains a symlink", current)
		}
	}
	return nil
}

func (s *reviewStore) rejectSymlinkComponents(full string) error {
	current := ""
	parts := strings.Split(filepath.Clean(full), string(filepath.Separator))
	for _, component := range parts {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := s.root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect review path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("review path %s contains a symlink", current)
		}
	}
	return nil
}

func (s *reviewStore) ReadRegular(name string) ([]byte, os.FileInfo, error) {
	clean, err := s.name(name)
	if err != nil {
		return nil, nil, err
	}
	pinned, err := s.openPinned()
	if err != nil {
		return nil, nil, err
	}
	defer pinned.Close()
	if err := rejectPinnedSymlinkComponents(pinned, clean); err != nil {
		return nil, nil, err
	}
	info, err := pinned.Lstat(clean)
	if err != nil {
		return nil, nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("%s is a symlink", s.display(name))
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s is not a regular file", s.display(name))
	}
	data, err := pinned.ReadFile(clean)
	if err != nil {
		return nil, nil, err
	}
	return data, info, nil
}

func (s *reviewStore) Lstat(name string) (os.FileInfo, error) {
	clean, err := s.name(name)
	if err != nil {
		return nil, err
	}
	pinned, err := s.openPinned()
	if err != nil {
		return nil, err
	}
	defer pinned.Close()
	return pinned.Lstat(clean)
}

func (s *reviewStore) Remove(name string) error {
	clean, err := s.name(name)
	if err != nil {
		return err
	}
	pinned, err := s.openPinned()
	if err != nil {
		return err
	}
	defer pinned.Close()
	return pinned.Remove(clean)
}

func (s *reviewStore) MkdirAll(name string, mode os.FileMode) error {
	clean, err := s.name(name)
	if err != nil {
		return err
	}
	pinned, err := s.openPinned()
	if err != nil {
		return err
	}
	defer pinned.Close()
	if err := pinned.MkdirAll(clean, mode); err != nil {
		return err
	}
	return rejectPinnedSymlinkComponents(pinned, clean)
}

func (s *reviewStore) WriteExclusive(name string, data []byte, mode os.FileMode) error {
	clean, err := s.name(name)
	if err != nil {
		return err
	}
	pinned, err := s.openPinned()
	if err != nil {
		return err
	}
	defer pinned.Close()
	if err := rejectPinnedSymlinkComponents(pinned, filepath.Dir(clean)); err != nil {
		return err
	}
	file, err := pinned.OpenFile(clean, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", s.display(name), err)
	}
	return writeDurableExclusiveWithCleanup(
		s.display(name),
		file,
		data,
		mode,
		func() error { return pinned.Remove(clean) },
	)
}

func (s *reviewStore) AtomicWrite(
	name string,
	data []byte,
	mode os.FileMode,
) (returnErr error) {
	clean, err := s.name(name)
	if err != nil {
		return err
	}
	pinned, err := s.openPinned()
	if err != nil {
		return err
	}
	defer pinned.Close()
	parent := filepath.Dir(clean)
	if err := rejectPinnedSymlinkComponents(pinned, parent); err != nil {
		return err
	}
	for attempt := 0; attempt < 32; attempt++ {
		suffix, err := randomReviewSuffix()
		if err != nil {
			return err
		}
		temp := filepath.Join(parent, ".ssb-prune-"+suffix)
		file, err := pinned.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("create temporary review file: %w", err)
		}
		if err := writeDurableExclusiveWithCleanup(
			temp,
			file,
			data,
			mode,
			func() error { return removeAtomicReviewFile(pinned, temp) },
		); err != nil {
			return err
		}
		cleanupTemp := true
		defer func() {
			if !cleanupTemp {
				return
			}
			if cleanupErr := removeAtomicReviewFile(pinned, temp); cleanupErr != nil &&
				!errors.Is(cleanupErr, os.ErrNotExist) {
				returnErr = errors.Join(
					returnErr,
					fmt.Errorf(
						"remove temporary review file %s: %w; inspect and remove this residual file after confirming no review transition is active",
						s.display(temp),
						cleanupErr,
					),
				)
			}
		}()
		if err := renameAtomicReviewFile(pinned, temp, clean); err != nil {
			return fmt.Errorf("replace review file %s: %w", s.display(name), err)
		}
		cleanupTemp = false
		return nil
	}
	return fmt.Errorf("create unique temporary review file: too many collisions")
}

func randomReviewSuffix() (string, error) {
	suffix := make([]byte, 16)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("create temporary review filename: %w", err)
	}
	return hex.EncodeToString(suffix), nil
}

func openPruneRepositoryRoot(repoRoot string) (*os.Root, os.FileInfo, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("open repository root for prune mutation: %w", err)
	}
	info, err := root.Lstat(".software-standards")
	if err != nil {
		_ = root.Close()
		return nil, nil, fmt.Errorf("inspect .software-standards for prune mutation: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		_ = root.Close()
		return nil, nil, preconditionError(".software-standards must be a real directory for prune mutation")
	}
	return root, info, nil
}

func openPinnedPack(root *os.Root, expected os.FileInfo) (*os.Root, error) {
	current, err := root.Lstat(".software-standards")
	if err != nil {
		return nil, err
	}
	if current.Mode()&os.ModeSymlink != 0 || !current.IsDir() ||
		!os.SameFile(current, expected) {
		return nil, fmt.Errorf(".software-standards changed after it was opened")
	}
	pinned, err := root.OpenRoot(".software-standards")
	if err != nil {
		return nil, err
	}
	info, err := pinned.Stat(".")
	if err != nil || !os.SameFile(info, expected) {
		_ = pinned.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(".software-standards changed while it was being opened")
	}
	current, err = root.Lstat(".software-standards")
	if err != nil || current.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(current, expected) {
		_ = pinned.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(".software-standards changed after it was pinned")
	}
	return pinned, nil
}

func writeDurableExclusiveWithCleanup(
	filePath string,
	file durableExclusiveFile,
	data []byte,
	mode os.FileMode,
	removeIncomplete func() error,
) (returnErr error) {
	closed := false
	complete := false
	defer func() {
		if !closed {
			returnErr = errors.Join(returnErr, file.Close())
		}
		if !complete {
			if err := removeIncomplete(); err != nil && !errors.Is(err, os.ErrNotExist) {
				returnErr = errors.Join(
					returnErr,
					fmt.Errorf("remove incomplete %s: %w", filePath, err),
				)
			}
		}
	}()
	if err := file.Chmod(mode); err != nil {
		return fmt.Errorf("set mode on %s: %w", filePath, err)
	}
	if written, err := file.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	} else if written != len(data) {
		return fmt.Errorf("write %s: %w", filePath, io.ErrShortWrite)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", filePath, err)
	}
	if err := file.Close(); err != nil {
		closed = true
		return fmt.Errorf("close %s: %w", filePath, err)
	}
	closed = true
	complete = true
	return nil
}

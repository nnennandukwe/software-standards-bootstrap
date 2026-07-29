package workspace

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestAtCommitRejectsIntermediateObjectIDLengthsBeforeGit(t *testing.T) {
	repo := &Repository{
		root:     t.TempDir(),
		baseline: strings.Repeat("b", 40),
		gitPath:  "git",
	}
	original := runRepositoryGit
	t.Cleanup(func() { runRepositoryGit = original })

	for _, length := range []int{41, 63} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			gitCalled := false
			runRepositoryGit = func(
				context.Context,
				string,
				string,
				[]byte,
				...string,
			) ([]byte, error) {
				gitCalled = true
				return nil, errors.New("unexpected Git invocation")
			}
			commit := strings.Repeat("a", length)
			_, err := repo.AtCommit(context.Background(), commit)
			if !errors.Is(err, ErrHistoricalCommit) ||
				!strings.Contains(err.Error(), "full lowercase object id") {
				t.Fatalf("error = %v, want non-canonical object-id rejection", err)
			}
			if gitCalled {
				t.Fatal("non-canonical object id reached Git")
			}
		})
	}
}

func TestAtCommitAcceptsFullSHA1AndSHA256ObjectIDs(t *testing.T) {
	original := runRepositoryGit
	t.Cleanup(func() { runRepositoryGit = original })

	for _, length := range []int{40, 64} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			commit := strings.Repeat("a", length)
			repo := &Repository{
				root:     t.TempDir(),
				baseline: strings.Repeat("b", length),
				gitPath:  "git",
			}
			runRepositoryGit = func(
				_ context.Context,
				_, _ string,
				input []byte,
				args ...string,
			) ([]byte, error) {
				switch args[0] {
				case "cat-file":
					if string(input) != commit+"^{commit}\n" {
						t.Fatalf("unexpected Git input: %q", input)
					}
					return []byte(commit + " commit\n"), nil
				case "merge-base":
					return nil, nil
				default:
					t.Fatalf("unexpected Git arguments: %#v", args)
					return nil, nil
				}
			}

			got, err := repo.AtCommit(context.Background(), commit)
			if err != nil {
				t.Fatal(err)
			}
			if got.Baseline() != commit {
				t.Fatalf("baseline = %q, want %q", got.Baseline(), commit)
			}
		})
	}
}

func TestAtCommitPreservesAncestryOperationalFailure(t *testing.T) {
	commit := strings.Repeat("a", 40)
	repo := &Repository{
		root:     t.TempDir(),
		baseline: strings.Repeat("b", 40),
		gitPath:  "git",
	}
	original := runRepositoryGit
	runRepositoryGit = func(
		_ context.Context,
		_, _ string,
		input []byte,
		args ...string,
	) ([]byte, error) {
		switch args[0] {
		case "cat-file":
			if string(input) != commit+"^{commit}\n" {
				t.Fatalf("unexpected Git input: %q", input)
			}
			return []byte(commit + " commit\n"), nil
		case "merge-base":
			return nil, context.Canceled
		default:
			t.Fatalf("unexpected Git arguments: %#v", args)
			return nil, nil
		}
	}
	t.Cleanup(func() { runRepositoryGit = original })

	if _, err := repo.AtCommit(context.Background(), commit); err == nil {
		t.Fatal("injected ancestry failure unexpectedly succeeded")
	} else {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if strings.Contains(err.Error(), "not an ancestor") {
			t.Fatalf("operational ancestry failure was misclassified: %v", err)
		}
	}
}

func TestAtCommitPreservesResolutionOperationalFailure(t *testing.T) {
	commit := strings.Repeat("a", 40)
	repo := &Repository{
		root:     t.TempDir(),
		baseline: strings.Repeat("b", 40),
		gitPath:  "git",
	}
	injected := errors.New("injected resolution failure")
	original := runRepositoryGit
	runRepositoryGit = func(
		_ context.Context,
		_, _ string,
		_ []byte,
		args ...string,
	) ([]byte, error) {
		if args[0] != "cat-file" {
			t.Fatalf("unexpected Git arguments: %#v", args)
		}
		return nil, injected
	}
	t.Cleanup(func() { runRepositoryGit = original })

	if _, err := repo.AtCommit(context.Background(), commit); err == nil {
		t.Fatal("injected resolution failure unexpectedly succeeded")
	} else {
		if !errors.Is(err, injected) {
			t.Fatalf("error = %v, want injected cause", err)
		}
		if errors.Is(err, ErrHistoricalCommit) {
			t.Fatalf("operational resolution failure was misclassified: %v", err)
		}
	}
}

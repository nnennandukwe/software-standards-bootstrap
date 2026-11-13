package workspace

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestAtCommitRejectsIntermediateObjectIDLengthsBeforeGit(t *testing.T) {
	t.Parallel()
	repo := &Repository{
		root:     t.TempDir(),
		baseline: strings.Repeat("b", 40),
		gitPath:  "git",
	}

	for _, length := range []int{41, 63} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			gitCalled := false
			repo.runGit = func(
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
			_, err := repo.AtCommit(t.Context(), commit)
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
	t.Parallel()

	for _, length := range []int{40, 64} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			commit := strings.Repeat("a", length)
			repo := &Repository{
				root:     t.TempDir(),
				baseline: strings.Repeat("b", length),
				gitPath:  "git",
			}
			repo.runGit = func(
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
				case "probe":
					return []byte("inherited runner"), nil
				default:
					t.Fatalf("unexpected Git arguments: %#v", args)
					return nil, nil
				}
			}

			got, err := repo.AtCommit(t.Context(), commit)
			if err != nil {
				t.Fatal(err)
			}
			if output, err := got.Git(t.Context(), "probe"); err != nil || string(output) != "inherited runner" {
				t.Fatalf("historical runner = %q, %v", output, err)
			}
			if got.Baseline() != commit {
				t.Fatalf("baseline = %q, want %q", got.Baseline(), commit)
			}
		})
	}
}

func TestAtCommitPreservesAncestryOperationalFailure(t *testing.T) {
	t.Parallel()
	commit := strings.Repeat("a", 40)
	repo := &Repository{
		root:     t.TempDir(),
		baseline: strings.Repeat("b", 40),
		gitPath:  "git",
	}
	repo.runGit = func(
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

	if _, err := repo.AtCommit(t.Context(), commit); err == nil {
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
	t.Parallel()
	commit := strings.Repeat("a", 40)
	repo := &Repository{
		root:     t.TempDir(),
		baseline: strings.Repeat("b", 40),
		gitPath:  "git",
	}
	injected := errors.New("injected resolution failure")
	repo.runGit = func(
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

	if _, err := repo.AtCommit(t.Context(), commit); err == nil {
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

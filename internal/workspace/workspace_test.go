package workspace_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestOpenForInspectAcceptsUntrackedFilesButRejectsTrackedChanges(t *testing.T) {
	repo := newRepository(t)
	writeFile(t, filepath.Join(repo, "tracked.txt"), "committed\n")
	git(t, repo, "add", "tracked.txt")
	git(t, repo, "commit", "-m", "baseline")

	writeFile(t, filepath.Join(repo, "notes.txt"), "untracked proposal\n")
	got, err := workspace.OpenForInspect(context.Background(), repo)
	if err != nil {
		t.Fatalf("untracked files must not block inspection: %v", err)
	}
	if got.Baseline() == "" {
		t.Fatal("expected a commit-backed baseline")
	}

	writeFile(t, filepath.Join(repo, "tracked.txt"), "changed\n")
	_, err = workspace.OpenForInspect(context.Background(), repo)
	if err == nil {
		t.Fatal("expected tracked changes to block inspection")
	}
	if !strings.Contains(err.Error(), "commit, stash, or restore tracked changes") {
		t.Fatalf("expected actionable recovery guidance, got %q", err)
	}
}

func TestOpenForInspectRejectsStagedDetachedUnbornAndNonGitStates(t *testing.T) {
	t.Run("staged", func(t *testing.T) {
		repo := newRepository(t)
		writeFile(t, filepath.Join(repo, "tracked.txt"), "committed\n")
		git(t, repo, "add", "tracked.txt")
		git(t, repo, "commit", "-m", "baseline")
		writeFile(t, filepath.Join(repo, "staged.txt"), "staged\n")
		git(t, repo, "add", "staged.txt")

		_, err := workspace.OpenForInspect(context.Background(), repo)
		assertErrorContains(t, err, "commit, stash, or restore tracked changes")
	})

	t.Run("detached", func(t *testing.T) {
		repo := committedRepository(t)
		git(t, repo, "checkout", "--detach")

		_, err := workspace.OpenForInspect(context.Background(), repo)
		assertErrorContains(t, err, "switch to a branch")
	})

	t.Run("unborn", func(t *testing.T) {
		repo := newRepository(t)

		_, err := workspace.OpenForInspect(context.Background(), repo)
		assertErrorContains(t, err, "create an initial commit")
	})

	t.Run("not git", func(t *testing.T) {
		_, err := workspace.OpenForInspect(context.Background(), t.TempDir())
		assertErrorContains(t, err, "run ssb from a non-bare Git repository")
	})
}

func TestOpenForInspectRejectsExistingPackWithoutChangingIt(t *testing.T) {
	repo := committedRepository(t)
	rule := filepath.Join(repo, ".software-standards", "rules", "keep-me.md")
	writeFile(t, rule, "developer work\n")

	_, err := workspace.OpenForInspect(context.Background(), repo)
	assertErrorContains(t, err, "edit or remove the existing pack")

	got, readErr := os.ReadFile(rule)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "developer work\n" {
		t.Fatalf("existing pack changed: %q", got)
	}
}

func TestOpenForInspectResolvesNestedPathToWorktreeRoot(t *testing.T) {
	repo := committedRepository(t)
	nested := filepath.Join(repo, "path", "with spaces", "日本語")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := workspace.OpenForInspect(context.Background(), nested)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root() != want {
		t.Fatalf("root = %q, want %q", got.Root(), want)
	}
}

func TestOpenForInspectPreservesHostValidWhitespaceAndUnicodeInRepositoryRoot(t *testing.T) {
	repositoryName := "repo \n日本語 "
	if runtime.GOOS == "windows" {
		// Win32 rejects newlines and trailing spaces in path components.
		repositoryName = "repo 日本語 path"
	}
	repo := filepath.Join(t.TempDir(), repositoryName)
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "-b", "main")
	writeFile(t, filepath.Join(repo, "README.md"), "fixture\n")
	git(t, repo, "add", "README.md")
	git(t, repo, "commit", "-m", "baseline")

	got, err := workspace.OpenForInspect(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root() != want {
		t.Fatalf("root = %q, want %q", got.Root(), want)
	}
}

func TestVerifyInspectSnapshotRejectsConcurrentHeadOrTrackedChanges(t *testing.T) {
	t.Run("tracked change", func(t *testing.T) {
		repo := committedRepository(t)
		ws, err := workspace.OpenForInspect(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(repo, "README.md"), "changed concurrently\n")

		err = ws.VerifyInspectSnapshot(context.Background())
		assertErrorContains(t, err, "tracked or staged files changed during inspection")
	})

	t.Run("head change", func(t *testing.T) {
		repo := committedRepository(t)
		ws, err := workspace.OpenForInspect(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(repo, "next.txt"), "next\n")
		git(t, repo, "add", "next.txt")
		git(t, repo, "commit", "-m", "concurrent")

		err = ws.VerifyInspectSnapshot(context.Background())
		assertErrorContains(t, err, "HEAD changed during inspection")
	})
}

func newRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	return dir
}

func committedRepository(t *testing.T) string {
	t.Helper()
	repo := newRepository(t)
	writeFile(t, filepath.Join(repo, "README.md"), "fixture\n")
	git(t, repo, "add", "README.md")
	git(t, repo, "commit", "-m", "baseline")
	return repo
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{"-c", "user.name=SSB Test", "-c", "user.email=ssb@example.invalid", "-C", dir}
	cmd := exec.Command("git", append(base, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertErrorContains(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", text)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error %q does not contain %q", err, text)
	}
}

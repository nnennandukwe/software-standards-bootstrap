package evaluation_test

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestPublicGoServiceFixtureProducesThePinnedSafeInventory(t *testing.T) {
	source := filepath.Join(repositoryRoot(t), "testdata", "fixtures", "go-service")
	repo := t.TempDir()
	copyTree(t, source, repo)
	git(t, repo, "init", "-b", "main")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "pinned public fixture")

	ws, err := workspace.OpenForInspect(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	report, err := inventory.Scan(context.Background(), ws, inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(report.Files))
	for _, file := range report.Files {
		got = append(got, file.Path)
	}
	want := []string{
		".env.example",
		"CONTRIBUTING.md",
		"EXPECTED.md",
		"Makefile",
		"README.md",
		"go.mod",
		"internal/payments/service.go",
		"internal/refunds/service.go",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inventory paths = %#v, want %#v", got, want)
	}
	if report.Excluded.Generated != 0 || report.Excluded.VendorTree != 2 {
		t.Fatalf("expected generated and vendor directories to be excluded before blob reads: %#v", report.Excluded)
	}
}

func repositoryRoot(tb testing.TB) string {
	tb.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("cannot resolve test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := append([]string{"-c", "user.name=SSB Test", "-c", "user.email=ssb@example.invalid", "-C", dir}, args...)
	cmd := exec.Command("git", command...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

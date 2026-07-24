// Package benchmarktest loads the exact public repository pins used by the
// opt-in inventory resource benchmarks.
package benchmarktest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

const RootEnvironment = "SSB_BENCHMARK_ROOT"

type manifest struct {
	Repositories []Repository `yaml:"repositories"`
}

// Repository identifies one exact benchmark checkout.
type Repository struct {
	Name   string `yaml:"name"`
	Commit string `yaml:"commit"`
}

// Root returns the absolute benchmark root or skips when the opt-in
// environment variable is unset.
func Root(tb testing.TB) string {
	tb.Helper()
	root := os.Getenv(RootEnvironment)
	if root == "" {
		tb.Skipf("%s is unset; public benchmarks remain opt-in", RootEnvironment)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		tb.Fatal(err)
	}
	return absolute
}

// Load reads the canonical benchmark manifest.
func Load(tb testing.TB) []Repository {
	tb.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("resolve benchmark helper path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	data, err := os.ReadFile(filepath.Join(repositoryRoot, "testdata", "benchmarks.yaml"))
	if err != nil {
		tb.Fatal(err)
	}
	var pins manifest
	if err := yaml.Unmarshal(data, &pins); err != nil {
		tb.Fatal(err)
	}
	if len(pins.Repositories) == 0 {
		tb.Fatal("benchmark manifest contains no repositories")
	}
	return pins.Repositories
}

// Open verifies an attached checkout's exact HEAD before opening it for
// read-only inspection.
func Open(tb testing.TB, root string, target Repository) *workspace.Repository {
	tb.Helper()
	repoPath := filepath.Join(root, target.Name)
	command := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := command.CombinedOutput()
	if err != nil {
		tb.Fatalf("resolve %s HEAD: %v\n%s", target.Name, err, output)
	}
	if got := strings.TrimSpace(string(output)); got != target.Commit {
		tb.Fatalf("%s HEAD = %s, want %s", target.Name, got, target.Commit)
	}
	repo, err := workspace.OpenForInspect(context.Background(), repoPath)
	if err != nil {
		tb.Fatal(err)
	}
	return repo
}

package evaluation_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

const benchmarkRootEnvironment = "SSB_BENCHMARK_ROOT"

type benchmarkManifest struct {
	Repositories []benchmarkRepository `yaml:"repositories"`
}

type benchmarkRepository struct {
	Name   string `yaml:"name"`
	Commit string `yaml:"commit"`
}

func TestPinnedInventoryResourceEnvelope(t *testing.T) {
	root := pinnedBenchmarkRoot(t)
	for _, target := range loadPinnedBenchmarks(t) {
		t.Run(target.Name, func(t *testing.T) {
			repo := openPinnedBenchmark(t, root, target)
			start := time.Now()
			report, err := inventory.Scan(context.Background(), repo, inventory.DefaultLimits())
			elapsed := time.Since(start)
			if err != nil {
				t.Fatal(err)
			}
			if report.Truncated {
				t.Fatalf("default inventory truncated: %#v", report)
			}
			if elapsed > 10*time.Second {
				t.Fatalf("inventory elapsed time %s exceeds 10s envelope", elapsed)
			}
			t.Logf(
				"candidate_files=%d candidate_bytes=%d indexed_files=%d indexed_bytes=%d elapsed=%s",
				report.CandidateFiles,
				report.CandidateBytes,
				len(report.Files),
				report.IndexedBytes,
				elapsed,
			)
		})
	}
}

func BenchmarkPinnedInventory(b *testing.B) {
	root := pinnedBenchmarkRoot(b)
	for _, target := range loadPinnedBenchmarks(b) {
		b.Run(target.Name, func(b *testing.B) {
			repo := openPinnedBenchmark(b, root, target)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				report, err := inventory.Scan(context.Background(), repo, inventory.DefaultLimits())
				if err != nil {
					b.Fatal(err)
				}
				if report.Truncated {
					b.Fatalf("default inventory truncated: %#v", report)
				}
			}
		})
	}
}

func pinnedBenchmarkRoot(tb testing.TB) string {
	tb.Helper()
	root := os.Getenv(benchmarkRootEnvironment)
	if root == "" {
		tb.Skipf("%s is unset; public benchmarks remain opt-in", benchmarkRootEnvironment)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		tb.Fatal(err)
	}
	return absolute
}

func loadPinnedBenchmarks(tb testing.TB) []benchmarkRepository {
	tb.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(tb), "testdata", "benchmarks.yaml"))
	if err != nil {
		tb.Fatal(err)
	}
	var manifest benchmarkManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		tb.Fatal(err)
	}
	if len(manifest.Repositories) == 0 {
		tb.Fatal("benchmark manifest contains no repositories")
	}
	return manifest.Repositories
}

func openPinnedBenchmark(
	tb testing.TB,
	root string,
	target benchmarkRepository,
) *workspace.Repository {
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

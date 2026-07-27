package inventory

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/evaluation/benchmarktest"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestDefaultBatchPolicyMatchesMeasuredSelection(t *testing.T) {
	if defaultBatchPolicy.MaxEntries != 512 || defaultBatchPolicy.MaxBytes != 4<<20 {
		t.Fatalf("default batch policy = %#v", defaultBatchPolicy)
	}
}

func TestScanRejectsAFileBoundaryLargerThanTheBatchByteCeiling(t *testing.T) {
	repo := t.TempDir()
	runFixtureGit(t, repo, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, repo, "add", "README.md")
	runFixtureGit(t, repo, "commit", "-m", "fixture")

	ws, err := workspace.OpenForInspect(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = scan(
		context.Background(),
		ws,
		Limits{
			MaxCandidateFiles: 10,
			MaxCandidateBytes: 16 << 20,
			MaxFileBytes:      5 << 20,
		},
		defaultBatchPolicy,
		ws.VerifyInspectSnapshot,
	)
	if err == nil || !strings.Contains(err.Error(), "max_file_bytes") {
		t.Fatalf("scan error = %v, want incompatible max_file_bytes", err)
	}
}

func runFixtureGit(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	arguments = append(
		[]string{
			"-c", "user.name=SSB Test",
			"-c", "user.email=ssb@example.invalid",
			"-C", directory,
		},
		arguments...,
	)
	output, err := exec.Command("git", arguments...).CombinedOutput()
	if err != nil {
		t.Fatalf("git failed: %v\n%s", err, output)
	}
}

func BenchmarkBatchPolicies(b *testing.B) {
	root := benchmarktest.Root(b)
	targets := benchmarktest.Load(b)
	repositories := make([]*workspace.Repository, 0, len(targets))
	for _, target := range targets {
		repositories = append(repositories, benchmarktest.Open(b, root, target))
	}

	for _, maxBytes := range []int64{1 << 20, 4 << 20, 8 << 20, 16 << 20} {
		for _, maxEntries := range []int{32, 128, 512} {
			policy := batchPolicy{MaxEntries: maxEntries, MaxBytes: maxBytes}
			name := fmt.Sprintf("%d_entries_%d_mib", maxEntries, maxBytes>>20)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					for _, repo := range repositories {
						report, err := scan(
							context.Background(),
							repo,
							DefaultLimits(),
							policy,
							repo.VerifyInspectSnapshot,
						)
						if err != nil {
							b.Fatal(err)
						}
						if report.Truncated {
							b.Fatalf("default inventory truncated: %#v", report)
						}
					}
				}
			})
		}
	}
}

package inventory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestDefaultBatchPolicyMatchesMeasuredSelection(t *testing.T) {
	if defaultBatchPolicy.MaxEntries != 512 || defaultBatchPolicy.MaxBytes != 4<<20 {
		t.Fatalf("default batch policy = %#v", defaultBatchPolicy)
	}
}

func BenchmarkBatchPolicies(b *testing.B) {
	root := os.Getenv("SSB_BENCHMARK_ROOT")
	if root == "" {
		b.Skip("SSB_BENCHMARK_ROOT is unset; public benchmarks remain opt-in")
	}
	repositories := make([]*workspace.Repository, 0, 4)
	for _, name := range []string{"cobra", "flask", "django", "next.js"} {
		repo, err := workspace.OpenForInspect(context.Background(), filepath.Join(root, name))
		if err != nil {
			b.Fatal(err)
		}
		repositories = append(repositories, repo)
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

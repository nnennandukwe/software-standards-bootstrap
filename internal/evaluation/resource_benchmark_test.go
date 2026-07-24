package evaluation_test

import (
	"context"
	"testing"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/evaluation/benchmarktest"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
)

func TestPinnedInventoryResourceEnvelope(t *testing.T) {
	root := benchmarktest.Root(t)
	for _, target := range benchmarktest.Load(t) {
		t.Run(target.Name, func(t *testing.T) {
			repo := benchmarktest.Open(t, root, target)
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
				report.IndexedFiles,
				report.IndexedBytes,
				elapsed,
			)
		})
	}
}

func BenchmarkPinnedInventory(b *testing.B) {
	root := benchmarktest.Root(b)
	for _, target := range benchmarktest.Load(b) {
		b.Run(target.Name, func(b *testing.B) {
			repo := benchmarktest.Open(b, root, target)
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

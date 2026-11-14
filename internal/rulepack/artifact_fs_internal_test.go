package rulepack

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestRegularFileReadStaysBoundedWhileFileGrows(t *testing.T) {
	target := filepath.Join(t.TempDir(), "partial.md")
	const limit = int64(64)
	if err := os.WriteFile(target, []byte("small"), 0o600); err != nil {
		t.Fatal(err)
	}
	writer, err := os.OpenFile(target, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		defer writer.Close()
		large := bytes.Repeat([]byte("x"), 4096)
		for {
			select {
			case <-stop:
				return
			default:
			}
			if writer.Truncate(5) != nil {
				return
			}
			runtime.Gosched()
			if _, err := writer.WriteAt(large, 0); err != nil {
				return
			}
			runtime.Gosched()
		}
	}()
	defer func() { close(stop); <-done }()
	for attempt := 0; attempt < 10_000; attempt++ {
		data, _, _ := readRegularFileLimited(target, "partial.md", limit)
		if int64(len(data)) > limit {
			t.Fatalf("runtime reader exceeded its ceiling: read=%d limit=%d", len(data), limit)
		}
		runtime.Gosched()
	}
}

func TestSharedSkillDiscoveryStopsAtItsBudgets(t *testing.T) {
	for _, budget := range []string{"entries", "names"} {
		t.Run(budget, func(t *testing.T) {
			root := t.TempDir()
			directory := filepath.Join(root, ".agents/skills")
			if err := os.MkdirAll(directory, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"one", "two", "three"} {
				if err := os.WriteFile(filepath.Join(directory, name), []byte("manual work"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			limits := inventory.DefaultLimits()
			if budget == "entries" {
				limits.MaxCandidateFiles = 2
			} else {
				limits.MaxFileBytes = 8
			}
			paths, err := unindexedRepositorySkills(context.Background(), root, Manifest{}, limits)
			if !errors.Is(err, workspace.ErrPrecondition) || len(paths) != 0 {
				t.Fatalf("shared skill budget wasn't enforced: paths=%q error=%v", paths, err)
			}
			for _, name := range []string{"one", "two", "three"} {
				if content, err := os.ReadFile(filepath.Join(directory, name)); err != nil || string(content) != "manual work" {
					t.Fatalf("discovery changed manual work: %q %v", content, err)
				}
			}
		})
	}
}

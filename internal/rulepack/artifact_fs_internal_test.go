package rulepack

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

type skillOpenContext struct {
	context.Context
	action func()
}

func (ctx *skillOpenContext) Err() error {
	if ctx.action != nil {
		action := ctx.action
		ctx.action = nil
		action()
	}
	return ctx.Context.Err()
}

func TestIndexedSkillSupportProbeStaysInsideItsRoot(t *testing.T) {
	root := t.TempDir()
	relative := ".agents/skills/indexed"
	directory := filepath.Join(root, relative)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte("Indexed entrypoint"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "outside-only.txt"), []byte("Outside fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	anchor, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer anchor.Close()
	ctx := &skillOpenContext{Context: context.Background(), action: func() {
		if err := os.Rename(directory, directory+"-held"); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, directory); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}}
	if support, err := repositorySkillHasSupport(ctx, anchor, relative); err == nil || support {
		t.Fatalf("support probe followed outside link: support=%t err=%v", support, err)
	}
}

func TestNativeArtifactDiscoveryEnforcesBudgetsBeforeCollectingEverything(t *testing.T) {
	for _, budget := range []string{"entries", "names"} {
		t.Run(budget, func(t *testing.T) {
			root := t.TempDir()
			directory := filepath.Join(root, ".software-standards/rules")
			if err := os.MkdirAll(directory, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"one.md", "two.md", "other.tmp"} {
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
			paths, diagnostics, err := unlistedNativeArtifactsLimited(context.Background(), root, map[string]AcceptedArtifact{}, limits)
			bounded := false
			for _, d := range diagnostics {
				bounded = bounded || strings.Contains(d.Message, "limit")
			}
			if err != nil || len(paths) != 0 || !bounded {
				t.Fatalf("native discovery budget wasn't enforced: paths=%q diagnostics=%#v err=%v", paths, diagnostics, err)
			}
			for _, name := range []string{"one.md", "two.md", "other.tmp"} {
				if content, err := os.ReadFile(filepath.Join(directory, name)); err != nil || string(content) != "manual work" {
					t.Fatalf("discovery changed native work: %q %v", content, err)
				}
			}
		})
	}
}

func TestNativeArtifactDiscoveryDoesntFollowDirectorySwaps(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, ".software-standards/rules")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "outside-only.md"), []byte("Outside fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := &skillOpenContext{Context: context.Background(), action: func() {
		if err := os.Rename(directory, directory+"-held"); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, directory); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}}
	paths, diagnostics, err := unlistedNativeArtifactsLimited(ctx, root, map[string]AcceptedArtifact{}, inventory.DefaultLimits())
	if err == nil {
		t.Fatalf("native discovery followed an outside directory: paths=%q diagnostics=%#v", paths, diagnostics)
	}
}

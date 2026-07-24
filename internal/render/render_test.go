package render_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestApplyPreservesExistingBytesAndProjectsOnlySurvivingRules(t *testing.T) {
	repo := committedRepository(t)
	agentsPath := filepath.Join(repo, "AGENTS.md")
	prefix := "# Existing guidance\n\nKeep this byte-for-byte.\n"
	writeFile(t, agentsPath, prefix)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}

	pack := testPack(ws.Baseline(), "first-rule", "First rule body.", "second-rule", "Second rule body.")
	first, err := render.Apply(ws, pack, false)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed {
		t.Fatal("expected first render to change AGENTS.md")
	}
	rendered, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(rendered), prefix) {
		t.Fatalf("existing bytes changed:\n%s", rendered)
	}
	for _, text := range []string{"First rule body.", "Second rule body.", "- Primary topic: `correctness`", render.StartMarker, render.EndMarker} {
		if !strings.Contains(string(rendered), text) {
			t.Fatalf("rendered section missing %q:\n%s", text, rendered)
		}
	}

	pack = testPack(ws.Baseline(), "second-rule", "Edited second rule body.")
	second, err := render.Apply(ws, pack, false)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Changed {
		t.Fatal("expected source edit and deletion to change projection")
	}
	rendered, err = os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(rendered), prefix) ||
		strings.Contains(string(rendered), "First rule body.") ||
		!strings.Contains(string(rendered), "Edited second rule body.") {
		t.Fatalf("surviving projection is wrong:\n%s", rendered)
	}

	third, err := render.Apply(ws, pack, false)
	if err != nil {
		t.Fatal(err)
	}
	if third.Changed {
		t.Fatal("byte-stable rerender should be a no-op")
	}
}

func TestApplyDryRunDoesNotCreateAgentsFile(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}

	result, err := render.Apply(ws, testPack(ws.Baseline(), "one-rule", "Body."), true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || !strings.Contains(string(result.Content), "Body.") {
		t.Fatalf("unexpected dry-run result: %#v", result)
	}
	if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created AGENTS.md: %v", err)
	}
}

func TestApplyBlocksDriftMalformedMarkersAndSymlinkTargets(t *testing.T) {
	t.Run("drift", func(t *testing.T) {
		repo := committedRepository(t)
		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		pack := testPack(ws.Baseline(), "one-rule", "Original body.")
		if _, err := render.Apply(ws, pack, false); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(repo, "AGENTS.md")
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		drifted := strings.Replace(string(before), "Original body.", "Direct edit.", 1)
		writeFile(t, path, drifted)

		_, err = render.Apply(ws, pack, false)
		if !errors.Is(err, render.ErrDrift) {
			t.Fatalf("expected drift error, got %v", err)
		}
		after, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(after) != drifted {
			t.Fatal("failed render modified drifted AGENTS.md")
		}
	})

	t.Run("duplicate markers", func(t *testing.T) {
		repo := committedRepository(t)
		path := filepath.Join(repo, "AGENTS.md")
		before := render.StartMarker + "\n" + render.StartMarker + "\n" + render.EndMarker + "\n"
		writeFile(t, path, before)
		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}

		_, err = render.Apply(ws, testPack(ws.Baseline(), "one-rule", "Body."), false)
		if !errors.Is(err, render.ErrMarkers) {
			t.Fatalf("expected marker error, got %v", err)
		}
		after, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(after) != before {
			t.Fatal("failed render repaired malformed markers")
		}
	})

	t.Run("symlink", func(t *testing.T) {
		repo := committedRepository(t)
		outside := filepath.Join(t.TempDir(), "outside.md")
		writeFile(t, outside, "outside\n")
		if err := os.Symlink(outside, filepath.Join(repo, "AGENTS.md")); err != nil {
			t.Fatal(err)
		}
		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}

		_, err = render.Apply(ws, testPack(ws.Baseline(), "one-rule", "Body."), false)
		if !errors.Is(err, render.ErrUnsafeTarget) {
			t.Fatalf("expected unsafe target error, got %v", err)
		}
		got, readErr := os.ReadFile(outside)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(got) != "outside\n" {
			t.Fatal("render followed AGENTS.md symlink")
		}
	})
}

func TestApplyWriteFailureLeavesExistingAgentsUntouched(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows directory permission semantics do not provide this failure injection")
	}
	repo := committedRepository(t)
	path := filepath.Join(repo, "AGENTS.md")
	before := "# Existing\n"
	writeFile(t, path, before)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(repo, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(repo, 0o755); err != nil {
			t.Errorf("restore permissions: %v", err)
		}
	}()

	_, err = render.Apply(ws, testPack(ws.Baseline(), "one-rule", "Body."), false)
	if err == nil {
		t.Fatal("expected staged write failure")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != before {
		t.Fatalf("failed staged write changed AGENTS.md: %q", after)
	}
	matches, globErr := filepath.Glob(filepath.Join(repo, ".ssb-agents-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("failed staged write left temporary files: %#v", matches)
	}
}

func testPack(baseline string, pairs ...string) rulepack.Pack {
	rules := make([]rulepack.Rule, 0, len(pairs)/2)
	for index := 0; index < len(pairs); index += 2 {
		id := pairs[index]
		rules = append(rules, rulepack.Rule{
			ID:             id,
			Title:          strings.ReplaceAll(strings.Title(strings.ReplaceAll(id, "-", " ")), " ", " "),
			Topic:          "correctness",
			Scopes:         []string{"**/*"},
			Classification: "guidance",
			Importance:     "high",
			Confidence:     "high",
			SourcePath:     ".software-standards/rules/" + id + ".md",
			Body:           pairs[index+1] + "\n",
		})
	}
	return rulepack.Pack{BaselineCommit: baseline, Rules: rules}
}

func committedRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	writeFile(t, filepath.Join(dir, "README.md"), "fixture\n")
	git(t, dir, "add", "README.md")
	git(t, dir, "commit", "-m", "baseline")
	return dir
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

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

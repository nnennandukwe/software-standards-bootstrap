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

func TestApplyRendersProgressiveStandingOrdersCommandsAndContextualIndex(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := rulepack.Pack{
		BaselineCommit: ws.Baseline(),
		Rules: []rulepack.Rule{
			{
				Schema:         rulepack.SchemaVersionV2,
				ID:             "never-delete-history",
				Title:          "Never delete history",
				Topic:          "reliability",
				Lenses:         []rulepack.Lens{{Kind: "base"}},
				Directive:      "never",
				Scopes:         []string{"**/*"},
				Classification: "guidance",
				Importance:     "very-high",
				Score:          rulepack.Score{Method: rulepack.ScoreMethod, Total: 90},
				Confidence:     "high",
				SourcePath:     ".software-standards/rules/never-delete-history.md",
				Body:           "Never delete Git history without explicit approval.\n",
				Verification:   rulepack.Verification{ProofGap: "No repository check proves operator intent."},
			},
			{
				Schema:         rulepack.SchemaVersionV2,
				ID:             "run-go-tests",
				Title:          "Run Go tests",
				Topic:          "correctness",
				Lenses:         []rulepack.Lens{{Kind: "base"}},
				Directive:      "always",
				Scopes:         []string{"**/*.go"},
				Classification: "deterministic",
				Importance:     "high",
				Score:          rulepack.Score{Method: rulepack.ScoreMethod, Total: 70},
				Confidence:     "high",
				SourcePath:     ".software-standards/rules/run-go-tests.md",
				Body:           "Run the retained Go assertions before handoff.\n",
				Verification: rulepack.Verification{
					Command:  "go test ./...",
					Coverage: "full",
					Proves:   "The retained Go assertions when the command passes.",
				},
			},
			{
				Schema:         rulepack.SchemaVersionV2,
				ID:             "review-cobra-command",
				Title:          "Review Cobra commands",
				Topic:          "maintainability",
				Lenses:         []rulepack.Lens{{Kind: "language", Value: "go"}, {Kind: "framework", Value: "cobra"}, {Kind: "task", Value: "review"}},
				Directive:      "prefer",
				Scopes:         []string{"cmd/**"},
				Classification: "guidance",
				Importance:     "medium",
				Score:          rulepack.Score{Method: rulepack.ScoreMethod, Total: 50},
				Confidence:     "medium",
				SourcePath:     ".software-standards/rules/review-cobra-command.md",
				Body:           "This contextual body must be loaded on demand.\n",
				Verification: rulepack.Verification{
					Command:  "go test ./...",
					Coverage: "partial",
					Proves:   "The command tests retained behavior but not review quality.",
				},
			},
		},
	}

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, required := range []string{
		"### How to apply these standards",
		"A rule is active only when its affected path scope matches.",
		"every represented lens dimension must also match",
		"If the language, framework, task, or affected path is uncertain",
		"Directives mean: `never` is prohibited",
		"Legacy v1 rules record no directive",
		"Linked rule files are canonical.",
		"`ssb` did not execute it, and its presence is not a passing result.",
		"### Standing orders",
		"#### Never",
		"#### Always",
		"Never delete Git history without explicit approval.",
		"Run the retained Go assertions before handoff.",
		"### Mapped verification commands",
		"`go test ./...` — mapped, not executed by ssb",
		"### Contextual rule index",
		"#### Language: `go`",
		"#### Framework: `cobra`",
		"#### Task: `review`",
		"[Review Cobra commands](.software-standards/rules/review-cobra-command.md)",
		"lenses: `language:go`, `framework:cobra`, `task:review`",
		"topic: `maintainability`",
		"`review-cobra-command`: `partial` — The command tests retained behavior but not review quality.",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("rendered router missing %q:\n%s", required, content)
		}
	}
	if strings.Contains(content, "This contextual body must be loaded on demand.") {
		t.Fatalf("contextual rule body was inlined:\n%s", content)
	}
	if strings.Count(content, "`go test ./...` — mapped, not executed by ssb") != 1 {
		t.Fatalf("verification command was not deduplicated:\n%s", content)
	}
	if strings.Index(content, "#### Never") > strings.Index(content, "#### Always") {
		t.Fatalf("standing-order severity is not deterministic:\n%s", content)
	}
}

func TestApplyRendersAllContextualProofGapsAndRelatedSkillsWithoutBaseBodies(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := rulepack.Pack{
		BaselineCommit: ws.Baseline(),
		Rules: []rulepack.Rule{{
			Schema:          rulepack.SchemaVersionV2,
			ID:              "review-security-boundary",
			Title:           "Review security boundary",
			Topic:           "security",
			Lenses:          []rulepack.Lens{{Kind: "task", Value: "security"}},
			Directive:       "ask-first",
			Scopes:          []string{"internal/security/**"},
			Classification:  "guidance",
			Importance:      "high",
			Score:           rulepack.Score{Method: rulepack.ScoreMethod, Total: 70},
			Confidence:      "high",
			SourcePath:      ".software-standards/rules/review-security-boundary.md",
			Body:            "This contextual body stays canonical.\n",
			Verification:    rulepack.Verification{ProofGap: "No check proves semantic review quality."},
			RelatedSkillIDs: []string{"review-security-change"},
		}},
	}

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, required := range []string{
		"No base standing orders were retained.",
		"#### Task: `security`",
		"Proof gap: No check proves semantic review quality.",
		"Related skills: `review-security-change`",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("all-contextual projection missing %q:\n%s", required, content)
		}
	}
	if strings.Contains(content, "This contextual body stays canonical.") {
		t.Fatalf("all-contextual projection inlined a contextual body:\n%s", content)
	}
}

func TestApplyOrdersRulesByDirectiveImportanceThenID(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	baseRule := func(id, importance, body string) rulepack.Rule {
		return rulepack.Rule{
			Schema:         rulepack.SchemaVersionV2,
			ID:             id,
			Title:          id,
			Topic:          "maintainability",
			Lenses:         []rulepack.Lens{{Kind: "base"}},
			Directive:      "prefer",
			Scopes:         []string{"**/*"},
			Classification: "guidance",
			Importance:     importance,
			Score:          rulepack.Score{Method: rulepack.ScoreMethod},
			Confidence:     "medium",
			SourcePath:     ".software-standards/rules/" + id + ".md",
			Body:           body + "\n",
			Verification:   rulepack.Verification{ProofGap: "No mapped check."},
		}
	}
	pack := rulepack.Pack{
		BaselineCommit: ws.Baseline(),
		Rules: []rulepack.Rule{
			baseRule("b-medium", "medium", "B medium body."),
			baseRule("a-medium", "medium", "A medium body."),
			baseRule("z-high", "high", "High body."),
		},
	}

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	high := strings.Index(content, "High body.")
	aMedium := strings.Index(content, "A medium body.")
	bMedium := strings.Index(content, "B medium body.")
	if high == -1 || aMedium == -1 || bMedium == -1 ||
		!(high < aMedium && aMedium < bMedium) {
		t.Fatalf("rules are not ordered by importance then ID:\n%s", content)
	}
}

func TestApplyOrdersLegacyThenAllV2DirectiveGroups(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	rule := func(schema, id, directive string) rulepack.Rule {
		result := rulepack.Rule{
			Schema:         schema,
			ID:             id,
			Title:          id,
			Topic:          "maintainability",
			Scopes:         []string{"**/*"},
			Classification: "guidance",
			Importance:     "high",
			Score:          rulepack.Score{Method: rulepack.ScoreMethod, Total: 70},
			Confidence:     "high",
			SourcePath:     ".software-standards/rules/" + id + ".md",
			Body:           id + " body.\n",
			Verification:   rulepack.Verification{ProofGap: "No mapped check."},
		}
		if schema == rulepack.SchemaVersionV2 {
			result.Lenses = []rulepack.Lens{{Kind: "base"}}
			result.Directive = directive
		}
		return result
	}
	pack := rulepack.Pack{
		BaselineCommit: ws.Baseline(),
		Rules: []rulepack.Rule{
			rule(rulepack.SchemaVersionV2, "prefer-rule", "prefer"),
			rule(rulepack.SchemaVersionV2, "always-rule", "always"),
			rule(rulepack.SchemaVersionV1, "legacy-rule", ""),
			rule(rulepack.SchemaVersionV2, "never-rule", "never"),
			rule(rulepack.SchemaVersionV2, "ask-first-rule", "ask-first"),
		},
	}

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	headings := []string{
		"#### Legacy v1 rules (directive not recorded)",
		"#### Never",
		"#### Ask first",
		"#### Always",
		"#### Prefer",
	}
	previous := -1
	for _, heading := range headings {
		index := strings.Index(content, heading)
		if index == -1 {
			t.Fatalf("mixed projection missing %q:\n%s", heading, content)
		}
		if index < previous {
			t.Fatalf("mixed projection headings are out of order:\n%s", content)
		}
		previous = index
	}
}

func TestApplyRendersRuleV1AsLegacyWithoutInventingDirectiveOrProofCoverage(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := rulepack.Pack{
		BaselineCommit: ws.Baseline(),
		Rules: []rulepack.Rule{{
			Schema:         rulepack.SchemaVersionV1,
			ID:             "verify-before-merge",
			Title:          "Verify before merge",
			Topic:          "correctness",
			Scopes:         []string{"**/*.go"},
			Classification: "deterministic",
			Importance:     "high",
			Score:          rulepack.Score{Method: rulepack.ScoreMethod, Total: 70},
			Confidence:     "high",
			SourcePath:     ".software-standards/rules/verify-before-merge.md",
			Body:           "Run the retained verification command.\n",
			Verification:   rulepack.Verification{Command: "go test ./..."},
		}},
	}

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, required := range []string{
		"#### Legacy v1 rules (directive not recorded)",
		"Run the retained verification command.",
		"- Classification: `deterministic`",
		"`go test ./...` — mapped, not executed by ssb",
		"rule v1 did not record coverage or a bounded proved property",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("v1 projection missing %q:\n%s", required, content)
		}
	}
	if strings.Contains(content, "legacy-unspecified") {
		t.Fatalf("v1 projection invented a coverage token:\n%s", content)
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
			Schema:         rulepack.SchemaVersionV1,
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

package cli_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestREADMEAndAgentSkillMatchTheExecutableCommandContract(t *testing.T) {
	root := repositoryRoot(t)
	readme := readText(t, filepath.Join(root, "README.md"))
	skill := readText(t, filepath.Join(root, "skills", "software-standards-bootstrap", "SKILL.md"))

	for _, form := range []string{
		"ssb inspect  [--repo PATH] [--format text|json] [resource limits]",
		"ssb validate [--repo PATH] [--format text|json]",
		"ssb render   [--repo PATH] [--review ID] [--dry-run]",
		"ssb adr      [--repo PATH] [--review ID] [--adr-dir PATH] [--dry-run]",
		"ssb prune    <inspect|validate|approve|apply|recover|status|verify> [options]",
	} {
		if !strings.Contains(readme, form) {
			t.Errorf("README missing canonical form %q", form)
		}
	}
	for _, required := range []string{
		"--max-candidate-files",
		"--max-candidate-bytes",
		"--allow-partial",
		"`4`: inventory coverage incomplete",
	} {
		if !strings.Contains(readme, required) {
			t.Errorf("README missing inspect contract %q", required)
		}
	}
	for _, required := range []string{
		"ssb prune inspect --repo . --review <id> --capabilities <profile>",
		"ssb prune validate --repo . --review <id> --format text",
		"ssb prune status --repo . --review <id>",
		"Application remains a dry run unless `--write` is explicitly authorized.",
	} {
		if !strings.Contains(skill, required) {
			t.Errorf("Agent Skill missing prune contract %q", required)
		}
	}

	for _, command := range []string{
		"ssb inspect --repo . --format json",
		"ssb validate --repo . --format text",
		"ssb render --repo . --dry-run",
		"ssb render --repo .",
		"ssb adr --repo . --dry-run",
		"ssb adr --repo .",
	} {
		if !strings.Contains(skill, command) {
			t.Errorf("Agent Skill missing executable command %q", command)
		}
	}
	if !strings.Contains(skill, "Do not run `ssb adr` as part of the initial generation workflow.") {
		t.Error("Agent Skill does not preserve the human review gate before ADR generation")
	}
	for _, required := range []string{
		"Never pass `--allow-partial`",
		"Do not create proposal files from incomplete inventory coverage",
	} {
		if !strings.Contains(skill, required) {
			t.Errorf("Agent Skill missing incomplete-coverage gate %q", required)
		}
	}
	for _, forbidden := range []string{"ssb commit", "ssb push", "ssb sync", "ssb model"} {
		if strings.Contains(skill, forbidden) || strings.Contains(readme, forbidden) {
			t.Errorf("documentation advertises nonexistent command %q", forbidden)
		}
	}
}

func TestPruneDocumentationKeepsLeadInsAdjacentToExamples(t *testing.T) {
	root := repositoryRoot(t)
	skill := readText(t, filepath.Join(root, "skills", "software-standards-bootstrap", "SKILL.md"))
	reference := readText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"references",
		"prune-review.md",
	))
	pruneDoc := readText(t, filepath.Join(root, "docs", "prune.md"))

	if !strings.Contains(skill, "proof. Run:\n\n```bash\nssb prune inspect") {
		t.Error("Agent Skill separates the prune-inspection lead-in from its command example")
	}
	if !strings.Contains(reference, "and provide:\n\n```yaml\ntarget:") {
		t.Error("prune reference separates the replacement lead-in from its target example")
	}
	if !strings.Contains(pruneDoc, "insufficient.\nFor example:\n\n```yaml\nevidence_gaps:") {
		t.Error("prune documentation separates the evidence-gap lead-in from its example")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

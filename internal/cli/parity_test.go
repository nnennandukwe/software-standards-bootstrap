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
		"ssb inspect  [--repo PATH] [--format text|json]",
		"ssb validate [--repo PATH] [--format text|json]",
		"ssb render   [--repo PATH] [--dry-run]",
		"ssb adr      [--repo PATH] [--adr-dir PATH] [--dry-run]",
	} {
		if !strings.Contains(readme, form) {
			t.Errorf("README missing canonical form %q", form)
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
	for _, forbidden := range []string{"ssb commit", "ssb push", "ssb sync", "ssb model"} {
		if strings.Contains(skill, forbidden) || strings.Contains(readme, forbidden) {
			t.Errorf("documentation advertises nonexistent command %q", forbidden)
		}
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

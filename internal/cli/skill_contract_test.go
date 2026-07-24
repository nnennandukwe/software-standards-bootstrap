package cli_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentSkillRequiresStructuralPatternDiscovery(t *testing.T) {
	root := repositoryRoot(t)
	skill := readText(t, filepath.Join(root, "skills", "software-standards-bootstrap", "SKILL.md"))

	const referenceLink = "[the structural-pattern workflow](references/structural-patterns.md)"
	if !strings.Contains(skill, referenceLink) {
		t.Errorf("Agent Skill missing mandatory structural-pattern reference %q", referenceLink)
	}
	if !strings.Contains(skill, "Complete the structural-pattern pass before scoring or writing candidates.") {
		t.Error("Agent Skill does not require structural discovery before candidate scoring")
	}

	workflow := readText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"references",
		"structural-patterns.md",
	))
	workflow = strings.Join(strings.Fields(workflow), " ")
	for _, required := range []string{
		"Package and dependency boundaries",
		"Parallel implementation families",
		"Platform and configuration seams",
		"Public compatibility surfaces",
		"Source, test, and documentation symmetry",
		"narrow the scope",
		"three consistent occurrences across at least two files",
		"ordinary ecosystem convention",
		"Structural pattern review",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("structural-pattern workflow missing contract %q", required)
		}
	}
}

func TestDocumentationCarriesStructuralPatternAcceptance(t *testing.T) {
	root := repositoryRoot(t)
	documents := map[string]string{
		"README":     readText(t, filepath.Join(root, "README.md")),
		"smoke test": readText(t, filepath.Join(root, "docs", "agent-smoke-tests.md")),
		"benchmark":  readText(t, filepath.Join(root, "docs", "benchmarks.md")),
	}

	for document, content := range documents {
		if !strings.Contains(content, "structural-pattern review") {
			t.Errorf("%s documentation missing structural-pattern review acceptance", document)
		}
	}
}

func TestAgentSkillRequiresPrimaryTopicMetadata(t *testing.T) {
	root := repositoryRoot(t)
	skill := readText(t, filepath.Join(root, "skills", "software-standards-bootstrap", "SKILL.md"))
	schema := readText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"references",
		"rule-schema.md",
	))

	for _, required := range []string{
		"assign exactly one primary topic",
		"`quality` only when no narrower topic fits",
		"`metadata.topic`",
	} {
		if !strings.Contains(skill, required) {
			t.Errorf("Agent Skill missing primary-topic contract %q", required)
		}
	}
	for _, topic := range []string{
		"architecture",
		"compatibility",
		"compliance",
		"correctness",
		"developer-experience",
		"documentation",
		"maintainability",
		"operability",
		"performance",
		"quality",
		"reliability",
		"security",
		"testability",
	} {
		if !strings.Contains(schema, topic) {
			t.Errorf("rule schema missing supported primary topic %q", topic)
		}
	}
}

func TestDocumentationCarriesPrimaryTopicAcceptance(t *testing.T) {
	root := repositoryRoot(t)
	documents := map[string]string{
		"README":     readText(t, filepath.Join(root, "README.md")),
		"smoke test": readText(t, filepath.Join(root, "docs", "agent-smoke-tests.md")),
		"benchmark":  readText(t, filepath.Join(root, "docs", "benchmarks.md")),
	}

	for document, content := range documents {
		if !strings.Contains(content, "primary topic") {
			t.Errorf("%s documentation missing primary topic acceptance", document)
		}
	}
}

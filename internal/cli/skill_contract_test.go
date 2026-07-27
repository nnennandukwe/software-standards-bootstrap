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

func TestAgentSkillSupportsExistingPackProgressiveSelection(t *testing.T) {
	root := repositoryRoot(t)
	skill := readText(t, filepath.Join(root, "skills", "software-standards-bootstrap", "SKILL.md"))
	skill = strings.Join(strings.Fields(skill), " ")

	for _, required := range []string{
		"existing-pack consumption mode",
		"Reviewed-pack maintenance mode",
		"Requested-ADR mode",
		"do not run `ssb inspect`, `ssb validate`, or `ssb render`",
		"reconcile each canonical source ID and its selection metadata against every managed-index occurrence",
		"expect one contextual rule to occur under each of its lens values",
		"stopping as stale",
		"base rule active when its path scope matches",
		"every represented lens dimension",
		"report the active rule IDs",
		"Legacy v1 rules have no directive",
		"mapped, not executed",
		"ssb validate --repo . --format text",
		"ssb adr --repo . --dry-run",
	} {
		if !strings.Contains(skill, required) {
			t.Errorf("Agent Skill missing existing-pack contract %q", required)
		}
	}
}

func TestRuleSchemaReferenceDefinesV2ActivationAndProofContract(t *testing.T) {
	root := repositoryRoot(t)
	schema := readText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"references",
		"rule-schema.md",
	))

	for _, required := range []string{
		"ssb.dev/rule/v2",
		"kind: language",
		"kind: framework",
		"kind: task",
		"directive: always",
		"coverage: full",
		"mapped, not executed",
		"implementation",
		"review",
		"testing",
		"security",
		"documentation",
		"release",
	} {
		if !strings.Contains(schema, required) {
			t.Errorf("rule schema missing v2 contract %q", required)
		}
	}
}

func TestSmokeTestsDefineHostAgnosticProgressiveSelectionConformance(t *testing.T) {
	root := repositoryRoot(t)
	smokeTests := readText(t, filepath.Join(root, "docs", "agent-smoke-tests.md"))
	smokeTests = strings.Join(strings.Fields(smokeTests), " ")

	for _, required := range []string{
		"Agent-host behavioral conformance tests",
		"not part of normal developer usage",
		"conforming agent host",
		"Run this suite once for every host compatibility claim",
		"Existing-pack progressive-selection matrix",
		"`implementation`",
		"`review`",
		"`testing`",
		"`security`",
		"reports the active rule IDs",
		"does not read irrelevant contextual rule bodies",
		"never reports a mapped verification command as executed or passed",
		"| Host/version | Skill exposure method |",
		"reference adapters",
	} {
		if !strings.Contains(smokeTests, required) {
			t.Errorf("agent-host conformance tests missing contract %q", required)
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

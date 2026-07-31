package cli_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentSkillRequiresActionableCandidateRouting(t *testing.T) {
	root := repositoryRoot(t)
	skill := normalizedText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"SKILL.md",
	))

	for _, required := range []string{
		"outside planning, implementation, or verification work",
		"Review existing commands, invocation, triggers, and automatic enforcement",
		"already handled completely and automatically",
		"exactly one primary destination",
		"implementation condition → semantic rule",
		"deliberately invoked existing command → verification recipe",
		"multi-step workflow with decisions, edits, setup, branching, or recovery → Agent Skill",
		"valuable proposed automatic check → automation proposal",
		"below `medium` confidence",
		"total below 45",
		"Review each semantic name",
		"Write accepted artifacts and then the final report manifest",
		"Do not persist rejected candidates, rejection reasons, or rejection counts",
	} {
		if !strings.Contains(skill, required) {
			t.Errorf("Agent Skill missing candidate-routing contract %q", required)
		}
	}
}

func TestAgentSkillRequiresStructuralAndExistingCheckReview(t *testing.T) {
	root := repositoryRoot(t)
	skill := normalizedText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"SKILL.md",
	))
	workflow := normalizedText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"references",
		"structural-patterns.md",
	))

	for _, required := range []string{
		"dependency boundaries",
		"parallel implementations",
		"platform seams",
		"compatibility surfaces",
		"source/test/documentation symmetry",
		"existing automatic enforcement",
	} {
		if !strings.Contains(skill, required) {
			t.Errorf("Agent Skill missing discovery contract %q", required)
		}
	}
	for _, required := range []string{
		"Package and dependency boundaries",
		"Parallel implementation families",
		"Platform and configuration seams",
		"Public compatibility surfaces",
		"Source, test, and documentation symmetry",
		"three consistent occurrences across two files",
		"ordinary ecosystem conventions",
		"verification recipe",
		"automation proposal",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("structural workflow missing %q", required)
		}
	}
}

func TestSchemaReferenceDefinesAllActionableContracts(t *testing.T) {
	root := repositoryRoot(t)
	schema := normalizedText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"references",
		"rule-schema.md",
	))

	for _, required := range []string{
		"ssb.dev/report/v1",
		"ssb.dev/rule/v2",
		"ssb.dev/verification/v1",
		"ssb.dev/automation/v1",
		"metadata.category",
		"ssb-utility-v1",
		"related_artifacts",
		"role: declares",
		"role: enforces",
		"source_evidence",
		"planning",
		"implementation",
		"verification",
	} {
		if !strings.Contains(schema, required) {
			t.Errorf("actionable schema reference missing %q", required)
		}
	}
	for _, category := range []string{
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
		if !strings.Contains(schema, category) {
			t.Errorf("actionable schema reference missing category %q", category)
		}
	}
}

func TestAgentSkillSupportsActionablePackConsumptionAndMaintenance(t *testing.T) {
	root := repositoryRoot(t)
	skill := normalizedText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"SKILL.md",
	))

	for _, required := range []string{
		"existing-pack consumption mode",
		"Reviewed-pack maintenance mode",
		"Requested-ADR mode",
		"Do not run `ssb inspect`, `ssb validate`, or `ssb render`",
		"no active guidance",
		"Automation proposals are not active policy",
		"every represented lens dimension matches",
		"Report active artifact IDs",
		"Run recipe commands only when",
		"ssb validate --repo . --format text",
		"ssb render --repo . --dry-run",
		"ssb adr --repo . --dry-run",
		"automation-only pack has nothing adoptable",
	} {
		if !strings.Contains(skill, required) {
			t.Errorf("Agent Skill missing consumption or maintenance contract %q", required)
		}
	}
}

func TestReadmeKeepsAutomationImplementationReviewSeparate(t *testing.T) {
	root := repositoryRoot(t)
	readme := normalizedText(t, filepath.Join(root, "README.md"))

	for _, required := range []string{
		"Keeping an automation proposal preserves it for separate design review.",
		"It is not included in `AGENTS.md` or the adoption ADR.",
		"Adopting or merging the standards pack does not authorize its implementation or activation.",
		"Implementing or activating the proposed check requires a separate reviewed change.",
	} {
		if !strings.Contains(readme, required) {
			t.Errorf("README missing automation review boundary %q", required)
		}
	}
}

func TestBundledSkillCarriesCreatorAttribution(t *testing.T) {
	root := repositoryRoot(t)
	skill := strings.TrimSpace(readText(t, filepath.Join(
		root,
		"skills",
		"software-standards-bootstrap",
		"SKILL.md",
	)))
	want := "<!-- Source: https://github.com/nnennandukwe/software-standards-bootstrap · Author: Nnenna Ndukwe · Apache-2.0 -->"

	if !strings.HasSuffix(skill, want) {
		t.Errorf("bundled Agent Skill must end with creator attribution %q", want)
	}
}

func TestPublicDocumentationUsesOnlyActionableContracts(t *testing.T) {
	root := repositoryRoot(t)
	documents := []string{
		"README.md",
		"CHANGELOG.md",
		"CONTEXT.md",
		"CONTRIBUTING.md",
		"docs/architecture.md",
		"docs/rule-format.md",
		"docs/agent-smoke-tests.md",
		"docs/benchmarks.md",
		"docs/verification.md",
		"skills/software-standards-bootstrap/SKILL.md",
		"skills/software-standards-bootstrap/references/rule-schema.md",
		"skills/software-standards-bootstrap/references/evidence-workflow.md",
		"skills/software-standards-bootstrap/references/structural-patterns.md",
		"skills/software-standards-bootstrap/references/categories.md",
	}

	for _, relative := range documents {
		content := readText(t, filepath.Join(root, filepath.FromSlash(relative)))
		for _, forbidden := range []string{
			".software-standards/assessment.md",
			"ssb.dev/rule/v1",
			"ssb-score-v1",
			"topic:",
			"metadata.topic",
			"proof_gap:",
			"related_skills:",
		} {
			if strings.Contains(content, forbidden) {
				t.Errorf("%s retains removed contract %q", relative, forbidden)
			}
		}
	}
}

func TestSmokeAcceptanceCoversAllArtifactKinds(t *testing.T) {
	root := repositoryRoot(t)
	smoke := normalizedText(t, filepath.Join(root, "docs", "agent-smoke-tests.md"))

	for _, required := range []string{
		"Agent-host behavioral conformance tests",
		"semantic rule, verification recipe, Agent Skill, or automation proposal",
		"`planning`",
		"`implementation`",
		"`verification`",
		"reports active artifact IDs",
		"does not treat automation proposals as active",
		"divide its keep plus edit-and-keep decisions by every final artifact emitted before developer review",
		"Defer and reject decisions remain in that denominator",
		"exact fraction must meet the threshold without rounding",
	} {
		if !strings.Contains(smoke, required) {
			t.Errorf("smoke-test contract missing %q", required)
		}
	}
}

func normalizedText(t *testing.T, path string) string {
	t.Helper()
	return strings.Join(strings.Fields(readText(t, path)), " ")
}

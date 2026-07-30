package evaluation_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBenchmarkAcceptanceCountsEveryActionableArtifact(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "docs", "benchmarks.md"))
	if err != nil {
		t.Fatal(err)
	}
	benchmark := strings.Join(strings.Fields(string(content)), " ")
	for _, required := range []string{
		"counts for every emitted rule, verification recipe, Agent Skill, and automation proposal",
		"100% of emitted artifacts have resolvable evidence",
		"Every accepted candidate has exactly one actionable artifact kind",
		"At least 70% of all final artifacts",
		"fresh blind pass over all four pins",
		"proposal generation",
		"developer retention",
		"rendering",
		"ADR creation",
		"release evidence",
	} {
		if !strings.Contains(benchmark, required) {
			t.Errorf("benchmark acceptance missing %q", required)
		}
	}
}

func TestFreshActionableBenchmarkLedgerDoesNotPromoteInventoryToAcceptance(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(
		repositoryRoot(t),
		"docs",
		"benchmarks",
		"results",
		"2026-07-29-actionable",
		"README.md",
	))
	if err != nil {
		t.Fatal(err)
	}
	ledger := strings.Join(strings.Fields(string(content)), " ")
	for _, required := range []string{
		"Cobra | 66 | 705,271 | 66 | 705,271 | 65 | 631,792 | 0 | 0",
		"Flask | 235 | 1,814,782 | 235 | 1,814,782 | 230 | 1,474,850 | 0 | 0",
		"Django | 7,001 | 45,506,636 | 7,001 | 45,506,636 | 5,619 | 36,820,618 | 0 | 0",
		"Next.js | 29,073 | 111,110,455 | 29,073 | 111,110,455 | 28,403 | 88,643,646 | 0 | 0",
		"Cobra | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0",
		"Flask | 5 | 0 | 0 | 1 | 0 | 0 | 0 | 0",
		"Django | 1,382 | 0 | 0 | 0 | 4 | 0 | 72 | 0",
		"Next.js | 652 | 18 | 21 | 113 | 29 | 0 | 1,060 | 0",
		"SSB and repository-tool network use: none",
		"Claude Code 2.1.220 with `claude-sonnet-5`; model-service transport was used",
		"Tree-level exclusions (`oversized`, `secret-like`, `symlink`, `submodule`, `vendor/generated tree`, and `non-regular`) are removed before candidate accounting",
		"Scan-level `binary` and `generated` exclusions explain the candidate-to-indexed delta",
		"Proposal generation | Not complete",
		"Developer retention | Not performed",
		"Actionable-artifact acceptance remains open",
	} {
		if !strings.Contains(ledger, required) {
			t.Errorf("fresh benchmark ledger missing %q", required)
		}
	}
}

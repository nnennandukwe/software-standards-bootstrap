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

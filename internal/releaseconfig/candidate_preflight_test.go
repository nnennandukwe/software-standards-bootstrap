package releaseconfig_test

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestReleaseCandidateRunsDocumentedPreflightBeforeTag(t *testing.T) {
	var workflow struct {
		On struct {
			Push struct{ Branches []string }
		}
		Jobs map[string]struct {
			If    string
			Steps []struct {
				If              string
				ContinueOnError bool `yaml:"continue-on-error"`
				Run             string
			}
		}
	}
	source := readText(t, filepath.Join(repositoryRoot(t), ".github", "workflows", "ci.yml"))
	if err := yaml.Unmarshal([]byte(source), &workflow); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(workflow.On.Push.Branches, "release/v*") {
		t.Fatal("candidate pushes must run CI")
	}
	job, ok := workflow.Jobs["release-archive-contract"]
	if !ok || job.If != "" {
		t.Fatal("release candidates require an unconditional archive contract job")
	}
	var commands []string
	for _, step := range job.Steps {
		if step.If == "" && !step.ContinueOnError {
			commands = append(commands, strings.Split(strings.TrimSpace(step.Run), "\n")...)
		}
	}
	for _, required := range []string{
		"go run github.com/goreleaser/goreleaser/v2@v2.17.0 check",
		"go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12",
		"make verify-release-archives",
	} {
		if !slices.Contains(commands, required) {
			t.Errorf("candidate archive job missing %q", required)
		}
	}
}

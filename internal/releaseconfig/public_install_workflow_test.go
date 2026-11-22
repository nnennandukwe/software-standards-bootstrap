package releaseconfig_test

import (
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestWindowsPublicInstallChecksPublishedTagWithReadOnlyAccess(t *testing.T) {
	var workflow struct {
		Jobs map[string]struct {
			Needs       string
			RunsOn      string `yaml:"runs-on"`
			Permissions map[string]string
			Steps       []struct {
				Uses string
				With map[string]string
				Run  string
			}
		}
	}

	source := readText(t, filepath.Join(repositoryRoot(t), ".github", "workflows", "release.yml"))
	if err := yaml.Unmarshal([]byte(source), &workflow); err != nil {
		t.Fatal(err)
	}
	job, ok := workflow.Jobs["windows-public-install"]
	if !ok || job.Needs != "release" || job.RunsOn != "windows-latest" {
		t.Fatal("expected a native Windows installation check after publication")
	}
	if len(job.Permissions) != 1 || job.Permissions["contents"] != "read" {
		t.Fatalf("unexpected public install permissions: %v", job.Permissions)
	}
	taggedCheckout := false
	for _, step := range job.Steps {
		if strings.HasPrefix(step.Uses, "actions/checkout@") {
			taggedCheckout = step.With["ref"] == "refs/tags/${{ github.ref_name }}" &&
				step.With["persist-credentials"] == "false"
		}
		if taggedCheckout && step.Run == `python scripts/verify-public-install.py "$env:GITHUB_REF_NAME"` {
			return
		}
	}
	t.Fatal("expected a credential-free tagged checkout followed by verification of the public installation")
}

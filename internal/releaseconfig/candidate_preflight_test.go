package releaseconfig_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseCandidateRunsDocumentedPreflightBeforeTag(t *testing.T) {
	root := repositoryRoot(t)
	workflow := readText(t, filepath.Join(root, ".github", "workflows", "ci.yml"))
	for _, required := range []string{
		"release/v*",
		"go run github.com/goreleaser/goreleaser/v2@v2.17.0 check",
		"go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12",
		"make verify-release-archives",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("candidate CI missing %q", required)
		}
	}
}

package releaseconfig_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var pinnedAction = regexp.MustCompile(`^\s*uses:\s+[^@\s]+@[0-9a-f]{40}(?:\s+#.*)?$`)

func TestEveryGitHubActionIsPinnedToAFullCommit(t *testing.T) {
	root := repositoryRoot(t)
	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) == 0 {
		t.Fatal("no workflows found")
	}
	for _, workflow := range workflows {
		file, err := os.Open(workflow)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()
			if strings.Contains(line, "uses:") && !pinnedAction.MatchString(line) {
				t.Errorf("%s:%d action is not pinned to a full commit: %s", workflow, lineNumber, line)
			}
		}
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseConfigurationKeepsToolchainTargetsAndAttestationGates(t *testing.T) {
	root := repositoryRoot(t)
	release := readText(t, filepath.Join(root, ".github", "workflows", "release.yml"))
	config := readText(t, filepath.Join(root, ".goreleaser.yaml"))
	goMod := readText(t, filepath.Join(root, "go.mod"))
	goVersion := strings.TrimSpace(readText(t, filepath.Join(root, ".go-version")))

	for _, required := range []string{
		"go-version: 1.26.5",
		"id-token: write",
		"attestations: write",
		"subject-checksums: dist/checksums.txt",
		"sbom-path:",
		"--draft=false",
	} {
		if !strings.Contains(release, required) {
			t.Errorf("release workflow missing %q", required)
		}
	}
	if goVersion != "1.26.5" {
		t.Errorf(".go-version = %q, want 1.26.5", goVersion)
	}
	for _, required := range []string{
		"go 1.26.5",
		"tool golang.org/x/vuln/cmd/govulncheck",
	} {
		if !strings.Contains(goMod, required) {
			t.Errorf("go.mod missing %q", required)
		}
	}
	for _, required := range []string{
		"- darwin",
		"- linux",
		"- windows",
		"- amd64",
		"- arm64",
		"algorithm: sha256",
		"artifacts: archive",
		"draft: true",
		"replace_existing_artifacts: false",
	} {
		if !strings.Contains(config, required) {
			t.Errorf("GoReleaser config missing %q", required)
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

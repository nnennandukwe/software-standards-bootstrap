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
		"skills/software-standards-bootstrap/SKILL.md",
		"draft: true",
		"replace_existing_artifacts: false",
	} {
		if !strings.Contains(config, required) {
			t.Errorf("GoReleaser config missing %q", required)
		}
	}
}

func TestReleaseRunbookRequiresPinnedConfigurationPreflight(t *testing.T) {
	root := repositoryRoot(t)
	releasing := readText(t, filepath.Join(root, "docs", "releasing.md"))
	verification := readText(t, filepath.Join(root, "docs", "verification.md"))

	for _, required := range []string{
		"go run github.com/goreleaser/goreleaser/v2@v2.17.0 check",
		"go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12",
		"make verify-release-archives",
		"./install.sh --version v0.1.1 --install-dir \"$install_root/bin\"",
		"\"$install_root/bin/ssb\" --help",
		"SSB_RELEASE_ARCHIVE_DIR=\"$release_root\" SSB_RELEASE_SOURCE_REF=v0.1.1 go test ./internal/releaseconfig -run '^TestGeneratedReleaseArchivesContainCompleteSkill$'",
	} {
		if !strings.Contains(releasing, required) {
			t.Errorf("release runbook missing %q", required)
		}
		if !strings.Contains(verification, required) {
			t.Errorf("verification contract missing %q", required)
		}
	}
}

func TestV011ReleaseNotesKeepHistoricalArchiveGapExplicit(t *testing.T) {
	root := repositoryRoot(t)
	changelog := strings.Join(strings.Fields(readText(t, filepath.Join(root, "CHANGELOG.md"))), " ")

	for _, required := range []string{
		"## [0.1.1] - 2026-07-31",
		"v0.1.0 archives omitted the root `SKILL.md`",
		"complete Agent Skill",
		"creator attribution",
	} {
		if !strings.Contains(changelog, required) {
			t.Errorf("v0.1.1 release notes missing %q", required)
		}
	}
}

func TestInstallerIsPackagedAndDocumentedBeforeProductDetail(t *testing.T) {
	root := repositoryRoot(t)
	config := readText(t, filepath.Join(root, ".goreleaser.yaml"))
	readme := readText(t, filepath.Join(root, "README.md"))

	if !strings.Contains(config, "      - install.sh") {
		t.Error("GoReleaser archive does not include install.sh")
	}

	installHeading := strings.Index(readme, "## Install")
	productDetailHeading := strings.Index(readme, "## What it generates")
	if installHeading < 0 {
		t.Fatal("README does not contain an Install section")
	}
	if productDetailHeading < 0 {
		t.Fatal("README does not contain the product-detail section")
	}
	if installHeading > productDetailHeading {
		t.Error("README hides installation below product detail")
	}

	for _, required := range []string{
		"curl -fsSL https://raw.githubusercontent.com/nnennandukwe/software-standards-bootstrap/main/install.sh | sh",
		"sh -s -- --skill-dir .agents/skills",
		"\"$HOME/.local/bin/ssb\" --help",
		"sh -s -- --version v0.1.0",
		"installs the binary into `~/.local/bin`",
		"verifies its SHA-256 checksum",
	} {
		if !strings.Contains(readme, required) {
			t.Errorf("README installer section missing %q", required)
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

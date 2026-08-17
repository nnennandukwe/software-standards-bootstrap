package releaseconfig_test

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
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
		"sh scripts/release-notes.sh \"$GITHUB_REF_NAME\"",
		"args: release --clean --release-notes ${{ runner.temp }}/release-notes.md",
		"cp \"$RUNNER_TEMP/release-notes.md\" \"$RUNNER_TEMP/expected-release-body.md\"",
		"printf '\\n' >> \"$RUNNER_TEMP/expected-release-body.md\"",
		"gh release view \"${GITHUB_REF_NAME}\" --json body --template '{{.body}}'",
		"diff -u \"$RUNNER_TEMP/expected-release-body.md\" \"$RUNNER_TEMP/draft-release-notes.md\"",
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

func TestReleaseNotesScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release workflow runs this POSIX shell script on Ubuntu")
	}

	root := repositoryRoot(t)
	script := filepath.Join(root, "scripts", "release-notes.sh")
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh is unavailable")
	}
	tests := []struct {
		name      string
		changelog string
		want      string
		wantErr   string
	}{
		{
			name: "extracts only the requested adjacent section",
			changelog: "# Changelog\n\n## Unreleased\n\nNo changes yet.\n\n" +
				"## [0.2.1] - 2026-08-15\n\n- Later.\n\n" +
				"## [0.2.0] - 2026-08-14\n\n### Added\n\n- Feature.\n\n" +
				"## [0.1.1] - 2026-07-31\n\n- Earlier.\n",
			want: "\n### Added\n\n- Feature.\n\n",
		},
		{
			name:      "rejects a missing section",
			changelog: "# Changelog\n\n## [0.1.1] - 2026-07-31\n\n- Earlier.\n",
			wantErr:   "release notes heading not found",
		},
		{
			name:      "rejects a whitespace-only section",
			changelog: "# Changelog\n\n## [0.2.0] - 2026-08-14\n\n \t\n## [0.1.1] - 2026-07-31\n",
			wantErr:   "release notes section is empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changelog := filepath.Join(t.TempDir(), "CHANGELOG.md")
			if err := os.WriteFile(changelog, []byte(test.changelog), 0o644); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			command := exec.Command(shell, script, "v0.2.0", changelog)
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("release-notes.sh unexpectedly succeeded: %q", stdout.String())
				}
				if stdout.Len() != 0 {
					t.Errorf("release-notes.sh wrote partial output: %q", stdout.String())
				}
				if !strings.Contains(stderr.String(), test.wantErr) {
					t.Errorf("release-notes.sh error = %q, want substring %q", stderr.String(), test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("release-notes.sh failed: %v: %s", err, stderr.String())
			}
			if got := stdout.String(); got != test.want {
				t.Errorf("release-notes.sh output = %q, want %q", got, test.want)
			}
		})
	}
}

func TestReleaseWorkflowPublishesAfterVerification(t *testing.T) {
	root := repositoryRoot(t)
	release := readText(t, filepath.Join(root, ".github", "workflows", "release.yml"))
	steps := []string{
		"- name: Prepare curated release notes",
		"- name: Build archives, checksums, SBOMs, and draft release",
		"- name: Verify draft release notes",
		"- name: Verify complete Agent Skill in every release archive",
		"- name: Attest archive provenance from the SHA-256 manifest",
		"- name: Attest macOS amd64 SBOM",
		"- name: Attest macOS arm64 SBOM",
		"- name: Attest Linux amd64 SBOM",
		"- name: Attest Linux arm64 SBOM",
		"- name: Attest Windows amd64 SBOM",
		"- name: Attest Windows arm64 SBOM",
		"- name: Publish the completed draft",
	}
	previous := -1
	for _, step := range steps {
		index := strings.Index(release, step)
		if index < 0 {
			t.Fatalf("release workflow missing %q", step)
		}
		if index <= previous {
			t.Fatalf("release workflow step %q is out of order", step)
		}
		previous = index
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
		"./install.sh --version v0.2.0 --install-dir \"$install_root/bin\"",
		"\"$install_root/bin/ssb\" --help",
		"gh release view v0.2.0 --repo nnennandukwe/software-standards-bootstrap --json body --jq .body",
		"SSB_RELEASE_ARCHIVE_DIR=\"$release_root\" SSB_RELEASE_SOURCE_REF=v0.2.0 go test ./internal/releaseconfig -run '^TestGeneratedReleaseArchivesContainCompleteSkill$'",
	} {
		if !strings.Contains(releasing, required) {
			t.Errorf("release runbook missing %q", required)
		}
		if !strings.Contains(verification, required) {
			t.Errorf("verification contract missing %q", required)
		}
	}
}

func TestV020ReleaseNotesStateMigrationBoundary(t *testing.T) {
	root := repositoryRoot(t)
	changelog := strings.Join(strings.Fields(readText(t, filepath.Join(root, "CHANGELOG.md"))), " ")

	for _, required := range []string{
		"## [0.2.0] - 2026-08-17",
		"Breaking for JSON consumers",
		"response schema 3",
		"Published v0.1.1 embedded-layout packs remain supported without migration",
	} {
		if !strings.Contains(changelog, required) {
			t.Errorf("v0.2.0 release notes missing %q", required)
		}
	}
}

func TestV020ReleaseNotesRoutingBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	changelog := strings.Join(strings.Fields(readText(t, filepath.Join(root, "CHANGELOG.md"))), " ")

	if !strings.Contains(
		changelog,
		"distinguish host-specific `AGENTS.md` discovery and precedence from SSB-defined routing metadata",
	) {
		t.Fatal("v0.2.0 release notes omit the generated routing ownership boundary")
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
	if !strings.Contains(config, "      - install.ps1") {
		t.Error("GoReleaser archive does not include install.ps1")
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
		"sh -s -- --version v0.2.0",
		"-Version v0.2.0",
		"go install github.com/nnennandukwe/software-standards-bootstrap/cmd/ssb@v0.2.0",
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

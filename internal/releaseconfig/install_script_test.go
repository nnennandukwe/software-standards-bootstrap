package releaseconfig_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	fixtureBinary    = "#!/bin/sh\nprintf 'ssb installer fixture\\n'\n"
	fixtureReference = "A tagged reference with LF.\n"
	fixtureSkill     = "---\nname: software-standards-bootstrap\n---\n"
)

func TestInstallScriptInstallsVerifiedRelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}

	for _, test := range []struct {
		name             string
		version          string
		arguments        []string
		usesLatestLookup bool
		installsSkill    bool
	}{
		{
			name:             "latest release into the user bin directory",
			version:          "v9.8.7",
			usesLatestLookup: true,
		},
		{
			name:          "exact release with its Agent Skill",
			version:       "v1.2.3",
			arguments:     []string{"--version", "v1.2.3", "--install-dir", "INSTALL_DIR", "--skill-dir", "SKILL_DIR"},
			installsSkill: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newInstallerFixture(t, test.version, true)
			arguments := append([]string(nil), test.arguments...)
			for index, argument := range arguments {
				if argument == "INSTALL_DIR" {
					arguments[index] = fixture.installDir
				}
				if argument == "SKILL_DIR" {
					arguments[index] = fixture.skillDir
				}
			}

			output, err := fixture.run(arguments...)
			if err != nil {
				t.Fatalf("install.sh failed: %v\n%s", err, output)
			}

			installed, err := os.ReadFile(filepath.Join(fixture.installDir, "ssb"))
			if err != nil {
				t.Fatal(err)
			}
			if string(installed) != fixtureBinary {
				t.Fatalf("installed binary = %q, want fixture binary", installed)
			}
			info, err := os.Stat(filepath.Join(fixture.installDir, "ssb"))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm()&0o111 == 0 {
				t.Fatalf("installed binary mode = %o, want executable", info.Mode().Perm())
			}
			if !strings.Contains(output, "Installed ssb "+test.version+" to "+filepath.Join(fixture.installDir, "ssb")) {
				t.Fatalf("install output does not identify the installed release and path:\n%s", output)
			}
			if test.installsSkill {
				skillPath := filepath.Join(fixture.skillDir, "software-standards-bootstrap", "SKILL.md")
				installedSkill, readErr := os.ReadFile(skillPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if string(installedSkill) != fixtureSkill {
					t.Fatalf("installed skill = %q, want fixture skill", installedSkill)
				}
				if !strings.Contains(output, "Installed Agent Skill to "+filepath.Dir(skillPath)) {
					t.Fatalf("install output does not identify the Agent Skill path:\n%s", output)
				}
				referencePath := filepath.Join(filepath.Dir(skillPath), "references", "routing.md")
				if got := readText(t, referencePath); got != fixtureReference {
					t.Fatalf("installed reference = %q, want exact release bytes", got)
				}
			}

			requests := readText(t, fixture.requestLog)
			hasLatestLookup := strings.Contains(requests, "/releases/latest")
			if hasLatestLookup != test.usesLatestLookup {
				t.Fatalf("latest lookup = %v, want %v; requests:\n%s", hasLatestLookup, test.usesLatestLookup, requests)
			}
			for _, required := range []string{
				"/releases/download/" + test.version + "/" + fixture.assetName,
				"/releases/download/" + test.version + "/checksums.txt",
			} {
				if !strings.Contains(requests, required) {
					t.Errorf("request log missing %q:\n%s", required, requests)
				}
			}
		})
	}
}

func TestInstallScriptPreservesDestinationsAfterDownloadFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}
	fixture := newInstallerFixture(t, "v1.2.3", true)
	if err := os.Remove(fixture.archive); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fixture.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(fixture.installDir, "ssb")
	if err := os.WriteFile(target, []byte("existing binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := fixture.run("--version", "v1.2.3", "--skill-dir", fixture.skillDir)
	if err == nil || !strings.Contains(output, "could not download") {
		t.Fatalf("expected a download error: %v\n%s", err, output)
	}
	if got := readText(t, target); got != "existing binary" {
		t.Fatalf("existing binary changed: %q", got)
	}
	if _, err := os.Stat(fixture.skillDir); !os.IsNotExist(err) {
		t.Fatalf("unexpected skill directory: %v", err)
	}
}

func TestInstallScriptRejectsChecksumMismatchBeforeReplacingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}

	fixture := newInstallerFixture(t, "v1.2.3", false)
	if err := os.MkdirAll(fixture.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(fixture.installDir, "ssb")
	if err := os.WriteFile(target, []byte("existing binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := fixture.run("--version", "v1.2.3", "--install-dir", fixture.installDir)
	if err == nil {
		t.Fatalf("install.sh succeeded with a mismatched checksum:\n%s", output)
	}
	if !strings.Contains(strings.ToLower(output), "checksum") {
		t.Fatalf("failure does not identify the checksum problem:\n%s", output)
	}
	installed, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(installed) != "existing binary\n" {
		t.Fatalf("existing binary changed after failed verification: %q", installed)
	}
}

func TestInstallScriptChecksSkillCopyToolBeforeDownloading(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}
	fixture := newInstallerFixture(t, "v1.2.3", true)
	for _, name := range []string{"uname", "tar", "awk", "mktemp", "install", "rm"} {
		command, err := exec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(command, filepath.Join(fixture.fakeBin, name)); err != nil {
			t.Fatal(err)
		}
	}
	fixture.environment = []string{"PATH=" + fixture.fakeBin}
	output, err := fixture.run("--version", "v1.2.3", "--skill-dir", fixture.skillDir)
	if err == nil || !strings.Contains(output, "required command not found for --skill-dir: cp") {
		t.Fatalf("expected the missing skill copy tool in the diagnostic: %v\n%s", err, output)
	}
	if requests := readText(t, fixture.requestLog); requests != "" {
		t.Fatalf("installer requested a download before checking for cp: %s", requests)
	}
}

func TestInstallScriptRejectsUnsupportedPlatformsBeforeDownloading(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}

	fixture := newInstallerFixture(t, "v1.2.3", true)
	writeExecutable(t, filepath.Join(fixture.fakeBin, "uname"), `#!/bin/sh
case "$1" in
  -s) printf 'FreeBSD\n' ;;
  -m) printf 'amd64\n' ;;
esac
`)

	output, err := fixture.run("--version", "v1.2.3")
	if err == nil {
		t.Fatalf("install.sh accepted an unsupported platform:\n%s", output)
	}
	if !strings.Contains(output, "Unsupported operating system: FreeBSD") {
		t.Fatalf("failure does not identify the unsupported platform:\n%s", output)
	}
	if requests := readText(t, fixture.requestLog); requests != "" {
		t.Fatalf("installer downloaded assets for an unsupported platform:\n%s", requests)
	}
}

func TestInstallScriptRejectsADirectoryAtTheBinaryTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}

	fixture := newInstallerFixture(t, "v1.2.3", true)
	target := filepath.Join(fixture.installDir, "ssb")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := fixture.run("--version", "v1.2.3", "--install-dir", fixture.installDir)
	if err == nil {
		t.Fatalf("install.sh accepted a directory at the binary target:\n%s", output)
	}
	if !strings.Contains(output, "install target is a directory") {
		t.Fatalf("failure does not identify the invalid install target:\n%s", output)
	}
	entries, readErr := os.ReadDir(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("installer moved files into the target directory: %v", entries)
	}
}

func TestInstallScriptRejectsSkillCreatedDuringPublication(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}
	for _, kind := range []string{"directory", "empty directory", "symlink", "file"} {
		t.Run(kind, func(t *testing.T) {
			fixture := newInstallerFixture(t, "v1.2.3", true)
			if err := os.MkdirAll(fixture.installDir, 0o755); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(fixture.installDir, "ssb")
			writeExecutable(t, target, "existing binary")
			skillTarget := filepath.Join(fixture.skillDir, "software-standards-bootstrap")
			developerDir := filepath.Join(fixture.root, "developer")
			if err := os.Mkdir(developerDir, 0o755); err != nil {
				t.Fatal(err)
			}
			writeExecutable(t, filepath.Join(developerDir, "SKILL.md"), "developer-owned skill\n")
			fixture.environment = append(fixture.environment, "RACE_SKILL_TARGET="+skillTarget, "RACE_KIND="+kind, "DEVELOPER_DIR="+developerDir)
			writeExecutable(t, filepath.Join(fixture.fakeBin, "mv"), `#!/bin/sh
if [ "$1" != -f ]; then
  case "$RACE_KIND" in
    directory) /bin/mkdir "$RACE_SKILL_TARGET"; /bin/cp "$DEVELOPER_DIR/SKILL.md" "$RACE_SKILL_TARGET/SKILL.md" ;;
    'empty directory') /bin/mkdir "$RACE_SKILL_TARGET" ;;
    symlink) /bin/ln -s "$DEVELOPER_DIR" "$RACE_SKILL_TARGET" ;;
    file) /bin/cp "$DEVELOPER_DIR/SKILL.md" "$RACE_SKILL_TARGET" ;;
  esac
fi
exec /bin/mv "$@"
`)
			output, err := fixture.run("--version", "v1.2.3", "--skill-dir", fixture.skillDir)
			if err == nil {
				t.Fatalf("installer accepted a skill created during publication:\n%s", output)
			}
			if got := readText(t, target); got != "existing binary" {
				t.Fatalf("existing binary changed: %q", got)
			}
			if kind == "file" {
				if got := readText(t, skillTarget); got != "developer-owned skill\n" {
					t.Fatalf("developer file changed: %q", got)
				}
				return
			}
			if kind != "empty directory" {
				if got := readText(t, filepath.Join(skillTarget, "SKILL.md")); got != "developer-owned skill\n" {
					t.Fatalf("developer skill changed: %q", got)
				}
			}
			entries, err := os.ReadDir(skillTarget)
			want := 1
			if kind == "empty directory" {
				want = 0
			}
			if err != nil || len(entries) != want {
				t.Fatalf("installer left files inside developer skill: %v, %v", entries, err)
			}
			if kind == "symlink" {
				if destination, err := os.Readlink(skillTarget); err != nil || destination != developerDir {
					t.Fatalf("developer link changed: %q, %v", destination, err)
				}
			}
		})
	}
}

func TestInstallScriptPublishesCompleteSkillAndRecoversAfterFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}
	fixture := newInstallerFixture(t, "v1.2.3", true)
	if err := os.MkdirAll(fixture.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(fixture.installDir, "ssb")
	writeExecutable(t, target, "existing binary")
	skillTarget := filepath.Join(fixture.skillDir, "software-standards-bootstrap")
	marker := filepath.Join(fixture.root, "publication-observed")
	fixture.environment = append(fixture.environment, "SKILL_TARGET="+skillTarget, "PUBLICATION_MARKER="+marker, "FAIL_PUBLICATION=1")
	writeExecutable(t, filepath.Join(fixture.fakeBin, "mv"), `#!/bin/sh
if [ "$1" = -n ]; then
  [ ! -e "$SKILL_TARGET" ] || exit 20
  [ -f "$2/SKILL.md" ] && [ -f "$2/references/routing.md" ] || exit 21
  printf observed > "$PUBLICATION_MARKER"
  [ "$FAIL_PUBLICATION" != 1 ] || exit 22
fi
/bin/mv "$@" || exit 1
if [ "$1" = -n ]; then
  [ -f "$SKILL_TARGET/SKILL.md" ] && [ -f "$SKILL_TARGET/references/routing.md" ] || exit 23
fi
`)
	if output, err := fixture.run("--version", "v1.2.3", "--skill-dir", fixture.skillDir); err == nil {
		t.Fatalf("failed publication reported success: %s", output)
	}
	if got := readText(t, marker); got != "observed" {
		t.Fatalf("publication wasn't observed: %q", got)
	}
	if got := readText(t, target); got != "existing binary" {
		t.Fatalf("binary changed: %q", got)
	}
	entries, err := os.ReadDir(fixture.skillDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial skill or staging remains: %v, %v", entries, err)
	}
	fixture.environment = append(fixture.environment, "FAIL_PUBLICATION=0")
	if output, err := fixture.run("--version", "v1.2.3", "--skill-dir", fixture.skillDir); err != nil {
		t.Fatalf("retry failed: %v\n%s", err, output)
	}
	if got := readText(t, filepath.Join(skillTarget, "SKILL.md")); got != fixtureSkill {
		t.Fatalf("retry installed wrong skill: %q", got)
	}
}

func TestInstallScriptDoesNotOverwriteAnExistingAgentSkill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}

	fixture := newInstallerFixture(t, "v1.2.3", true)
	skillTarget := filepath.Join(fixture.skillDir, "software-standards-bootstrap")
	if err := os.MkdirAll(skillTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	skillFile := filepath.Join(skillTarget, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("developer-owned skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := fixture.run("--version", "v1.2.3", "--skill-dir", fixture.skillDir)
	if err == nil {
		t.Fatalf("install.sh overwrote an existing Agent Skill:\n%s", output)
	}
	if !strings.Contains(output, "Agent Skill already exists") {
		t.Fatalf("failure does not identify the existing Agent Skill:\n%s", output)
	}
	installedSkill, readErr := os.ReadFile(skillFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(installedSkill) != "developer-owned skill\n" {
		t.Fatalf("existing Agent Skill changed: %q", installedSkill)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.installDir, "ssb")); !os.IsNotExist(statErr) {
		t.Fatalf("binary was installed before the Agent Skill conflict was reported: %v", statErr)
	}
	if requests := readText(t, fixture.requestLog); requests != "" {
		t.Fatalf("installer downloaded assets before the Agent Skill conflict was reported:\n%s", requests)
	}
}

// v0.1.0 packaged references but omitted the entrypoint.
func fixtureArchiveSkill(version string) []archiveEntry {
	entries := []archiveEntry{{name: bundledSkillDirectory + "/references/routing.md", data: []byte(fixtureReference), mode: 0o644}}
	if version != "v0.1.0" {
		entries = append(entries, archiveEntry{name: bundledSkillDirectory + "/SKILL.md", data: []byte(fixtureSkill), mode: 0o644})
	}
	return entries
}

// Redirect the historical source fetch to a real local tag. Checkout conversion
// is deliberately enabled so comparing raw bytes exercises the user's Git configuration.
func historicalSkillEnvironment(t *testing.T) []string {
	t.Helper()
	root := t.TempDir()
	skill := filepath.Join(root, filepath.FromSlash(bundledSkillDirectory))
	if err := os.MkdirAll(filepath.Join(skill, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"SKILL.md": fixtureSkill, "references/routing.md": fixtureReference} {
		if err := os.WriteFile(filepath.Join(skill, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runGitCommand(t, root, "init", "--quiet")
	runGitCommand(t, root, "-c", "core.autocrlf=false", "add", "skills")
	runGitCommand(t, root, "-c", "user.name=Fixture", "-c", "user.email=test@example.org", "-c", "commit.gpgSign=false", "commit", "--quiet", "-m", "historical skill")
	runGitCommand(t, root, "-c", "tag.gpgSign=false", "tag", "v0.1.0")
	return []string{
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=url." + filepath.ToSlash(root) + ".insteadOf",
		"GIT_CONFIG_VALUE_0=https://github.com/nnennandukwe/software-standards-bootstrap.git",
		"GIT_CONFIG_KEY_1=core.autocrlf",
		"GIT_CONFIG_VALUE_1=true",
	}
}

func TestInstallScriptHistoricalTagSkillBytes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}
	fixture := newInstallerFixture(t, "v0.1.0", true)
	fixture.environment = historicalSkillEnvironment(t)
	output, err := fixture.run("--version", "v0.1.0", "--skill-dir", fixture.skillDir)
	if err != nil {
		t.Fatalf("historical install failed: %v\n%s", err, output)
	}
	assertHistoricalSkillBytes(t, fixture.skillDir)
}

func assertHistoricalSkillBytes(t *testing.T, skillDir string) {
	t.Helper()
	root := filepath.Join(skillDir, "software-standards-bootstrap")
	if got := readText(t, filepath.Join(root, "SKILL.md")); got != fixtureSkill {
		t.Fatalf("installed historical entrypoint = %q, want tagged LF bytes", got)
	}
	if got := readText(t, filepath.Join(root, "references", "routing.md")); got != fixtureReference {
		t.Fatalf("installed historical reference = %q, want tagged LF bytes", got)
	}
}

func TestInstallScriptInterruptAtRename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux")
	}
	for _, tc := range []struct {
		name          string
		afterBinary   bool
		signalCommand bool
	}{
		{name: "after skill publication"},
		{name: "after binary commit", afterBinary: true},
		{name: "command and shell interrupted after skill publication", signalCommand: true},
		{name: "command and shell interrupted after binary commit", afterBinary: true, signalCommand: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newInstallerFixture(t, "v1.2.3", true)
			if err := os.MkdirAll(fixture.installDir, 0o755); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(fixture.installDir, "ssb")
			writeExecutable(t, target, "existing binary")
			fixture.environment = append(fixture.environment, fmt.Sprintf("SIGNAL_AFTER_BINARY=%t", tc.afterBinary), fmt.Sprintf("SIGNAL_COMMAND=%t", tc.signalCommand))
			writeExecutable(t, filepath.Join(fixture.fakeBin, "mv"), `#!/bin/sh
/bin/mv "$@" || exit 1
case "$1:$SIGNAL_AFTER_BINARY" in
  -f:true) ;;
  -f:*) exit 0 ;;
  *:false) ;;
  *) exit 0 ;;
esac
kill -TERM "$PPID"
if [ "$SIGNAL_COMMAND" = true ]; then kill -TERM "$$"; fi
`)
			output, err := fixture.run("--version", "v1.2.3", "--skill-dir", fixture.skillDir)
			skillPath := filepath.Join(fixture.skillDir, "software-standards-bootstrap", "SKILL.md")
			if tc.afterBinary {
				if err != nil || !strings.Contains(output, "Installed Agent Skill") {
					t.Fatalf("complete install reported wrong status: %v\n%s", err, output)
				}
				if got := readText(t, target); got != fixtureBinary {
					t.Fatalf("committed binary = %q", got)
				}
				if got := readText(t, skillPath); got != fixtureSkill {
					t.Fatalf("committed skill = %q", got)
				}
			} else {
				if err == nil {
					t.Fatalf("pre-commit interrupt reported success: %s", output)
				}
				if got := readText(t, target); got != "existing binary" {
					t.Fatalf("old binary changed: %q", got)
				}
				if _, err := os.Stat(filepath.Dir(skillPath)); !os.IsNotExist(err) {
					t.Fatalf("pre-commit skill remains: %v", err)
				}
			}
			for _, dest := range []string{fixture.skillDir, fixture.installDir} {
				entries, err := os.ReadDir(dest)
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), ".") {
						t.Errorf("staging path remains: %s", entry.Name())
					}
				}
			}
		})
	}
}

type installerFixture struct {
	environment   []string
	t             *testing.T
	root          string
	fakeBin       string
	home          string
	installDir    string
	skillDir      string
	archive       string
	checksums     string
	requestLog    string
	assetName     string
	latestVersion string
}

func newInstallerFixture(t *testing.T, version string, validChecksum bool) *installerFixture {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("POSIX sh is unavailable")
	}

	root := t.TempDir()
	fakeBin := filepath.Join(root, "bin")
	home := filepath.Join(root, "home")
	installDir := filepath.Join(home, ".local", "bin")
	skillDir := filepath.Join(root, "repository", ".agents", "skills")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goarch == "x86_64" {
		goarch = "amd64"
	}
	assetName := fmt.Sprintf("ssb_%s_%s_%s.tar.gz", version, goos, goarch)
	archive := filepath.Join(root, assetName)
	entries := []archiveEntry{{name: "ssb", data: []byte(fixtureBinary), mode: 0o755}}
	entries = append(entries, fixtureArchiveSkill(version)...)
	writeTarGzip(t, archive, entries)
	digest, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(digest)
	if !validChecksum {
		sum[0] ^= 0xff
	}
	checksums := filepath.Join(root, "checksums.txt")
	if err := os.WriteFile(checksums, []byte(fmt.Sprintf("%x  %s\n", sum, assetName)), 0o644); err != nil {
		t.Fatal(err)
	}

	requestLog := filepath.Join(root, "requests.log")
	if err := os.WriteFile(requestLog, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	writeExecutable(t, filepath.Join(fakeBin, "curl"), `#!/bin/sh
output=
write_format=
url=
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      output=$2
      shift 2
      ;;
    -w)
      write_format=$2
      shift 2
      ;;
    -* )
      shift
      ;;
    *)
      url=$1
      shift
      ;;
  esac
done
printf '%s\n' "$url" >>"$FAKE_REQUEST_LOG"
case "$url" in
  */releases/latest)
    if [ -n "$write_format" ]; then
      printf '%s' "$FAKE_LATEST_URL"
    fi
    ;;
  */checksums.txt)
    cp "$FAKE_CHECKSUMS" "$output"
    ;;
  */"$FAKE_ASSET_NAME")
    cp "$FAKE_ARCHIVE" "$output"
    ;;
  *)
    printf 'unexpected URL: %s\n' "$url" >&2
    exit 22
    ;;
esac
`)

	return &installerFixture{
		t:             t,
		root:          root,
		fakeBin:       fakeBin,
		home:          home,
		installDir:    installDir,
		skillDir:      skillDir,
		archive:       archive,
		checksums:     checksums,
		requestLog:    requestLog,
		assetName:     assetName,
		latestVersion: version,
	}
}

func (fixture *installerFixture) run(arguments ...string) (string, error) {
	fixture.t.Helper()
	script := filepath.Join(repositoryRoot(fixture.t), "install.sh")
	command := exec.Command("sh", append([]string{script}, arguments...)...)
	command.Env = append(os.Environ(),
		"GIT_ALLOW_PROTOCOL=file",
		"PATH="+fixture.fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+fixture.home,
		"FAKE_ARCHIVE="+fixture.archive,
		"FAKE_ASSET_NAME="+fixture.assetName,
		"FAKE_CHECKSUMS="+fixture.checksums,
		"FAKE_LATEST_URL=https://github.com/nnennandukwe/software-standards-bootstrap/releases/tag/"+fixture.latestVersion,
		"FAKE_REQUEST_LOG="+fixture.requestLog,
	)
	command.Env = append(command.Env, fixture.environment...)
	output, err := command.CombinedOutput()
	return string(output), err
}

type archiveEntry struct {
	name string
	data []byte
	mode int64
}

func writeTarGzip(t *testing.T, path string, entries []archiveEntry) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Mode: entry.mode, Size: int64(len(entry.data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

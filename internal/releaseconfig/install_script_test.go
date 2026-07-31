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
	fixtureBinary = "#!/bin/sh\nprintf 'ssb installer fixture\\n'\n"
	fixtureSkill  = "---\nname: software-standards-bootstrap\n---\n"
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
				cloneLog := readText(t, fixture.cloneLog)
				for _, required := range []string{
					"--branch " + test.version,
					"https://github.com/nnennandukwe/software-standards-bootstrap.git",
				} {
					if !strings.Contains(cloneLog, required) {
						t.Errorf("clone log missing %q:\n%s", required, cloneLog)
					}
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

type installerFixture struct {
	t             *testing.T
	root          string
	fakeBin       string
	home          string
	installDir    string
	skillDir      string
	archive       string
	checksums     string
	requestLog    string
	cloneLog      string
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
	writeTarGzip(t, archive, []archiveEntry{
		{name: "ssb", data: []byte(fixtureBinary), mode: 0o755},
	})
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
	cloneLog := filepath.Join(root, "clone.log")
	if err := os.WriteFile(cloneLog, nil, 0o644); err != nil {
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
	writeExecutable(t, filepath.Join(fakeBin, "git"), `#!/bin/sh
printf '%s\n' "$*" >>"$FAKE_CLONE_LOG"
destination=
for argument in "$@"; do
  destination=$argument
done
mkdir -p "$destination/skills/software-standards-bootstrap"
cp "$FAKE_SKILL_SOURCE" "$destination/skills/software-standards-bootstrap/SKILL.md"
`)
	skillSource := filepath.Join(root, "fixture-SKILL.md")
	if err := os.WriteFile(skillSource, []byte(fixtureSkill), 0o644); err != nil {
		t.Fatal(err)
	}

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
		cloneLog:      cloneLog,
		assetName:     assetName,
		latestVersion: version,
	}
}

func (fixture *installerFixture) run(arguments ...string) (string, error) {
	fixture.t.Helper()
	script := filepath.Join(repositoryRoot(fixture.t), "install.sh")
	command := exec.Command("sh", append([]string{script}, arguments...)...)
	command.Env = append(os.Environ(),
		"PATH="+fixture.fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+fixture.home,
		"FAKE_ARCHIVE="+fixture.archive,
		"FAKE_ASSET_NAME="+fixture.assetName,
		"FAKE_CHECKSUMS="+fixture.checksums,
		"FAKE_LATEST_URL=https://github.com/nnennandukwe/software-standards-bootstrap/releases/tag/"+fixture.latestVersion,
		"FAKE_REQUEST_LOG="+fixture.requestLog,
		"FAKE_CLONE_LOG="+fixture.cloneLog,
		"FAKE_SKILL_SOURCE="+filepath.Join(fixture.root, "fixture-SKILL.md"),
	)
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

package releaseconfig_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// install.ps1 is the Windows counterpart of install.sh. These tests assert the
// same user-visible contract: a verified release is installed, and every
// rejected path stops before it can replace developer-owned files.

func TestInstallPowerShellScriptInstallsVerifiedRelease(t *testing.T) {
	for _, test := range []struct {
		name             string
		version          string
		arguments        []string
		environment      []string
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
			arguments:     []string{"-Version", "v1.2.3", "-InstallDir", "{{INSTALL_DIR}}", "-SkillDir", "{{SKILL_DIR}}"},
			installsSkill: true,
		},
		{
			name:          "release and directories supplied by environment variables",
			version:       "v1.2.3",
			environment:   []string{"SSB_VERSION=v1.2.3", "SSB_INSTALL_DIR={{INSTALL_DIR}}", "SSB_SKILL_DIR={{SKILL_DIR}}"},
			installsSkill: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPowerShellInstallerFixture(t, test.version, true)
			arguments := fixture.expand(test.arguments)
			fixture.environment = fixture.expand(test.environment)

			output, err := fixture.run(arguments...)
			if err != nil {
				t.Fatalf("install.ps1 failed: %v\n%s", err, output)
			}

			target := filepath.Join(fixture.installDir, "ssb.exe")
			installed, readErr := os.ReadFile(target)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(installed, fixture.expectedBinary) {
				t.Fatalf("installed binary does not match the fixture binary")
			}
			if !strings.Contains(output, "Installed ssb "+test.version+" to "+target) {
				t.Fatalf("install output does not identify the installed release and path:\n%s", output)
			}
			// The install directory is not on PATH in the fixture, so the hint
			// has to name the exact directory the binary landed in.
			wantHint := "$env:PATH = \"" + fixture.installDir + ";$env:PATH\""
			if !strings.Contains(output, wantHint) {
				t.Fatalf("install output does not contain the PATH hint %q:\n%s", wantHint, output)
			}
			if test.installsSkill {
				skillPath := filepath.Join(fixture.skillDir, "software-standards-bootstrap", "SKILL.md")
				installedSkill, skillErr := os.ReadFile(skillPath)
				if skillErr != nil {
					t.Fatal(skillErr)
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
					"--depth 1",
					"--single-branch",
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

// A release binary that writes a warning to stderr and exits zero is working.
// PowerShell wraps redirected native stderr in an ErrorRecord, so an installer
// that treats that as failure would reject a perfectly good release.
func TestInstallPowerShellScriptAcceptsABinaryThatWritesToStandardError(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)
	fixture.useBinarySource(t, fixtureStandardErrorSSBSource)

	output, err := fixture.run("-Version", "v1.2.3", "-InstallDir", fixture.installDir)
	if err != nil {
		t.Fatalf("install.ps1 rejected a working binary that writes to stderr: %v\n%s", err, output)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.installDir, "ssb.exe")); statErr != nil {
		t.Fatalf("binary was not installed: %v", statErr)
	}
}

// The mirror of the case above: the smoke run exists to reject an archive whose
// binary cannot execute here, so a non-zero exit must stop the install.
func TestInstallPowerShellScriptRejectsABinaryThatCannotRun(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)
	fixture.useBinarySource(t, fixtureFailingSSBSource)

	output, err := fixture.run("-Version", "v1.2.3", "-InstallDir", fixture.installDir)
	if err == nil {
		t.Fatalf("install.ps1 installed a binary that cannot run:\n%s", output)
	}
	if !strings.Contains(output, "could not run on windows/") {
		t.Fatalf("failure does not identify the unusable binary:\n%s", output)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.installDir, "ssb.exe")); !os.IsNotExist(statErr) {
		t.Fatalf("binary was installed despite failing the smoke run: %v", statErr)
	}
}

func TestInstallPowerShellScriptShowsUsageAndRejectsUnknownArguments(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)

	output, err := fixture.run("-Help")
	if err != nil {
		t.Fatalf("install.ps1 -Help failed: %v\n%s", err, output)
	}
	for _, required := range []string{"Usage:", "-Version", "-InstallDir", "-SkillDir", "SSB_VERSION"} {
		if !strings.Contains(output, required) {
			t.Errorf("usage output missing %q:\n%s", required, output)
		}
	}
	if requests := readText(t, fixture.requestLog); requests != "" {
		t.Fatalf("installer downloaded assets while showing usage:\n%s", requests)
	}

	unknown := newPowerShellInstallerFixture(t, "v1.2.3", true)
	output, err = unknown.run("--skill-dir", "some-directory")
	if err == nil {
		t.Fatalf("install.ps1 accepted an unknown option:\n%s", output)
	}
	if !strings.Contains(output, "unknown option: --skill-dir") {
		t.Fatalf("failure does not identify the unknown option:\n%s", output)
	}
	if requests := readText(t, unknown.requestLog); requests != "" {
		t.Fatalf("installer downloaded assets for an unknown option:\n%s", requests)
	}
}

func TestInstallPowerShellScriptRejectsChecksumMismatchBeforeReplacingBinary(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", false)
	if err := os.MkdirAll(fixture.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(fixture.installDir, "ssb.exe")
	if err := os.WriteFile(target, []byte("existing binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := fixture.run("-Version", "v1.2.3", "-InstallDir", fixture.installDir)
	if err == nil {
		t.Fatalf("install.ps1 succeeded with a mismatched checksum:\n%s", output)
	}
	if !strings.Contains(strings.ToLower(output), "checksum verification failed") {
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

func TestInstallPowerShellScriptReplacesAnExistingBinary(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)
	if err := os.MkdirAll(fixture.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(fixture.installDir, "ssb.exe")
	if err := os.WriteFile(target, []byte("existing binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := fixture.run("-Version", "v1.2.3", "-InstallDir", fixture.installDir)
	if err != nil {
		t.Fatalf("install.ps1 failed to upgrade an existing binary: %v\n%s", err, output)
	}
	installed, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(installed, fixture.expectedBinary) {
		t.Fatalf("existing binary was not replaced with the release binary")
	}
	// A staged copy or a rollback backup left behind would put an extra entry
	// on the user's PATH.
	entries, readErr := os.ReadDir(fixture.installDir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 1 || entries[0].Name() != "ssb.exe" {
		t.Fatalf("install directory = %v, want only ssb.exe", entries)
	}
}

// A checksums.txt carrying a blank or single-token line is still well formed.
// The entry for this asset has to be selected without tripping over them.
func TestInstallPowerShellScriptSelectsTheAssetEntryAmongUnrelatedLines(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)
	existing := readText(t, fixture.checksums)
	noise := "\n" + strings.Repeat("-", 16) + "\n\n" +
		"0000000000000000000000000000000000000000000000000000000000000000  ssb_v1.2.3_linux_amd64.tar.gz\n"
	if err := os.WriteFile(fixture.checksums, []byte(noise+existing), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := fixture.run("-Version", "v1.2.3", "-InstallDir", fixture.installDir)
	if err != nil {
		t.Fatalf("install.ps1 failed on a checksums.txt with unrelated lines: %v\n%s", err, output)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.installDir, "ssb.exe")); statErr != nil {
		t.Fatalf("binary was not installed: %v", statErr)
	}
}

func TestInstallPowerShellScriptRejectsUnsupportedArchitectureBeforeDownloading(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)
	fixture.processorArchitecture = "IA64"

	output, err := fixture.run("-Version", "v1.2.3")
	if err == nil {
		t.Fatalf("install.ps1 accepted an unsupported architecture:\n%s", output)
	}
	if !strings.Contains(output, "Unsupported architecture: IA64") {
		t.Fatalf("failure does not identify the unsupported architecture:\n%s", output)
	}
	if requests := readText(t, fixture.requestLog); requests != "" {
		t.Fatalf("installer downloaded assets for an unsupported architecture:\n%s", requests)
	}
}

func TestInstallPowerShellScriptRejectsInvalidReleaseTagBeforeDownloading(t *testing.T) {
	for _, tag := range []string{"latest", "v1.0.0\n"} {
		fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)

		output, err := fixture.run("-Version", tag)
		if err == nil {
			t.Fatalf("install.ps1 accepted the invalid release tag %q:\n%s", tag, output)
		}
		if !strings.Contains(output, "invalid release tag") {
			t.Fatalf("failure does not identify the invalid release tag %q:\n%s", tag, output)
		}
		if requests := readText(t, fixture.requestLog); requests != "" {
			t.Fatalf("installer downloaded assets for the invalid release tag %q:\n%s", tag, requests)
		}
	}
}

func TestInstallPowerShellScriptRejectsADirectoryAtTheBinaryTarget(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)
	target := filepath.Join(fixture.installDir, "ssb.exe")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := fixture.run("-Version", "v1.2.3", "-InstallDir", fixture.installDir)
	if err == nil {
		t.Fatalf("install.ps1 accepted a directory at the binary target:\n%s", output)
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

func TestInstallPowerShellScriptDoesNotOverwriteAnExistingAgentSkill(t *testing.T) {
	fixture := newPowerShellInstallerFixture(t, "v1.2.3", true)
	skillTarget := filepath.Join(fixture.skillDir, "software-standards-bootstrap")
	if err := os.MkdirAll(skillTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	skillFile := filepath.Join(skillTarget, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("developer-owned skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := fixture.run("-Version", "v1.2.3", "-SkillDir", fixture.skillDir)
	if err == nil {
		t.Fatalf("install.ps1 overwrote an existing Agent Skill:\n%s", output)
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
	if _, statErr := os.Stat(filepath.Join(fixture.installDir, "ssb.exe")); !os.IsNotExist(statErr) {
		t.Fatalf("binary was installed before the Agent Skill conflict was reported: %v", statErr)
	}
	if requests := readText(t, fixture.requestLog); requests != "" {
		t.Fatalf("installer downloaded assets before the Agent Skill conflict was reported:\n%s", requests)
	}
}

type powerShellInstallerFixture struct {
	t                     *testing.T
	root                  string
	fakeBin               string
	home                  string
	installDir            string
	skillDir              string
	archive               string
	checksums             string
	requestLog            string
	cloneLog              string
	assetName             string
	latestVersion         string
	processorArchitecture string
	environment           []string
	expectedBinary        []byte
	validChecksum         bool
}

func newPowerShellInstallerFixture(t *testing.T, version string, validChecksum bool) *powerShellInstallerFixture {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("install.ps1 targets Windows")
	}
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		t.Skip("Windows PowerShell is unavailable")
	}

	root := t.TempDir()
	fakeBin := filepath.Join(root, "bin")
	home := filepath.Join(root, "home")
	installDir := filepath.Join(home, ".local", "bin")
	skillDir := filepath.Join(root, "repository", ".agents", "skills")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}

	assetName := fmt.Sprintf("ssb_%s_windows_%s.zip", version, runtime.GOARCH)

	requestLog := filepath.Join(root, "requests.log")
	if err := os.WriteFile(requestLog, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cloneLog := filepath.Join(root, "clone.log")
	if err := os.WriteFile(cloneLog, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	skillSource := filepath.Join(root, "fixture-SKILL.md")
	if err := os.WriteFile(skillSource, []byte(fixtureSkill), 0o644); err != nil {
		t.Fatal(err)
	}

	// install.ps1 shells out to curl.exe and git.exe, so the fakes must be real
	// executables that Windows will resolve from PATH.
	copyFile(t, sharedFakeExecutable(t, "curl", fakeCurlSource), filepath.Join(fakeBin, "curl.exe"))
	copyFile(t, sharedFakeExecutable(t, "git", fakeGitSource), filepath.Join(fakeBin, "git.exe"))

	fixture := &powerShellInstallerFixture{
		t:                     t,
		root:                  root,
		fakeBin:               fakeBin,
		home:                  home,
		installDir:            installDir,
		skillDir:              skillDir,
		archive:               filepath.Join(root, assetName),
		checksums:             filepath.Join(root, "checksums.txt"),
		requestLog:            requestLog,
		cloneLog:              cloneLog,
		assetName:             assetName,
		latestVersion:         version,
		processorArchitecture: os.Getenv("PROCESSOR_ARCHITECTURE"),
		validChecksum:         validChecksum,
	}
	fixture.useBinarySource(t, fixtureSSBSource)
	return fixture
}

// useBinarySource rebuilds the release archive around a different fixture
// binary, so tests can cover what happens when the downloaded ssb.exe misbehaves.
func (fixture *powerShellInstallerFixture) useBinarySource(t *testing.T, source string) {
	t.Helper()
	binaryPath := sharedFakeExecutable(t, fixtureBinaryName(source), source)
	expectedBinary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	writeZip(t, fixture.archive, []archiveEntry{
		{name: "ssb.exe", data: expectedBinary, mode: 0o755},
	})

	archiveBytes, err := os.ReadFile(fixture.archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveBytes)
	if !fixture.validChecksum {
		sum[0] ^= 0xff
	}
	contents := fmt.Sprintf("%x  %s\n", sum, fixture.assetName)
	if err := os.WriteFile(fixture.checksums, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.expectedBinary = expectedBinary
}

// Placeholders are delimited so they cannot match inside a variable name such
// as SSB_SKILL_DIR, which would rewrite the name along with the value.
func (fixture *powerShellInstallerFixture) expand(values []string) []string {
	expanded := append([]string(nil), values...)
	for index, value := range expanded {
		expanded[index] = strings.ReplaceAll(value, "{{INSTALL_DIR}}", fixture.installDir)
		expanded[index] = strings.ReplaceAll(expanded[index], "{{SKILL_DIR}}", fixture.skillDir)
	}
	return expanded
}

func (fixture *powerShellInstallerFixture) run(arguments ...string) (string, error) {
	fixture.t.Helper()
	script := filepath.Join(repositoryRoot(fixture.t), "install.ps1")
	command := exec.Command("powershell.exe",
		append([]string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script}, arguments...)...)
	command.Env = append(os.Environ(),
		"PATH="+fixture.fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"USERPROFILE="+fixture.home,
		"HOME="+fixture.home,
		// PROCESSOR_ARCHITEW6432 is deliberately not set: on Windows it is the
		// WOW64 signal, and setting it at all makes the OS rewrite
		// PROCESSOR_ARCHITECTURE in the child process.
		"PROCESSOR_ARCHITECTURE="+fixture.processorArchitecture,
		"SSB_VERSION=",
		"SSB_INSTALL_DIR=",
		"SSB_SKILL_DIR=",
		"FAKE_ARCHIVE="+fixture.archive,
		"FAKE_ASSET_NAME="+fixture.assetName,
		"FAKE_CHECKSUMS="+fixture.checksums,
		"FAKE_LATEST_URL=https://github.com/nnennandukwe/software-standards-bootstrap/releases/tag/"+fixture.latestVersion,
		"FAKE_REQUEST_LOG="+fixture.requestLog,
		"FAKE_CLONE_LOG="+fixture.cloneLog,
		"FAKE_SKILL_SOURCE="+filepath.Join(fixture.root, "fixture-SKILL.md"),
	)
	command.Env = append(command.Env, fixture.environment...)
	output, err := command.CombinedOutput()
	return string(output), err
}

func copyFile(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0o755); err != nil {
		t.Fatal(err)
	}
}

var (
	fakeExecutableMutex sync.Mutex
	fakeExecutableDir   string
	fakeExecutables     = map[string]string{}
)

// The shared fake-executable directory is package-level state with no owning
// test, so nothing removes it via t.TempDir; without this it survives every
// run.
func TestMain(m *testing.M) {
	code := m.Run()
	if fakeExecutableDir != "" {
		os.RemoveAll(fakeExecutableDir)
	}
	os.Exit(code)
}

// Each fake costs a Go compile, so build one copy per distinct source and share
// it across every fixture rather than rebuilding per test.
func sharedFakeExecutable(t *testing.T, name, source string) string {
	t.Helper()
	fakeExecutableMutex.Lock()
	defer fakeExecutableMutex.Unlock()

	if path, ok := fakeExecutables[name]; ok {
		return path
	}
	if fakeExecutableDir == "" {
		directory, err := os.MkdirTemp("", "ssb-fake-executables")
		if err != nil {
			t.Fatal(err)
		}
		fakeExecutableDir = directory
	}

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "main.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(fakeExecutableDir, name+".exe")
	command := exec.Command("go", "build", "-o", path, sourcePath)
	command.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if output, err := command.CombinedOutput(); err != nil {
		// A skip here would report the whole Windows installer suite as green
		// without running any of it.
		t.Fatalf("could not build the fake executable %s: %v\n%s", name, err, output)
	}
	fakeExecutables[name] = path
	return path
}

func fixtureBinaryName(source string) string {
	digest := sha256.Sum256([]byte(source))
	return fmt.Sprintf("ssb-fixture-%x", digest[:8])
}

const fixtureSSBSource = `package main

import "fmt"

func main() {
	fmt.Println("Software Standards Bootstrap")
}
`

const fixtureStandardErrorSSBSource = `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "warning: this release prints a notice on stderr")
	fmt.Println("Software Standards Bootstrap")
}
`

const fixtureFailingSSBSource = `package main

import "os"

func main() {
	os.Exit(1)
}
`

const fakeCurlSource = `package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	var output, writeFormat, url string
	arguments := os.Args[1:]
	for index := 0; index < len(arguments); index++ {
		switch {
		case arguments[index] == "-o" && index+1 < len(arguments):
			output = arguments[index+1]
			index++
		case arguments[index] == "-w" && index+1 < len(arguments):
			writeFormat = arguments[index+1]
			index++
		case strings.HasPrefix(arguments[index], "-"):
		default:
			url = arguments[index]
		}
	}

	log, err := os.OpenFile(os.Getenv("FAKE_REQUEST_LOG"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		os.Exit(22)
	}
	fmt.Fprintln(log, url)
	log.Close()

	copyFile := func(source string) {
		data, readErr := os.ReadFile(source)
		if readErr != nil {
			os.Exit(22)
		}
		if writeErr := os.WriteFile(output, data, 0o644); writeErr != nil {
			os.Exit(22)
		}
	}

	switch {
	case strings.HasSuffix(url, "/releases/latest"):
		if writeFormat != "" {
			io.WriteString(os.Stdout, os.Getenv("FAKE_LATEST_URL"))
		}
	case strings.HasSuffix(url, "/checksums.txt"):
		copyFile(os.Getenv("FAKE_CHECKSUMS"))
	case strings.HasSuffix(url, "/"+os.Getenv("FAKE_ASSET_NAME")):
		copyFile(os.Getenv("FAKE_ARCHIVE"))
	default:
		fmt.Fprintf(os.Stderr, "unexpected URL: %s\n", url)
		os.Exit(22)
	}
}
`

const fakeGitSource = `package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	arguments := os.Args[1:]
	log, err := os.OpenFile(os.Getenv("FAKE_CLONE_LOG"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		os.Exit(1)
	}
	fmt.Fprintln(log, strings.Join(arguments, " "))
	log.Close()

	destination := arguments[len(arguments)-1]
	skillDirectory := filepath.Join(destination, "skills", "software-standards-bootstrap")
	if mkdirErr := os.MkdirAll(skillDirectory, 0o755); mkdirErr != nil {
		os.Exit(1)
	}
	data, readErr := os.ReadFile(os.Getenv("FAKE_SKILL_SOURCE"))
	if readErr != nil {
		os.Exit(1)
	}
	if writeErr := os.WriteFile(filepath.Join(skillDirectory, "SKILL.md"), data, 0o644); writeErr != nil {
		os.Exit(1)
	}
}
`

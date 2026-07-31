package releaseconfig_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	archivepath "path"
	"path/filepath"
	"strings"
	"testing"
)

const (
	releaseArchiveDirectoryEnv = "SSB_RELEASE_ARCHIVE_DIR"
	bundledSkillDirectory      = "skills/software-standards-bootstrap"
)

var releaseArchiveTargets = map[string]string{
	"darwin_amd64":  ".tar.gz",
	"darwin_arm64":  ".tar.gz",
	"linux_amd64":   ".tar.gz",
	"linux_arm64":   ".tar.gz",
	"windows_amd64": ".zip",
	"windows_arm64": ".zip",
}

func TestReleaseArchiveContractRejectsIncompleteSkillBundles(t *testing.T) {
	root := repositoryRoot(t)

	for _, test := range []struct {
		name          string
		missingTarget string
		missingPath   string
		extraPath     string
		want          string
	}{
		{
			name:        "missing skill entrypoint",
			missingPath: bundledSkillDirectory + "/SKILL.md",
			want:        "darwin_amd64 missing " + bundledSkillDirectory + "/SKILL.md",
		},
		{
			name:        "missing referenced file",
			missingPath: bundledSkillDirectory + "/references/rule-schema.md",
			want:        "darwin_amd64 missing " + bundledSkillDirectory + "/references/rule-schema.md",
		},
		{
			name:          "missing platform archive",
			missingTarget: "windows_arm64",
			want:          "missing target windows_arm64",
		},
		{
			name:      "extra skill file",
			extraPath: bundledSkillDirectory + "/references/stale.md",
			want:      "darwin_amd64 contains unexpected " + bundledSkillDirectory + "/references/stale.md",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			archives := t.TempDir()
			writeReleaseArchiveMatrix(t, root, archives, test.missingTarget, test.missingPath, test.extraPath)

			err := verifyReleaseArchives(root, archives, "HEAD")
			if err == nil {
				t.Fatal("incomplete release archives passed verification")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("verification error = %q, want it to contain %q", err, test.want)
			}
		})
	}
}

func TestReleaseArchiveContractPinsExpectedSkillToSourceRef(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, filepath.FromSlash(bundledSkillDirectory), "SKILL.md")
	reference := filepath.Join(root, filepath.FromSlash(bundledSkillDirectory), "references", "rule-schema.md")
	if err := os.MkdirAll(filepath.Dir(reference), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skill, []byte("tagged skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reference, []byte("tagged reference\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, root, "init", "--quiet")
	runGitCommand(t, root, "add", bundledSkillDirectory)
	runGitCommand(t, root, "-c", "user.name=Release Test", "-c", "user.email=release-test@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "tagged source")
	runGitCommand(t, root, "tag", "v1.0.0")

	if err := os.WriteFile(skill, []byte("dirty later skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	archives := t.TempDir()
	writeReleaseArchiveMatrix(t, root, archives, "", "", "")

	err := verifyReleaseArchives(root, archives, "v1.0.0")
	if err == nil {
		t.Fatal("archives matching a dirty later checkout passed as tagged-source evidence")
	}
	if !strings.Contains(err.Error(), "does not match source ref v1.0.0") {
		t.Fatalf("verification error = %q, want tagged-source mismatch", err)
	}
}

func TestGeneratedReleaseArchivesContainCompleteSkill(t *testing.T) {
	archives := os.Getenv(releaseArchiveDirectoryEnv)
	if archives == "" {
		t.Skip(releaseArchiveDirectoryEnv + " is not set; the snapshot and release workflows exercise the generated archives")
	}
	root := repositoryRoot(t)
	if !filepath.IsAbs(archives) {
		archives = filepath.Join(root, archives)
	}

	sourceRef := os.Getenv("SSB_RELEASE_SOURCE_REF")
	if sourceRef == "" {
		t.Fatal("SSB_RELEASE_SOURCE_REF is required when " + releaseArchiveDirectoryEnv + " is set")
	}
	if err := verifyReleaseArchives(root, archives, sourceRef); err != nil {
		t.Fatal(err)
	}
}

func verifyReleaseArchives(root, archives, sourceRef string) error {
	expected, err := readExpectedSkillFiles(root, sourceRef)
	if err != nil {
		return err
	}
	for target, extension := range releaseArchiveTargets {
		pattern := filepath.Join(archives, "ssb_*_"+target+extension)
		matches, globErr := filepath.Glob(pattern)
		if globErr != nil {
			return fmt.Errorf("match release archive target %s: %w", target, globErr)
		}
		if len(matches) == 0 {
			return fmt.Errorf("release archives missing target %s; run the pinned GoReleaser build", target)
		}
		if len(matches) != 1 {
			return fmt.Errorf("release archive target %s matched %d files; keep exactly one archive per target", target, len(matches))
		}

		actual, readErr := readArchiveFiles(matches[0])
		if readErr != nil {
			return fmt.Errorf("inspect release archive %s: %w", target, readErr)
		}
		for name, expectedData := range expected {
			actualData, found := actual[name]
			if !found {
				return fmt.Errorf("release archive %s missing %s; rebuild with the complete Agent Skill bundle", target, name)
			}
			if !bytes.Equal(actualData, expectedData) {
				return fmt.Errorf("release archive %s contains %s that does not match source ref %s", target, name, sourceRef)
			}
		}
		for name := range actual {
			if !strings.HasPrefix(name, bundledSkillDirectory+"/") {
				continue
			}
			if _, found := expected[name]; !found {
				return fmt.Errorf("release archive %s contains unexpected %s; rebuild from the exact source ref %s", target, name, sourceRef)
			}
		}
	}
	return nil
}

func runGitCommand(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func writeReleaseArchiveMatrix(t *testing.T, root, archives, missingTarget, missingPath, extraPath string) {
	t.Helper()
	expected := expectedSkillFiles(t, root)
	for target, extension := range releaseArchiveTargets {
		if target == missingTarget {
			continue
		}
		entries := []archiveEntry{{name: "ssb", data: []byte("fixture\n"), mode: 0o755}}
		for name, data := range expected {
			if name == missingPath && target == "darwin_amd64" {
				continue
			}
			entries = append(entries, archiveEntry{name: name, data: data, mode: 0o644})
		}
		if extraPath != "" && target == "darwin_amd64" {
			entries = append(entries, archiveEntry{name: extraPath, data: []byte("stale\n"), mode: 0o644})
		}
		archive := filepath.Join(archives, "ssb_v9.9.9_"+target+extension)
		if extension == ".zip" {
			writeZip(t, archive, entries)
			continue
		}
		writeTarGzip(t, archive, entries)
	}
}

func expectedSkillFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	expected, err := readWorkingSkillFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	return expected
}

func readWorkingSkillFiles(root string) (map[string][]byte, error) {
	expected := make(map[string][]byte)
	skillRoot := filepath.Join(root, filepath.FromSlash(bundledSkillDirectory))
	err := filepath.WalkDir(skillRoot, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return readErr
		}
		relative, relativeErr := filepath.Rel(root, filePath)
		if relativeErr != nil {
			return relativeErr
		}
		expected[filepath.ToSlash(relative)] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read bundled Agent Skill: %w", err)
	}
	if len(expected) == 0 {
		return nil, fmt.Errorf("bundled Agent Skill has no regular files")
	}
	return expected, nil
}

func readExpectedSkillFiles(root, sourceRef string) (map[string][]byte, error) {
	if strings.TrimSpace(sourceRef) == "" {
		return nil, fmt.Errorf("release source ref is required")
	}
	commitBytes, err := gitOutput(root, "rev-parse", "--verify", sourceRef+"^{commit}")
	if err != nil {
		return nil, fmt.Errorf("resolve release source ref %s: %w", sourceRef, err)
	}
	commit := strings.TrimSpace(string(commitBytes))
	tree, err := gitOutput(root, "ls-tree", "-r", "-z", commit, "--", bundledSkillDirectory)
	if err != nil {
		return nil, fmt.Errorf("list bundled Agent Skill at source ref %s: %w", sourceRef, err)
	}

	expected := make(map[string][]byte)
	for _, rawRecord := range bytes.Split(tree, []byte{0}) {
		if len(rawRecord) == 0 {
			continue
		}
		parts := bytes.SplitN(rawRecord, []byte{'\t'}, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("parse bundled Agent Skill tree entry at source ref %s", sourceRef)
		}
		metadata := strings.Fields(string(parts[0]))
		if len(metadata) != 3 {
			return nil, fmt.Errorf("parse bundled Agent Skill tree metadata at source ref %s", sourceRef)
		}
		if metadata[1] != "blob" || (metadata[0] != "100644" && metadata[0] != "100755") {
			continue
		}
		name := filepath.ToSlash(string(parts[1]))
		data, readErr := gitOutput(root, "cat-file", "blob", commit+":"+name)
		if readErr != nil {
			return nil, fmt.Errorf("read %s at source ref %s: %w", name, sourceRef, readErr)
		}
		expected[name] = data
	}
	if len(expected) == 0 {
		return nil, fmt.Errorf("bundled Agent Skill has no regular files at source ref %s", sourceRef)
	}
	return expected, nil
}

func gitOutput(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitError.Stderr)))
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

func readArchiveFiles(filePath string) (map[string][]byte, error) {
	if strings.HasSuffix(filePath, ".tar.gz") {
		return readTarGzipFiles(filePath)
	}
	if strings.HasSuffix(filePath, ".zip") {
		return readZipFiles(filePath)
	}
	return nil, fmt.Errorf("unsupported archive format: %s", filepath.Base(filePath))
}

func readTarGzipFiles(filePath string) (map[string][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	files := make(map[string][]byte)
	reader := tar.NewReader(gzipReader)
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nil, nextErr
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		name, normalizeErr := normalizedArchivePath(header.Name)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		data, readErr := io.ReadAll(reader)
		if readErr != nil {
			return nil, readErr
		}
		if _, duplicate := files[name]; duplicate {
			return nil, fmt.Errorf("duplicate archive path %s", name)
		}
		files[name] = data
	}
	return files, nil
}

func readZipFiles(filePath string) (map[string][]byte, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	files := make(map[string][]byte)
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name, normalizeErr := normalizedArchivePath(entry.Name)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		entryReader, openErr := entry.Open()
		if openErr != nil {
			return nil, openErr
		}
		data, readErr := io.ReadAll(entryReader)
		closeErr := entryReader.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if _, duplicate := files[name]; duplicate {
			return nil, fmt.Errorf("duplicate archive path %s", name)
		}
		files[name] = data
	}
	return files, nil
}

func normalizedArchivePath(name string) (string, error) {
	cleaned := archivepath.Clean(strings.TrimPrefix(filepath.ToSlash(name), "./"))
	if cleaned == "." || archivepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return cleaned, nil
}

func writeZip(t *testing.T, filePath string, entries []archiveEntry) {
	t.Helper()
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(os.FileMode(entry.mode))
		entryWriter, createErr := writer.CreateHeader(header)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entryWriter.Write(entry.data); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

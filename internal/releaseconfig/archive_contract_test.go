package releaseconfig_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
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
	} {
		t.Run(test.name, func(t *testing.T) {
			archives := t.TempDir()
			writeReleaseArchiveMatrix(t, root, archives, test.missingTarget, test.missingPath)

			err := verifyReleaseArchives(root, archives)
			if err == nil {
				t.Fatal("incomplete release archives passed verification")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("verification error = %q, want it to contain %q", err, test.want)
			}
		})
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

	if err := verifyReleaseArchives(root, archives); err != nil {
		t.Fatal(err)
	}
}

func verifyReleaseArchives(root, archives string) error {
	expected, err := readExpectedSkillFiles(root)
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
				return fmt.Errorf("release archive %s contains %s that does not match the release source", target, name)
			}
		}
	}
	return nil
}

func writeReleaseArchiveMatrix(t *testing.T, root, archives, missingTarget, missingPath string) {
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
	expected, err := readExpectedSkillFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	return expected
}

func readExpectedSkillFiles(root string) (map[string][]byte, error) {
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

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRoutingSnapshotRestoresExistingAndAbsentTrees(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		root := t.TempDir()
		const relativeTarget = ".software-standards/routing"
		target := filepath.Join(root, filepath.FromSlash(relativeTarget))
		if runtime.GOOS != "windows" {
			t.Cleanup(func() {
				_ = os.Chmod(target, 0o700)
				_ = os.Chmod(filepath.Join(target, "bundles"), 0o700)
			})
		}
		writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "before catalog\n")
		writeSnapshotTestFile(t, filepath.Join(target, "bundles", "route-one.md"), "before bundle\n")
		if runtime.GOOS != "windows" {
			if err := os.Chmod(target, 0o500); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Join(target, "bundles"), 0o500); err != nil {
				t.Fatal(err)
			}
		}
		snapshot, err := captureDirectory(root, relativeTarget)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" {
			if err := os.Chmod(target, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Join(target, "bundles"), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.RemoveAll(target); err != nil {
			t.Fatal(err)
		}
		writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "after\n")
		expectedCurrent, err := captureDirectory(root, relativeTarget)
		if err != nil {
			t.Fatal(err)
		}
		if err := restoreDirectoryIfCurrent(root, relativeTarget, snapshot, expectedCurrent); err != nil {
			t.Fatal(err)
		}
		assertSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "before catalog\n")
		assertSnapshotTestFile(t, filepath.Join(target, "bundles", "route-one.md"), "before bundle\n")
		if runtime.GOOS != "windows" {
			assertSnapshotTestMode(t, target, 0o500)
			assertSnapshotTestMode(t, filepath.Join(target, "bundles"), 0o500)
		}
	})

	t.Run("absent", func(t *testing.T) {
		root := t.TempDir()
		const relativeTarget = ".software-standards/routing"
		target := filepath.Join(root, filepath.FromSlash(relativeTarget))
		snapshot, err := captureDirectory(root, relativeTarget)
		if err != nil {
			t.Fatal(err)
		}
		writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "generated\n")
		expectedCurrent, err := captureDirectory(root, relativeTarget)
		if err != nil {
			t.Fatal(err)
		}
		if err := restoreDirectoryIfCurrent(root, relativeTarget, snapshot, expectedCurrent); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("absent snapshot left directory behind: %v", err)
		}
	})
}

func TestRoutingSnapshotRollbackPreservesConcurrentChange(t *testing.T) {
	root := t.TempDir()
	const relativeTarget = ".software-standards/routing"
	target := filepath.Join(root, filepath.FromSlash(relativeTarget))
	writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "before\n")
	before, err := captureDirectory(root, relativeTarget)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "rendered\n")
	expectedCurrent, err := captureDirectory(root, relativeTarget)
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "concurrent\n")

	if err := restoreDirectoryIfCurrent(root, relativeTarget, before, expectedCurrent); err == nil {
		t.Fatal("rollback replaced a routing tree that changed after render")
	}
	assertSnapshotTestFile(t, filepath.Join(target, "catalog.md"), "concurrent\n")
}

func TestCaptureRoutingSnapshotRejectsParentSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeSnapshotTestFile(t, filepath.Join(outside, "routing", "catalog.md"), "outside\n")
	if err := os.Symlink(outside, filepath.Join(root, ".software-standards")); err != nil {
		t.Fatal(err)
	}
	if _, err := captureDirectory(root, ".software-standards/routing"); err == nil {
		t.Fatal("capture followed a routing parent symlink outside the repository")
	}
	assertSnapshotTestFile(t, filepath.Join(outside, "routing", "catalog.md"), "outside\n")
}

func TestRoutingSnapshotRollbackRejectsParentSymlink(t *testing.T) {
	root := t.TempDir()
	standards := filepath.Join(root, ".software-standards")
	target := filepath.Join(standards, "routing", "catalog.md")
	writeSnapshotTestFile(t, target, "before\n")
	before, err := captureDirectory(root, ".software-standards/routing")
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotTestFile(t, target, "rendered\n")
	expectedCurrent, err := captureDirectory(root, ".software-standards/routing")
	if err != nil {
		t.Fatal(err)
	}
	parked := filepath.Join(root, ".software-standards-parked")
	if err := os.Rename(standards, parked); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	writeSnapshotTestFile(t, filepath.Join(outside, "routing", "catalog.md"), "outside\n")
	if err := os.Symlink(outside, standards); err != nil {
		t.Fatal(err)
	}

	if err := restoreDirectoryIfCurrent(root, ".software-standards/routing", before, expectedCurrent); err == nil {
		t.Fatal("rollback followed a routing parent symlink outside the repository")
	}
	assertSnapshotTestFile(t, filepath.Join(outside, "routing", "catalog.md"), "outside\n")
}

func TestCaptureRoutingSnapshotRejectsFileSwappedToOutsideSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.md")
	target := filepath.Join(root, ".software-standards", "routing", "catalog.md")
	writeSnapshotTestFile(t, target, "inside\n")
	writeSnapshotTestFile(t, outside, "outside\n")
	baseFileSystem, err := openSnapshotFileSystem(root)
	if err != nil {
		t.Fatal(err)
	}
	defer baseFileSystem.close()
	fileSystem := &beforeOpenSnapshotFileSystem{snapshotFileSystem: baseFileSystem}
	fileSystem.beforeOpen = func(name string) {
		if name != ".software-standards/routing/catalog.md" {
			return
		}
		fileSystem.beforeOpen = nil
		if err := os.Remove(target); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, target); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := captureDirectoryWithFS(fileSystem, ".software-standards/routing"); err == nil {
		t.Fatal("capture accepted a file swapped to an outside symlink")
	}
	assertSnapshotTestFile(t, outside, "outside\n")
}

type beforeOpenSnapshotFileSystem struct {
	snapshotFileSystem
	beforeOpen func(string)
}

func (fileSystem *beforeOpenSnapshotFileSystem) open(name string) (snapshotFile, error) {
	if fileSystem.beforeOpen != nil {
		fileSystem.beforeOpen(name)
	}
	return fileSystem.snapshotFileSystem.open(name)
}

func TestFileSnapshotRollbackPreservesConcurrentChange(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "AGENTS.md")
	writeSnapshotTestFile(t, target, "before\n")
	before, err := captureFile(root, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotTestFile(t, target, "rendered\n")
	expectedCurrent, err := captureFile(root, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotTestFile(t, target, "concurrent\n")

	if err := restoreFileIfCurrent(root, "AGENTS.md", before, expectedCurrent); err == nil {
		t.Fatal("rollback replaced AGENTS.md after it changed")
	}
	assertSnapshotTestFile(t, target, "concurrent\n")
}

func TestProjectionSnapshotRollbackPreflightsBothArtifacts(t *testing.T) {
	root := t.TempDir()
	agentsTarget := filepath.Join(root, "AGENTS.md")
	routingTarget := filepath.Join(root, ".software-standards", "routing", "catalog.md")
	writeSnapshotTestFile(t, agentsTarget, "before agents\n")
	writeSnapshotTestFile(t, routingTarget, "before routing\n")
	beforeAgents, err := captureFile(root, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	beforeRouting, err := captureDirectory(root, ".software-standards/routing")
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotTestFile(t, agentsTarget, "rendered agents\n")
	writeSnapshotTestFile(t, routingTarget, "rendered routing\n")
	expectedAgents, err := captureFile(root, "AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	expectedRouting, err := captureDirectory(root, ".software-standards/routing")
	if err != nil {
		t.Fatal(err)
	}
	writeSnapshotTestFile(t, routingTarget, "concurrent routing\n")

	err = restoreSnapshotsIfCurrent(
		root,
		&fileRollback{target: "AGENTS.md", before: beforeAgents, expectedCurrent: expectedAgents},
		&directoryRollback{
			target: ".software-standards/routing", before: beforeRouting, expectedCurrent: expectedRouting,
		},
	)
	if !errors.Is(err, errSnapshotDrift) {
		t.Fatalf("rollback error = %v, want projection drift", err)
	}
	assertSnapshotTestFile(t, agentsTarget, "rendered agents\n")
	assertSnapshotTestFile(t, routingTarget, "concurrent routing\n")
}

func TestCaptureSnapshotRejectsOversizedFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "AGENTS.md")
	writeSnapshotTestFile(t, target, "placeholder\n")
	if err := os.Truncate(target, maxSnapshotFileBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := captureFile(root, "AGENTS.md"); err == nil {
		t.Fatal("snapshot accepted a file larger than its configured limit")
	}
}

func TestCaptureSnapshotRejectsInPlaceMutation(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "AGENTS.md")
	writeSnapshotTestFile(t, target, "first\n")
	baseFileSystem, err := openSnapshotFileSystem(root)
	if err != nil {
		t.Fatal(err)
	}
	defer baseFileSystem.close()
	fileSystem := &mutateOnSeekSnapshotFileSystem{
		snapshotFileSystem: baseFileSystem,
		target:             "AGENTS.md",
		mutate: func() {
			if err := os.WriteFile(target, []byte("other\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
	}
	if _, err := captureFileWithFS(fileSystem, "AGENTS.md"); err == nil {
		t.Fatal("snapshot accepted bytes from an in-place mutation")
	}
}

type mutateOnSeekSnapshotFileSystem struct {
	snapshotFileSystem
	target string
	mutate func()
}

func (fileSystem *mutateOnSeekSnapshotFileSystem) open(name string) (snapshotFile, error) {
	file, err := fileSystem.snapshotFileSystem.open(name)
	if err != nil || name != fileSystem.target {
		return file, err
	}
	return &mutateOnSeekSnapshotFile{snapshotFile: file, mutate: fileSystem.mutate}, nil
}

type mutateOnSeekSnapshotFile struct {
	snapshotFile
	mutate  func()
	mutated bool
}

func (file *mutateOnSeekSnapshotFile) Seek(offset int64, whence int) (int64, error) {
	if !file.mutated {
		file.mutated = true
		file.mutate()
	}
	return file.snapshotFile.Seek(offset, whence)
}

func writeSnapshotTestFile(t *testing.T, target, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertSnapshotTestFile(t *testing.T, target, want string) {
	t.Helper()
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("%s = %q, want %q", target, content, want)
	}
}

func assertSnapshotTestMode(t *testing.T, target string, want os.FileMode) {
	t.Helper()
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", target, got, want)
	}
}

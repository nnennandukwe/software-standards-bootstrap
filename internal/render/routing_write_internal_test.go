package render

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishRoutingTargetRejectsConcurrentValidChange(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".software-standards"), 0o755); err != nil {
		t.Fatal(err)
	}
	desiredCatalog, err := wrapRoutingFile(struct{ Version string }{"desired"}, []byte("# Catalog\n"))
	if err != nil {
		t.Fatal(err)
	}
	desired := &RoutingResult{
		Path: RoutingDirectory, Exists: true,
		Files: []FileResult{{Path: ".software-standards/routing/catalog.md", Content: desiredCatalog, SHA256: digest(desiredCatalog), Bytes: len(desiredCatalog)}},
	}
	target := filepath.Join(root, filepath.FromSlash(RoutingDirectory))
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "catalog.md"), desiredCatalog, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := inspectRoutingTarget(root, desired); err != nil {
		t.Fatal(err)
	}
	desired.Changed = true
	staged, err := stageRoutingTarget(root, desired)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(staged) })

	concurrentCatalog, err := wrapRoutingFile(struct{ Version string }{"concurrent"}, []byte("# Concurrent catalog\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "catalog.md"), concurrentCatalog, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := publishRoutingTarget(root, desired, staged); !errors.Is(err, ErrDrift) {
		t.Fatalf("publish error = %v, want concurrent drift rejection", err)
	}
	after, err := os.ReadFile(filepath.Join(target, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(concurrentCatalog) {
		t.Fatal("concurrent routing change was overwritten")
	}
}

func TestRoutingRelativePathRejectsRootedRelativeSuffix(t *testing.T) {
	if _, err := routingRelativePath(RoutingDirectory + "//outside.md"); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("rooted suffix error = %v, want unsafe target", err)
	}
}

func TestPublishProjectionRestoresRoutingWhenAgentsWriteFails(t *testing.T) {
	root := t.TempDir()
	standards := filepath.Join(root, ".software-standards")
	if err := os.MkdirAll(standards, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "AGENTS.md")
	originalAgents := []byte("# Original guidance\n")
	if err := os.WriteFile(target, originalAgents, 0o644); err != nil {
		t.Fatal(err)
	}
	originalCatalog, err := wrapRoutingFile(struct{ Version string }{"original"}, []byte("# Original catalog\n"))
	if err != nil {
		t.Fatal(err)
	}
	routingTarget := filepath.Join(root, filepath.FromSlash(RoutingDirectory))
	if err := os.MkdirAll(routingTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(routingTarget, "catalog.md"), originalCatalog, 0o644); err != nil {
		t.Fatal(err)
	}
	desiredCatalog, err := wrapRoutingFile(struct{ Version string }{"desired"}, []byte("# Desired catalog\n"))
	if err != nil {
		t.Fatal(err)
	}
	routing := &RoutingResult{
		Path: RoutingDirectory, Exists: true,
		Files: []FileResult{{
			Path: RoutingDirectory + "/catalog.md", Content: desiredCatalog,
			SHA256: digest(desiredCatalog), Bytes: len(desiredCatalog),
		}},
	}
	if err := inspectRoutingTarget(root, routing); err != nil {
		t.Fatal(err)
	}
	if !routing.Changed {
		t.Fatal("test setup did not create a changed routing tree")
	}

	injected := errors.New("injected AGENTS.md write failure")
	err = publishProjection(
		root, routing, target, originalAgents, []byte("# Desired guidance\n"), 0o644, true, true,
		func(_ string, _, _ []byte, _ os.FileMode, _ bool) error {
			published, readErr := os.ReadFile(filepath.Join(routingTarget, "catalog.md"))
			if readErr != nil {
				t.Fatalf("read published routing tree: %v", readErr)
			}
			if string(published) != string(desiredCatalog) {
				t.Fatal("failure was not injected after routing publication")
			}
			return injected
		},
	)
	if !errors.Is(err, injected) {
		t.Fatalf("publish error = %v, want injected AGENTS.md failure", err)
	}
	afterAgents, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	afterCatalog, err := os.ReadFile(filepath.Join(routingTarget, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(afterAgents) != string(originalAgents) || string(afterCatalog) != string(originalCatalog) {
		t.Fatal("failed projection left partial root or routing output")
	}
}

func TestPublishRoutingTargetCopiesBackupWhenRenameRestoreFails(t *testing.T) {
	root := t.TempDir()
	standards := filepath.Join(root, ".software-standards")
	if err := os.MkdirAll(standards, 0o755); err != nil {
		t.Fatal(err)
	}
	originalCatalog, err := wrapRoutingFile(struct{ Version string }{"original"}, []byte("# Original catalog\n"))
	if err != nil {
		t.Fatal(err)
	}
	routingTarget := filepath.Join(root, filepath.FromSlash(RoutingDirectory))
	if err := os.MkdirAll(routingTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(routingTarget, "catalog.md"), originalCatalog, 0o644); err != nil {
		t.Fatal(err)
	}
	desiredCatalog, err := wrapRoutingFile(struct{ Version string }{"desired"}, []byte("# Desired catalog\n"))
	if err != nil {
		t.Fatal(err)
	}
	routing := &RoutingResult{
		Path: RoutingDirectory, Exists: true,
		Files: []FileResult{{
			Path: RoutingDirectory + "/catalog.md", Content: desiredCatalog,
			SHA256: digest(desiredCatalog), Bytes: len(desiredCatalog),
		}},
	}
	if err := inspectRoutingTarget(root, routing); err != nil {
		t.Fatal(err)
	}
	staged, err := stageRoutingTarget(root, routing)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(staged) })

	publishFailure := errors.New("injected staged-tree publish failure")
	restoreFailure := errors.New("injected backup rename failure")
	renameCalls := 0
	rename := func(oldPath, newPath string) error {
		renameCalls++
		switch renameCalls {
		case 1:
			return os.Rename(oldPath, newPath)
		case 2:
			return publishFailure
		case 3:
			return restoreFailure
		default:
			t.Fatalf("unexpected rename call %d: %s -> %s", renameCalls, oldPath, newPath)
			return nil
		}
	}
	if _, err := publishRoutingTargetWithRename(root, routing, staged, rename); !errors.Is(err, publishFailure) {
		t.Fatalf("publish error = %v, want staged-tree failure", err)
	}
	if renameCalls != 3 {
		t.Fatalf("rename calls = %d, want publish plus failed rename restoration", renameCalls)
	}
	after, err := os.ReadFile(filepath.Join(routingTarget, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(originalCatalog) {
		t.Fatal("copy fallback did not restore the original routing tree")
	}
	backups, err := filepath.Glob(filepath.Join(standards, ".ssb-routing-backup-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Fatalf("successful copy fallback left routing backups: %v", backups)
	}
}

func TestPublishProjectionPreservesConcurrentRoutingChangeDuringRollback(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".software-standards"), 0o755); err != nil {
		t.Fatal(err)
	}
	agentsTarget := filepath.Join(root, "AGENTS.md")
	originalAgents := []byte("# Original guidance\n")
	if err := os.WriteFile(agentsTarget, originalAgents, 0o644); err != nil {
		t.Fatal(err)
	}
	originalCatalog := testRoutingCatalog(t, "original")
	desiredCatalog := testRoutingCatalog(t, "desired")
	concurrentCatalog := testRoutingCatalog(t, "concurrent")
	routingTarget := filepath.Join(root, filepath.FromSlash(RoutingDirectory))
	if err := os.MkdirAll(routingTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(routingTarget, "catalog.md"), originalCatalog, 0o644); err != nil {
		t.Fatal(err)
	}
	routing := &RoutingResult{
		Path: RoutingDirectory, Exists: true,
		TreeDigest: routingTreeDigest([]FileResult{{Path: RoutingDirectory + "/catalog.md", SHA256: digest(desiredCatalog)}}),
		Files: []FileResult{{
			Path: RoutingDirectory + "/catalog.md", Content: desiredCatalog,
			SHA256: digest(desiredCatalog), Bytes: len(desiredCatalog),
		}},
	}
	if err := inspectRoutingTarget(root, routing); err != nil {
		t.Fatal(err)
	}

	injected := errors.New("injected AGENTS.md write failure")
	err := publishProjection(
		root, routing, agentsTarget, originalAgents, []byte("# Desired guidance\n"), 0o644, true, true,
		func(_ string, _, _ []byte, _ os.FileMode, _ bool) error {
			if err := os.WriteFile(filepath.Join(routingTarget, "catalog.md"), concurrentCatalog, 0o644); err != nil {
				t.Fatal(err)
			}
			return injected
		},
	)
	if !errors.Is(err, injected) || !strings.Contains(err.Error(), "routing directory changed") {
		t.Fatalf("publish error = %v, want write failure plus rollback drift", err)
	}
	after, err := os.ReadFile(filepath.Join(routingTarget, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(concurrentCatalog) {
		t.Fatal("rollback deleted the concurrent routing change")
	}
	backups, err := filepath.Glob(filepath.Join(root, ".software-standards", ".ssb-routing-backup-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("routing backups = %v, want original tree retained for recovery", backups)
	}
}

func TestPublishProjectionPreservesRoutingTreeThatAppearsDuringRemovalRollback(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".software-standards"), 0o755); err != nil {
		t.Fatal(err)
	}
	agentsTarget := filepath.Join(root, "AGENTS.md")
	originalAgents := []byte("# Original guidance\n")
	if err := os.WriteFile(agentsTarget, originalAgents, 0o644); err != nil {
		t.Fatal(err)
	}
	originalCatalog := testRoutingCatalog(t, "original")
	concurrentCatalog := testRoutingCatalog(t, "concurrent")
	routingTarget := filepath.Join(root, filepath.FromSlash(RoutingDirectory))
	if err := os.MkdirAll(routingTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(routingTarget, "catalog.md"), originalCatalog, 0o644); err != nil {
		t.Fatal(err)
	}
	routing := &RoutingResult{
		Path: RoutingDirectory, Exists: false,
		TreeDigest: routingTreeDigest(nil), Files: make([]FileResult, 0),
	}
	if err := inspectRoutingTarget(root, routing); err != nil {
		t.Fatal(err)
	}

	injected := errors.New("injected AGENTS.md write failure")
	err := publishProjection(
		root, routing, agentsTarget, originalAgents, []byte("# Desired guidance\n"), 0o644, true, true,
		func(_ string, _, _ []byte, _ os.FileMode, _ bool) error {
			if err := os.MkdirAll(routingTarget, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(routingTarget, "catalog.md"), concurrentCatalog, 0o644); err != nil {
				t.Fatal(err)
			}
			return injected
		},
	)
	if !errors.Is(err, injected) || !errors.Is(err, ErrDrift) {
		t.Fatalf("publish error = %v, want write failure plus rollback drift", err)
	}
	after, err := os.ReadFile(filepath.Join(routingTarget, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(concurrentCatalog) {
		t.Fatal("rollback deleted the routing tree that appeared after removal")
	}
	backups, err := filepath.Glob(filepath.Join(root, ".software-standards", ".ssb-routing-backup-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("routing backups = %v, want original tree retained for recovery", backups)
	}
}

func TestPublishRoutingTargetRechecksTreeMovedToBackup(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".software-standards"), 0o755); err != nil {
		t.Fatal(err)
	}
	originalCatalog := testRoutingCatalog(t, "original")
	desiredCatalog := testRoutingCatalog(t, "desired")
	concurrentCatalog := testRoutingCatalog(t, "concurrent")
	routingTarget := filepath.Join(root, filepath.FromSlash(RoutingDirectory))
	if err := os.MkdirAll(routingTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(routingTarget, "catalog.md"), originalCatalog, 0o644); err != nil {
		t.Fatal(err)
	}
	routing := &RoutingResult{
		Path: RoutingDirectory, Exists: true,
		Files: []FileResult{{
			Path: RoutingDirectory + "/catalog.md", Content: desiredCatalog,
			SHA256: digest(desiredCatalog), Bytes: len(desiredCatalog),
		}},
	}
	if err := inspectRoutingTarget(root, routing); err != nil {
		t.Fatal(err)
	}
	staged, err := stageRoutingTarget(root, routing)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(staged) })

	renameCalls := 0
	rename := func(oldPath, newPath string) error {
		renameCalls++
		if renameCalls == 1 {
			if err := os.WriteFile(filepath.Join(routingTarget, "catalog.md"), concurrentCatalog, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return os.Rename(oldPath, newPath)
	}
	if _, err := publishRoutingTargetWithRename(root, routing, staged, rename); !errors.Is(err, ErrDrift) {
		t.Fatalf("publish error = %v, want moved-prestate drift rejection", err)
	}
	after, err := os.ReadFile(filepath.Join(routingTarget, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(concurrentCatalog) {
		t.Fatal("publish overwrote the concurrent prestate")
	}
}

func TestStageRoutingTargetRejectsStandardsParentSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".software-standards")); err != nil {
		t.Fatal(err)
	}
	catalog := testRoutingCatalog(t, "desired")
	routing := &RoutingResult{
		Path: RoutingDirectory, Exists: true, Changed: true,
		Files: []FileResult{{
			Path: RoutingDirectory + "/catalog.md", Content: catalog,
			SHA256: digest(catalog), Bytes: len(catalog),
		}},
	}
	if _, err := stageRoutingTarget(root, routing); err == nil {
		t.Fatal("staging followed .software-standards symlink outside the repository")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unsafe staging wrote outside the repository: %v", entries)
	}
}

func testRoutingCatalog(t *testing.T, version string) []byte {
	t.Helper()
	catalog, err := wrapRoutingFile(struct{ Version string }{version}, []byte("# "+version+" catalog\n"))
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

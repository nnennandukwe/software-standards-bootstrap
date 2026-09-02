package render

import (
	"errors"
	"os"
	"path/filepath"
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

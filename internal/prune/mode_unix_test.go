//go:build !windows

package prune

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestWriteNewExclusiveMaterializesExactModeDespiteUmask(t *testing.T) {
	previous := syscall.Umask(0o077)
	t.Cleanup(func() { syscall.Umask(previous) })

	target := filepath.Join(t.TempDir(), "candidate")
	if err := writeNewExclusive(target, []byte("candidate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("mode = %04o, want 0644", got)
	}
}

func TestWriteNewExclusiveRequiresDurableDirectoryPublication(t *testing.T) {
	original := syncApplicationDirectory
	syncApplicationDirectory = func(string) error {
		return errors.New("injected directory sync failure")
	}
	t.Cleanup(func() { syncApplicationDirectory = original })

	target := filepath.Join(t.TempDir(), "candidate")
	err := writeNewExclusive(target, []byte("candidate\n"), 0o644)
	if err == nil || !strings.Contains(err.Error(), "injected directory sync failure") {
		t.Fatalf("error = %v, want directory durability failure", err)
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil || string(got) != "candidate\n" {
		t.Fatalf("published bytes = %q, %v, want complete target", got, readErr)
	}
}

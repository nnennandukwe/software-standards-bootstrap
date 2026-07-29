//go:build !windows

package prune

import (
	"os"
	"path/filepath"
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

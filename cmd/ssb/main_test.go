package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestBuiltVersionForms(t *testing.T) {
	const version = "v0.2.1-test"
	const commit = "1234567890abcdef1234567890abcdef12345678"
	binary := filepath.Join(t.TempDir(), "ssb"+executableSuffix())
	linker := "-X github.com/nnennandukwe/software-standards-bootstrap/internal/buildinfo.Version=" + version +
		" -X github.com/nnennandukwe/software-standards-bootstrap/internal/buildinfo.Commit=" + commit
	build := exec.CommandContext(t.Context(), "go", "build", "-ldflags", linker, "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	for _, command := range []string{"version", "--version"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			process := exec.CommandContext(t.Context(), binary, command)
			process.Stdout, process.Stderr = &stdout, &stderr
			if err := process.Run(); err != nil || stdout.String() != "ssb "+version+" (commit "+commit+")\n" || stderr.Len() != 0 {
				t.Fatalf("run: %v stdout=%q stderr=%q", err, &stdout, &stderr)
			}
		})
	}
}

func TestInterruptDuringGit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Process.Signal can't send os.Interrupt on Windows; context cancellation is tested separately")
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	toolsDir := t.TempDir()
	binary := filepath.Join(toolsDir, "ssb")
	for _, target := range []struct{ output, source string }{
		{output: binary, source: "."},
		{output: filepath.Join(toolsDir, "git"), source: "../../testdata/git-helper"},
	} {
		build := exec.CommandContext(t.Context(), "go", "build", "-o", target.output, target.source)
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build: %v\n%s", err, output)
		}
	}
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-b", "main"}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "baseline"}} {
		command := exec.CommandContext(t.Context(), realGit, append([]string{"-C", root}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("fixture Git: %v\n%s", err, output)
		}
	}
	for _, match := range []string{"--version", "rev-parse --show-toplevel", "symbolic-ref", "HEAD^{commit}", "status", "ls-tree"} {
		t.Run(match, func(t *testing.T) {
			ready := filepath.Join(t.TempDir(), "ready")
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			process := exec.CommandContext(ctx, binary, "inspect", "--repo", root, "--format", "json")
			process.Env = append(os.Environ(),
				"PATH="+toolsDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"SSB_REAL_GIT="+realGit, "SSB_GIT_BLOCK_MATCH="+match, "SSB_GIT_READY="+ready,
			)
			var stdout, stderr bytes.Buffer
			process.Stdout, process.Stderr = &stdout, &stderr
			if err := process.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = process.Process.Kill() })
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				if _, err := os.Stat(ready); err == nil {
					break
				}
				select {
				case <-ctx.Done():
					_ = process.Wait()
					t.Fatal("Git readiness handshake timed out")
				case <-ticker.C:
				}
			}
			if err := process.Process.Signal(os.Interrupt); err != nil {
				t.Fatal(err)
			}
			var failure *exec.ExitError
			err := process.Wait()
			if !errors.As(err, &failure) || failure.ExitCode() != 130 || stdout.Len() != 0 || stderr.String() != "error: interrupted\n" {
				t.Fatalf("run: %v stdout=%q stderr=%q", err, &stdout, &stderr)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 || entries[0].Name() != ".git" {
				t.Fatalf("interrupted inspection changed the fixture: %v %v", entries, err)
			}
		})
	}
}

func executableSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

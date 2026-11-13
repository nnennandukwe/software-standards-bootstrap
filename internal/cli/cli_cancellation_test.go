package cli_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/cli"
)

func TestVersionForms(t *testing.T) {
	for _, command := range []string{"version", "--version"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli.Run([]string{command}, &stdout, &stderr)
			if code != 0 || stdout.String() != "ssb dev (commit unknown)\n" || stderr.Len() != 0 {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
		})
	}
}

func TestVersionHelpAndArguments(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{name: "help", args: []string{"version", "--help"}, code: 0},
		{name: "short help", args: []string{"version", "-h"}, code: 0},
		{name: "extra version argument", args: []string{"version", "extra"}, code: 2},
		{name: "extra flag argument", args: []string{"--version", "extra"}, code: 2},
		{name: "flag help argument", args: []string{"--version", "--help"}, code: 2},
		{name: "extra help argument", args: []string{"version", "--help", "extra"}, code: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := cli.Run(test.args, &stdout, &stderr); code != test.code {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
			if test.code == 0 && (stdout.Len() == 0 || stderr.Len() != 0) {
				t.Fatalf("help streams: stdout=%q stderr=%q", &stdout, &stderr)
			}
			if test.code == 2 && (stdout.Len() != 0 || stderr.Len() == 0) {
				t.Fatalf("usage streams: stdout=%q stderr=%q", &stdout, &stderr)
			}
		})
	}
}

func TestRunContextPreCanceled(t *testing.T) {
	for _, args := range [][]string{
		nil, {"--help"}, {"version"}, {"--version"}, {"unknown"},
		{"inspect"}, {"validate"}, {"render"}, {"adr"},
		{"prune", "inspect"}, {"prune", "validate"}, {"prune", "approve"},
		{"prune", "apply", "--write"}, {"prune", "recover"},
		{"prune", "status"}, {"prune", "verify"},
	} {
		t.Run(commandName(args), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			var stdout, stderr bytes.Buffer
			code := cli.RunContext(ctx, args, &stdout, &stderr)
			if code != 130 || stdout.Len() != 0 || stderr.String() != "error: interrupted\n" {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
		})
	}
}

func commandName(args []string) string {
	if len(args) == 0 {
		return "empty"
	}
	name := args[0]
	if len(args) > 1 {
		name += " " + args[1]
	}
	return name
}

func TestRunContextCancellationAfterPublicationKeepsInterruptionContract(t *testing.T) {
	for _, test := range []struct {
		name   string
		target string
	}{
		{name: "render", target: "AGENTS.md"},
		{name: "adr", target: "docs/adr/0001-actionable-standards.md"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, baseline := evidenceRepository(t)
			writeValidPack(t, root, baseline)
			parent, cancel := context.WithCancel(t.Context())
			defer cancel()
			ctx := cancelAfterPublicationContext{
				Context: parent, cancel: cancel, target: filepath.Join(root, filepath.FromSlash(test.target)),
			}
			var stdout, stderr bytes.Buffer
			code := cli.RunContext(&ctx, []string{test.name, "--repo", root}, &stdout, &stderr)
			if code != 130 || stdout.Len() != 0 || stderr.String() != "error: interrupted\n" {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
			if data, err := os.ReadFile(ctx.target); err != nil || len(data) == 0 {
				t.Fatalf("bounded publication didn't finish: bytes=%d, %v", len(data), err)
			}
		})
	}
}

// Observing the published target triggers cancellation at the CLI's final check.
// Preparation still uses a live caller context through the real write path.
type cancelAfterPublicationContext struct {
	context.Context
	cancel context.CancelFunc
	target string
}

func (ctx *cancelAfterPublicationContext) Err() error {
	if _, err := os.Stat(ctx.target); err == nil {
		ctx.cancel()
	}
	return ctx.Context.Err()
}

func TestRunContextCancellationDuringDeliveryReturnsCommandResult(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{name: "stdout", args: []string{"version"}, code: 0},
		{name: "stderr", args: []string{"unknown"}, code: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			var wantOutput, wantDiagnostics bytes.Buffer
			if code := cli.Run(test.args, &wantOutput, &wantDiagnostics); code != test.code {
				t.Fatalf("normal exit = %d, want %d", code, test.code)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var output, diagnostics bytes.Buffer
			var stdout io.Writer = &output
			var stderr io.Writer = &diagnostics
			if test.name == "stdout" {
				stdout = cancelDuringDeliveryWriter{Writer: stdout, cancel: cancel}
			} else {
				stderr = cancelDuringDeliveryWriter{Writer: stderr, cancel: cancel}
			}
			code := cli.RunContext(ctx, test.args, stdout, stderr)
			if ctx.Err() != context.Canceled || code != test.code ||
				output.String() != wantOutput.String() || diagnostics.String() != wantDiagnostics.String() {
				t.Fatalf("cancellation=%v exit=%d stdout=%q stderr=%q", ctx.Err(), code, &output, &diagnostics)
			}
		})
	}
}

type cancelDuringDeliveryWriter struct {
	io.Writer
	cancel context.CancelFunc
}

func (writer cancelDuringDeliveryWriter) Write(data []byte) (int, error) {
	writer.cancel()
	return writer.Writer.Write(data)
}

func TestCancellationPreservesRecoveryFailure(t *testing.T) {
	root, _ := evidenceRepository(t)
	lock := filepath.Join(root, ".software-standards", "reviews", "review-one", ".transition.lock")
	obstruction := filepath.Join(lock, "retained-file")
	writeFile(t, obstruction, "keep this recovery evidence\n")
	args := []string{"prune", "recover", "--repo", root, "--review", "review-one", "--clear-stale-lock"}
	var normalOutput, normalDiagnostics bytes.Buffer
	wantCode := cli.Run(args, &normalOutput, &normalDiagnostics)
	if wantCode != 3 || !strings.Contains(normalDiagnostics.String(), "clear stale review transition lock") {
		t.Fatalf("normal recovery failure: exit=%d stdout=%q stderr=%q", wantCode, &normalOutput, &normalDiagnostics)
	}
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := cancelAtRecoveryBoundaryContext{Context: parent, cancel: cancel}
	var output, diagnostics bytes.Buffer
	code := cli.RunContext(&ctx, args, &output, &diagnostics)
	if !ctx.triggered || parent.Err() != context.Canceled {
		t.Fatal("recovery didn't reach its bounded cleanup after the caller was canceled")
	}
	if code != wantCode || output.Len() != 0 || diagnostics.String() != normalDiagnostics.String() {
		t.Fatalf("canceled recovery failure: exit=%d stdout=%q stderr=%q; want exit=%d stderr=%q", code, &output, &diagnostics, wantCode, &normalDiagnostics)
	}
	if data, err := os.ReadFile(obstruction); err != nil || string(data) != "keep this recovery evidence\n" {
		t.Fatalf("recovery evidence changed: %q, %v", data, err)
	}
	if err := os.Remove(obstruction); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	diagnostics.Reset()
	if code := cli.Run(args, &output, &diagnostics); code != 0 {
		t.Fatalf("recovery retry: exit=%d stdout=%q stderr=%q", code, &output, &diagnostics)
	}
	if _, err := os.Lstat(lock); !os.IsNotExist(err) {
		t.Fatalf("recovery retry left its lock: %v", err)
	}
}

// Simulate an interrupt immediately after Recover accepts its last cancellable
// check. Its real filesystem cleanup then fails inside the bounded sequence.
type cancelAtRecoveryBoundaryContext struct {
	context.Context
	cancel    context.CancelFunc
	triggered bool
}

func (ctx *cancelAtRecoveryBoundaryContext) Err() error {
	err := ctx.Context.Err()
	pc, _, _, ok := runtime.Caller(1)
	if ok && strings.HasSuffix(runtime.FuncForPC(pc).Name(), "/internal/prune.Recover") {
		ctx.triggered = true
		ctx.cancel()
	}
	return err
}

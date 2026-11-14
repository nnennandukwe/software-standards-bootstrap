// Package cli owns ssb's public command and exit-code contract.
package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

// Run executes one command and returns the documented process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunContext(
		context.Background(),
		args,
		stdout,
		stderr,
	)
}

// RunContext executes a command using the caller's cancellation lifetime.
// It holds output until execution completes so cancellation before delivery
// never exposes partial results. Once final delivery starts, finish it and return
// the command's result. An independent failure retains its result and recovery
// diagnostics despite caller cancellation.
func RunContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if ctx.Err() != nil {
		fmt.Fprintln(stderr, "error: interrupted")
		return 130
	}
	var output, diagnostics bytes.Buffer
	code := dispatch(
		ctx,
		args,
		&output,
		&diagnostics,
	)
	canceled := ctx.Err() != nil
	if code == 130 || (canceled && code == 0) {
		fmt.Fprintln(stderr, "error: interrupted")
		return 130
	}
	if _, err := io.Copy(stdout, &output); err != nil {
		fmt.Fprintf(stderr, "error: write command result: %s\n", err)
		return 3
	}
	if _, err := io.Copy(stderr, &diagnostics); err != nil {
		return 3
	}
	return code
}

func commandErrorExitCode(err error, fallback int) int {
	if isOnlyCancellation(err) {
		return 130
	}
	return fallback
}

// A cleanup failure joined with cancellation must keep its failure diagnostic.
func isOnlyCancellation(err error) bool {
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, cause := range joined.Unwrap() {
			// Git adds this classification marker alongside the cancel cause.
			if cause == nil || cause == workspace.ErrGitOperation {
				continue
			}
			if !isOnlyCancellation(cause) {
				return false
			}
		}
		return true
	}
	if cause := errors.Unwrap(err); cause != nil {
		return isOnlyCancellation(cause)
	}
	return true
}

func dispatch(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, helpText)
		return 2
	}
	switch args[0] {
	case "-h", "--help", "help":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "error: help does not accept arguments")
			return 2
		}
		fmt.Fprint(stdout, helpText)
		return 0
	case "version", "--version":
		return runVersion(args, stdout, stderr)
	case "inspect":
		return runInspect(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "resume":
		return runResume(ctx, args[1:], stdout, stderr)
	case "validate":
		return runValidate(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "render":
		return runRender(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "adr":
		return runADR(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "prune":
		return runPrune(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	default:
		fmt.Fprintf(
			stderr,
			"error: unknown command %q\n\n%s",
			args[0],
			helpText,
		)
		return 2
	}
}

func cleanFlagError(err error) string {
	message := strings.TrimPrefix(err.Error(), "flag provided but not defined: ")
	if strings.HasPrefix(message, "-") && !strings.HasPrefix(message, "--") {
		message = "-" + message
	}
	return strings.Replace(
		message,
		"for flag -",
		"for flag --",
		1,
	)
}

const versionHelp = `Usage: ssb version
       ssb --version

Print the build version and full source commit.
`

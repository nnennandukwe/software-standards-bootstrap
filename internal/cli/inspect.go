package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func runInspect(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	defaults := inventory.DefaultLimits()
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	format := flags.String("format", "text", "output format")
	maxCandidateFiles := flags.Int(
		"max-candidate-files",
		defaults.MaxCandidateFiles,
		"candidate files that may be read",
	)
	maxCandidateBytes := flags.Int64(
		"max-candidate-bytes",
		defaults.MaxCandidateBytes,
		"candidate bytes that may be read",
	)
	allowPartial := flags.Bool(
		"allow-partial",
		false,
		"return success for explicitly diagnostic partial coverage",
	)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, inspectHelp)
			return 0
		}
		fmt.Fprintf(stderr, "error: %s\nnext: ssb inspect --help\n", cleanFlagError(err))
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "error: inspect does not accept positional arguments")
		fmt.Fprintln(stderr, "next: ssb inspect --help")
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(stderr, "error: --format must be text or json")
		fmt.Fprintln(stderr, "next: ssb inspect --repo PATH --format text")
		return 2
	}
	if *maxCandidateFiles <= 0 {
		fmt.Fprintln(stderr, "error: --max-candidate-files must be greater than zero")
		fmt.Fprintln(stderr, "next: pass --max-candidate-files N with a positive integer")
		return 2
	}
	if *maxCandidateBytes <= 0 {
		fmt.Fprintln(stderr, "error: --max-candidate-bytes must be greater than zero")
		fmt.Fprintln(stderr, "next: pass --max-candidate-bytes N with a positive integer")
		return 2
	}

	repo, err := workspace.OpenForInspect(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb inspect --repo <clean-repository-path>\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			return 2
		}
		return commandErrorExitCode(err, 3)
	}
	report, err := inventory.Scan(ctx, repo, inventory.Limits{
		MaxCandidateFiles: *maxCandidateFiles,
		MaxCandidateBytes: *maxCandidateBytes,
		MaxFileBytes:      defaults.MaxFileBytes,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: inventory failed: %s\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			fmt.Fprintln(stderr, "next: stabilize the repository and rerun ssb inspect --repo PATH.")
			return 2
		}
		return commandErrorExitCode(err, 3)
	}

	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "error: write JSON inventory: %s\n", err)
			return 3
		}
	} else {
		writeTextInventory(stdout, report)
	}
	if report.Truncated && !*allowPartial {
		fmt.Fprintf(stderr, "error: inventory coverage incomplete: %s\n", report.TruncationReason)
		fmt.Fprintln(
			stderr,
			"next: raise --max-candidate-files or --max-candidate-bytes and rerun; use --allow-partial only for diagnostic inventory.",
		)
		return 4
	}
	return 0
}

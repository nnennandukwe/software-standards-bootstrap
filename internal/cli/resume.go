package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func runResume(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("resume", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, resumeHelp)
			return 0
		}
		fmt.Fprintf(stderr, "error: %s\nnext: ssb resume --help\n", cleanFlagError(err))
		return 2
	}
	if flags.NArg() != 0 || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "error: resume accepts no positional arguments; --format must be text or json\nnext: ssb resume --help")
		return 2
	}
	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		return writeResumeError(stderr, err)
	}
	result, pack, err := rulepack.AssessContinuation(ctx, repo)
	if err != nil {
		return writeResumeError(stderr, err)
	}
	if result.ProposalReusable {
		projection, err := render.Apply(ctx, repo, pack, true)
		if err != nil {
			if !errors.Is(err, render.ErrDrift) && !errors.Is(err, render.ErrMarkers) && !errors.Is(err, render.ErrUnsafeTarget) {
				return writeResumeError(stderr, err)
			}
			result.State = "blocked"
			result.ProposalReusable = false
			result.Repeat = []string{"projection_recovery", "validation", "rendering", "developer_review"}
			result.Guidance = []string{"Projection checks blocked continuation. Preserve canonical sources and manual edits; reconcile the reported derived files through explicit review, then rerun ssb resume. Inventory verification remains reusable."}
			result.Diagnostics = append(result.Diagnostics, rulepack.Diagnostic{Path: "AGENTS.md", Field: "projection", Message: err.Error(), Recovery: "preserve manual edits; restore derived surfaces through explicit review, then rerun ssb resume"})
		} else if !projection.Changed {
			result.State = "complete"
			result.Repeat = []string{"developer_review"}
			result.Guidance = []string{"The proposal validates and its projection is current. Review the complete uncommitted result; generation completion isn't adoption."}
		}
	}
	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return writeResumeError(stderr, err)
		}
	} else {
		fmt.Fprintf(stdout, "Generation continuation: %s; baseline %s\nInventory reusable: %t; proposal reusable: %t\n", result.State, result.BaselineCommit, result.InventoryReusable, result.ProposalReusable)
		for _, reference := range result.Files {
			fmt.Fprintf(stdout, "Retained: %s %s\n", reference.Path, reference.SHA256)
		}
		for _, stage := range result.Repeat {
			fmt.Fprintf(stdout, "Repeat: %s\n", stage)
		}
		for _, guidance := range result.Guidance {
			fmt.Fprintln(stdout, guidance)
		}
		if len(result.Diagnostics) != 0 {
			writeValidationDiagnostics(stderr, result.Diagnostics)
		}
	}
	if len(result.Diagnostics) != 0 || result.State == "blocked" || result.State == "needs_correction" || result.State == "existing_pack" {
		return 1
	}
	return 0
}

func writeResumeError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: continuation assessment failed: %s\nnext: preserve the pack and rerun ssb resume --repo PATH after resolving the reported problem.\n", err)
	if errors.Is(err, workspace.ErrPrecondition) {
		return 2
	}
	return commandErrorExitCode(err, 3)
}

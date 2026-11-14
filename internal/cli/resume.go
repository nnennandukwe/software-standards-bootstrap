package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

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
		fmt.Fprintf(stderr, "error: %s\nnext: ssb resume --help\n", escapeResumeText(cleanFlagError(err)))
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
	var projection *render.Result
	var projectionErr error
	if result.ProposalReusable {
		value, applyErr := render.Apply(ctx, repo, pack, true)
		projection, projectionErr = &value, applyErr
		if projectionErr != nil && !errors.Is(projectionErr, render.ErrDrift) && !errors.Is(projectionErr, render.ErrMarkers) && !errors.Is(projectionErr, render.ErrUnsafeTarget) {
			return writeResumeError(stderr, projectionErr)
		}
	}
	if result.InventoryReusable {
		result, err = rulepack.RecheckContinuation(ctx, repo, result, pack)
		if err != nil {
			return writeResumeError(stderr, err)
		}
	}
	if projection != nil {
		if projectionErr == nil {
			if err := projection.VerifyUnchanged(ctx, repo); err != nil {
				return writeResumeError(stderr, err)
			}
		}
		if projectionErr != nil {
			result.State = "blocked"
			result.ProposalReusable = false
			result.Repeat = []string{"projection_recovery", "validation", "rendering", "developer_review"}
			result.Guidance[0] = "Projection checks blocked continuation. Preserve canonical sources and manual edits; reconcile the reported derived files through explicit review, then rerun ssb resume. Inventory verification remains reusable."
			result.Diagnostics = append(result.Diagnostics, rulepack.Diagnostic{Path: ".", Field: "projection", Message: projectionErr.Error(), Recovery: "preserve manual edits; review AGENTS.md and .software-standards/routing, reconcile the reported projection through explicit review, then rerun ssb resume"})
		} else if !projection.Changed {
			result.State = "complete"
			result.Repeat = []string{"developer_review"}
			result.Guidance[0] = "The proposal validates and its projection is current. Review the complete uncommitted result; generation completion isn't adoption."
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
			fmt.Fprintf(stdout, "Retained: %s %s\n", escapeResumeText(reference.Path), reference.SHA256)
		}
		for _, stage := range result.Repeat {
			fmt.Fprintf(stdout, "Repeat: %s\n", stage)
		}
		for _, guidance := range result.Guidance {
			fmt.Fprintln(stdout, escapeResumeText(guidance))
		}
		if len(result.Diagnostics) != 0 {
			diagnostics := append([]rulepack.Diagnostic(nil), result.Diagnostics...)
			for index := range diagnostics {
				item := &diagnostics[index]
				item.Path = escapeResumeText(item.Path)
				item.Field = escapeResumeText(item.Field)
				item.Message = escapeResumeText(item.Message)
				item.Recovery = escapeResumeText(item.Recovery)
			}
			writeDiagnostics(stderr, diagnostics, "Generation continuation blocked",
				"Next: preserve retained work, follow the reported recovery steps, and rerun ssb resume --repo PATH.")
		}
	}
	if len(result.Diagnostics) != 0 || result.State == "blocked" || result.State == "needs_correction" || result.State == "existing_pack" {
		return 1
	}
	return 0
}

func writeResumeError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: continuation assessment failed: %s\nnext: preserve the pack and rerun ssb resume --repo PATH after resolving the reported problem.\n", escapeResumeText(err.Error()))
	if errors.Is(err, workspace.ErrPrecondition) {
		return 2
	}
	return commandErrorExitCode(err, 3)
}

func escapeResumeText(value string) string {
	var output strings.Builder
	for _, character := range value {
		if unicode.IsPrint(character) {
			output.WriteRune(character)
			continue
		}
		quoted := strconv.QuoteRune(character)
		output.WriteString(quoted[1 : len(quoted)-1])
	}
	return output.String()
}

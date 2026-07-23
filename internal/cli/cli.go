// Package cli owns ssb's public command and exit-code contract.
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

	"github.com/nnennandukwe/software-standards-bootstrap/internal/adr"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const helpText = `Software Standards Bootstrap

Usage:
  ssb inspect  [--repo PATH] [--format text|json]
  ssb validate [--repo PATH] [--format text|json]
  ssb render   [--repo PATH] [--dry-run]
  ssb adr      [--repo PATH] [--adr-dir PATH] [--dry-run]

Exit codes:
  0  success
  1  rule-pack validation failure
  2  usage or repository precondition failure
  3  unexpected internal failure
`

const (
	inspectHelp = `Usage: ssb inspect [--repo PATH] [--format text|json]

  --repo PATH              target Git repository (default ".")
  --format text|json       stable output format (default "text")
`
	validateHelp = `Usage: ssb validate [--repo PATH] [--format text|json]

  --repo PATH              target Git repository (default ".")
  --format text|json       validation output format (default "text")
`
	renderHelp = `Usage: ssb render [--repo PATH] [--dry-run]

  --repo PATH              target Git repository (default ".")
  --dry-run                print the complete proposed AGENTS.md without writing
`
	adrHelp = `Usage: ssb adr [--repo PATH] [--adr-dir PATH] [--dry-run]

  --repo PATH              target Git repository (default ".")
  --adr-dir PATH           explicit repository-contained ADR directory
  --dry-run                print the complete proposed ADR without writing
`
)

// Run executes one command and returns the documented process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
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
	case "inspect":
		return runInspect(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "render":
		return runRender(args[1:], stdout, stderr)
	case "adr":
		return runADR(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n%s", args[0], helpText)
		return 2
	}
}

func runADR(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("adr", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	adrDir := flags.String("adr-dir", "", "ADR directory")
	dryRun := flags.Bool("dry-run", false, "print without writing")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, adrHelp)
			return 0
		}
		fmt.Fprintf(stderr, "error: %s\nnext: ssb adr --repo PATH [--adr-dir PATH] [--dry-run]\n", cleanFlagError(err))
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "error: adr does not accept positional arguments")
		fmt.Fprintln(stderr, "next: ssb adr --repo PATH [--adr-dir PATH] [--dry-run]")
		return 2
	}

	ctx := context.Background()
	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb adr --repo <repository-path>\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			return 2
		}
		return 3
	}
	pack, diagnostics, err := rulepack.Validate(ctx, repo)
	if err != nil {
		fmt.Fprintf(stderr, "error: validation failed unexpectedly: %s\n", err)
		return 3
	}
	if len(diagnostics) != 0 {
		writeValidationDiagnostics(stderr, diagnostics)
		return 1
	}
	result, err := adr.Create(ctx, repo, pack, adr.Options{Directory: *adrDir, DryRun: *dryRun})
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		if errors.Is(err, adr.ErrAmbiguousDirectory) || errors.Is(err, adr.ErrUnsafeTarget) || errors.Is(err, adr.ErrCollision) {
			fmt.Fprintln(stderr, "next: choose a safe repository ADR directory with --adr-dir PATH and rerun.")
			return 2
		}
		return 3
	}
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run — proposed %s:\n\n", result.Path)
		_, _ = stdout.Write(result.Content)
		if len(result.Content) == 0 || result.Content[len(result.Content)-1] != '\n' {
			fmt.Fprintln(stdout)
		}
		return 0
	}
	fmt.Fprintf(stdout, "Created %s with Proposed status.\n", result.Path)
	fmt.Fprintln(stdout, "Next: review every uncommitted path and create the adoption pull request yourself.")
	return 0
}

func runRender(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	dryRun := flags.Bool("dry-run", false, "print without writing")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, renderHelp)
			return 0
		}
		fmt.Fprintf(stderr, "error: %s\nnext: ssb render --repo PATH [--dry-run]\n", cleanFlagError(err))
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "error: render does not accept positional arguments")
		fmt.Fprintln(stderr, "next: ssb render --repo PATH [--dry-run]")
		return 2
	}

	ctx := context.Background()
	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb render --repo <repository-path>\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			return 2
		}
		return 3
	}
	pack, diagnostics, err := rulepack.Validate(ctx, repo)
	if err != nil {
		fmt.Fprintf(stderr, "error: validation failed unexpectedly: %s\n", err)
		return 3
	}
	if len(diagnostics) != 0 {
		writeValidationDiagnostics(stderr, diagnostics)
		return 1
	}
	result, err := render.Apply(repo, pack, *dryRun)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		if errors.Is(err, render.ErrDrift) || errors.Is(err, render.ErrMarkers) || errors.Is(err, render.ErrUnsafeTarget) {
			fmt.Fprintln(stderr, "next: fix AGENTS.md or its rule sources, then rerun ssb render --repo PATH.")
			return 2
		}
		return 3
	}
	if *dryRun {
		fmt.Fprintf(stdout, "Dry run — proposed %s:\n\n", result.Path)
		_, _ = stdout.Write(result.Content)
		if len(result.Content) == 0 || result.Content[len(result.Content)-1] != '\n' {
			fmt.Fprintln(stdout)
		}
		return 0
	}
	if result.Changed {
		fmt.Fprintf(stdout, "Rendered %s from %d validated rule(s).\n", result.Path, len(pack.Rules))
	} else {
		fmt.Fprintf(stdout, "%s is already byte-stable for the current rule sources.\n", result.Path)
	}
	fmt.Fprintln(stdout, "Next: review the uncommitted diff; edit or delete rule source files and rerun as needed.")
	return 0
}

type validationResponse struct {
	SchemaVersion  int                   `json:"schema_version"`
	Valid          bool                  `json:"valid"`
	BaselineCommit string                `json:"baseline_commit,omitempty"`
	RuleCount      int                   `json:"rule_count"`
	SkillCount     int                   `json:"skill_count"`
	Diagnostics    []rulepack.Diagnostic `json:"diagnostics"`
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, validateHelp)
			return 0
		}
		fmt.Fprintf(stderr, "error: %s\nnext: ssb validate --repo PATH [--format text|json]\n", cleanFlagError(err))
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "error: validate does not accept positional arguments")
		fmt.Fprintln(stderr, "next: ssb validate --repo PATH [--format text|json]")
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(stderr, "error: --format must be text or json")
		fmt.Fprintln(stderr, "next: ssb validate --repo PATH --format text")
		return 2
	}

	ctx := context.Background()
	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb validate --repo <repository-path>\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			return 2
		}
		return 3
	}
	pack, diagnostics, err := rulepack.Validate(ctx, repo)
	if err != nil {
		fmt.Fprintf(stderr, "error: validation failed unexpectedly: %s\n", err)
		return 3
	}
	response := validationResponse{
		SchemaVersion:  1,
		Valid:          len(diagnostics) == 0,
		BaselineCommit: pack.BaselineCommit,
		RuleCount:      len(pack.Rules),
		SkillCount:     len(pack.Skills),
		Diagnostics:    diagnostics,
	}
	if response.Diagnostics == nil {
		response.Diagnostics = make([]rulepack.Diagnostic, 0)
	}

	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(response); err != nil {
			fmt.Fprintf(stderr, "error: write JSON validation result: %s\n", err)
			return 3
		}
	} else if response.Valid {
		fmt.Fprintf(stdout, "Rule pack valid: %d rule(s), %d related skill(s), baseline %s\n", response.RuleCount, response.SkillCount, response.BaselineCommit)
		fmt.Fprintln(stdout, "Next: review and edit source files, then run ssb render --repo PATH.")
	} else {
		writeValidationDiagnostics(stderr, diagnostics)
	}
	if !response.Valid {
		return 1
	}
	return 0
}

func writeValidationDiagnostics(stderr io.Writer, diagnostics []rulepack.Diagnostic) {
	fmt.Fprintf(stderr, "Rule pack invalid: %d problem(s)\n", len(diagnostics))
	for _, item := range diagnostics {
		location := item.Path
		if item.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, item.Line)
		}
		if item.Field != "" {
			location += " [" + item.Field + "]"
		}
		fmt.Fprintf(stderr, "- %s: %s\n", location, item.Message)
		if item.Recovery != "" {
			fmt.Fprintf(stderr, "  fix: %s\n", item.Recovery)
		}
	}
	fmt.Fprintln(stderr, "Next: edit the reported source files and rerun ssb validate --repo PATH.")
}

func runInspect(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, inspectHelp)
			return 0
		}
		fmt.Fprintf(stderr, "error: %s\nnext: ssb inspect --repo PATH [--format text|json]\n", cleanFlagError(err))
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "error: inspect does not accept positional arguments\nnext: ssb inspect --repo PATH [--format text|json]\n")
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(stderr, "error: --format must be text or json")
		fmt.Fprintln(stderr, "next: ssb inspect --repo PATH --format text")
		return 2
	}

	ctx := context.Background()
	repo, err := workspace.OpenForInspect(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb inspect --repo <clean-repository-path>\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			return 2
		}
		return 3
	}
	report, err := inventory.Scan(ctx, repo, inventory.DefaultLimits())
	if err != nil {
		fmt.Fprintf(stderr, "error: inventory failed: %s\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			fmt.Fprintln(stderr, "next: stabilize the repository and rerun ssb inspect --repo PATH.")
			return 2
		}
		return 3
	}

	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "error: write JSON inventory: %s\n", err)
			return 3
		}
		return 0
	}
	writeTextInventory(stdout, report)
	return 0
}

func writeTextInventory(out io.Writer, report inventory.Report) {
	fmt.Fprintln(out, "Software Standards Bootstrap inventory")
	fmt.Fprintf(out, "Baseline: %s\n", report.BaselineCommit)
	fmt.Fprintf(out, "Files indexed: %d (%d bytes)\n", len(report.Files), report.IndexedBytes)
	fmt.Fprintf(
		out,
		"Excluded: binary=%d generated=%d oversized=%d secret-like=%d symlink=%d submodule=%d vendor/generated-tree=%d non-regular=%d\n",
		report.Excluded.Binary,
		report.Excluded.Generated,
		report.Excluded.Oversized,
		report.Excluded.SecretLike,
		report.Excluded.Symlink,
		report.Excluded.Submodule,
		report.Excluded.VendorTree,
		report.Excluded.NonRegular,
	)
	if report.Truncated {
		fmt.Fprintf(out, "Coverage: TRUNCATED — %s\n", report.TruncationReason)
	} else {
		fmt.Fprintln(out, "Coverage: complete within configured limits")
	}
	fmt.Fprintln(out, "Files:")
	for _, file := range report.Files {
		language := file.Language
		if language == "" {
			language = "text"
		}
		fmt.Fprintf(out, "- %s [%s, %d bytes, %d lines, %s]\n", strconv.Quote(file.Path), language, file.Bytes, file.Lines, file.SHA256)
	}
	fmt.Fprintln(out, "Next: perform targeted semantic reads, then create .software-standards/assessment.md and evidence-backed rule files.")
}

func cleanFlagError(err error) string {
	return strings.TrimPrefix(err.Error(), "flag provided but not defined: ")
}

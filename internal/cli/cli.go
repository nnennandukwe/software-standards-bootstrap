// Package cli owns ssb's public command and exit-code contract.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/adr"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const helpText = `Software Standards Bootstrap

Usage:
  ssb inspect  [--repo PATH] [--format text|json] [resource limits]
  ssb validate [--repo PATH] [--format text|json]
  ssb render   [--repo PATH] [--review ID] [--dry-run]
  ssb adr      [--repo PATH] [--review ID] [--adr-dir PATH] [--dry-run]
  ssb prune    <inspect|validate|approve|apply|recover|status|verify> [options]

Exit codes:
  0  success
  1  actionable-pack or prune-proposal validation failure
  2  usage or repository precondition failure
  3  unexpected internal failure
  4  inventory coverage incomplete
`

const (
	inspectHelp = `Usage: ssb inspect [--repo PATH] [--format text|json] [--max-candidate-files N] [--max-candidate-bytes N] [--allow-partial]

  --repo PATH                 target Git repository (default ".")
  --format text|json          stable output format (default "text")
  --max-candidate-files N     candidate files that may be read (default 40000)
  --max-candidate-bytes N     candidate bytes that may be read (default 134217728)
  --allow-partial             return success for explicitly diagnostic partial coverage
`
	validateHelp = `Usage: ssb validate [--repo PATH] [--format text|json]

  --repo PATH              target Git repository (default ".")
  --format text|json       validation output format (default "text")
`
	renderHelp = `Usage: ssb render [--repo PATH] [--review ID] [--dry-run]

  --repo PATH              target Git repository (default ".")
  --review ID              record rerendering for an applied prune review
  --dry-run                preview the proposed AGENTS.md or report that no write applies
`
	adrHelp = `Usage: ssb adr [--repo PATH] [--review ID] [--adr-dir PATH] [--dry-run]

  --repo PATH              target Git repository (default ".")
  --review ID              record the ADR as a separate prune review state
  --adr-dir PATH           explicit repository-contained ADR directory
  --dry-run                print the complete proposed ADR without writing
`
	pruneHelp = `Usage: ssb prune <command> [options]

Commands:
  inspect   create an immutable, complete review context
  validate  validate a host-agent proposal without approving it
  approve   record one explicit decision for every action
  apply     show a dry run; pass --write to apply approved changes
  recover   restore an interrupted application
  status    report proposal, approval, no-change, application, render, ADR, and verification separately
  verify    validate external content-addressed check receipts
`
	pruneInspectHelp = `Usage: ssb prune inspect --review ID --capabilities PATH [--provenance PATH] [--repo PATH] [--format text|json] [resource limits]

  --review ID                  lower-case kebab-case review identifier
  --capabilities PATH          local point-in-time capability profile
  --provenance PATH            optional explicit artifact provenance manifest
  --repo PATH                  target Git repository (default ".")
  --format text|json           compact result format (default "text")
  --max-candidate-files N      candidate files that may be read (default 40000)
  --max-candidate-bytes N      candidate bytes that may be read (default 134217728)
`
	pruneValidateHelp = `Usage: ssb prune validate --review ID [--repo PATH] [--format text|json]

  --review ID              review bundle identifier
  --repo PATH              target Git repository (default ".")
  --format text|json       compact validation format (default "text")
`
	pruneApproveHelp = `Usage: ssb prune approve --review ID [--approve ID[,ID...]] [--reject ID[,ID...]] [--repo PATH]

  --review ID              validated review bundle identifier
  --approve ID[,ID...]     actions explicitly approved by a human
  --reject ID[,ID...]      actions explicitly rejected by a human
  --repo PATH              target Git repository (default ".")
`
	pruneApplyHelp = `Usage: ssb prune apply --review ID [--repo PATH] [--format text|json] [--write]

  Without --write, this command is a reviewable dry run.
`
	pruneRecoverHelp = `Usage: ssb prune recover --review ID [--repo PATH] [--clear-stale-lock]

  --review ID              interrupted review bundle identifier
  --repo PATH              target Git repository (default ".")
  --clear-stale-lock       clear review-owned crash-left locks after confirming no process is active
`
	pruneStatusHelp = `Usage: ssb prune status --review ID [--repo PATH] [--format text|json]

  --review ID              review bundle identifier
  --repo PATH              target Git repository (default ".")
  --format text|json       compact lifecycle-state format (default "text")
`
	pruneVerifyHelp = `Usage: ssb prune verify --review ID --receipts PATH [--repo PATH]

  --review ID              applied review bundle identifier
  --receipts PATH          directory of external content-addressed receipts
  --repo PATH              target Git repository (default ".")
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
	case "prune":
		return runPrune(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n%s", args[0], helpText)
		return 2
	}
}

func runADR(args []string, stdout, stderr io.Writer) (exitCode int) {
	flags := flag.NewFlagSet("adr", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "prune review id")
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
	pack, diagnostics, err := validateRulePackForCommand(ctx, repo, *reviewID)
	if err != nil {
		fmt.Fprintf(stderr, "error: validation failed unexpectedly: %s\n", err)
		return 3
	}
	if len(diagnostics) != 0 {
		writeValidationDiagnostics(stderr, diagnostics)
		return 1
	}
	var transition *prune.Transition
	var adrRollbackDirs []string
	if *reviewID != "" {
		transition, err = prune.BeginTransition(repo.Root(), *reviewID, prune.EventADR, nil)
		if err != nil {
			return writePruneError(stderr, err)
		}
		defer func() {
			if cancelErr := transition.Cancel(); cancelErr != nil {
				fmt.Fprintf(stderr, "error: release ADR review transition: %s\n", cancelErr)
				exitCode = 3
			}
		}()
		if !*dryRun {
			preview, previewErr := adr.Create(ctx, repo, pack, adr.Options{Directory: *adrDir, DryRun: true})
			if previewErr != nil {
				fmt.Fprintf(stderr, "error: preview review-aware ADR target: %s\n", previewErr)
				return 3
			}
			adrRollbackDirs, err = missingDirectories(
				repo.Root(),
				filepath.Dir(filepath.Join(repo.Root(), filepath.FromSlash(preview.Path))),
			)
			if err != nil {
				fmt.Fprintf(stderr, "error: inspect ADR rollback boundary: %s\n", err)
				return 3
			}
		}
	}
	result, err := adr.Create(ctx, repo, pack, adr.Options{Directory: *adrDir, DryRun: *dryRun})
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		if errors.Is(err, adr.ErrNoAdoptableArtifacts) {
			fmt.Fprintln(stderr, "next: retain a semantic rule, verification recipe, or Agent Skill before creating an ADR.")
			return 2
		}
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
	if *reviewID != "" {
		event, err := transition.Complete(result)
		transitionErr := reconcileTransitionCompletion(
			event,
			err,
			"ADR",
			"created ADR was removed",
			func() error {
				rollbackErr := os.Remove(filepath.Join(repo.Root(), filepath.FromSlash(result.Path)))
				if rollbackErr == nil {
					rollbackErr = removeEmptyDirectories(adrRollbackDirs)
				}
				return rollbackErr
			},
		)
		if transitionErr != nil {
			return writePruneError(stderr, transitionErr)
		}
	}
	fmt.Fprintf(stdout, "Created %s with Proposed status.\n", result.Path)
	fmt.Fprintln(stdout, "Next: review every uncommitted path and create the adoption pull request yourself.")
	return 0
}

func validateRulePackForCommand(
	ctx context.Context,
	repo *workspace.Repository,
	reviewID string,
) (rulepack.Pack, []rulepack.Diagnostic, error) {
	if reviewID != "" {
		return rulepack.ValidateRetainedPack(ctx, repo)
	}
	return rulepack.Validate(ctx, repo)
}

func runRender(args []string, stdout, stderr io.Writer) (exitCode int) {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "prune review id")
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
	pack, diagnostics, err := validateRulePackForCommand(ctx, repo, *reviewID)
	if err != nil {
		fmt.Fprintf(stderr, "error: validation failed unexpectedly: %s\n", err)
		return 3
	}
	if len(diagnostics) != 0 {
		writeValidationDiagnostics(stderr, diagnostics)
		return 1
	}
	var transition *prune.Transition
	var before fileSnapshot
	if *reviewID != "" {
		transition, err = prune.BeginTransition(repo.Root(), *reviewID, prune.EventRendered, nil)
		if err != nil {
			return writePruneError(stderr, err)
		}
		defer func() {
			if cancelErr := transition.Cancel(); cancelErr != nil {
				fmt.Fprintf(stderr, "error: release render review transition: %s\n", cancelErr)
				exitCode = 3
			}
		}()
		before, err = captureFile(filepath.Join(repo.Root(), "AGENTS.md"))
		if err != nil {
			fmt.Fprintf(stderr, "error: capture AGENTS.md before review-aware render: %s\n", err)
			return 3
		}
	}
	result, err := render.Apply(repo, pack, *dryRun)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		if errors.Is(err, render.ErrDrift) || errors.Is(err, render.ErrMarkers) || errors.Is(err, render.ErrUnsafeTarget) {
			fmt.Fprintln(stderr, "next: fix AGENTS.md or its canonical artifact sources, then rerun ssb render --repo PATH.")
			return 2
		}
		return 3
	}
	if *dryRun {
		if !result.Changed {
			fmt.Fprintf(
				stdout,
				"%s would not be changed: the pack has no active semantic rule, verification recipe, or Agent Skill to project.\n",
				result.Path,
			)
			return 0
		}
		fmt.Fprintf(stdout, "Dry run — proposed %s:\n\n", result.Path)
		_, _ = stdout.Write(result.Content)
		if len(result.Content) == 0 || result.Content[len(result.Content)-1] != '\n' {
			fmt.Fprintln(stdout)
		}
		return 0
	}
	if *reviewID != "" {
		event, err := transition.Complete(result)
		transitionErr := reconcileTransitionCompletion(
			event,
			err,
			"render",
			"AGENTS.md was restored",
			func() error {
				return restoreFile(filepath.Join(repo.Root(), "AGENTS.md"), before)
			},
		)
		if transitionErr != nil {
			return writePruneError(stderr, transitionErr)
		}
	}
	if result.Changed {
		fmt.Fprintf(
			stdout,
			"Rendered %s from %d semantic rule(s), %d verification recipe(s), and %d Agent Skill(s).\n",
			result.Path,
			len(pack.Rules),
			len(pack.Recipes),
			len(pack.Skills),
		)
	} else {
		fmt.Fprintf(stdout, "%s requires no write for the current actionable artifacts.\n", result.Path)
	}
	fmt.Fprintln(stdout, "Next: review the uncommitted diff; edit canonical artifact sources and the report manifest together.")
	return 0
}

func reconcileTransitionCompletion(
	event prune.Event,
	completionErr error,
	label, rollbackSuccess string,
	rollback func() error,
) error {
	if completionErr == nil {
		return nil
	}
	if event.EventDigest != "" {
		return completionErr
	}
	if rollbackErr := rollback(); rollbackErr != nil {
		return fmt.Errorf(
			"%s event was not recorded: %w; rollback failed: %v",
			label,
			completionErr,
			rollbackErr,
		)
	}
	return fmt.Errorf(
		"%s event was not recorded: %w; %s",
		label,
		completionErr,
		rollbackSuccess,
	)
}

type validationResponse struct {
	SchemaVersion   int                   `json:"schema_version"`
	Valid           bool                  `json:"valid"`
	BaselineCommit  string                `json:"baseline_commit,omitempty"`
	ArtifactCount   int                   `json:"artifact_count"`
	RuleCount       int                   `json:"rule_count"`
	RecipeCount     int                   `json:"verification_recipe_count"`
	SkillCount      int                   `json:"skill_count"`
	AutomationCount int                   `json:"automation_proposal_count"`
	Diagnostics     []rulepack.Diagnostic `json:"diagnostics"`
	Pack            *rulepack.Pack        `json:"pack,omitempty"`
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
		SchemaVersion:   2,
		Valid:           len(diagnostics) == 0,
		BaselineCommit:  pack.BaselineCommit,
		ArtifactCount:   len(pack.Report.Artifacts),
		RuleCount:       len(pack.Rules),
		RecipeCount:     len(pack.Recipes),
		SkillCount:      len(pack.Skills),
		AutomationCount: len(pack.Automations),
		Diagnostics:     diagnostics,
	}
	if response.Diagnostics == nil {
		response.Diagnostics = make([]rulepack.Diagnostic, 0)
	}
	if response.Valid {
		response.Pack = &pack
	}

	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(response); err != nil {
			fmt.Fprintf(stderr, "error: write JSON validation result: %s\n", err)
			return 3
		}
	} else if response.Valid {
		fmt.Fprintf(
			stdout,
			"Actionable pack valid: %d artifact(s) — %d semantic rule(s), %d verification recipe(s), %d Agent Skill(s), %d automation proposal(s); baseline %s\n",
			response.ArtifactCount,
			response.RuleCount,
			response.RecipeCount,
			response.SkillCount,
			response.AutomationCount,
			response.BaselineCommit,
		)
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
	fmt.Fprintf(stderr, "Actionable pack invalid: %d problem(s)\n", len(diagnostics))
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

	ctx := context.Background()
	repo, err := workspace.OpenForInspect(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb inspect --repo <clean-repository-path>\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			return 2
		}
		return 3
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
		return 3
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

func writeTextInventory(out io.Writer, report inventory.Report) {
	fmt.Fprintln(out, "Software Standards Bootstrap inventory")
	fmt.Fprintf(out, "Baseline: %s\n", report.BaselineCommit)
	fmt.Fprintf(
		out,
		"Candidates: files=%d bytes=%d; scanned: files=%d bytes=%d; remaining: files=%d bytes=%d\n",
		report.CandidateFiles,
		report.CandidateBytes,
		report.ScannedFiles,
		report.ScannedBytes,
		report.RemainingCandidateFiles,
		report.RemainingCandidateBytes,
	)
	fmt.Fprintf(out, "Files indexed: %d (%d bytes)\n", report.IndexedFiles, report.IndexedBytes)
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
	if report.Truncated {
		return
	}
	fmt.Fprintln(out, "Next: perform targeted semantic reads, route accepted candidates to actionable artifacts, then create .software-standards/report.md.")
}

func runPrune(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, pruneHelp)
		return 2
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		if len(args) != 1 {
			fmt.Fprintln(stderr, "error: prune help does not accept arguments")
			return 2
		}
		fmt.Fprint(stdout, pruneHelp)
		return 0
	}
	switch args[0] {
	case "inspect":
		return runPruneInspect(args[1:], stdout, stderr)
	case "validate":
		return runPruneValidate(args[1:], stdout, stderr)
	case "approve":
		return runPruneApprove(args[1:], stdout, stderr)
	case "apply":
		return runPruneApply(args[1:], stdout, stderr)
	case "recover":
		return runPruneRecover(args[1:], stdout, stderr)
	case "status":
		return runPruneStatus(args[1:], stdout, stderr)
	case "verify":
		return runPruneVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown prune command %q\n\n%s", args[0], pruneHelp)
		return 2
	}
}

func runPruneInspect(args []string, stdout, stderr io.Writer) int {
	defaults := inventory.DefaultLimits()
	flags := flag.NewFlagSet("prune inspect", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	capabilities := flags.String("capabilities", "", "capability profile path")
	provenance := flags.String("provenance", "", "provenance manifest path")
	format := flags.String("format", "text", "output format")
	maxFiles := flags.Int("max-candidate-files", defaults.MaxCandidateFiles, "candidate file limit")
	maxBytes := flags.Int64("max-candidate-bytes", defaults.MaxCandidateBytes, "candidate byte limit")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(err, pruneInspectHelp, stdout, stderr)
	}
	if flags.NArg() != 0 || *reviewID == "" || *capabilities == "" ||
		(*format != "text" && *format != "json") || *maxFiles <= 0 || *maxBytes <= 0 {
		fmt.Fprintln(stderr, "error: inspect requires --review ID, --capabilities PATH, valid limits, and --format text|json")
		return 2
	}
	repo, err := workspace.OpenForPrune(context.Background(), *repoPath)
	if err != nil {
		return writePruneError(stderr, err)
	}
	result, err := prune.CreateReview(context.Background(), repo, prune.ContextOptions{
		ReviewID:     *reviewID,
		Capabilities: *capabilities,
		Provenance:   *provenance,
		InventoryLimits: inventory.Limits{
			MaxCandidateFiles: *maxFiles,
			MaxCandidateBytes: *maxBytes,
			MaxFileBytes:      defaults.MaxFileBytes,
		},
	})
	if err != nil {
		return writePruneError(stderr, err)
	}
	if *format == "json" {
		summary := struct {
			ContextPath    string `json:"context_path"`
			ReviewID       string `json:"review_id"`
			ContextDigest  string `json:"context_digest"`
			BaselineCommit string `json:"baseline_commit"`
			ArtifactCount  int    `json:"artifact_count"`
			InventoryFiles int    `json:"inventory_files"`
			InventoryBytes int64  `json:"inventory_bytes"`
		}{
			ContextPath:    result.ContextPath,
			ReviewID:       result.Context.ReviewID,
			ContextDigest:  result.Context.ContextDigest,
			BaselineCommit: result.Context.BaselineCommit,
			ArtifactCount:  len(result.Context.Artifacts),
			InventoryFiles: result.Context.Inventory.IndexedFiles,
			InventoryBytes: result.Context.Inventory.IndexedBytes,
		}
		return writeJSON(stdout, stderr, summary)
	}
	fmt.Fprintf(stdout, "Created prune review context %s (%d rule/skill artifact(s)).\n", result.ContextPath, len(result.Context.Artifacts))
	fmt.Fprintln(stdout, "Next: use the repository Agent Skill to write proposal.yaml, then run ssb prune validate.")
	return 0
}

func runPruneValidate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(err, pruneValidateHelp, stdout, stderr)
	}
	if flags.NArg() != 0 || *reviewID == "" || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "error: validate requires --review ID and --format text|json")
		return 2
	}
	repo, err := workspace.Open(context.Background(), *repoPath)
	if err != nil {
		return writePruneError(stderr, err)
	}
	review, diagnostics, err := prune.ValidateReview(context.Background(), repo, *reviewID)
	if err != nil {
		return writePruneError(stderr, err)
	}
	response := struct {
		Valid          bool                 `json:"valid"`
		ProposalDigest string               `json:"proposal_digest"`
		Diagnostics    []prune.Diagnostic   `json:"diagnostics"`
		Summary        pruneProposalSummary `json:"summary"`
	}{len(diagnostics) == 0, review.ProposalDigest, diagnostics, summarizePruneProposal(review.Proposal)}
	if *format == "json" {
		if code := writeJSON(stdout, stderr, response); code != 0 {
			return code
		}
	} else if len(diagnostics) == 0 {
		fmt.Fprintf(stdout, "Prune proposal valid: %d action(s), digest %s\n", len(review.Proposal.Actions), review.ProposalDigest)
		writePruneSummary(stdout, response.Summary)
	} else {
		writePruneDiagnostics(stderr, diagnostics)
	}
	if len(diagnostics) != 0 {
		return 1
	}
	return 0
}

func runPruneApprove(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune approve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	approved := flags.String("approve", "", "comma-separated approved action ids")
	rejected := flags.String("reject", "", "comma-separated rejected action ids")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(err, pruneApproveHelp, stdout, stderr)
	}
	if flags.NArg() != 0 || *reviewID == "" {
		fmt.Fprintln(stderr, "error: approve requires --review ID and explicit --approve/--reject decisions")
		return 2
	}
	event, err := prune.Approve(context.Background(), *repoPath, prune.ApprovalOptions{
		ReviewID: *reviewID,
		Approved: commaList(*approved),
		Rejected: commaList(*rejected),
	})
	if err != nil {
		return writePruneError(stderr, err)
	}
	fmt.Fprintf(stdout, "Recorded approval event %s in .software-standards/reviews/%s/events.jsonl, bound to proposal %s.\n", event.ID, *reviewID, event.ProposalDigest)
	fmt.Fprintln(stdout, "Next: run ssb prune apply --review ID for a dry run.")
	return 0
}

func runPruneApply(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune apply", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	write := flags.Bool("write", false, "apply approved changes")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(err, pruneApplyHelp, stdout, stderr)
	}
	if flags.NArg() != 0 || *reviewID == "" || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "error: apply requires --review ID and --format text|json")
		return 2
	}
	result, err := prune.Apply(context.Background(), *repoPath, prune.ApplyOptions{ReviewID: *reviewID, Write: *write})
	if err != nil {
		return writePruneError(stderr, err)
	}
	if *format == "json" {
		return writeJSON(stdout, stderr, result)
	}
	if result.DryRun {
		fmt.Fprintf(stdout, "Dry run: %d approved file change(s).\n", len(result.Changes))
		fmt.Fprintf(stdout, "Plan: %s\n", result.PlanDigest)
		for _, change := range result.Changes {
			fmt.Fprintf(stdout, "- %s %s (%s)\n", change.Kind, strconv.Quote(change.Path), change.ActionID)
		}
		fmt.Fprintln(stdout, "Next: review this plan, then rerun with --write.")
	} else if result.NoChangesApproved {
		fmt.Fprintln(stdout, "No changes were approved; the review is complete without application or verification.")
		fmt.Fprintf(stdout, "Plan: %s\n", result.PlanDigest)
		fmt.Fprintf(stdout, "Next: run ssb prune status --review %s to inspect the recorded review outcome.\n", *reviewID)
	} else {
		fmt.Fprintf(stdout, "Applied %d approved file change(s); rerender and verify remain separate states.\n", len(result.Changes))
		fmt.Fprintf(stdout, "Plan: %s\n", result.PlanDigest)
		for _, change := range result.Changes {
			fmt.Fprintf(stdout, "- %s %s (%s)\n", change.Kind, strconv.Quote(change.Path), change.ActionID)
		}
		fmt.Fprintf(stdout, "Next: run ssb prune status --review %s; rerender with ssb render --review %s when rules changed, then attach receipts and verify.\n", *reviewID, *reviewID)
	}
	return 0
}

func runPruneRecover(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune recover", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	clearStaleLock := flags.Bool("clear-stale-lock", false, "clear crash-left transition lock")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(err, pruneRecoverHelp, stdout, stderr)
	}
	if flags.NArg() != 0 || *reviewID == "" {
		fmt.Fprintln(stderr, "error: recover requires --review ID")
		return 2
	}
	repo, err := workspace.Open(context.Background(), *repoPath)
	if err != nil {
		return writePruneError(stderr, err)
	}
	if err := prune.Recover(context.Background(), repo.Root(), *reviewID, *clearStaleLock); err != nil {
		return writePruneError(stderr, err)
	}
	fmt.Fprintln(stdout, "Recovery completed; crash-left locks and any application journal were reconciled.")
	return 0
}

func runPruneStatus(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(err, pruneStatusHelp, stdout, stderr)
	}
	if flags.NArg() != 0 || *reviewID == "" || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "error: status requires --review ID and --format text|json")
		return 2
	}
	repo, err := workspace.Open(context.Background(), *repoPath)
	if err != nil {
		return writePruneError(stderr, err)
	}
	status, diagnostics, err := prune.ReviewStatus(repo.Root(), *reviewID)
	if err != nil {
		return writePruneError(stderr, err)
	}
	if *format == "json" {
		summary := pruneProposalSummary{Counts: map[string]int{}, Rows: []pruneProposalRow{}}
		if status.Proposed {
			if review, _, loadErr := prune.LoadReview(repo.Root(), *reviewID); loadErr == nil {
				summary = summarizePruneProposal(review.Proposal)
			}
		}
		return writeJSON(stdout, stderr, struct {
			Status      prune.Status         `json:"status"`
			Diagnostics []prune.Diagnostic   `json:"diagnostics"`
			Summary     pruneProposalSummary `json:"summary"`
		}{status, diagnostics, summary})
	}
	fmt.Fprintf(stdout, "Review %s: inspected=%t proposed=%t valid=%t approved=%t no-changes-approved=%t applied=%t rendered=%t adr=%t verified=%t\n",
		status.ReviewID, status.Inspected, status.Proposed, status.ProposalValid, status.Approved,
		status.NoChangesApproved, status.Applied, status.Rendered, status.ADRRecorded, status.Verified)
	if status.Proposed {
		if review, _, loadErr := prune.LoadReview(repo.Root(), *reviewID); loadErr == nil {
			writePruneSummary(stdout, summarizePruneProposal(review.Proposal))
		}
	}
	return 0
}

func runPruneVerify(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune verify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	receipts := flags.String("receipts", "", "receipt directory")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(err, pruneVerifyHelp, stdout, stderr)
	}
	if flags.NArg() != 0 || *reviewID == "" || *receipts == "" {
		fmt.Fprintln(stderr, "error: verify requires --review ID and --receipts PATH")
		return 2
	}
	result, err := prune.Verify(context.Background(), *repoPath, *reviewID, *receipts, nil)
	if err != nil {
		return writePruneError(stderr, err)
	}
	fmt.Fprintf(stdout, "Verified %d external check receipt(s) and recorded the event in .software-standards/reviews/%s/events.jsonl; ssb executed no repository command.\n", len(result.Receipts), *reviewID)
	return 0
}

func pruneFlagError(err error, help string, stdout, stderr io.Writer) int {
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(stdout, help)
		return 0
	}
	fmt.Fprintf(stderr, "error: %s\n\n%s", cleanFlagError(err), help)
	return 2
}

func writePruneError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: %s\n", err)
	switch {
	case errors.Is(err, prune.ErrIncompleteInventory):
		fmt.Fprintln(stderr, "next: raise the inventory limits and rerun prune inspection; partial coverage cannot create a review.")
		return 4
	case errors.Is(err, prune.ErrValidation):
		fmt.Fprintln(stderr, "next: correct the reported review input or proposal evidence and rerun the same command.")
		return 1
	case errors.Is(err, workspace.ErrPrecondition):
		return 2
	case errors.Is(err, prune.ErrPrecondition):
		fmt.Fprintln(stderr, "next: resolve the reported review state, then inspect it with ssb prune status --review ID.")
		return 2
	default:
		return 3
	}
}

func writePruneDiagnostics(stderr io.Writer, diagnostics []prune.Diagnostic) {
	fmt.Fprintf(stderr, "Prune proposal invalid: %d problem(s)\n", len(diagnostics))
	for _, item := range diagnostics {
		fmt.Fprintf(stderr, "- %s [%s]: %s\n", item.Path, item.Field, item.Message)
		if item.Recovery != "" {
			fmt.Fprintf(stderr, "  fix: %s\n", item.Recovery)
		}
	}
}

type pruneProposalRow struct {
	Artifact    string `json:"artifact"`
	Kind        string `json:"kind"`
	Disposition string `json:"disposition"`
	Action      string `json:"action"`
	Confidence  string `json:"confidence"`
}

type pruneProposalSummary struct {
	Counts map[string]int     `json:"counts"`
	Rows   []pruneProposalRow `json:"rows"`
}

func summarizePruneProposal(proposal prune.Proposal) pruneProposalSummary {
	summary := pruneProposalSummary{
		Counts: make(map[string]int),
		Rows:   make([]pruneProposalRow, 0),
	}
	for _, action := range proposal.Actions {
		for _, source := range action.Sources {
			summary.Counts[source.Kind+"/"+action.Disposition]++
			summary.Rows = append(summary.Rows, pruneProposalRow{
				Artifact:    source.Path,
				Kind:        source.Kind,
				Disposition: action.Disposition,
				Action:      action.ID,
				Confidence:  action.Confidence,
			})
		}
	}
	return summary
}

func writePruneSummary(output io.Writer, summary pruneProposalSummary) {
	keys := make([]string, 0, len(summary.Counts))
	for key := range summary.Counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Fprint(output, "Disposition counts:")
	for _, key := range keys {
		fmt.Fprintf(output, " %s=%d", key, summary.Counts[key])
	}
	fmt.Fprintln(output)
	for _, row := range summary.Rows {
		fmt.Fprintf(output, "- %s | %s | %s | action=%s | confidence=%s\n",
			row.Kind, row.Disposition, strconv.Quote(row.Artifact), row.Action, row.Confidence)
	}
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(stderr, "error: write JSON result: %s\n", err)
		return 3
	}
	return 0
}

func commaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		result = append(result, strings.TrimSpace(part))
	}
	return result
}

type fileSnapshot struct {
	existed bool
	mode    os.FileMode
	content []byte
}

func captureFile(filePath string) (fileSnapshot, error) {
	info, err := os.Lstat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return fileSnapshot{}, nil
	}
	if err != nil {
		return fileSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return fileSnapshot{}, fmt.Errorf("target is not a regular file")
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{existed: true, mode: info.Mode().Perm(), content: content}, nil
}

func restoreFile(filePath string, snapshot fileSnapshot) error {
	if !snapshot.existed {
		if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	temp, err := os.CreateTemp(filepath.Dir(filePath), ".ssb-restore-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(snapshot.mode); err != nil {
		temp.Close()
		return err
	}
	if written, err := temp.Write(snapshot.content); err != nil {
		temp.Close()
		return err
	} else if written != len(snapshot.content) {
		temp.Close()
		return io.ErrShortWrite
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, filePath)
}

func missingDirectories(root, target string) ([]string, error) {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("directory escapes repository")
	}
	current := root
	missing := make([]string, 0)
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			missing = append(missing, current)
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", current)
		}
	}
	return missing, nil
}

func removeEmptyDirectories(directories []string) error {
	for index := len(directories) - 1; index >= 0; index-- {
		if err := os.Remove(directories[index]); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func cleanFlagError(err error) string {
	message := strings.TrimPrefix(err.Error(), "flag provided but not defined: ")
	if strings.HasPrefix(message, "-") && !strings.HasPrefix(message, "--") {
		message = "-" + message
	}
	return strings.Replace(message, "for flag -", "for flag --", 1)
}

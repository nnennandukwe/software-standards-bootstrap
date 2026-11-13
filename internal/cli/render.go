package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

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

func runRender(ctx context.Context, args []string, stdout, stderr io.Writer) (exitCode int) {
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

	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb render --repo <repository-path>\n", err)
		if errors.Is(err, workspace.ErrPrecondition) {
			return 2
		}
		return commandErrorExitCode(err, 3)
	}
	pack, diagnostics, err := validateRulePackForCommand(ctx, repo, *reviewID)
	if err != nil {
		fmt.Fprintf(stderr, "error: validation failed unexpectedly: %s\n", err)
		return commandErrorExitCode(err, 3)
	}
	if len(diagnostics) != 0 {
		writeValidationDiagnostics(stderr, diagnostics)
		return 1
	}
	var transition *prune.Transition
	var before fileSnapshot
	var beforeRouting directorySnapshot
	var after fileSnapshot
	var afterRouting directorySnapshot
	if *reviewID != "" {
		transition, err = prune.BeginTransition(ctx, repo.Root(), prune.TransitionOptions{ReviewID: *reviewID, Kind: prune.EventRendered, Now: nil})
		if err != nil {
			return writePruneError(stderr, err)
		}
		defer func() {
			if cancelErr := transition.Cancel(); cancelErr != nil {
				fmt.Fprintf(stderr, "error: release render review transition: %s\n", cancelErr)
				exitCode = 3
			}
		}()
		before, err = captureFile(repo.Root(), "AGENTS.md")
		if err != nil {
			fmt.Fprintf(stderr, "error: capture AGENTS.md before review-aware render: %s\n", err)
			return 3
		}
		if pack.Layout == rulepack.LayoutManifest {
			beforeRouting, err = captureDirectory(repo.Root(), render.RoutingDirectory)
			if err != nil {
				fmt.Fprintf(stderr, "error: capture routing tree before review-aware render: %s\n", err)
				return 3
			}
		}
	}
	result, err := render.Apply(
		ctx,
		repo,
		pack,
		*dryRun,
	)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		if errors.Is(err, render.ErrDrift) || errors.Is(err, render.ErrMarkers) || errors.Is(err, render.ErrUnsafeTarget) {
			if pack.Layout == rulepack.LayoutManifest {
				fmt.Fprintln(stderr, "next: restore generated AGENTS.md and .software-standards/routing from reviewed version-control state, then edit canonical sources and rerun ssb render --repo PATH.")
			} else {
				fmt.Fprintln(stderr, "next: fix AGENTS.md or its canonical artifact sources, then rerun ssb render --repo PATH.")
			}
			return 2
		}
		return commandErrorExitCode(err, 3)
	}
	if *dryRun {
		if !result.Changed {
			if len(pack.Rules) == 0 && len(pack.Recipes) == 0 && len(pack.Skills) == 0 {
				fmt.Fprintf(
					stdout,
					"%s would not be changed: the pack has no active semantic rule, verification recipe, or Agent Skill to project.\n",
					result.Path,
				)
			} else {
				fmt.Fprintf(stdout, "%s is already current; no write would occur.\n", result.Path)
			}
		}
		if result.Changed && len(pack.Rules) == 0 && len(pack.Recipes) == 0 && len(pack.Skills) == 0 {
			fmt.Fprintf(
				stdout,
				"Dry run - %s would remove its managed Software Standards Bootstrap section.\n",
				result.Path,
			)
		}
		fmt.Fprintln(stdout, "Dry run - proposed render write set:")
		writeRenderFilePreview(stdout, result.Path, result.Content)
		if result.Routing != nil {
			if result.Routing.Exists {
				for _, file := range result.Routing.Files {
					writeRenderFilePreview(stdout, file.Path, file.Content)
				}
			} else if result.Routing.Changed {
				fmt.Fprintf(stdout, "--- %s (remove generated directory) ---\n", result.Routing.Path)
			}
		}
		writeProjectionMeasurements(stdout, result.Metrics)
		writeProjectionWarnings(stderr, result.Warnings)
		return 0
	}
	if *reviewID != "" {
		after = projectedFileSnapshot(before, result.Content)
		if pack.Layout == rulepack.LayoutManifest {
			afterRouting = projectedRoutingSnapshot(beforeRouting, result.Routing)
		}
		completionContext := ctx
		if result.Changed {
			completionContext = context.WithoutCancel(ctx)
		}
		event, err := transition.Complete(completionContext, result)
		isCompletionCanceled := errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
		if !result.Changed && isCompletionCanceled {
			return writePruneError(stderr, err)
		}
		transitionErr := reconcileTransitionCompletion(
			event,
			err,
			"render",
			"AGENTS.md and the routing tree were restored",
			func() error {
				var routingRollback *directoryRollback
				if pack.Layout == rulepack.LayoutManifest {
					routingRollback = &directoryRollback{
						target: render.RoutingDirectory, before: beforeRouting, expectedCurrent: afterRouting,
					}
				}
				return restoreSnapshotsIfCurrent(
					repo.Root(),
					&fileRollback{target: "AGENTS.md", before: before, expectedCurrent: after},
					routingRollback,
				)
			},
		)
		if transitionErr != nil {
			return writePruneError(stderr, transitionErr)
		}
	}
	for _, warning := range result.OperationalWarnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	if result.Changed {
		if len(pack.Rules) == 0 && len(pack.Recipes) == 0 && len(pack.Skills) == 0 {
			fmt.Fprintf(
				stdout,
				"Removed the managed Software Standards Bootstrap section from %s because the pack has no active semantic rule, verification recipe, or Agent Skill.\n",
				result.Path,
			)
		} else {
			fmt.Fprintf(
				stdout,
				"Rendered %s from %d semantic rule(s), %d verification recipe(s), and %d Agent Skill(s).\n",
				result.Path,
				len(pack.Rules),
				len(pack.Recipes),
				len(pack.Skills),
			)
		}
		if result.Routing != nil && result.Routing.Changed {
			if result.Routing.Exists {
				fmt.Fprintf(
					stdout,
					"Rendered %s with %d generated routing file(s).\n",
					result.Routing.Path,
					len(result.Routing.Files),
				)
			} else {
				fmt.Fprintf(stdout, "Removed generated routing tree %s.\n", result.Routing.Path)
			}
		}
	} else {
		fmt.Fprintf(stdout, "%s requires no write for the current actionable artifacts.\n", result.Path)
	}
	writeProjectionMeasurements(stdout, result.Metrics)
	writeProjectionWarnings(stderr, result.Warnings)
	if pack.Layout == rulepack.LayoutManifest {
		fmt.Fprintln(stdout, "Next: review the uncommitted diff; edit digest-bound sources and update manifest.yaml SHA-256 values together.")
	} else {
		fmt.Fprintln(stdout, "Next: review the uncommitted diff; edit canonical artifact sources and the report manifest together.")
	}
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

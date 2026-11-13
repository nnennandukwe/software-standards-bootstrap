package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/adr"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func runADR(ctx context.Context, args []string, stdout, stderr io.Writer) (exitCode int) {
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

	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\nnext: ssb adr --repo <repository-path>\n", err)
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
	var adrRollbackDirs []string
	if *reviewID != "" {
		transition, err = prune.BeginTransition(ctx, repo.Root(), prune.TransitionOptions{ReviewID: *reviewID, Kind: prune.EventADR, Now: nil})
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
			preview, previewErr := adr.Create(
				ctx,
				repo,
				pack,
				adr.Options{Directory: *adrDir, DryRun: true},
			)
			if previewErr != nil {
				return writeADRError(stderr, previewErr)
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
	result, err := adr.Create(
		ctx,
		repo,
		pack,
		adr.Options{Directory: *adrDir, DryRun: *dryRun},
	)
	if err != nil {
		return writeADRError(stderr, err)
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
		event, err := transition.Complete(context.WithoutCancel(ctx), result)
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

func writeADRError(stderr io.Writer, err error) int {
	if isOnlyCancellation(err) {
		return 130
	}
	fmt.Fprintf(stderr, "error: %s\n", err)
	if errors.Is(err, adr.ErrNoAdoptableArtifacts) {
		fmt.Fprintln(stderr, "next: retain a semantic rule, verification recipe, or Agent Skill before creating an ADR.")
		return 2
	}
	if errors.Is(err, adr.ErrAmbiguousDirectory) ||
		errors.Is(err, adr.ErrUnsafeTarget) ||
		errors.Is(err, adr.ErrCollision) {
		fmt.Fprintln(stderr, "next: choose a safe repository ADR directory with --adr-dir PATH and rerun.")
		return 2
	}
	return 3
}

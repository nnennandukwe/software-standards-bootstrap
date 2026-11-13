package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func runPruneRecover(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune recover", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	clearStaleLock := flags.Bool("clear-stale-lock", false, "clear crash-left transition lock")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(
			err,
			pruneRecoverHelp,
			stdout,
			stderr,
		)
	}
	if flags.NArg() != 0 || *reviewID == "" {
		fmt.Fprintln(stderr, "error: recover requires --review ID")
		return 2
	}
	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		return writePruneError(stderr, err)
	}
	if err := prune.Recover(
		ctx,
		repo.Root(),
		*reviewID,
		*clearStaleLock,
	); err != nil {
		return writePruneError(stderr, err)
	}
	fmt.Fprintln(stdout, "Recovery completed; crash-left locks and any application journal were reconciled.")
	return 0
}

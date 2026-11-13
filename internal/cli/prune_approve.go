package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
)

func runPruneApprove(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune approve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	approved := flags.String("approve", "", "comma-separated approved action ids")
	rejected := flags.String("reject", "", "comma-separated rejected action ids")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(
			err,
			pruneApproveHelp,
			stdout,
			stderr,
		)
	}
	if flags.NArg() != 0 || *reviewID == "" {
		fmt.Fprintln(stderr, "error: approve requires --review ID and explicit --approve/--reject decisions")
		return 2
	}
	event, err := prune.Approve(ctx, *repoPath, prune.ApprovalOptions{
		ReviewID: *reviewID,
		Approved: commaList(*approved),
		Rejected: commaList(*rejected),
	})
	if err != nil {
		return writePruneError(stderr, err)
	}
	fmt.Fprintf(
		stdout,
		"Recorded approval event %s in .software-standards/reviews/%s/events.jsonl, bound to proposal %s.\n",
		event.ID,
		*reviewID,
		event.ProposalDigest,
	)
	fmt.Fprintln(stdout, "Next: run ssb prune apply --review ID for a dry run.")
	return 0
}

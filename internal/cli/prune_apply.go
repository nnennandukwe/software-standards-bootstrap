package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
)

func runPruneApply(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune apply", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	write := flags.Bool("write", false, "apply approved changes")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(
			err,
			pruneApplyHelp,
			stdout,
			stderr,
		)
	}
	if flags.NArg() != 0 || *reviewID == "" || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "error: apply requires --review ID and --format text|json")
		return 2
	}
	result, err := prune.Apply(ctx, *repoPath, prune.ApplyOptions{ReviewID: *reviewID, Write: *write})
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
			fmt.Fprintf(
				stdout,
				"- %s %s (%s)\n",
				change.Kind,
				strconv.Quote(change.Path),
				change.ActionID,
			)
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
			fmt.Fprintf(
				stdout,
				"- %s %s (%s)\n",
				change.Kind,
				strconv.Quote(change.Path),
				change.ActionID,
			)
		}
		fmt.Fprintf(
			stdout,
			"Next: run ssb prune status --review %s; rerender with ssb render --review %s when rules changed, then attach receipts and verify.\n",
			*reviewID,
			*reviewID,
		)
	}
	return 0
}

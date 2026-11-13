package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func runPruneStatus(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(
			err,
			pruneStatusHelp,
			stdout,
			stderr,
		)
	}
	if flags.NArg() != 0 || *reviewID == "" || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "error: status requires --review ID and --format text|json")
		return 2
	}
	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		return writePruneError(stderr, err)
	}
	status, diagnostics, err := prune.ReviewStatus(ctx, repo.Root(), *reviewID)
	if err != nil {
		return writePruneError(stderr, err)
	}
	if *format == "json" {
		summary := pruneProposalSummary{Counts: map[string]int{}, Rows: []pruneProposalRow{}}
		if status.Proposed {
			if review, _, loadErr := prune.LoadReview(ctx, repo.Root(), *reviewID); loadErr == nil {
				summary = summarizePruneProposal(review.Proposal)
			}
		}
		return writeJSON(stdout, stderr, struct {
			Status      prune.Status         `json:"status"`
			Diagnostics []prune.Diagnostic   `json:"diagnostics"`
			Summary     pruneProposalSummary `json:"summary"`
		}{status, diagnostics, summary})
	}
	fmt.Fprintf(
		stdout,
		"Review %s: inspected=%t proposed=%t valid=%t approved=%t no-changes-approved=%t applied=%t rendered=%t adr=%t verified=%t\n",
		status.ReviewID,
		status.Inspected,
		status.Proposed,
		status.ProposalValid,
		status.Approved,
		status.NoChangesApproved,
		status.Applied,
		status.Rendered,
		status.ADRRecorded,
		status.Verified,
	)
	if status.Proposed {
		if review, _, loadErr := prune.LoadReview(ctx, repo.Root(), *reviewID); loadErr == nil {
			writePruneSummary(stdout, summarizePruneProposal(review.Proposal))
		}
	}
	return 0
}

package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func runPruneValidate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	format := flags.String("format", "text", "output format")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(
			err,
			pruneValidateHelp,
			stdout,
			stderr,
		)
	}
	if flags.NArg() != 0 || *reviewID == "" || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, "error: validate requires --review ID and --format text|json")
		return 2
	}
	repo, err := workspace.Open(ctx, *repoPath)
	if err != nil {
		return writePruneError(stderr, err)
	}
	review, diagnostics, err := prune.ValidateReview(ctx, repo, *reviewID)
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
		fmt.Fprintf(
			stdout,
			"Prune proposal valid: %d action(s), digest %s\n",
			len(review.Proposal.Actions),
			review.ProposalDigest,
		)
		writePruneSummary(stdout, response.Summary)
	} else {
		writePruneDiagnostics(stderr, diagnostics)
	}
	if len(diagnostics) != 0 {
		return 1
	}
	return 0
}

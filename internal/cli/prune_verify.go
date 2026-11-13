package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
)

func runPruneVerify(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prune verify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoPath := flags.String("repo", ".", "repository path")
	reviewID := flags.String("review", "", "review id")
	receipts := flags.String("receipts", "", "receipt directory")
	if err := flags.Parse(args); err != nil {
		return pruneFlagError(
			err,
			pruneVerifyHelp,
			stdout,
			stderr,
		)
	}
	if flags.NArg() != 0 || *reviewID == "" || *receipts == "" {
		fmt.Fprintln(stderr, "error: verify requires --review ID and --receipts PATH")
		return 2
	}
	result, err := prune.Verify(ctx, *repoPath, prune.VerificationOptions{ReviewID: *reviewID, ReceiptsDir: *receipts, Now: nil})
	if err != nil {
		return writePruneError(stderr, err)
	}
	fmt.Fprintf(
		stdout,
		"Verified %d external check receipt(s) and recorded the event in .software-standards/reviews/%s/events.jsonl; ssb executed no repository command.\n",
		len(result.Receipts),
		*reviewID,
	)
	return 0
}

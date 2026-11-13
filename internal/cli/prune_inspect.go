package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func runPruneInspect(ctx context.Context, args []string, stdout, stderr io.Writer) int {
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
		return pruneFlagError(
			err,
			pruneInspectHelp,
			stdout,
			stderr,
		)
	}
	if flags.NArg() != 0 || *reviewID == "" || *capabilities == "" ||
		(*format != "text" && *format != "json") || *maxFiles <= 0 || *maxBytes <= 0 {
		fmt.Fprintln(stderr, "error: inspect requires --review ID, --capabilities PATH, valid limits, and --format text|json")
		return 2
	}
	repo, err := workspace.OpenForPrune(ctx, *repoPath)
	if err != nil {
		return writePruneError(stderr, err)
	}
	result, err := prune.CreateReview(ctx, repo, prune.CreateReviewOptions{
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
	fmt.Fprintf(
		stdout,
		"Created prune review context %s (%d actionable artifact(s)).\n",
		result.ContextPath,
		len(result.Context.Artifacts),
	)
	fmt.Fprintln(stdout, "Next: use the repository Agent Skill to write proposal.yaml, then run ssb prune validate.")
	return 0
}

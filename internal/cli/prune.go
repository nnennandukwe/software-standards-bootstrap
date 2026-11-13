package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/prune"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func runPrune(ctx context.Context, args []string, stdout, stderr io.Writer) int {
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
		return runPruneInspect(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "validate":
		return runPruneValidate(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "approve":
		return runPruneApprove(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "apply":
		return runPruneApply(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "recover":
		return runPruneRecover(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "status":
		return runPruneStatus(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	case "verify":
		return runPruneVerify(
			ctx,
			args[1:],
			stdout,
			stderr,
		)
	default:
		fmt.Fprintf(
			stderr,
			"error: unknown prune command %q\n\n%s",
			args[0],
			pruneHelp,
		)
		return 2
	}
}

func pruneFlagError(err error, help string, stdout, stderr io.Writer) int {
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(stdout, help)
		return 0
	}
	fmt.Fprintf(
		stderr,
		"error: %s\n\n%s",
		cleanFlagError(err),
		help,
	)
	return 2
}

func writePruneError(stderr io.Writer, err error) int {
	if isOnlyCancellation(err) {
		return 130
	}
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
		fmt.Fprintf(
			stderr,
			"- %s [%s]: %s\n",
			item.Path,
			item.Field,
			item.Message,
		)
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
		fmt.Fprintf(
			output,
			" %s=%d",
			key,
			summary.Counts[key],
		)
	}
	fmt.Fprintln(output)
	for _, row := range summary.Rows {
		fmt.Fprintf(
			output,
			"- %s | %s | %s | action=%s | confidence=%s\n",
			row.Kind,
			row.Disposition,
			strconv.Quote(row.Artifact),
			row.Action,
			row.Confidence,
		)
	}
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

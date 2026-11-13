package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
)

func writeRenderFilePreview(output io.Writer, path string, content []byte) {
	fmt.Fprintf(
		output,
		"--- %s (%d bytes) ---\n",
		path,
		len(content),
	)
	_, _ = output.Write(content)
	if len(content) == 0 || content[len(content)-1] != '\n' {
		fmt.Fprintln(output)
	}
}

func writeProjectionMeasurements(output io.Writer, metrics render.ProjectionMetrics) {
	fmt.Fprintf(
		output,
		"Projection sizes: %s %d bytes (warning threshold %d)",
		metrics.Root.Path,
		metrics.Root.Bytes,
		metrics.Root.WarningThresholdBytes,
	)
	if metrics.Catalog != nil {
		fmt.Fprintf(
			output,
			"; %s %d bytes (warning threshold %d)",
			metrics.Catalog.Path,
			metrics.Catalog.Bytes,
			metrics.Catalog.WarningThresholdBytes,
		)
	}
	fmt.Fprintln(output, ".")
}

func writeProjectionWarnings(stderr io.Writer, warnings []render.SizeWarning) {
	for _, warning := range warnings {
		fmt.Fprintf(
			stderr,
			"warning: %s is %d bytes; warning threshold is %d bytes\n",
			warning.Path,
			warning.ActualBytes,
			warning.ThresholdBytes,
		)
	}
}

func writeValidationDiagnostics(stderr io.Writer, diagnostics []rulepack.Diagnostic) {
	fmt.Fprintf(stderr, "Actionable pack invalid: %d problem(s)\n", len(diagnostics))
	for _, item := range diagnostics {
		location := item.Path
		if item.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, item.Line)
		}
		if item.Field != "" {
			location += " [" + item.Field + "]"
		}
		fmt.Fprintf(
			stderr,
			"- %s: %s\n",
			location,
			item.Message,
		)
		if item.Recovery != "" {
			fmt.Fprintf(stderr, "  fix: %s\n", item.Recovery)
		}
	}
	fmt.Fprintln(stderr, "Next: edit the reported source files and rerun ssb validate --repo PATH.")
}

func writeTextInventory(out io.Writer, report inventory.Report) {
	fmt.Fprintln(out, "Software Standards Bootstrap inventory")
	fmt.Fprintf(out, "Baseline: %s\n", report.BaselineCommit)
	fmt.Fprintf(
		out,
		"Candidates: files=%d bytes=%d; scanned: files=%d bytes=%d; remaining: files=%d bytes=%d\n",
		report.CandidateFiles,
		report.CandidateBytes,
		report.ScannedFiles,
		report.ScannedBytes,
		report.RemainingCandidateFiles,
		report.RemainingCandidateBytes,
	)
	fmt.Fprintf(
		out,
		"Files indexed: %d (%d bytes)\n",
		report.IndexedFiles,
		report.IndexedBytes,
	)
	fmt.Fprintf(
		out,
		"Excluded: binary=%d generated=%d oversized=%d secret-like=%d symlink=%d submodule=%d vendor/generated-tree=%d non-regular=%d\n",
		report.Excluded.Binary,
		report.Excluded.Generated,
		report.Excluded.Oversized,
		report.Excluded.SecretLike,
		report.Excluded.Symlink,
		report.Excluded.Submodule,
		report.Excluded.VendorTree,
		report.Excluded.NonRegular,
	)
	if report.Truncated {
		fmt.Fprintf(out, "Coverage: TRUNCATED — %s\n", report.TruncationReason)
	} else {
		fmt.Fprintln(out, "Coverage: complete within configured limits")
	}
	fmt.Fprintln(out, "Files:")
	for _, file := range report.Files {
		language := file.Language
		if language == "" {
			language = "text"
		}
		fmt.Fprintf(
			out,
			"- %s [%s, %d bytes, %d lines, %s]\n",
			strconv.Quote(file.Path),
			language,
			file.Bytes,
			file.Lines,
			file.SHA256,
		)
	}
	if report.Truncated {
		return
	}
	fmt.Fprintln(out, "Next: perform targeted semantic reads, route accepted candidates to actionable artifacts, then create inventory.json, human-facing artifacts, and manifest.yaml in .software-standards.")
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(stderr, "error: write JSON result: %s\n", err)
		return 3
	}
	return 0
}

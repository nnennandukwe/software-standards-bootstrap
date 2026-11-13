package rulepack

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func validateHumanReport(sourcePath string, data []byte) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	const heading = "# Software standards report"
	if !hasExactOpeningHeading(data, heading) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			"body",
			"report.md must begin at byte zero with # Software standards report",
			"remove frontmatter and begin the human report with its H1 title",
		))
		return diagnostics
	}
	body := bodyAfterOpeningHeading(data)
	if strings.TrimSpace(string(body)) == "" {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			"body",
			"report narrative must not be empty",
			"record run-wide limitations and accepted-output summaries",
		))
	}
	for _, target := range []string{"manifest.yaml", "inventory.json"} {
		if !bytes.Contains(data, []byte("]("+target+")")) &&
			!bytes.Contains(data, []byte("](.software-standards/"+target+")")) {
			diagnostics = append(diagnostics, diagnostic(
				sourcePath,
				"body",
				"report.md must link to "+target,
				"add a Markdown link to the canonical machine artifact",
			))
		}
	}
	return diagnostics
}

func hasExactOpeningHeading(data []byte, heading string) bool {
	return bytes.HasPrefix(data, []byte(heading+"\n")) || bytes.HasPrefix(data, []byte(heading+"\r\n"))
}

func bodyAfterOpeningHeading(data []byte) []byte {
	newline := bytes.IndexByte(data, '\n')
	if newline < 0 {
		return nil
	}
	body := data[newline+1:]
	if bytes.HasPrefix(body, []byte("\r\n")) {
		return body[2:]
	}
	if bytes.HasPrefix(body, []byte("\n")) {
		return body[1:]
	}
	return body
}

func validateReport(repo *workspace.Repository, report Report, retained bool) []Diagnostic {
	const reportPath = ".software-standards/report.md"
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			reportPath,
			field,
			message,
			recovery,
		))
	}
	if report.Schema != ReportSchema {
		add("schema", "schema must be "+ReportSchema, "update the report schema value")
	}
	if !commitPattern.MatchString(report.BaselineCommit) {
		add("baseline_commit", "baseline_commit must be a 40-character lowercase Git object id", "copy the exact commit from ssb inspect")
	} else if !retained && report.BaselineCommit != repo.Baseline() {
		add(
			"baseline_commit",
			fmt.Sprintf("baseline_commit must equal current HEAD %s", repo.Baseline()),
			"reinspect the new commit and refresh the report and every evidence hash",
		)
	}
	diagnostics = append(diagnostics, validateInventoryMetadata(reportPath, report.BaselineCommit, report.Inventory)...)
	return diagnostics
}

func validateInventoryMetadata(sourcePath, baselineCommit string, reportInventory ReportInventory) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field,
			message,
			recovery,
		))
	}
	if reportInventory.SchemaVersion != 2 || reportInventory.InventoryVersion != "ssb-inventory-v2" {
		add("inventory", "inventory must preserve schema 2 ssb-inventory-v2 accounting", "copy the complete successful ssb inspect inventory")
	}
	if reportInventory.BaselineCommit != baselineCommit {
		add("inventory.baseline_commit", "inventory baseline_commit must match the report baseline_commit", "copy one complete inventory for the report baseline")
	}
	if reportInventory.Truncated {
		add("inventory.truncated", "report inventory coverage must be complete", "rerun inspection with sufficient limits before producing artifacts")
	}
	if reportInventory.Limits.MaxCandidateFiles <= 0 ||
		reportInventory.Limits.MaxCandidateBytes <= 0 ||
		reportInventory.Limits.MaxFileBytes <= 0 {
		add("inventory.limits", "inventory limits must be positive", "copy the exact limits from ssb inspect")
	}
	if reportInventory.Limits.MaxFileBytes != inventory.DefaultLimits().MaxFileBytes {
		add(
			"inventory.limits.max_file_bytes",
			fmt.Sprintf("max_file_bytes must remain %d", inventory.DefaultLimits().MaxFileBytes),
			"copy the fixed per-file limit from ssb inspect",
		)
	}
	if reportInventory.CandidateFiles != reportInventory.ScannedFiles+reportInventory.RemainingCandidateFiles ||
		reportInventory.CandidateBytes != reportInventory.ScannedBytes+reportInventory.RemainingCandidateBytes {
		add("inventory", "candidate, scanned, and remaining inventory accounting is inconsistent", "copy the complete inventory without editing its counts")
	}
	if reportInventory.IndexedFiles != len(reportInventory.Files) {
		add("inventory.indexed_files", "indexed_files must equal the number of inventory file records", "copy every indexed file record from ssb inspect")
	}
	var indexedBytes int64
	for _, file := range reportInventory.Files {
		indexedBytes += file.Bytes
	}
	if reportInventory.IndexedBytes != indexedBytes {
		add("inventory.indexed_bytes", "indexed_bytes must equal the sum of inventory file bytes", "copy the complete inventory without editing its byte counts")
	}
	return diagnostics
}

func validateReportInventory(
	ctx context.Context,
	repo *workspace.Repository,
	recorded ReportInventory,
	sourcePath string,
) ([]Diagnostic, error) {
	actual, err := inventory.ScanAtBaseline(ctx, repo, inventory.Limits{
		MaxCandidateFiles: recorded.Limits.MaxCandidateFiles,
		MaxCandidateBytes: recorded.Limits.MaxCandidateBytes,
		MaxFileBytes:      recorded.Limits.MaxFileBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("rebuild report inventory: %w", err)
	}
	if reflect.DeepEqual(recorded, actual) {
		return nil, nil
	}
	return []Diagnostic{diagnostic(
		sourcePath,
		"inventory",
		"recorded inventory does not exactly match the pinned baseline and limits",
		"copy the complete unedited schema 2 response from ssb inspect for this baseline",
	)}, nil
}

func validateEmbeddedArtifact(sourcePath, field string, artifact AcceptedArtifact) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field+suffix,
			message,
			recovery,
		))
	}
	if !stableIDPattern.MatchString(artifact.ID) {
		add(".id", "artifact id must be lower-case kebab-case", "choose a stable id such as verify-test-migrations")
	}
	expectedPath := canonicalArtifactPath(artifact.Kind, artifact.ID)
	switch artifact.Kind {
	case "rule", "verification", "skill", "automation":
	default:
		add(".kind", fmt.Sprintf("artifact kind %q is not supported", artifact.Kind), "use rule, verification, skill, or automation")
	}
	if expectedPath != "" && artifact.Path != expectedPath {
		add(
			".path",
			fmt.Sprintf("%s artifact path must be %s", artifact.Kind, expectedPath),
			"move the artifact to its canonical path or correct the manifest entry",
		)
	}
	if artifact.Confidence != "medium" && artifact.Confidence != "high" {
		add(
			".confidence",
			"accepted artifact confidence must be medium or high",
			"remove the candidate instead of preserving a low-confidence artifact",
		)
	}
	diagnostics = append(diagnostics, validateUtility(sourcePath, field+".utility", artifact.Utility)...)
	if artifact.Kind != "skill" &&
		(artifact.Category != "" || len(artifact.Lenses) != 0 ||
			len(artifact.Scopes) != 0 || artifact.Derivation != "" ||
			len(artifact.Evidence) != 0) {
		add("", "native artifact provenance belongs in its source file", "remove category, lenses, scopes, derivation, and evidence from this manifest entry")
	}
	if artifact.Kind == "skill" {
		if _, supported := supportedCategories[artifact.Category]; artifact.Category == "" || !supported {
			add(".category", fmt.Sprintf("category %q is not supported", artifact.Category), categoryRecovery)
		}
		diagnostics = append(diagnostics, validateActionableLenses(sourcePath, field+".lenses", artifact.Lenses)...)
		diagnostics = append(
			diagnostics,
			prefixDiagnosticFields(validateScopes(sourcePath, artifact.Scopes), field+".")...,
		)
	}
	return diagnostics
}

func canonicalArtifactPath(kind, id string) string {
	if !stableIDPattern.MatchString(id) {
		return ""
	}
	switch kind {
	case "rule":
		return path.Join(".software-standards/rules", id+".md")
	case "verification":
		return path.Join(".software-standards/verification", id+".yaml")
	case "skill":
		return path.Join(".agents/skills", id, "SKILL.md")
	case "automation":
		return path.Join(".software-standards/automation", id+".yaml")
	default:
		return ""
	}
}

func validateUtility(sourcePath, field string, utility Utility) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field+suffix,
			message,
			recovery,
		))
	}
	if utility.Method != UtilityMethod {
		add(".method", "utility method must be "+UtilityMethod, "use the versioned actionable-artifact utility method")
	}
	factors := []struct {
		name  string
		value int
		max   int
	}{
		{"marginal_value", utility.Factors.MarginalValue, 30},
		{"risk_reduction", utility.Factors.RiskReduction, 25},
		{"actionability", utility.Factors.Actionability, 20},
		{"applicability", utility.Factors.Applicability, 15},
		{"earlier_feedback", utility.Factors.EarlierFeedback, 10},
	}
	sum := 0
	for _, factor := range factors {
		sum += factor.value
		if factor.value < 0 || factor.value > factor.max {
			add(
				".factors."+factor.name,
				fmt.Sprintf(
					"%s utility %d is outside 0-%d",
					factor.name,
					factor.value,
					factor.max,
				),
				"correct the factor within its documented range",
			)
		}
	}
	if utility.Total != sum {
		add(".total", fmt.Sprintf("utility total %d does not equal factor sum %d", utility.Total, sum), "recalculate the transparent utility score")
	}
	if utility.Total < 45 {
		add(
			".total",
			fmt.Sprintf("utility %d is below the 45-point acceptance threshold", utility.Total),
			"remove the candidate instead of preserving a rejected artifact",
		)
	} else if utility.Total > 100 {
		add(".total", fmt.Sprintf("utility %d is above 100", utility.Total), "correct the factor arithmetic")
	}
	return diagnostics
}

func loadReport(repo *workspace.Repository) (Report, []Diagnostic, error) {
	const relative = ".software-standards/report.md"
	data, diagnostics, err := readRequiredRegularFile(
		filepath.Join(repo.Root(), filepath.FromSlash(relative)),
		relative,
	)
	if err != nil || len(data) == 0 {
		return Report{}, diagnostics, err
	}
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Report{}, append(diagnostics, diagnostic(
			relative,
			"frontmatter",
			splitErr.Error(),
			"add strict YAML frontmatter between --- markers",
		)), nil
	}
	var report Report
	if err := yaml.Load(
		frontmatter,
		&report,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return Report{}, append(diagnostics, yamlDiagnostic(
			relative,
			err,
			"remove unknown or duplicate fields and use the ssb.dev/report/v1 schema",
		)), nil
	}
	report.Body = string(body)
	diagnostics = append(diagnostics, validateReport(repo, report, true)...)
	if strings.TrimSpace(report.Body) == "" {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"body",
			"report narrative must not be empty",
			"record run-wide limitations and accepted-output summaries",
		))
	}
	return report, diagnostics, nil
}

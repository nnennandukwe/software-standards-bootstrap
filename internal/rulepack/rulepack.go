// Package rulepack validates the editable, evidence-backed standards pack.
package rulepack

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const (
	ReportSchema       = "ssb.dev/report/v1"
	RuleSchema         = "ssb.dev/rule/v2"
	VerificationSchema = "ssb.dev/verification/v1"
	AutomationSchema   = "ssb.dev/automation/v1"
	UtilityMethod      = "ssb-utility-v1"
	categoryRecovery   = "use one primary category: architecture, compatibility, compliance, correctness, developer-experience, documentation, maintainability, operability, performance, quality, reliability, security, or testability"
)

var (
	stableIDPattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	digestPattern       = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	supportedCategories = map[string]struct{}{
		"architecture":         {},
		"compatibility":        {},
		"compliance":           {},
		"correctness":          {},
		"developer-experience": {},
		"documentation":        {},
		"maintainability":      {},
		"operability":          {},
		"performance":          {},
		"quality":              {},
		"reliability":          {},
		"security":             {},
		"testability":          {},
	}
	supportedDirectives = map[string]struct{}{
		"always":    {},
		"ask-first": {},
		"never":     {},
		"prefer":    {},
	}
)

// Diagnostic is a recoverable, file-specific validation problem.
type Diagnostic struct {
	Path     string `json:"path"`
	Line     int    `json:"line,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Recovery string `json:"recovery,omitempty"`
}

// Evidence points to an exact line range in the baseline commit.
type Evidence struct {
	Ref           string `yaml:"ref,omitempty" json:"ref,omitempty"`
	Role          string `yaml:"role,omitempty" json:"role,omitempty"`
	Path          string `yaml:"path" json:"path"`
	Lines         string `yaml:"lines" json:"lines"`
	ExcerptSHA256 string `yaml:"excerpt_sha256" json:"excerpt_sha256"`
}

// Utility records the transparent ssb-utility-v1 ranking arithmetic owned by
// the report manifest.
type Utility struct {
	Method  string         `yaml:"method" json:"method"`
	Total   int            `yaml:"total" json:"total"`
	Factors UtilityFactors `yaml:"factors" json:"factors"`
}

// UtilityFactors are the bounded ssb-utility-v1 inputs.
type UtilityFactors struct {
	MarginalValue   int `yaml:"marginal_value" json:"marginal_value"`
	RiskReduction   int `yaml:"risk_reduction" json:"risk_reduction"`
	Actionability   int `yaml:"actionability" json:"actionability"`
	Applicability   int `yaml:"applicability" json:"applicability"`
	EarlierFeedback int `yaml:"earlier_feedback" json:"earlier_feedback"`
}

// ReportInventory is the exact ssb-inventory-v2 response recorded in report.md.
type ReportInventory = inventory.Report

// ManifestArtifact is one accepted output indexed by report.md.
type ManifestArtifact struct {
	ID                 string     `yaml:"id" json:"id"`
	Kind               string     `yaml:"kind" json:"kind"`
	Path               string     `yaml:"path" json:"path"`
	Confidence         string     `yaml:"confidence" json:"confidence"`
	Utility            Utility    `yaml:"utility" json:"utility"`
	RelatedArtifactIDs []string   `yaml:"related_artifacts,omitempty" json:"related_artifacts,omitempty"`
	Category           string     `yaml:"category,omitempty" json:"category,omitempty"`
	Lenses             []Lens     `yaml:"lenses,omitempty" json:"lenses,omitempty"`
	Scopes             []string   `yaml:"scopes,omitempty" json:"scopes,omitempty"`
	Derivation         string     `yaml:"derivation,omitempty" json:"derivation,omitempty"`
	Evidence           []Evidence `yaml:"evidence,omitempty" json:"evidence,omitempty"`
}

// Report is the accepted artifact index and run narrative.
type Report struct {
	Schema         string             `yaml:"schema" json:"schema"`
	BaselineCommit string             `yaml:"baseline_commit" json:"baseline_commit"`
	Inventory      ReportInventory    `yaml:"inventory" json:"inventory"`
	Artifacts      []ManifestArtifact `yaml:"artifacts" json:"artifacts"`
	Body           string             `yaml:"-" json:"body"`
}

// RemoveManifestArtifacts removes accepted artifacts and relationships to
// them while preserving the report inventory and narrative. Lifecycle
// mutation uses this to keep report.md in the same atomic write set as the
// governed artifacts it removes.
func RemoveManifestArtifacts(data []byte, removedIDs map[string]struct{}) ([]byte, error) {
	if len(removedIDs) == 0 {
		return data, nil
	}
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}
	var report Report
	if err := yaml.Load(frontmatter, &report, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return nil, fmt.Errorf("parse report frontmatter: %w", err)
	}
	found := make(map[string]struct{}, len(removedIDs))
	artifacts := make([]ManifestArtifact, 0, len(report.Artifacts))
	for _, artifact := range report.Artifacts {
		if _, removed := removedIDs[artifact.ID]; removed {
			found[artifact.ID] = struct{}{}
			continue
		}
		relationships := artifact.RelatedArtifactIDs[:0]
		for _, relatedID := range artifact.RelatedArtifactIDs {
			if _, removed := removedIDs[relatedID]; !removed {
				relationships = append(relationships, relatedID)
			}
		}
		artifact.RelatedArtifactIDs = relationships
		artifacts = append(artifacts, artifact)
	}
	if len(found) != len(removedIDs) {
		missing := make([]string, 0, len(removedIDs)-len(found))
		for id := range removedIDs {
			if _, exists := found[id]; !exists {
				missing = append(missing, id)
			}
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("report does not list artifact %s", missing[0])
	}
	report.Artifacts = artifacts
	encoded, err := yaml.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode report frontmatter: %w", err)
	}
	result := make([]byte, 0, len(encoded)+len(body)+10)
	result = append(result, "---\n"...)
	result = append(result, encoded...)
	result = append(result, "---\n"...)
	result = append(result, body...)
	return result, nil
}

// Lens identifies one context dimension used to select a rule. Values within
// one kind are alternatives; represented kinds are matched together.
type Lens struct {
	Kind  string `yaml:"kind" json:"kind"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

// Rule is one editable rule source file plus its exact Markdown body.
type Rule struct {
	Schema     string     `yaml:"schema" json:"schema"`
	ID         string     `yaml:"id" json:"id"`
	Title      string     `yaml:"title" json:"title"`
	Category   string     `yaml:"category" json:"category"`
	Lenses     []Lens     `yaml:"lenses" json:"lenses"`
	Directive  string     `yaml:"directive" json:"directive"`
	Scopes     []string   `yaml:"scopes" json:"scopes"`
	Derivation string     `yaml:"derivation" json:"derivation"`
	Evidence   []Evidence `yaml:"evidence" json:"evidence"`

	SourcePath string `yaml:"-" json:"source_path"`
	Body       string `yaml:"-" json:"body"`
}

// Skill is a validated portable Agent Skill indexed by the report manifest.
type Skill struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category"`
	SourcePath  string `json:"source_path"`
	Body        string `json:"body"`
}

// VerificationStep is one existing command in a recorded recipe.
type VerificationStep struct {
	Run            string `yaml:"run" json:"run"`
	SourceEvidence string `yaml:"source_evidence" json:"source_evidence"`
	ExpectedResult string `yaml:"expected_result" json:"expected_result"`
}

// VerificationRecipe records an ordered, deliberately invoked existing
// command sequence. SSB validates but never executes it.
type VerificationRecipe struct {
	Schema     string             `yaml:"schema" json:"schema"`
	ID         string             `yaml:"id" json:"id"`
	Title      string             `yaml:"title" json:"title"`
	Category   string             `yaml:"category" json:"category"`
	Lenses     []Lens             `yaml:"lenses" json:"lenses"`
	Scopes     []string           `yaml:"scopes" json:"scopes"`
	Derivation string             `yaml:"derivation" json:"derivation"`
	Evidence   []Evidence         `yaml:"evidence" json:"evidence"`
	When       string             `yaml:"when" json:"when"`
	Steps      []VerificationStep `yaml:"steps" json:"steps"`
	SourcePath string             `yaml:"-" json:"source_path"`
}

// AutomationProposal is a reviewable checker design, not adopted behavior.
type AutomationProposal struct {
	Schema          string     `yaml:"schema" json:"schema"`
	ID              string     `yaml:"id" json:"id"`
	Title           string     `yaml:"title" json:"title"`
	Category        string     `yaml:"category" json:"category"`
	Lenses          []Lens     `yaml:"lenses" json:"lenses"`
	Scopes          []string   `yaml:"scopes" json:"scopes"`
	Derivation      string     `yaml:"derivation" json:"derivation"`
	Evidence        []Evidence `yaml:"evidence" json:"evidence"`
	Condition       string     `yaml:"condition" json:"condition"`
	SuggestedCheck  string     `yaml:"suggested_check" json:"suggested_check"`
	Trigger         string     `yaml:"trigger" json:"trigger"`
	ExpectedSuccess string     `yaml:"expected_success" json:"expected_success"`
	ExpectedFailure string     `yaml:"expected_failure" json:"expected_failure"`
	SourcePath      string     `yaml:"-" json:"source_path"`
}

// Pack contains parsed artifacts even when diagnostics are returned. Consumers
// must not render or create an ADR unless diagnostics is empty.
type Pack struct {
	BaselineCommit string               `json:"baseline_commit"`
	ReportPath     string               `json:"report_path,omitempty"`
	Report         Report               `json:"report,omitempty"`
	Rules          []Rule               `json:"rules"`
	Recipes        []VerificationRecipe `json:"verification_recipes"`
	Skills         []Skill              `json:"skills"`
	Automations    []AutomationProposal `json:"automation_proposals"`
}

type skillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
}

// Validate parses the current editable pack and verifies all evidence against
// the repository's pinned HEAD commit. Validation never writes files.
func Validate(ctx context.Context, repo *workspace.Repository) (Pack, []Diagnostic, error) {
	return validateActionablePack(ctx, repo, false)
}

// ValidateRetainedPack parses an adopted pack and verifies artifact evidence
// against the historical baseline recorded in report.md. It is intended for
// review-aware post-application rendering and ADR creation; ordinary editable
// pack validation remains pinned to the repository's current HEAD.
func ValidateRetainedPack(ctx context.Context, repo *workspace.Repository) (Pack, []Diagnostic, error) {
	return validateActionablePack(ctx, repo, true)
}

type semanticRuleSource struct {
	Schema     string     `yaml:"schema"`
	ID         string     `yaml:"id"`
	Title      string     `yaml:"title"`
	Category   string     `yaml:"category"`
	Lenses     []Lens     `yaml:"lenses"`
	Directive  string     `yaml:"directive"`
	Scopes     []string   `yaml:"scopes"`
	Derivation string     `yaml:"derivation"`
	Evidence   []Evidence `yaml:"evidence"`
}

func validateActionablePack(
	ctx context.Context,
	repo *workspace.Repository,
	retained bool,
) (Pack, []Diagnostic, error) {
	const reportPath = ".software-standards/report.md"
	pack := Pack{
		BaselineCommit: repo.Baseline(),
		ReportPath:     reportPath,
		Rules:          make([]Rule, 0),
		Recipes:        make([]VerificationRecipe, 0),
		Skills:         make([]Skill, 0),
		Automations:    make([]AutomationProposal, 0),
	}
	diagnostics := make([]Diagnostic, 0)

	packRootPath := filepath.Join(repo.Root(), ".software-standards")
	info, err := os.Lstat(packRootPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			diagnostics = append(diagnostics, diagnostic(
				".software-standards",
				"file",
				".software-standards does not exist",
				"create .software-standards/report.md and rerun ssb validate",
			))
			return pack, diagnostics, nil
		}
		return Pack{}, nil, fmt.Errorf("inspect .software-standards: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		diagnostics = append(diagnostics, diagnostic(
			".software-standards",
			"file",
			".software-standards must be a real directory, not a symlink",
			"move the editable pack inside the repository",
		))
		return pack, diagnostics, nil
	}

	reportBytes, reportDiagnostics, err := readRequiredRegularFile(
		filepath.Join(repo.Root(), filepath.FromSlash(reportPath)),
		reportPath,
	)
	if err != nil {
		return Pack{}, nil, err
	}
	diagnostics = append(diagnostics, reportDiagnostics...)
	if len(reportBytes) == 0 {
		return pack, diagnostics, nil
	}
	frontmatter, body, splitErr := splitFrontmatter(reportBytes)
	if splitErr != nil {
		diagnostics = append(diagnostics, diagnostic(
			reportPath,
			"frontmatter",
			splitErr.Error(),
			"add strict YAML frontmatter between --- markers",
		))
		return pack, diagnostics, nil
	}
	if err := yaml.Load(
		frontmatter,
		&pack.Report,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		diagnostics = append(diagnostics, yamlDiagnostic(
			reportPath,
			err,
			"remove unknown or duplicate fields and use the ssb.dev/report/v1 schema",
		))
		return pack, diagnostics, nil
	}
	pack.Report.Body = string(body)
	pack.BaselineCommit = pack.Report.BaselineCommit
	diagnostics = append(diagnostics, validateReport(repo, pack.Report, retained)...)
	if strings.TrimSpace(pack.Report.Body) == "" {
		diagnostics = append(diagnostics, diagnostic(
			reportPath,
			"body",
			"report narrative must not be empty",
			"record run-wide limitations and accepted-output summaries",
		))
	}

	evidenceRepo := repo
	if retained {
		historical, historicalErr := repo.AtCommit(ctx, pack.Report.BaselineCommit)
		if historicalErr != nil {
			if !errors.Is(historicalErr, workspace.ErrHistoricalCommit) {
				return Pack{}, nil, historicalErr
			}
			diagnostics = append(diagnostics, diagnostic(
				reportPath,
				"baseline_commit",
				fmt.Sprintf(
					"recorded baseline_commit %q is not a reachable ancestor; artifact evidence cannot be verified: %v",
					pack.Report.BaselineCommit,
					historicalErr,
				),
				"restore the recorded baseline to current repository history or update this pack through a new approved review",
			))
			return pack, diagnostics, nil
		}
		evidenceRepo = historical
	}
	inventoryDiagnostics, inventoryErr := validateReportInventory(
		ctx,
		evidenceRepo,
		pack.Report.Inventory,
	)
	if inventoryErr != nil {
		return Pack{}, nil, inventoryErr
	}
	diagnostics = append(diagnostics, inventoryDiagnostics...)

	entriesByID := make(map[string]ManifestArtifact, len(pack.Report.Artifacts))
	entriesByPath := make(map[string]ManifestArtifact, len(pack.Report.Artifacts))
	for index, artifact := range pack.Report.Artifacts {
		field := fmt.Sprintf("artifacts[%d]", index)
		diagnostics = append(diagnostics, validateManifestArtifact(reportPath, field, artifact)...)
		if prior, exists := entriesByID[artifact.ID]; exists {
			diagnostics = append(diagnostics, diagnostic(
				reportPath,
				field+".id",
				fmt.Sprintf("duplicate artifact id %q also used by %s", artifact.ID, prior.Path),
				"give every accepted artifact a globally unique stable id",
			))
		} else {
			entriesByID[artifact.ID] = artifact
		}
		if prior, exists := entriesByPath[artifact.Path]; exists {
			diagnostics = append(diagnostics, diagnostic(
				reportPath,
				field+".path",
				fmt.Sprintf("duplicate artifact path %q also used by %s", artifact.Path, prior.ID),
				"list each canonical artifact path once",
			))
		} else {
			entriesByPath[artifact.Path] = artifact
		}
	}

	for index, artifact := range pack.Report.Artifacts {
		if canonicalArtifactPath(artifact.Kind, artifact.ID) != artifact.Path {
			continue
		}
		switch artifact.Kind {
		case "rule":
			rule, artifactDiagnostics, loadErr := loadActionableRule(
				ctx,
				evidenceRepo,
				repo.Root(),
				artifact,
			)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if rule.ID != "" {
				pack.Rules = append(pack.Rules, rule)
			}
		case "verification":
			recipe, artifactDiagnostics, loadErr := loadVerificationRecipe(
				ctx,
				evidenceRepo,
				repo.Root(),
				artifact,
			)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if recipe.ID != "" {
				pack.Recipes = append(pack.Recipes, recipe)
			}
		case "skill":
			skill, artifactDiagnostics, loadErr := loadActionableSkill(
				ctx,
				evidenceRepo,
				repo.Root(),
				reportPath,
				fmt.Sprintf("artifacts[%d]", index),
				artifact,
			)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if skill.ID != "" {
				pack.Skills = append(pack.Skills, skill)
			}
		case "automation":
			automation, artifactDiagnostics, loadErr := loadAutomationProposal(
				ctx,
				evidenceRepo,
				repo.Root(),
				artifact,
			)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if automation.ID != "" {
				pack.Automations = append(pack.Automations, automation)
			}
		}
	}

	for _, artifact := range pack.Report.Artifacts {
		seenRelated := make(map[string]struct{}, len(artifact.RelatedArtifactIDs))
		for _, relatedID := range artifact.RelatedArtifactIDs {
			if relatedID == artifact.ID {
				diagnostics = append(diagnostics, diagnostic(
					reportPath,
					"related_artifacts",
					fmt.Sprintf("artifact %s cannot relate to itself", artifact.ID),
					"remove the self relationship",
				))
			}
			if _, duplicate := seenRelated[relatedID]; duplicate {
				diagnostics = append(diagnostics, diagnostic(
					reportPath,
					"related_artifacts",
					fmt.Sprintf("artifact %s repeats relationship %s", artifact.ID, relatedID),
					"list each related artifact once",
				))
			}
			seenRelated[relatedID] = struct{}{}
			if _, exists := entriesByID[relatedID]; !exists {
				diagnostics = append(diagnostics, diagnostic(
					reportPath,
					"related_artifacts",
					fmt.Sprintf("artifact %s references missing related artifact %s", artifact.ID, relatedID),
					"restore the related artifact or remove the dangling relationship",
				))
			}
		}
	}

	unlisted, scanDiagnostics, scanErr := unlistedNativeArtifacts(repo.Root(), entriesByPath)
	if scanErr != nil {
		return Pack{}, nil, scanErr
	}
	diagnostics = append(diagnostics, scanDiagnostics...)
	for _, relative := range unlisted {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"file",
			relative+" is not listed in .software-standards/report.md",
			"add the accepted artifact to the report or remove the unaccepted file",
		))
	}
	sort.Slice(pack.Rules, func(i, j int) bool { return pack.Rules[i].ID < pack.Rules[j].ID })
	sort.Slice(pack.Recipes, func(i, j int) bool { return pack.Recipes[i].ID < pack.Recipes[j].ID })
	sort.Slice(pack.Skills, func(i, j int) bool { return pack.Skills[i].ID < pack.Skills[j].ID })
	sort.Slice(pack.Automations, func(i, j int) bool { return pack.Automations[i].ID < pack.Automations[j].ID })
	return pack, diagnostics, nil
}

func validateReport(repo *workspace.Repository, report Report, retained bool) []Diagnostic {
	const reportPath = ".software-standards/report.md"
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(reportPath, field, message, recovery))
	}
	if report.Schema != ReportSchema {
		add("schema", "schema must be "+ReportSchema, "update the report schema value")
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(report.BaselineCommit) {
		add("baseline_commit", "baseline_commit must be a 40-character lowercase Git object id", "copy the exact commit from ssb inspect")
	} else if !retained && report.BaselineCommit != repo.Baseline() {
		add(
			"baseline_commit",
			fmt.Sprintf("baseline_commit must equal current HEAD %s", repo.Baseline()),
			"reinspect the new commit and refresh the report and every evidence hash",
		)
	}
	reportInventory := report.Inventory
	if reportInventory.SchemaVersion != 2 || reportInventory.InventoryVersion != "ssb-inventory-v2" {
		add("inventory", "inventory must preserve schema 2 ssb-inventory-v2 accounting", "copy the complete successful ssb inspect inventory")
	}
	if reportInventory.BaselineCommit != report.BaselineCommit {
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
		".software-standards/report.md",
		"inventory",
		"recorded inventory does not exactly match the pinned baseline and limits",
		"copy the complete unedited schema 2 response from ssb inspect for this baseline",
	)}, nil
}

func validateManifestArtifact(sourcePath, field string, artifact ManifestArtifact) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(sourcePath, field+suffix, message, recovery))
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
		diagnostics = append(diagnostics, diagnostic(sourcePath, field+suffix, message, recovery))
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
				fmt.Sprintf("%s utility %d is outside 0-%d", factor.name, factor.value, factor.max),
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

func loadActionableRule(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	manifest ManifestArtifact,
) (Rule, []Diagnostic, error) {
	data, diagnostics, err := readManifestArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		if len(data) == 0 && len(diagnostics) != 0 {
			for index := range diagnostics {
				diagnostics[index].Recovery = "remove its manifest entry or restore the artifact at the canonical path"
			}
		}
		return Rule{}, diagnostics, err
	}
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Rule{}, append(diagnostics, diagnostic(
			manifest.Path,
			"frontmatter",
			splitErr.Error(),
			"add strict YAML frontmatter between --- markers",
		)), nil
	}
	var source semanticRuleSource
	if err := yaml.Load(frontmatter, &source, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return Rule{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"remove old proof-oriented fields and use the current ssb.dev/rule/v2 schema",
		)), nil
	}
	rule := Rule{
		Schema:     source.Schema,
		ID:         source.ID,
		Title:      source.Title,
		Category:   source.Category,
		Lenses:     source.Lenses,
		Directive:  source.Directive,
		Scopes:     source.Scopes,
		Derivation: source.Derivation,
		Evidence:   source.Evidence,
		SourcePath: manifest.Path,
		Body:       string(body),
	}
	diagnostics = append(diagnostics, validateActionableRule(ctx, evidenceRepo, rule, manifest)...)
	return rule, diagnostics, nil
}

func validateActionableRule(
	ctx context.Context,
	repo *workspace.Repository,
	rule Rule,
	manifest ManifestArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(rule.SourcePath, field, message, recovery))
	}
	if rule.Schema != RuleSchema {
		add("schema", "schema must be "+RuleSchema, "rewrite this pre-release rule using the current semantic rule contract")
	}
	if rule.ID != manifest.ID {
		add("id", fmt.Sprintf("rule id %q must match manifest id %q", rule.ID, manifest.ID), "align the rule id, filename, and report entry")
	}
	if strings.TrimSpace(rule.Title) == "" {
		add("title", "title is required", "add a concise developer-facing title")
	}
	if _, supported := supportedCategories[rule.Category]; rule.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", rule.Category), categoryRecovery)
	}
	if strings.TrimSpace(rule.Body) == "" {
		add("body", "rule body is required", "write the actionable semantic obligation")
	}
	diagnostics = append(diagnostics, validateActionableLenses(rule.SourcePath, "lenses", rule.Lenses)...)
	if _, supported := supportedDirectives[rule.Directive]; !supported {
		add("directive", fmt.Sprintf("directive %q is not supported", rule.Directive), "use always, ask-first, never, or prefer")
	}
	diagnostics = append(diagnostics, validateScopes(rule.SourcePath, rule.Scopes)...)
	diagnostics = append(diagnostics, validateDerivationEvidence(ctx, repo, rule.SourcePath, rule.Derivation, rule.Evidence)...)
	return diagnostics
}

func validateActionableLenses(sourcePath, field string, lenses []Lens) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(sourcePath, field+suffix, message, recovery))
	}
	if len(lenses) == 0 {
		add("", "at least one activation lens is required", "add one base lens or one or more language, framework, and task lenses")
		return diagnostics
	}
	baseCount := 0
	seen := make(map[string]struct{}, len(lenses))
	for index, lens := range lenses {
		suffix := fmt.Sprintf("[%d]", index)
		switch lens.Kind {
		case "base":
			baseCount++
			if lens.Value != "" {
				add(suffix+".value", "base lens must not have a value", "remove the value from the base lens")
			}
		case "language", "framework":
			if !stableIDPattern.MatchString(lens.Value) {
				add(suffix+".value", lens.Kind+" lens requires a lower-case kebab-case value", "add a value such as go, python, cobra, or django")
			}
		case "task":
			switch lens.Value {
			case "planning", "implementation", "verification":
			default:
				add(suffix+".value", fmt.Sprintf("task lens value %q is not supported", lens.Value), "use planning, implementation, or verification")
			}
		default:
			add(suffix+".kind", fmt.Sprintf("lens kind %q is not supported", lens.Kind), "use base, language, framework, or task")
		}
		key := lens.Kind + ":" + lens.Value
		if _, duplicate := seen[key]; duplicate {
			add("", "duplicate activation lens "+key, "list each lens once")
		}
		seen[key] = struct{}{}
	}
	if baseCount != 0 && len(lenses) != 1 {
		add("", "base must be the sole activation lens", "remove contextual lenses or remove the base lens")
	}
	return diagnostics
}

func validateScopes(sourcePath string, scopes []string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if len(scopes) == 0 {
		diagnostics = append(diagnostics, diagnostic(sourcePath, "scopes", "at least one path scope is required", "add one or more repository-relative glob scopes"))
	}
	for index, scope := range scopes {
		if unsafeScope(scope) {
			diagnostics = append(diagnostics, diagnostic(
				sourcePath,
				fmt.Sprintf("scopes[%d]", index),
				fmt.Sprintf("unsafe or empty scope %q", scope),
				"use a repository-relative glob without path traversal",
			))
		}
	}
	return diagnostics
}

func validateDerivationEvidence(
	ctx context.Context,
	repo *workspace.Repository,
	sourcePath string,
	derivation string,
	evidence []Evidence,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(sourcePath, field, message, recovery))
	}
	if derivation != "extracted" && derivation != "inferred" {
		add("derivation", "derivation must be extracted or inferred", "record how the artifact was derived from repository evidence")
	}
	if len(evidence) == 0 {
		add("evidence", "at least one exact evidence citation is required", "cite eligible baseline lines and their excerpt hash")
	}
	seenLocations := make(map[string]struct{}, len(evidence))
	demonstrates := 0
	demonstratedFiles := make(map[string]struct{})
	hasAuthority := false
	for index, item := range evidence {
		field := fmt.Sprintf("evidence[%d]", index)
		switch item.Role {
		case "declares", "enforces":
			hasAuthority = true
		case "demonstrates":
			demonstrates++
			demonstratedFiles[item.Path] = struct{}{}
		default:
			add(field+".role", fmt.Sprintf("evidence role %q is not supported", item.Role), "use declares, demonstrates, or enforces")
		}
		diagnostics = append(diagnostics, validateEvidence(ctx, repo, sourcePath, field, item)...)
		location := item.Path + "\x00" + item.Lines
		if _, duplicate := seenLocations[location]; duplicate {
			add("evidence", fmt.Sprintf("duplicate evidence location %s:%s", item.Path, item.Lines), "cite each distinct occurrence once")
		}
		seenLocations[location] = struct{}{}
	}
	if derivation == "extracted" && !hasAuthority {
		add("evidence", "extracted artifacts require at least one declares or enforces citation", "cite the repository-maintained obligation or active enforcement mechanism")
	}
	if derivation == "inferred" && (demonstrates < 3 || len(demonstratedFiles) < 2) {
		add("evidence", "inferred artifacts require three demonstrates citations across at least two files", "add consistent implementation occurrences or remove the candidate")
	}
	return diagnostics
}

func loadVerificationRecipe(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	manifest ManifestArtifact,
) (VerificationRecipe, []Diagnostic, error) {
	data, diagnostics, err := readManifestArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return VerificationRecipe{}, diagnostics, err
	}
	var recipe VerificationRecipe
	if err := yaml.Load(data, &recipe, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return VerificationRecipe{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"use only fields from the ssb.dev/verification/v1 schema",
		)), nil
	}
	recipe.SourcePath = manifest.Path
	diagnostics = append(diagnostics, validateVerificationRecipe(ctx, evidenceRepo, recipe, manifest)...)
	return recipe, diagnostics, nil
}

func validateVerificationRecipe(
	ctx context.Context,
	repo *workspace.Repository,
	recipe VerificationRecipe,
	manifest ManifestArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(recipe.SourcePath, field, message, recovery))
	}
	if recipe.Schema != VerificationSchema {
		add("schema", "schema must be "+VerificationSchema, "update the verification recipe schema value")
	}
	if recipe.ID != manifest.ID {
		add("id", fmt.Sprintf("recipe id %q must match manifest id %q", recipe.ID, manifest.ID), "align the recipe id, filename, and report entry")
	}
	if strings.TrimSpace(recipe.Title) == "" {
		add("title", "title is required", "add a concise developer-facing title")
	}
	if _, supported := supportedCategories[recipe.Category]; recipe.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", recipe.Category), categoryRecovery)
	}
	diagnostics = append(diagnostics, validateActionableLenses(recipe.SourcePath, "lenses", recipe.Lenses)...)
	diagnostics = append(diagnostics, validateScopes(recipe.SourcePath, recipe.Scopes)...)
	diagnostics = append(diagnostics, validateDerivationEvidence(ctx, repo, recipe.SourcePath, recipe.Derivation, recipe.Evidence)...)
	if strings.TrimSpace(recipe.When) == "" {
		add("when", "when is required", "state the exact handoff context in which the recipe applies")
	}
	enforcesByRef := make(map[string]struct{})
	seenRefs := make(map[string]struct{})
	for index, evidence := range recipe.Evidence {
		field := fmt.Sprintf("evidence[%d].ref", index)
		if !stableIDPattern.MatchString(evidence.Ref) {
			add(field, "recipe evidence ref must be lower-case kebab-case", "give every evidence citation a stable ref")
		}
		if _, duplicate := seenRefs[evidence.Ref]; duplicate {
			add(field, fmt.Sprintf("duplicate evidence ref %q", evidence.Ref), "give every recipe evidence citation a unique ref")
		}
		seenRefs[evidence.Ref] = struct{}{}
		if evidence.Role == "enforces" {
			enforcesByRef[evidence.Ref] = struct{}{}
		}
	}
	if len(recipe.Steps) == 0 {
		add("steps", "verification recipe requires at least one ordered command", "record an existing deliberately invoked command")
	}
	for index, step := range recipe.Steps {
		field := fmt.Sprintf("steps[%d]", index)
		if strings.TrimSpace(step.Run) == "" {
			add(field+".run", "run is required", "record the exact existing repository command")
		}
		if _, exists := enforcesByRef[step.SourceEvidence]; !exists {
			add(
				field+".source_evidence",
				fmt.Sprintf("step references missing enforces evidence %q", step.SourceEvidence),
				"reference an evidence ref whose role is enforces",
			)
		}
		if strings.TrimSpace(step.ExpectedResult) == "" {
			add(field+".expected_result", "expected_result is required", "state the observable successful result")
		}
	}
	return diagnostics
}

func loadActionableSkill(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	reportPath string,
	manifestField string,
	manifest ManifestArtifact,
) (Skill, []Diagnostic, error) {
	data, diagnostics, err := readManifestArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return Skill{}, diagnostics, err
	}
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Skill{}, append(diagnostics, diagnostic(
			manifest.Path,
			"frontmatter",
			splitErr.Error(),
			"add portable Agent Skill YAML frontmatter",
		)), nil
	}
	var metadata skillFrontmatter
	if err := yaml.Load(frontmatter, &metadata, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return Skill{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"use only Agent Skills core specification fields",
		)), nil
	}
	diagnostics = append(diagnostics, validateDeprecatedSkillMetadata(manifest.Path, metadata)...)
	skill := Skill{
		ID:          manifest.ID,
		Description: metadata.Description,
		Category:    metadata.Metadata["category"],
		SourcePath:  manifest.Path,
		Body:        string(body),
	}
	if metadata.Name != manifest.ID {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"name",
			fmt.Sprintf("skill name %q must match manifest id %q", metadata.Name, manifest.ID),
			"align the skill name, directory, and report entry",
		))
	}
	if len(metadata.Name) > 64 {
		diagnostics = append(diagnostics, diagnostic(manifest.Path, "name", "skill name must be at most 64 characters", "shorten the portable skill name"))
	}
	if strings.TrimSpace(metadata.Description) == "" || len(metadata.Description) > 1024 {
		diagnostics = append(diagnostics, diagnostic(manifest.Path, "description", "skill description must contain 1-1024 characters", "describe what the skill does and when to use it"))
	}
	if skill.Category != manifest.Category {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"metadata.category",
			fmt.Sprintf("skill metadata.category %q must match manifest category %q", skill.Category, manifest.Category),
			"align the portable metadata and report manifest",
		))
	}
	if strings.TrimSpace(skill.Body) == "" {
		diagnostics = append(diagnostics, diagnostic(manifest.Path, "body", "skill body is required", "document the procedural workflow"))
	}
	diagnostics = append(
		diagnostics,
		prefixDiagnosticFields(validateDerivationEvidence(
			ctx,
			evidenceRepo,
			reportPath,
			manifest.Derivation,
			manifest.Evidence,
		), manifestField+".")...,
	)
	return skill, diagnostics, nil
}

func loadAutomationProposal(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	manifest ManifestArtifact,
) (AutomationProposal, []Diagnostic, error) {
	data, diagnostics, err := readManifestArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return AutomationProposal{}, diagnostics, err
	}
	var proposal AutomationProposal
	if err := yaml.Load(data, &proposal, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return AutomationProposal{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"use only fields from the ssb.dev/automation/v1 schema",
		)), nil
	}
	proposal.SourcePath = manifest.Path
	diagnostics = append(diagnostics, validateAutomationProposal(ctx, evidenceRepo, proposal, manifest)...)
	return proposal, diagnostics, nil
}

func validateAutomationProposal(
	ctx context.Context,
	repo *workspace.Repository,
	proposal AutomationProposal,
	manifest ManifestArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(proposal.SourcePath, field, message, recovery))
	}
	if proposal.Schema != AutomationSchema {
		add("schema", "schema must be "+AutomationSchema, "update the automation proposal schema value")
	}
	if proposal.ID != manifest.ID {
		add("id", fmt.Sprintf("automation id %q must match manifest id %q", proposal.ID, manifest.ID), "align the automation id, filename, and report entry")
	}
	if strings.TrimSpace(proposal.Title) == "" {
		add("title", "title is required", "add a concise developer-facing title")
	}
	if _, supported := supportedCategories[proposal.Category]; proposal.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", proposal.Category), categoryRecovery)
	}
	diagnostics = append(diagnostics, validateActionableLenses(proposal.SourcePath, "lenses", proposal.Lenses)...)
	diagnostics = append(diagnostics, validateScopes(proposal.SourcePath, proposal.Scopes)...)
	diagnostics = append(diagnostics, validateDerivationEvidence(ctx, repo, proposal.SourcePath, proposal.Derivation, proposal.Evidence)...)
	required := []struct {
		field string
		value string
	}{
		{"condition", proposal.Condition},
		{"suggested_check", proposal.SuggestedCheck},
		{"trigger", proposal.Trigger},
		{"expected_success", proposal.ExpectedSuccess},
		{"expected_failure", proposal.ExpectedFailure},
	}
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			add(item.field, item.field+" is required", "complete the reviewable automation design")
		}
	}
	return diagnostics
}

func readManifestArtifact(
	root string,
	manifest ManifestArtifact,
) ([]byte, []Diagnostic, error) {
	if component, found, err := findSymlinkComponent(root, manifest.Path); err != nil {
		return nil, nil, err
	} else if found {
		return nil, []Diagnostic{diagnostic(
			manifest.Path,
			"file",
			manifest.Path+" contains a symlink component "+component,
			"place the accepted artifact inside the repository",
		)}, nil
	}
	data, diagnostics, err := readRequiredRegularFile(
		filepath.Join(root, filepath.FromSlash(manifest.Path)),
		manifest.Path,
	)
	if len(data) == 0 && len(diagnostics) != 0 {
		for index := range diagnostics {
			diagnostics[index].Recovery = "remove its manifest entry or restore the artifact at the canonical path"
		}
	}
	return data, diagnostics, err
}

func findSymlinkComponent(root, relative string) (string, bool, error) {
	current := root
	for _, component := range strings.Split(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		if err != nil {
			return "", false, fmt.Errorf("inspect %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return filepath.ToSlash(current), true, nil
		}
	}
	return "", false, nil
}

func yamlDiagnostic(sourcePath string, err error, recovery string) Diagnostic {
	line := 1
	var loadError *yaml.LoadError
	if errors.As(err, &loadError) && loadError.Mark.Line > 0 {
		line = loadError.Mark.Line + 1
	}
	return Diagnostic{
		Path:     sourcePath,
		Line:     line,
		Field:    "frontmatter",
		Message:  err.Error(),
		Recovery: recovery,
	}
}

func unlistedNativeArtifacts(
	root string,
	listed map[string]ManifestArtifact,
) ([]string, []Diagnostic, error) {
	directories := []struct {
		relative string
		suffix   string
	}{
		{relative: ".software-standards/rules", suffix: ".md"},
		{relative: ".software-standards/verification", suffix: ".yaml"},
		{relative: ".software-standards/automation", suffix: ".yaml"},
	}
	unlisted := make([]string, 0)
	diagnostics := make([]Diagnostic, 0)
	for _, directory := range directories {
		absolute := filepath.Join(root, filepath.FromSlash(directory.relative))
		info, err := os.Lstat(absolute)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("inspect artifact directory %s: %w", directory.relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			diagnostics = append(diagnostics, diagnostic(
				directory.relative,
				"file",
				"artifact directory "+directory.relative+" must be a real directory, not a symlink",
				"replace it with a directory inside the repository",
			))
			continue
		}
		entries, err := os.ReadDir(absolute)
		if err != nil {
			return nil, nil, fmt.Errorf("read artifact directory %s: %w", directory.relative, err)
		}
		for _, entry := range entries {
			relative := path.Join(directory.relative, entry.Name())
			if entry.IsDir() {
				diagnostics = append(diagnostics, diagnostic(
					relative,
					"file",
					"native artifact directories must be flat",
					"move the artifact to the top-level canonical directory or remove the nested directory",
				))
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return nil, nil, fmt.Errorf("inspect artifact path %s: %w", relative, err)
			}
			if entry.Type()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				diagnostics = append(diagnostics, diagnostic(
					relative,
					"file",
					"native artifact paths must be real regular files",
					"replace the path with a canonical regular artifact file or remove it",
				))
				continue
			}
			if !strings.HasSuffix(entry.Name(), directory.suffix) {
				diagnostics = append(diagnostics, diagnostic(
					relative,
					"file",
					relative+" is not a supported artifact path",
					"use the canonical "+directory.suffix+" extension at the top level or remove the file",
				))
				continue
			}
			if _, exists := listed[relative]; !exists {
				unlisted = append(unlisted, relative)
			}
		}
	}
	sort.Strings(unlisted)
	return unlisted, diagnostics, nil
}

func parseActionableRuleBytes(relative string, data []byte) (Rule, *Diagnostic) {
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		item := diagnostic(relative, "frontmatter", err.Error(), "add strict YAML frontmatter between --- markers")
		return Rule{}, &item
	}
	var source semanticRuleSource
	if err := yaml.Load(frontmatter, &source, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		item := yamlDiagnostic(
			relative,
			err,
			"remove old proof-oriented fields and use the current ssb.dev/rule/v2 schema",
		)
		return Rule{}, &item
	}
	return Rule{
		Schema:     source.Schema,
		ID:         source.ID,
		Title:      source.Title,
		Category:   source.Category,
		Lenses:     source.Lenses,
		Directive:  source.Directive,
		Scopes:     source.Scopes,
		Derivation: source.Derivation,
		Evidence:   source.Evidence,
		SourcePath: relative,
		Body:       string(body),
	}, nil
}

// ValidateCandidateRule validates complete proposed semantic-rule bytes against
// the current commit without requiring the candidate to exist in the worktree.
func ValidateCandidateRule(
	ctx context.Context,
	repo *workspace.Repository,
	relative string,
	data []byte,
) (Rule, []Diagnostic) {
	rule, parseDiagnostic := parseActionableRuleBytes(relative, data)
	if parseDiagnostic != nil {
		return Rule{}, []Diagnostic{*parseDiagnostic}
	}
	manifestID := strings.TrimSuffix(path.Base(relative), path.Ext(relative))
	manifest := ManifestArtifact{ID: manifestID, Kind: "rule", Path: relative}
	return rule, validateActionableRule(ctx, repo, rule, manifest)
}

// ValidateRetainedRule validates a proposed update against the historical
// baseline owned by report.md and the rule's existing manifest entry.
func ValidateRetainedRule(
	ctx context.Context,
	repo *workspace.Repository,
	relative string,
	data []byte,
) (Rule, []Diagnostic, error) {
	rule, parseDiagnostic := parseActionableRuleBytes(relative, data)
	if parseDiagnostic != nil {
		return Rule{}, []Diagnostic{*parseDiagnostic}, nil
	}
	report, diagnostics, err := loadReport(repo)
	if err != nil || len(diagnostics) != 0 {
		return rule, diagnostics, err
	}
	manifest := ManifestArtifact{}
	for _, candidate := range report.Artifacts {
		if candidate.Kind == "rule" && candidate.Path == relative {
			manifest = candidate
			break
		}
	}
	if manifest.ID == "" {
		return rule, []Diagnostic{diagnostic(
			relative,
			"manifest",
			relative+" is not listed as a rule in .software-standards/report.md",
			"restore its report entry before proposing a retained-rule update",
		)}, nil
	}
	historical, historicalErr := repo.AtCommit(ctx, report.BaselineCommit)
	if historicalErr != nil {
		if !errors.Is(historicalErr, workspace.ErrHistoricalCommit) {
			return rule, nil, historicalErr
		}
		return rule, []Diagnostic{diagnostic(
			".software-standards/report.md",
			"baseline_commit",
			fmt.Sprintf(
				"recorded baseline_commit %q is not a reachable ancestor; retained-rule evidence cannot be verified: %v",
				report.BaselineCommit,
				historicalErr,
			),
			"restore the recorded baseline to current repository history or refresh the accepted pack through review",
		)}, nil
	}
	return rule, validateActionableRule(ctx, historical, rule, manifest), nil
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
	if err := yaml.Load(frontmatter, &report, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
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

func validateEvidence(ctx context.Context, repo *workspace.Repository, sourcePath, field string, evidence Evidence) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(sourcePath, field, message, recovery))
	}
	if evidence.Path == "" {
		add("evidence path is required", "cite a tracked regular file at the baseline commit")
		return diagnostics
	}
	if !digestPattern.MatchString(evidence.ExcerptSHA256) {
		add("excerpt_sha256 must use sha256:<64 lowercase hex characters>", "hash the exact cited bytes including line endings")
	}
	start, end, err := parseLineRange(evidence.Lines)
	if err != nil {
		add(err.Error(), "use a one-based range such as 12-18")
		return diagnostics
	}
	content, err := inventory.ReadEvidence(ctx, repo, evidence.Path)
	if err != nil {
		add(err.Error(), "cite an eligible tracked regular file listed by ssb inspect")
		return diagnostics
	}
	excerpt, err := lineExcerpt(content, start, end)
	if err != nil {
		add(err.Error(), "correct the cited line range")
		return diagnostics
	}
	sum := sha256.Sum256(excerpt)
	actual := "sha256:" + hex.EncodeToString(sum[:])
	if evidence.ExcerptSHA256 != actual {
		add(fmt.Sprintf("excerpt hash does not match %s lines %s; expected %s", evidence.Path, evidence.Lines, actual), "refresh the evidence after inspecting the exact baseline bytes")
	}
	return diagnostics
}

// ValidateCandidateSkill validates complete portable Agent Skill bytes without
// requiring the candidate to exist in the worktree. SSB provenance remains in
// report.md and is validated after the skill is accepted.
func ValidateCandidateSkill(relative, skillID string, data []byte) (Skill, []Diagnostic) {
	diagnostics := make([]Diagnostic, 0)
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Skill{}, append(diagnostics, diagnostic(relative, "frontmatter", splitErr.Error(), "add portable Agent Skill YAML frontmatter"))
	}
	var metadata skillFrontmatter
	if err := yaml.Load(frontmatter, &metadata, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return Skill{}, append(diagnostics, yamlDiagnostic(relative, err, "use only Agent Skills core specification fields"))
	}
	diagnostics = append(diagnostics, validateDeprecatedSkillMetadata(relative, metadata)...)
	if metadata.Name != skillID {
		diagnostics = append(diagnostics, diagnostic(relative, "name", fmt.Sprintf("skill name %q must match directory %q", metadata.Name, skillID), "align the skill name and directory"))
	}
	if len(metadata.Name) > 64 {
		diagnostics = append(diagnostics, diagnostic(relative, "name", "skill name must be at most 64 characters", "shorten the portable skill name"))
	}
	if strings.TrimSpace(metadata.Description) == "" || len(metadata.Description) > 1024 {
		diagnostics = append(diagnostics, diagnostic(relative, "description", "skill description must contain 1-1024 characters", "describe what the skill does and when to use it"))
	}
	category := metadata.Metadata["category"]
	if _, supported := supportedCategories[category]; category == "" || !supported {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"metadata.category",
			fmt.Sprintf("category %q is not supported", category),
			categoryRecovery,
		))
	}
	if strings.TrimSpace(string(body)) == "" {
		diagnostics = append(diagnostics, diagnostic(relative, "body", "skill body is required", "document the procedural workflow"))
	}
	return Skill{
		ID:          skillID,
		Description: metadata.Description,
		Category:    category,
		SourcePath:  relative,
		Body:        string(body),
	}, diagnostics
}

func validateDeprecatedSkillMetadata(relative string, metadata skillFrontmatter) []Diagnostic {
	if _, exists := metadata.Metadata["topic"]; !exists {
		return nil
	}
	return []Diagnostic{diagnostic(
		relative,
		"metadata.topic",
		"metadata.topic is not supported",
		"remove metadata.topic and use metadata.category",
	)}
}

func prefixDiagnosticFields(diagnostics []Diagnostic, prefix string) []Diagnostic {
	for index := range diagnostics {
		diagnostics[index].Field = prefix + diagnostics[index].Field
	}
	return diagnostics
}

func readRequiredRegularFile(absolute, relative string) ([]byte, []Diagnostic, error) {
	info, err := os.Lstat(absolute)
	if errors.Is(err, os.ErrNotExist) {
		return nil, []Diagnostic{diagnostic(relative, "file", relative+" does not exist", "create the required file and rerun ssb validate")}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("inspect %s: %w", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, []Diagnostic{diagnostic(relative, "file", relative+" must be a real regular file, not a symlink", "replace it with a file inside the repository")}, nil
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", relative, err)
	}
	if len(data) == 0 {
		return data, []Diagnostic{diagnostic(relative, "file", relative+" must not be empty", "write the required artifact and rerun ssb validate")}, nil
	}
	return data, nil, nil
}

func splitFrontmatter(data []byte) ([]byte, []byte, error) {
	var opening []byte
	var closing []byte
	switch {
	case bytes.HasPrefix(data, []byte("---\n")):
		opening = []byte("---\n")
		closing = []byte("\n---\n")
	case bytes.HasPrefix(data, []byte("---\r\n")):
		opening = []byte("---\r\n")
		closing = []byte("\r\n---\r\n")
	default:
		return nil, nil, errors.New("file must begin with --- on its own line")
	}
	rest := data[len(opening):]
	index := bytes.Index(rest, closing)
	if index < 0 {
		return nil, nil, errors.New("frontmatter closing --- marker is missing")
	}
	frontmatter := rest[:index]
	body := rest[index+len(closing):]
	return frontmatter, body, nil
}

func parseLineRange(value string) (int, int, error) {
	parts := strings.Split(value, "-")
	if len(parts) < 1 || len(parts) > 2 {
		return 0, 0, fmt.Errorf("invalid evidence line range %q", value)
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil || start < 1 {
		return 0, 0, fmt.Errorf("invalid evidence line range %q", value)
	}
	end := start
	if len(parts) == 2 {
		end, err = strconv.Atoi(parts[1])
		if err != nil || end < start {
			return 0, 0, fmt.Errorf("invalid evidence line range %q", value)
		}
	}
	return start, end, nil
}

func lineExcerpt(content []byte, start, end int) ([]byte, error) {
	lines := splitLines(content)
	if start > len(lines) || end > len(lines) {
		return nil, fmt.Errorf("line range %d-%d exceeds file length %d", start, end, len(lines))
	}
	var excerpt []byte
	for _, line := range lines[start-1 : end] {
		excerpt = append(excerpt, line...)
	}
	return excerpt, nil
}

func splitLines(content []byte) [][]byte {
	if len(content) == 0 {
		return nil
	}
	lines := make([][]byte, 0, bytes.Count(content, []byte{'\n'})+1)
	for len(content) > 0 {
		index := bytes.IndexByte(content, '\n')
		if index < 0 {
			lines = append(lines, content)
			break
		}
		lines = append(lines, content[:index+1])
		content = content[index+1:]
	}
	return lines
}

func unsafeScope(scope string) bool {
	if strings.TrimSpace(scope) == "" || strings.HasPrefix(scope, "/") || strings.HasPrefix(scope, "\\") {
		return true
	}
	normalized := strings.ReplaceAll(scope, "\\", "/")
	if len(normalized) >= 3 && normalized[1] == ':' &&
		((normalized[0] >= 'A' && normalized[0] <= 'Z') || (normalized[0] >= 'a' && normalized[0] <= 'z')) &&
		normalized[2] == '/' {
		return true
	}
	for _, component := range strings.Split(normalized, "/") {
		if component == ".." {
			return true
		}
	}
	return false
}

func diagnostic(filePath, field, message, recovery string) Diagnostic {
	return Diagnostic{
		Path:     filePath,
		Field:    field,
		Message:  message,
		Recovery: recovery,
	}
}

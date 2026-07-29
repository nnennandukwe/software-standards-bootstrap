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
	SchemaVersionV1    = "ssb.dev/rule/v1"
	SchemaVersionV2    = "ssb.dev/rule/v2"
	RuleSchema         = SchemaVersionV2
	VerificationSchema = "ssb.dev/verification/v1"
	AutomationSchema   = "ssb.dev/automation/v1"
	ScoreMethod        = "ssb-score-v1"
	UtilityMethod      = "ssb-utility-v1"
	topicRecovery      = "use one primary topic: architecture, compatibility, compliance, correctness, developer-experience, documentation, maintainability, operability, performance, quality, reliability, security, or testability"
)

var (
	stableIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	digestPattern   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	supportedTopics = map[string]struct{}{
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
	supportedTasks = map[string]struct{}{
		"implementation": {},
		"review":         {},
		"testing":        {},
		"security":       {},
		"documentation":  {},
		"release":        {},
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

// Score records transparent ranking arithmetic.
type Score struct {
	Method  string       `yaml:"method" json:"method"`
	Total   int          `yaml:"total" json:"total"`
	Factors ScoreFactors `yaml:"factors" json:"factors"`
}

// ScoreFactors are the bounded ssb-score-v1 inputs.
type ScoreFactors struct {
	Prevalence    int `yaml:"prevalence" json:"prevalence"`
	Consistency   int `yaml:"consistency" json:"consistency"`
	Authority     int `yaml:"authority" json:"authority"`
	Risk          int `yaml:"risk" json:"risk"`
	Applicability int `yaml:"applicability" json:"applicability"`
}

// Evidence points to an exact line range in the baseline commit.
type Evidence struct {
	Ref           string `yaml:"ref,omitempty" json:"ref,omitempty"`
	Role          string `yaml:"role,omitempty" json:"role,omitempty"`
	Path          string `yaml:"path" json:"path"`
	Lines         string `yaml:"lines" json:"lines"`
	ExcerptSHA256 string `yaml:"excerpt_sha256" json:"excerpt_sha256"`
	Authoritative bool   `yaml:"authoritative,omitempty" json:"authoritative,omitempty"`
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

// InventoryLimits and the following inventory types preserve the complete
// ssb-inventory-v2 coverage accounting inside report.md without coupling the
// report schema to inventory's JSON-only representation.
type InventoryLimits struct {
	MaxCandidateFiles int   `yaml:"max_candidate_files" json:"max_candidate_files"`
	MaxCandidateBytes int64 `yaml:"max_candidate_bytes" json:"max_candidate_bytes"`
	MaxFileBytes      int64 `yaml:"max_file_bytes" json:"max_file_bytes"`
}

type InventoryExclusions struct {
	Binary     int `yaml:"binary" json:"binary"`
	Generated  int `yaml:"generated" json:"generated"`
	Oversized  int `yaml:"oversized" json:"oversized"`
	SecretLike int `yaml:"secret_like" json:"secret_like"`
	Symlink    int `yaml:"symlink" json:"symlink"`
	Submodule  int `yaml:"submodule" json:"submodule"`
	VendorTree int `yaml:"vendor_or_generated_tree" json:"vendor_or_generated_tree"`
	NonRegular int `yaml:"non_regular" json:"non_regular"`
}

type InventoryFile struct {
	Path     string `yaml:"path" json:"path"`
	BlobOID  string `yaml:"blob_oid" json:"blob_oid"`
	Bytes    int64  `yaml:"bytes" json:"bytes"`
	Lines    int    `yaml:"lines" json:"lines"`
	Language string `yaml:"language,omitempty" json:"language,omitempty"`
	SHA256   string `yaml:"sha256" json:"sha256"`
}

type ReportInventory struct {
	SchemaVersion           int                 `yaml:"schema_version" json:"schema_version"`
	InventoryVersion        string              `yaml:"inventory_version" json:"inventory_version"`
	BaselineCommit          string              `yaml:"baseline_commit" json:"baseline_commit"`
	Limits                  InventoryLimits     `yaml:"limits" json:"limits"`
	CandidateFiles          int                 `yaml:"candidate_files" json:"candidate_files"`
	CandidateBytes          int64               `yaml:"candidate_bytes" json:"candidate_bytes"`
	ScannedFiles            int                 `yaml:"scanned_files" json:"scanned_files"`
	ScannedBytes            int64               `yaml:"scanned_bytes" json:"scanned_bytes"`
	IndexedFiles            int                 `yaml:"indexed_files" json:"indexed_files"`
	IndexedBytes            int64               `yaml:"indexed_bytes" json:"indexed_bytes"`
	Files                   []InventoryFile     `yaml:"files" json:"files"`
	Excluded                InventoryExclusions `yaml:"excluded" json:"excluded"`
	Truncated               bool                `yaml:"truncated" json:"truncated"`
	TruncationReason        string              `yaml:"truncation_reason,omitempty" json:"truncation_reason,omitempty"`
	RemainingCandidateFiles int                 `yaml:"remaining_candidate_files" json:"remaining_candidate_files"`
	RemainingCandidateBytes int64               `yaml:"remaining_candidate_bytes" json:"remaining_candidate_bytes"`
}

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

// Lens identifies one context dimension used to select a rule. Values within
// one kind are alternatives; represented kinds are matched together.
type Lens struct {
	Kind  string `yaml:"kind" json:"kind"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
}

// Verification either cites an existing repository check or records a proof
// gap. ssb never executes the command.
type Verification struct {
	Command  string    `yaml:"command,omitempty" json:"command,omitempty"`
	Source   *Evidence `yaml:"source,omitempty" json:"source,omitempty"`
	ProofGap string    `yaml:"proof_gap,omitempty" json:"proof_gap,omitempty"`
	Coverage string    `yaml:"coverage,omitempty" json:"coverage,omitempty"`
	Proves   string    `yaml:"proves,omitempty" json:"proves,omitempty"`
}

// Rule is one editable rule source file plus its exact Markdown body.
type Rule struct {
	Schema          string       `yaml:"schema" json:"schema"`
	ID              string       `yaml:"id" json:"id"`
	Title           string       `yaml:"title" json:"title"`
	Category        string       `yaml:"category,omitempty" json:"category,omitempty"`
	Topic           string       `yaml:"topic" json:"topic"`
	Lenses          []Lens       `yaml:"lenses,omitempty" json:"lenses,omitempty"`
	Directive       string       `yaml:"directive,omitempty" json:"directive,omitempty"`
	Scopes          []string     `yaml:"scopes" json:"scopes"`
	Derivation      string       `yaml:"derivation,omitempty" json:"derivation,omitempty"`
	Classification  string       `yaml:"classification" json:"classification"`
	Importance      string       `yaml:"importance" json:"importance"`
	Score           Score        `yaml:"score" json:"score"`
	Confidence      string       `yaml:"confidence" json:"confidence"`
	BaselineCommit  string       `yaml:"baseline_commit" json:"baseline_commit"`
	Evidence        []Evidence   `yaml:"evidence" json:"evidence"`
	Verification    Verification `yaml:"verification" json:"verification"`
	RelatedSkillIDs []string     `yaml:"related_skills,omitempty" json:"related_skills,omitempty"`

	SourcePath string `yaml:"-" json:"source_path"`
	Body       string `yaml:"-" json:"body"`
}

// Skill is a validated portable Agent Skill referenced by a rule.
type Skill struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Topic       string `json:"topic"`
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
	AssessmentPath string               `json:"assessment_path"`
	Assessment     string               `json:"assessment"`
	Rules          []Rule               `json:"rules"`
	Recipes        []VerificationRecipe `json:"verification_recipes,omitempty"`
	Skills         []Skill              `json:"skills"`
	Automations    []AutomationProposal `json:"automation_proposals,omitempty"`
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
	if reportExists(repo.Root()) {
		return validateActionablePack(ctx, repo, false)
	}
	return validatePack(ctx, repo, false)
}

// ValidateRetainedPack parses an adopted pack and verifies each rule against
// the historical baseline recorded in that rule. It is intended for
// review-aware post-application rendering and ADR creation; ordinary editable
// pack validation remains pinned to the repository's current HEAD.
func ValidateRetainedPack(ctx context.Context, repo *workspace.Repository) (Pack, []Diagnostic, error) {
	if reportExists(repo.Root()) {
		return validateActionablePack(ctx, repo, true)
	}
	return validatePack(ctx, repo, true)
}

func reportExists(root string) bool {
	_, err := os.Lstat(filepath.Join(root, ".software-standards", "report.md"))
	return err == nil
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

	for _, artifact := range pack.Report.Artifacts {
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

	unlisted, scanErr := unlistedNativeArtifacts(repo.Root(), entriesByPath)
	if scanErr != nil {
		return Pack{}, nil, scanErr
	}
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
	inventory := report.Inventory
	if inventory.SchemaVersion != 2 || inventory.InventoryVersion != "ssb-inventory-v2" {
		add("inventory", "inventory must preserve schema 2 ssb-inventory-v2 accounting", "copy the complete successful ssb inspect inventory")
	}
	if inventory.BaselineCommit != report.BaselineCommit {
		add("inventory.baseline_commit", "inventory baseline_commit must match the report baseline_commit", "copy one complete inventory for the report baseline")
	}
	if inventory.Truncated {
		add("inventory.truncated", "report inventory coverage must be complete", "rerun inspection with sufficient limits before producing artifacts")
	}
	if inventory.Limits.MaxCandidateFiles <= 0 ||
		inventory.Limits.MaxCandidateBytes <= 0 ||
		inventory.Limits.MaxFileBytes <= 0 {
		add("inventory.limits", "inventory limits must be positive", "copy the exact limits from ssb inspect")
	}
	if inventory.CandidateFiles != inventory.ScannedFiles+inventory.RemainingCandidateFiles ||
		inventory.CandidateBytes != inventory.ScannedBytes+inventory.RemainingCandidateBytes {
		add("inventory", "candidate, scanned, and remaining inventory accounting is inconsistent", "copy the complete inventory without editing its counts")
	}
	if inventory.IndexedFiles != len(inventory.Files) {
		add("inventory.indexed_files", "indexed_files must equal the number of inventory file records", "copy every indexed file record from ssb inspect")
	}
	var indexedBytes int64
	for _, file := range inventory.Files {
		indexedBytes += file.Bytes
	}
	if inventory.IndexedBytes != indexedBytes {
		add("inventory.indexed_bytes", "indexed_bytes must equal the sum of inventory file bytes", "copy the complete inventory without editing its byte counts")
	}
	return diagnostics
}

func validateManifestArtifact(sourcePath, field string, artifact ManifestArtifact) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(sourcePath, field+suffix, message, recovery))
	}
	if !stableIDPattern.MatchString(artifact.ID) {
		add(".id", "artifact id must be lower-case kebab-case", "choose a stable id such as verify-test-migrations")
	}
	expectedPath := ""
	switch artifact.Kind {
	case "rule":
		expectedPath = path.Join(".software-standards/rules", artifact.ID+".md")
	case "verification":
		expectedPath = path.Join(".software-standards/verification", artifact.ID+".yaml")
	case "skill":
		expectedPath = path.Join(".agents/skills", artifact.ID, "SKILL.md")
	case "automation":
		expectedPath = path.Join(".software-standards/automation", artifact.ID+".yaml")
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
		if _, supported := supportedTopics[artifact.Category]; artifact.Category == "" || !supported {
			add(".category", fmt.Sprintf("category %q is not supported", artifact.Category), strings.ReplaceAll(topicRecovery, "topic", "category"))
		}
		diagnostics = append(diagnostics, validateActionableLenses(sourcePath, field+".lenses", artifact.Lenses)...)
		diagnostics = append(diagnostics, validateScopes(sourcePath, artifact.Scopes)...)
	}
	return diagnostics
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
	data, diagnostics, err := readRequiredRegularFile(
		filepath.Join(root, filepath.FromSlash(manifest.Path)),
		manifest.Path,
	)
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
		Confidence: manifest.Confidence,
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
	if _, supported := supportedTopics[rule.Category]; rule.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", rule.Category), strings.ReplaceAll(topicRecovery, "topic", "category"))
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
	if _, supported := supportedTopics[recipe.Category]; recipe.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", recipe.Category), strings.ReplaceAll(topicRecovery, "topic", "category"))
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
	diagnostics = append(diagnostics, validateDerivationEvidence(
		ctx,
		evidenceRepo,
		manifest.Path,
		manifest.Derivation,
		manifest.Evidence,
	)...)
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
	if _, supported := supportedTopics[proposal.Category]; proposal.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", proposal.Category), strings.ReplaceAll(topicRecovery, "topic", "category"))
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

func unlistedNativeArtifacts(root string, listed map[string]ManifestArtifact) ([]string, error) {
	directories := []struct {
		relative string
		suffix   string
	}{
		{relative: ".software-standards/rules", suffix: ".md"},
		{relative: ".software-standards/verification", suffix: ".yaml"},
		{relative: ".software-standards/automation", suffix: ".yaml"},
	}
	unlisted := make([]string, 0)
	for _, directory := range directories {
		absolute := filepath.Join(root, filepath.FromSlash(directory.relative))
		info, err := os.Lstat(absolute)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect artifact directory %s: %w", directory.relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("artifact directory %s must be a real directory", directory.relative)
		}
		entries, err := os.ReadDir(absolute)
		if err != nil {
			return nil, fmt.Errorf("read artifact directory %s: %w", directory.relative, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), directory.suffix) {
				continue
			}
			relative := path.Join(directory.relative, entry.Name())
			if _, exists := listed[relative]; !exists {
				unlisted = append(unlisted, relative)
			}
		}
	}
	sort.Strings(unlisted)
	return unlisted, nil
}

func validatePack(
	ctx context.Context,
	repo *workspace.Repository,
	retained bool,
) (Pack, []Diagnostic, error) {
	pack := Pack{
		BaselineCommit: repo.Baseline(),
		AssessmentPath: ".software-standards/assessment.md",
		Rules:          make([]Rule, 0),
		Skills:         make([]Skill, 0),
	}
	diagnostics := make([]Diagnostic, 0)

	packRootPath := filepath.Join(repo.Root(), ".software-standards")
	if info, err := os.Lstat(packRootPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			diagnostics = append(diagnostics, diagnostic(".software-standards", "file", ".software-standards must be a real directory, not a symlink", "move the editable pack inside the repository"))
			return pack, diagnostics, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Pack{}, nil, fmt.Errorf("inspect .software-standards: %w", err)
	}

	assessmentPath := filepath.Join(repo.Root(), filepath.FromSlash(pack.AssessmentPath))
	assessment, assessmentDiagnostics, err := readRequiredRegularFile(assessmentPath, pack.AssessmentPath)
	if err != nil {
		return Pack{}, nil, err
	}
	diagnostics = append(diagnostics, assessmentDiagnostics...)
	if len(assessment) != 0 {
		pack.Assessment = string(assessment)
		if strings.TrimSpace(pack.Assessment) == "" {
			diagnostics = append(diagnostics, diagnostic(pack.AssessmentPath, "assessment", "assessment must not be empty", "record repository context and candidate rationale"))
		}
	}

	rulesDirPath := filepath.Join(repo.Root(), ".software-standards", "rules")
	info, err := os.Lstat(rulesDirPath)
	if errors.Is(err, os.ErrNotExist) {
		diagnostics = append(diagnostics, diagnostic(".software-standards/rules", "rules", "rule directory does not exist", "create one Markdown file per retained rule"))
		return pack, diagnostics, nil
	}
	if err != nil {
		return Pack{}, nil, fmt.Errorf("inspect rule directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		diagnostics = append(diagnostics, diagnostic(".software-standards/rules", "rules", "rule directory must be a real directory, not a symlink", "replace it with a directory inside the repository"))
		return pack, diagnostics, nil
	}
	entries, err := os.ReadDir(rulesDirPath)
	if err != nil {
		return Pack{}, nil, fmt.Errorf("read rule directory: %w", err)
	}
	ruleFiles := make([]string, 0)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			ruleFiles = append(ruleFiles, entry.Name())
		}
	}
	sort.Strings(ruleFiles)
	if len(ruleFiles) == 0 {
		diagnostics = append(diagnostics, diagnostic(".software-standards/rules", "rules", "no rule files found", "retain at least one scored candidate or keep candidates in the assessment only"))
		return pack, diagnostics, nil
	}

	skillCache := make(map[string]Skill)
	seenRuleIDs := make(map[string]string)
	for _, fileName := range ruleFiles {
		relative := path.Join(".software-standards/rules", fileName)
		absolute := filepath.Join(rulesDirPath, fileName)
		data, fileDiagnostics, readErr := readRequiredRegularFile(absolute, relative)
		if readErr != nil {
			return Pack{}, nil, readErr
		}
		diagnostics = append(diagnostics, fileDiagnostics...)
		if len(data) == 0 {
			continue
		}
		rule, parseDiagnostic := parseRule(relative, data)
		if parseDiagnostic != nil {
			diagnostics = append(diagnostics, *parseDiagnostic)
			continue
		}
		pack.Rules = append(pack.Rules, rule)
		if retained {
			retainedDiagnostics, retainedErr := validateRetainedRule(ctx, repo, rule, fileName)
			if retainedErr != nil {
				return Pack{}, nil, retainedErr
			}
			diagnostics = append(diagnostics, retainedDiagnostics...)
		} else {
			diagnostics = append(diagnostics, validateRule(ctx, repo, rule, fileName)...)
		}

		if prior, exists := seenRuleIDs[rule.ID]; exists {
			diagnostics = append(diagnostics, diagnostic(relative, "id", fmt.Sprintf("duplicate rule id %q also used by %s", rule.ID, prior), "give every rule a stable unique id"))
		} else {
			seenRuleIDs[rule.ID] = relative
		}
		for _, skillID := range rule.RelatedSkillIDs {
			if _, loaded := skillCache[skillID]; loaded {
				continue
			}
			skill, skillDiagnostics, loadErr := loadSkill(repo.Root(), skillID)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, skillDiagnostics...)
			if skill.ID != "" {
				skillCache[skillID] = skill
			}
		}
	}
	for _, skill := range skillCache {
		pack.Skills = append(pack.Skills, skill)
	}
	sort.Slice(pack.Skills, func(i, j int) bool { return pack.Skills[i].ID < pack.Skills[j].ID })
	return pack, diagnostics, nil
}

func parseRule(relative string, data []byte) (Rule, *Diagnostic) {
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		item := diagnostic(relative, "frontmatter", err.Error(), "add strict YAML frontmatter between --- markers")
		return Rule{}, &item
	}
	var rule Rule
	if err := yaml.Load(frontmatter, &rule, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		line := 1
		var loadError *yaml.LoadError
		if errors.As(err, &loadError) && loadError.Mark.Line > 0 {
			line = loadError.Mark.Line + 1
		}
		item := Diagnostic{
			Path:     relative,
			Line:     line,
			Field:    "frontmatter",
			Message:  err.Error(),
			Recovery: "remove unknown or duplicate fields and use the ssb.dev/rule/v1 or ssb.dev/rule/v2 schema",
		}
		return Rule{}, &item
	}
	rule.SourcePath = relative
	rule.Body = string(body)
	return rule, nil
}

// ValidateCandidateRule validates complete proposed rule bytes against the
// current commit without requiring the candidate to exist in the worktree.
func ValidateCandidateRule(
	ctx context.Context,
	repo *workspace.Repository,
	relative string,
	data []byte,
) (Rule, []Diagnostic) {
	rule, parseDiagnostic := parseRule(relative, data)
	if parseDiagnostic != nil {
		return Rule{}, []Diagnostic{*parseDiagnostic}
	}
	return rule, validateRule(ctx, repo, rule, path.Base(relative))
}

// ValidateRetainedRule validates an adopted rule against the historical
// baseline explicitly recorded in its frontmatter.
func ValidateRetainedRule(
	ctx context.Context,
	repo *workspace.Repository,
	relative string,
	data []byte,
) (Rule, []Diagnostic, error) {
	rule, parseDiagnostic := parseRule(relative, data)
	if parseDiagnostic != nil {
		return Rule{}, []Diagnostic{*parseDiagnostic}, nil
	}
	diagnostics, err := validateRetainedRule(ctx, repo, rule, path.Base(relative))
	return rule, diagnostics, err
}

func validateRetainedRule(
	ctx context.Context,
	repo *workspace.Repository,
	rule Rule,
	fileName string,
) ([]Diagnostic, error) {
	historical, err := repo.AtCommit(ctx, rule.BaselineCommit)
	if err != nil {
		if !errors.Is(err, workspace.ErrHistoricalCommit) {
			return nil, err
		}
		return []Diagnostic{diagnostic(
			rule.SourcePath,
			"baseline_commit",
			fmt.Sprintf(
				"recorded baseline_commit %q is not a reachable ancestor; retained-rule evidence cannot be verified: %v",
				rule.BaselineCommit,
				err,
			),
			"restore the recorded baseline to current repository history or update this rule through a new approved prune review",
		)}, nil
	}
	return validateRule(ctx, historical, rule, fileName), nil
}

func validateRule(ctx context.Context, repo *workspace.Repository, rule Rule, fileName string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(rule.SourcePath, field, message, recovery))
	}

	if rule.Schema != SchemaVersionV1 && rule.Schema != SchemaVersionV2 {
		add("schema", fmt.Sprintf("schema must be %s or %s", SchemaVersionV1, SchemaVersionV2), "update the rule schema value")
	}
	if rule.Schema == SchemaVersionV1 {
		if len(rule.Lenses) != 0 || rule.Directive != "" ||
			rule.Verification.Coverage != "" || rule.Verification.Proves != "" {
			add("schema", "rule v1 must not declare v2 activation or proof metadata", "remove lenses, directive, coverage, and proves or update schema to ssb.dev/rule/v2")
		}
	} else if rule.Schema == SchemaVersionV2 {
		diagnostics = append(diagnostics, validateRuleV2Activation(rule)...)
	}
	if !stableIDPattern.MatchString(rule.ID) {
		add("id", "id must be lower-case kebab-case", "choose a stable id such as verify-before-merge")
	}
	if fileName != rule.ID+".md" {
		add("id", fmt.Sprintf("filename %q must match rule id %q", fileName, rule.ID+".md"), "rename the source file or correct its id")
	}
	if strings.TrimSpace(rule.Title) == "" {
		add("title", "title is required", "add a concise developer-facing title")
	}
	if rule.Topic == "" {
		add("topic", "topic is required", topicRecovery)
	} else if _, supported := supportedTopics[rule.Topic]; !supported {
		add("topic", fmt.Sprintf("topic %q is not supported", rule.Topic), topicRecovery)
	}
	if strings.TrimSpace(rule.Body) == "" {
		add("body", "rule body is required", "write the exact guidance that should be projected")
	}
	if len(rule.Scopes) == 0 {
		add("scopes", "at least one path scope is required", "add one or more repository-relative glob scopes")
	}
	for _, scope := range rule.Scopes {
		if unsafeScope(scope) {
			add("scopes", fmt.Sprintf("unsafe or empty scope %q", scope), "use a repository-relative glob without path traversal")
		}
	}
	if rule.Classification != "guidance" && rule.Classification != "deterministic" {
		add("classification", "classification must be guidance or deterministic", "classify judgment separately from existing deterministic proof")
	}
	if rule.Confidence != "low" && rule.Confidence != "medium" && rule.Confidence != "high" {
		add("confidence", "confidence must be low, medium, or high", "record an honest confidence band")
	}
	if rule.BaselineCommit != repo.Baseline() {
		add("baseline_commit", fmt.Sprintf("baseline_commit must equal current HEAD %s", repo.Baseline()), "reinspect the new commit and refresh every evidence hash")
	}
	diagnostics = append(diagnostics, validateScore(rule)...)

	authoritative := false
	evidencePaths := make(map[string]struct{})
	evidenceLocations := make(map[string]struct{})
	uniqueEvidenceCount := 0
	if len(rule.Evidence) == 0 {
		add("evidence", "at least one evidence location is required", "cite exact baseline lines and their excerpt hash")
	}
	for index, evidence := range rule.Evidence {
		field := fmt.Sprintf("evidence[%d]", index)
		diagnostics = append(diagnostics, validateEvidence(ctx, repo, rule.SourcePath, field, evidence)...)
		authoritative = authoritative || evidence.Authoritative
		evidencePaths[evidence.Path] = struct{}{}
		location := evidence.Path + "\x00" + evidence.Lines
		if _, duplicate := evidenceLocations[location]; duplicate {
			add("evidence", fmt.Sprintf("duplicate evidence location %s:%s", evidence.Path, evidence.Lines), "cite each distinct occurrence once")
		} else {
			evidenceLocations[location] = struct{}{}
			uniqueEvidenceCount++
		}
	}
	if !authoritative && (uniqueEvidenceCount < 3 || len(evidencePaths) < 2) {
		add("evidence", "candidate requires one authoritative source or three occurrences across two files", "add consistent evidence or keep the candidate in assessment.md")
	}

	hasCommand := strings.TrimSpace(rule.Verification.Command) != ""
	hasSource := rule.Verification.Source != nil
	hasGap := strings.TrimSpace(rule.Verification.ProofGap) != ""
	hasCoverage := strings.TrimSpace(rule.Verification.Coverage) != ""
	hasProves := strings.TrimSpace(rule.Verification.Proves) != ""
	if rule.Classification == "deterministic" {
		if !hasCommand || !hasSource || hasGap {
			add("verification", "deterministic rules must cite an existing verification command and source without a proof gap", "cite the existing check or reclassify the rule as guidance")
		}
	} else if rule.Classification == "guidance" {
		if hasGap == (hasCommand || hasSource) {
			add("verification", "guidance must cite one existing check or one explicit proof gap", "set command plus source, or set proof_gap")
		}
		if hasCommand != hasSource {
			add("verification", "verification command and source must be provided together", "cite the repository location that defines the command")
		}
	}
	if rule.Schema == SchemaVersionV2 {
		switch {
		case hasGap:
			if hasCoverage || hasProves {
				add("verification", "proof gaps must not declare verification coverage or a proved property", "remove coverage and proves from the proof gap")
			}
		case hasCommand && hasSource:
			if !hasProves {
				add("verification.proves", "mapped verification must state the bounded property it proves", "describe only the property established when the cited command passes")
			}
			if rule.Classification == "deterministic" && rule.Verification.Coverage != "full" {
				add("verification.coverage", "deterministic rules require full verification coverage", "set coverage to full only when the cited check proves the complete rule")
			}
			if rule.Classification == "guidance" && rule.Verification.Coverage != "partial" {
				add("verification.coverage", "guidance with a mapped check requires partial verification coverage", "set coverage to partial or replace the mapping with a proof gap")
			}
		}
	}
	if hasSource {
		diagnostics = append(diagnostics, validateEvidence(ctx, repo, rule.SourcePath, "verification.source", *rule.Verification.Source)...)
	}

	seenSkills := make(map[string]struct{})
	for _, skillID := range rule.RelatedSkillIDs {
		if !stableIDPattern.MatchString(skillID) {
			add("related_skills", fmt.Sprintf("invalid related skill id %q", skillID), "use lower-case kebab-case skill directory names")
		}
		if _, duplicate := seenSkills[skillID]; duplicate {
			add("related_skills", fmt.Sprintf("duplicate related skill id %q", skillID), "list each related skill once")
		}
		seenSkills[skillID] = struct{}{}
	}
	return diagnostics
}

func validateRuleV2Activation(rule Rule) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(rule.SourcePath, field, message, recovery))
	}
	if len(rule.Lenses) == 0 {
		add("lenses", "at least one activation lens is required", "use one base lens or one or more language, framework, and task lenses")
	}
	baseCount := 0
	seen := make(map[string]struct{})
	for index, lens := range rule.Lenses {
		field := fmt.Sprintf("lenses[%d]", index)
		switch lens.Kind {
		case "base":
			baseCount++
			if lens.Value != "" {
				add(field+".value", "base lens must not have a value", "remove the value from the base lens")
			}
		case "language", "framework":
			if !stableIDPattern.MatchString(lens.Value) {
				add(field+".value", lens.Kind+" lens requires a lower-case kebab-case value", "add a value such as go, python, cobra, or django")
			}
		case "task":
			if _, supported := supportedTasks[lens.Value]; !supported {
				add(field+".value", fmt.Sprintf("task lens value %q is not supported", lens.Value), "use implementation, review, testing, security, documentation, or release")
			}
		default:
			add(field+".kind", fmt.Sprintf("lens kind %q is not supported", lens.Kind), "use base, language, framework, or task")
		}
		key := lens.Kind + ":" + lens.Value
		if _, duplicate := seen[key]; duplicate {
			add("lenses", "duplicate activation lens "+key, "list each lens once")
		}
		seen[key] = struct{}{}
	}
	if baseCount != 0 && len(rule.Lenses) != 1 {
		add("lenses", "base must be the sole activation lens", "remove contextual lenses or remove the base lens")
	}
	if _, supported := supportedDirectives[rule.Directive]; !supported {
		add("directive", fmt.Sprintf("directive %q is not supported", rule.Directive), "use always, ask-first, never, or prefer")
	}
	return diagnostics
}

func validateScore(rule Rule) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(rule.SourcePath, field, message, recovery))
	}
	if rule.Score.Method != ScoreMethod {
		add("score.method", fmt.Sprintf("score method must be %s", ScoreMethod), "use the versioned v0.1 ranking method")
	}
	factors := []struct {
		name  string
		value int
		max   int
	}{
		{"prevalence", rule.Score.Factors.Prevalence, 25},
		{"consistency", rule.Score.Factors.Consistency, 20},
		{"authority", rule.Score.Factors.Authority, 20},
		{"risk", rule.Score.Factors.Risk, 20},
		{"applicability", rule.Score.Factors.Applicability, 15},
	}
	sum := 0
	for _, factor := range factors {
		sum += factor.value
		if factor.value < 0 || factor.value > factor.max {
			add("score.factors."+factor.name, fmt.Sprintf("%s score %d is outside 0-%d", factor.name, factor.value, factor.max), "correct the factor within its documented range")
		}
	}
	if rule.Score.Total != sum {
		add("score.total", fmt.Sprintf("score total %d does not equal factor sum %d", rule.Score.Total, sum), "recalculate the transparent score")
	}
	expectedBand := importanceBand(rule.Score.Total)
	if expectedBand == "assessment-only" {
		add("score.total", fmt.Sprintf("score %d is below the 25-point rule threshold", rule.Score.Total), "keep this candidate in assessment.md instead of emitting a rule")
	} else if rule.Importance != expectedBand {
		add("importance", fmt.Sprintf("importance %s does not match score %d (%s)", rule.Importance, rule.Score.Total, expectedBand), "use the band determined by ssb-score-v1")
	}
	return diagnostics
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

func loadSkill(root, skillID string) (Skill, []Diagnostic, error) {
	relative := path.Join(".agents/skills", skillID, "SKILL.md")
	if !stableIDPattern.MatchString(skillID) {
		return Skill{}, nil, nil
	}
	if component, found, err := findSymlinkComponent(root, relative); err != nil {
		return Skill{}, nil, err
	} else if found {
		return Skill{}, []Diagnostic{diagnostic(relative, "file", relative+" contains a symlink component "+component, "place the portable skill inside the repository")}, nil
	}
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	data, diagnostics, err := readRequiredRegularFile(absolute, relative)
	if err != nil || len(data) == 0 {
		return Skill{}, diagnostics, err
	}
	skill, parsed := validateSkillBytes(relative, skillID, data)
	return skill, append(diagnostics, parsed...), nil
}

// ValidateCandidateSkill validates complete proposed Agent Skill bytes without
// requiring the candidate to exist in the worktree.
func ValidateCandidateSkill(relative, skillID string, data []byte) (Skill, []Diagnostic) {
	return validateSkillBytes(relative, skillID, data)
}

func validateSkillBytes(relative, skillID string, data []byte) (Skill, []Diagnostic) {
	diagnostics := make([]Diagnostic, 0)
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Skill{}, append(diagnostics, diagnostic(relative, "frontmatter", splitErr.Error(), "add Agent Skill YAML frontmatter"))
	}
	var metadata skillFrontmatter
	if err := yaml.Load(frontmatter, &metadata, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return Skill{}, append(diagnostics, diagnostic(relative, "frontmatter", err.Error(), "use only Agent Skills core specification fields"))
	}
	if metadata.Name != skillID {
		diagnostics = append(diagnostics, diagnostic(relative, "name", fmt.Sprintf("skill name %q must match directory %q", metadata.Name, skillID), "align the skill name and directory"))
	}
	if len(metadata.Name) > 64 {
		diagnostics = append(diagnostics, diagnostic(relative, "name", "skill name must be at most 64 characters", "shorten the portable skill name"))
	}
	if strings.TrimSpace(metadata.Description) == "" || len(metadata.Description) > 1024 {
		diagnostics = append(diagnostics, diagnostic(relative, "description", "skill description must contain 1-1024 characters", "describe what the skill does and when to use it"))
	}
	topic := metadata.Metadata["topic"]
	if topic == "" {
		diagnostics = append(diagnostics, diagnostic(relative, "metadata.topic", "topic is required", topicRecovery))
	} else if _, supported := supportedTopics[topic]; !supported {
		diagnostics = append(diagnostics, diagnostic(relative, "metadata.topic", fmt.Sprintf("topic %q is not supported", topic), topicRecovery))
	}
	if strings.TrimSpace(string(body)) == "" {
		diagnostics = append(diagnostics, diagnostic(relative, "body", "skill body is required", "document the procedural workflow"))
	}
	return Skill{
		ID:          skillID,
		Description: metadata.Description,
		Topic:       topic,
		SourcePath:  relative,
		Body:        string(body),
	}, diagnostics
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

func importanceBand(score int) string {
	switch {
	case score >= 80 && score <= 100:
		return "very-high"
	case score >= 65 && score <= 79:
		return "high"
	case score >= 45 && score <= 64:
		return "medium"
	case score >= 25 && score <= 44:
		return "low"
	default:
		return "assessment-only"
	}
}

func diagnostic(filePath, field, message, recovery string) Diagnostic {
	return Diagnostic{
		Path:     filePath,
		Field:    field,
		Message:  message,
		Recovery: recovery,
	}
}

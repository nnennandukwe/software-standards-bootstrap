package rulepack

import (
	"regexp"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
)

// Layout identifies how an actionable standards pack stores machine metadata.
type Layout string

const (
	ManifestSchema                = "ssb.dev/manifest/v1"
	ReportSchema                  = "ssb.dev/report/v1"
	RuleSchema                    = "ssb.dev/rule/v2"
	VerificationSchemaV1          = "ssb.dev/verification/v1"
	VerificationSchemaV2          = "ssb.dev/verification/v2"
	VerificationSchema            = VerificationSchemaV1
	AutomationSchema              = "ssb.dev/automation/v1"
	UtilityMethod                 = "ssb-utility-v1"
	RoutingCatalogPath            = ".software-standards/routing/catalog.md"
	RoutingBundleDirectory        = ".software-standards/routing/bundles"
	RootCoreMaxRules              = 16
	LayoutManifest         Layout = "manifest"
	LayoutEmbedded         Layout = "embedded"
	categoryRecovery              = "use one primary category: architecture, compatibility, compliance, correctness, developer-experience, documentation, maintainability, operability, performance, quality, reliability, security, or testability"
)

var (
	stableIDPattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	digestPattern       = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	commitPattern       = regexp.MustCompile(`^[0-9a-f]{40}$`)
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

// AcceptedArtifact is one accepted output indexed by a pack.
type AcceptedArtifact struct {
	ID                 string     `yaml:"id" json:"id"`
	Kind               string     `yaml:"kind" json:"kind"`
	Path               string     `yaml:"path" json:"path"`
	SHA256             string     `yaml:"sha256,omitempty" json:"sha256,omitempty"`
	Confidence         string     `yaml:"confidence" json:"confidence"`
	Utility            Utility    `yaml:"utility" json:"utility"`
	RelatedArtifactIDs []string   `yaml:"related_artifacts,omitempty" json:"related_artifacts,omitempty"`
	Category           string     `yaml:"category,omitempty" json:"category,omitempty"`
	Lenses             []Lens     `yaml:"lenses,omitempty" json:"lenses,omitempty"`
	Directive          string     `yaml:"directive,omitempty" json:"directive,omitempty"`
	Scopes             []string   `yaml:"scopes,omitempty" json:"scopes,omitempty"`
	Derivation         string     `yaml:"derivation,omitempty" json:"derivation,omitempty"`
	Evidence           []Evidence `yaml:"evidence,omitempty" json:"evidence,omitempty"`
}

// FileReference binds one canonical manifest-layout file to its exact raw bytes.
type FileReference struct {
	Path   string `yaml:"path" json:"path"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// IsZero lets validation JSON omit manifest-layout file references for
// embedded packs while preserving one normalized manifest shape.
func (reference FileReference) IsZero() bool {
	return reference.Path == "" && reference.SHA256 == ""
}

// Manifest owns manifest-layout machine metadata. Embedded reports are
// normalized into the same shape without separate file references.
type Manifest struct {
	Schema         string             `yaml:"schema" json:"schema"`
	BaselineCommit string             `yaml:"baseline_commit" json:"baseline_commit"`
	Inventory      FileReference      `yaml:"inventory,omitempty" json:"inventory,omitzero"`
	Report         FileReference      `yaml:"report,omitempty" json:"report,omitzero"`
	Orientation    FileReference      `yaml:"orientation,omitempty" json:"orientation,omitzero"`
	RootCore       []string           `yaml:"root_core,omitempty" json:"root_core,omitempty"`
	Artifacts      []AcceptedArtifact `yaml:"artifacts" json:"artifacts"`
}

// RoutingCatalog is the normalized, portable route index for a manifest pack.
// The renderer projects it without reinterpreting repository metadata.
type RoutingCatalog struct {
	CatalogPath string          `json:"catalog_path"`
	Bundles     []RoutingBundle `json:"bundles"`
}

// RoutingBundle groups actionable artifacts with one exact normalized selector.
type RoutingBundle struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Lenses      []Lens   `json:"lenses"`
	Scopes      []string `json:"scopes"`
	ArtifactIDs []string `json:"artifact_ids"`
}

// HumanReport is the human-facing report document after layout detection.
type HumanReport struct {
	Body string `json:"body"`
}

// Report is the accepted artifact index and run narrative.
type Report struct {
	Schema         string             `yaml:"schema" json:"schema"`
	BaselineCommit string             `yaml:"baseline_commit" json:"baseline_commit"`
	Inventory      ReportInventory    `yaml:"inventory" json:"inventory"`
	Artifacts      []AcceptedArtifact `yaml:"artifacts" json:"artifacts"`
	Body           string             `yaml:"-" json:"body"`
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
	Run              string `yaml:"run" json:"run"`
	WorkingDirectory string `yaml:"working_directory" json:"working_directory"`
	SourceEvidence   string `yaml:"source_evidence" json:"source_evidence"`
	ExpectedResult   string `yaml:"expected_result" json:"expected_result"`
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
	Layout          Layout               `json:"layout"`
	BaselineCommit  string               `json:"baseline_commit"`
	ManifestPath    string               `json:"manifest_path,omitempty"`
	InventoryPath   string               `json:"inventory_path,omitempty"`
	ReportPath      string               `json:"report_path,omitempty"`
	OrientationPath string               `json:"orientation_path,omitempty"`
	Manifest        Manifest             `json:"manifest"`
	Orientation     *Orientation         `json:"orientation,omitempty"`
	Inventory       ReportInventory      `json:"inventory"`
	HumanReport     HumanReport          `json:"report"`
	Report          Report               `json:"-"`
	Routing         *RoutingCatalog      `json:"routing,omitempty"`
	Rules           []Rule               `json:"rules"`
	Recipes         []VerificationRecipe `json:"verification_recipes"`
	Skills          []Skill              `json:"skills"`
	Automations     []AutomationProposal `json:"automation_proposals"`
}

type skillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
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

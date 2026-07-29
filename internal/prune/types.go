// Package prune owns the governed lifecycle review protocol for adopted rules
// and repository Agent Skills.
package prune

import (
	"errors"
	"fmt"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"go.yaml.in/yaml/v4"
)

var (
	// ErrValidation identifies malformed or internally inconsistent review
	// evidence that the developer or host agent must correct.
	ErrValidation = errors.New("prune validation failed")
	// ErrPrecondition identifies a valid operation attempted in the wrong
	// review or repository state.
	ErrPrecondition = errors.New("prune precondition failed")
)

func validationError(format string, values ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, values...))
}

func preconditionError(format string, values ...any) error {
	return fmt.Errorf("%w: %s", ErrPrecondition, fmt.Sprintf(format, values...))
}

const (
	ContextSchema    = "ssb.dev/prune-context/v1"
	CapabilitySchema = "ssb.dev/capability-profile/v1"
	ProvenanceSchema = "ssb.dev/artifact-provenance/v1"
	ProposalSchema   = "ssb.dev/prune-proposal/v1"
	EventSchema      = "ssb.dev/prune-event/v1"
	CheckSchema      = "ssb.dev/prune-check-receipt/v1"

	ArtifactRule  = "rule"
	ArtifactSkill = "skill"

	OriginGenerated    = "generated"
	OriginUserAuthored = "user-authored"
	OriginMixed        = "mixed"
	OriginUnknown      = "unknown"

	CapabilitySupported   = "supported"
	CapabilityUnsupported = "unsupported"
	CapabilityUnknown     = "unknown"

	DispositionKeep              = "keep"
	DispositionUpdate            = "update"
	DispositionConsolidate       = "consolidate"
	DispositionRemove            = "remove"
	DispositionUnableToDetermine = "unable-to-determine"

	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"

	EvidenceGapInventory  = "inventory"
	EvidenceGapProvenance = "provenance"
	EvidenceGapCapability = "capability"
	EvidenceGapRepository = "repository"
	EvidenceGapConflict   = "conflict"
)

// Diagnostic is one actionable contract failure.
type Diagnostic struct {
	Path     string `json:"path"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Recovery string `json:"recovery,omitempty"`
}

// HostIdentity identifies the exact agent host observed by a capability run.
type HostIdentity struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}

// ModelIdentity identifies the exact provider/model pairing used by the host.
type ModelIdentity struct {
	Provider string `yaml:"provider" json:"provider"`
	ID       string `yaml:"id" json:"id"`
}

// CapabilityEvidence is one content-addressed local evidence artifact.
type CapabilityEvidence struct {
	ID     string `yaml:"id" json:"id"`
	Kind   string `yaml:"kind" json:"kind"`
	Path   string `yaml:"path" json:"path"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// Capability is one observed host/model behavior.
type Capability struct {
	ID          string   `yaml:"id" json:"id"`
	Status      string   `yaml:"status" json:"status"`
	EvidenceIDs []string `yaml:"evidence_ids,omitempty" json:"evidence_ids,omitempty"`
}

// Timestamp preserves an RFC3339 timestamp as text while accepting YAML's
// native timestamp scalar resolution.
type Timestamp string

// UnmarshalYAML rejects non-scalar timestamps and normalizes valid values.
func (timestamp *Timestamp) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("timestamp must be a scalar")
	}
	parsed, err := time.Parse(time.RFC3339, node.Value)
	if err != nil {
		return err
	}
	*timestamp = Timestamp(parsed.Format(time.RFC3339))
	return nil
}

// CapabilityProfile is an explicitly selected point-in-time observation.
type CapabilityProfile struct {
	Schema       string               `yaml:"schema" json:"schema"`
	ID           string               `yaml:"id" json:"id"`
	Host         HostIdentity         `yaml:"host" json:"host"`
	Model        ModelIdentity        `yaml:"model" json:"model"`
	ObservedAt   Timestamp            `yaml:"observed_at" json:"observed_at"`
	Evidence     []CapabilityEvidence `yaml:"evidence" json:"evidence"`
	Capabilities []Capability         `yaml:"capabilities" json:"capabilities"`
}

// ProvenanceEntry is an explicit local declaration for exact artifact bytes.
type ProvenanceEntry struct {
	Path        string `yaml:"path" json:"path"`
	SHA256      string `yaml:"sha256" json:"sha256"`
	Origin      string `yaml:"origin" json:"origin"`
	Declaration string `yaml:"declaration" json:"declaration"`
}

// ProvenanceManifest records declarations without inferring authorship.
type ProvenanceManifest struct {
	Schema    string            `yaml:"schema" json:"schema"`
	Artifacts []ProvenanceEntry `yaml:"artifacts" json:"artifacts"`
}

// Artifact is one canonical lifecycle candidate at current HEAD.
type Artifact struct {
	Kind            string         `json:"kind"`
	ID              string         `json:"id"`
	Path            string         `json:"path"`
	SHA256          string         `json:"sha256"`
	Mode            string         `json:"mode"`
	Origin          string         `json:"origin"`
	SupportingFiles []ArtifactFile `json:"supporting_files,omitempty"`
}

// ArtifactFile binds one tracked supporting file in a repository skill.
type ArtifactFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   string `json:"mode"`
}

// Context is the immutable deterministic input to semantic proposal work.
type Context struct {
	Schema                   string            `json:"schema"`
	ReviewID                 string            `json:"review_id"`
	BaselineCommit           string            `json:"baseline_commit"`
	Inventory                inventory.Report  `json:"inventory"`
	Artifacts                []Artifact        `json:"artifacts"`
	Capabilities             CapabilityProfile `json:"capabilities"`
	CapabilityProfileDigest  string            `json:"capability_profile_digest"`
	CapabilityProfilePath    string            `json:"capability_profile_path"`
	ProvenanceManifestDigest string            `json:"provenance_manifest_digest,omitempty"`
	ProvenanceManifestPath   string            `json:"provenance_manifest_path,omitempty"`
	ContextDigest            string            `json:"context_digest"`
}

// ArtifactRef binds a proposal action to exact canonical bytes.
type ArtifactRef struct {
	Kind   string `yaml:"kind" json:"kind"`
	ID     string `yaml:"id" json:"id"`
	Path   string `yaml:"path" json:"path"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// CandidateRef points to a complete replacement file inside the review bundle.
type CandidateRef struct {
	Kind            string             `yaml:"kind" json:"kind"`
	ID              string             `yaml:"id" json:"id"`
	TargetPath      string             `yaml:"target_path" json:"target_path"`
	SourcePath      string             `yaml:"source_path" json:"source_path"`
	SHA256          string             `yaml:"sha256" json:"sha256"`
	Mode            string             `yaml:"mode" json:"mode"`
	SupportingFiles []CandidateFileRef `yaml:"supporting_files,omitempty" json:"supporting_files,omitempty"`
}

// CandidateFileRef binds one complete supporting file in a replacement skill.
type CandidateFileRef struct {
	TargetPath string `yaml:"target_path" json:"target_path"`
	SourcePath string `yaml:"source_path" json:"source_path"`
	SHA256     string `yaml:"sha256" json:"sha256"`
	Mode       string `yaml:"mode" json:"mode"`
}

// EvidenceRef points to current repository evidence used by one disposition.
type EvidenceRef struct {
	Path   string `yaml:"path" json:"path"`
	Lines  string `yaml:"lines" json:"lines"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// EvidenceGap records the exact context subject and missing fact that prevents
// an evidence-backed lifecycle disposition.
type EvidenceGap struct {
	Kind         string `yaml:"kind" json:"kind"`
	ArtifactPath string `yaml:"artifact_path" json:"artifact_path"`
	Detail       string `yaml:"detail" json:"detail"`
}

// CheckRequirement declares external evidence required before verification.
type CheckRequirement struct {
	ID      string `yaml:"id" json:"id"`
	Command string `yaml:"command" json:"command"`
}

// Action is one complete proposed lifecycle disposition.
type Action struct {
	ID                   string             `yaml:"id" json:"id"`
	Disposition          string             `yaml:"disposition" json:"disposition"`
	Sources              []ArtifactRef      `yaml:"sources" json:"sources"`
	Target               *CandidateRef      `yaml:"target,omitempty" json:"target,omitempty"`
	Dependencies         []string           `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Rationale            string             `yaml:"rationale" json:"rationale"`
	Confidence           string             `yaml:"confidence" json:"confidence"`
	RepositoryEvidence   []EvidenceRef      `yaml:"repository_evidence,omitempty" json:"repository_evidence,omitempty"`
	CapabilityRefs       []string           `yaml:"capability_refs,omitempty" json:"capability_refs,omitempty"`
	EvidenceGaps         []EvidenceGap      `yaml:"evidence_gaps,omitempty" json:"evidence_gaps,omitempty"`
	UnresolvedQuestions  []string           `yaml:"unresolved_questions,omitempty" json:"unresolved_questions,omitempty"`
	RequiredVerification []CheckRequirement `yaml:"required_verification,omitempty" json:"required_verification,omitempty"`
}

// Proposal is the host-agent semantic output validated by ssb.
type Proposal struct {
	Schema        string   `yaml:"schema" json:"schema"`
	ReviewID      string   `yaml:"review_id" json:"review_id"`
	ContextDigest string   `yaml:"context_digest" json:"context_digest"`
	Actions       []Action `yaml:"actions" json:"actions"`
}

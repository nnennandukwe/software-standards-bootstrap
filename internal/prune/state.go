package prune

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

// Status preserves each lifecycle state instead of collapsing completion.
type Status struct {
	ReviewID      string `json:"review_id"`
	Inspected     bool   `json:"inspected"`
	Proposed      bool   `json:"proposed"`
	ProposalValid bool   `json:"proposal_valid"`
	Approved      bool   `json:"approved"`
	Applied       bool   `json:"applied"`
	Rendered      bool   `json:"rendered"`
	ADRRecorded   bool   `json:"adr_recorded"`
	Verified      bool   `json:"verified"`
}

// ReceiptEvidence is one content-addressed external check artifact.
type ReceiptEvidence struct {
	Path   string `yaml:"path" json:"path"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// CheckReceipt records externally executed proof. SSB verifies the receipt
// and evidence bytes but never executes its command.
type CheckReceipt struct {
	Schema         string            `yaml:"schema" json:"schema"`
	ReviewID       string            `yaml:"review_id" json:"review_id"`
	ProposalDigest string            `yaml:"proposal_digest" json:"proposal_digest"`
	CheckID        string            `yaml:"check_id" json:"check_id"`
	Command        string            `yaml:"command" json:"command"`
	Status         string            `yaml:"status" json:"status"`
	ObservedAt     Timestamp         `yaml:"observed_at" json:"observed_at"`
	Evidence       []ReceiptEvidence `yaml:"evidence" json:"evidence"`
}

// VerificationResult identifies the exact receipts accepted for a review.
type VerificationResult struct {
	Receipts map[string]string `json:"receipts"`
}

// ReviewStatus reads the bundle and derives every independent state.
func ReviewStatus(repoRoot, reviewID string) (Status, []Diagnostic, error) {
	root, err := reviewRoot(repoRoot, reviewID)
	if err != nil {
		return Status{}, nil, err
	}
	status := Status{ReviewID: reviewID}
	if _, err := os.Stat(filepath.Join(root, "context.json")); err != nil {
		return Status{}, nil, fmt.Errorf("read review context: %w", err)
	}
	status.Inspected = true
	if _, err := os.Stat(filepath.Join(root, "proposal.yaml")); err != nil {
		if os.IsNotExist(err) {
			return status, nil, nil
		}
		return Status{}, nil, err
	}
	status.Proposed = true
	review, diagnostics, err := LoadReview(repoRoot, reviewID)
	if err != nil {
		return Status{}, nil, err
	}
	status.ProposalValid = len(diagnostics) == 0
	for _, event := range review.Events {
		switch event.Kind {
		case EventApproved:
			status.Approved = true
		case EventApplied:
			status.Applied = true
		case EventRendered:
			status.Rendered = true
		case EventADR:
			status.ADRRecorded = true
		case EventVerified:
			status.Verified = true
		}
	}
	return status, diagnostics, nil
}

// RecordTransition appends a result for rerendering or ADR recording after a
// successful application. The command doing the work supplies its own payload.
func RecordTransition(repoPath, reviewID, kind string, payload any, now func() time.Time) (Event, error) {
	transition, err := BeginTransition(repoPath, reviewID, kind, now)
	if err != nil {
		return Event{}, err
	}
	defer transition.Cancel()
	return transition.Complete(payload)
}

// Transition holds the review lock between preflight and a successful
// renderer/ADR write so invalid state cannot create an unrecorded artifact.
type Transition struct {
	review Review
	kind   string
	now    func() time.Time
	unlock func()
	done   bool
}

// BeginTransition validates and locks a separate rerender or ADR state before
// its corresponding file write begins.
func BeginTransition(repoPath, reviewID, kind string, now func() time.Time) (*Transition, error) {
	unlock, err := acquireReviewLock(repoPath, reviewID)
	if err != nil {
		return nil, err
	}
	review, diagnostics, err := LoadReview(repoPath, reviewID)
	if err != nil {
		unlock()
		return nil, err
	}
	if len(diagnostics) != 0 {
		unlock()
		return nil, validationError("proposal has %d error(s)", len(diagnostics))
	}
	if kind != EventRendered && kind != EventADR {
		unlock()
		return nil, fmt.Errorf("unsupported transition kind %s", kind)
	}
	applied := false
	for _, event := range review.Events {
		if event.Kind == EventApplied {
			applied = true
		}
		if event.Kind == kind {
			unlock()
			return nil, preconditionError("review already has a %s event", kind)
		}
	}
	if !applied {
		unlock()
		return nil, preconditionError("%s requires a completed application event", kind)
	}
	return &Transition{review: review, kind: kind, now: now, unlock: unlock}, nil
}

// Complete appends the transition result and releases the review lock.
func (transition *Transition) Complete(payload any) (Event, error) {
	if transition.done {
		return Event{}, fmt.Errorf("transition is already complete")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	if transition.now == nil {
		transition.now = time.Now
	}
	event := newEvent(transition.review, transition.kind, data, transition.now().UTC())
	if err := appendEvent(transition.review, event); err != nil {
		return Event{}, err
	}
	transition.done = true
	transition.unlock()
	return event, nil
}

// Cancel releases a transition that did not write or whose write was rolled
// back.
func (transition *Transition) Cancel() {
	if transition == nil || transition.done {
		return
	}
	transition.done = true
	transition.unlock()
}

// Verify accepts only exact, passing, content-addressed receipts for every
// required external check and records verification as a separate event.
func Verify(ctx context.Context, repoPath, reviewID, receiptsDir string, now func() time.Time) (VerificationResult, error) {
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return VerificationResult{}, err
	}
	unlock, err := acquireReviewLock(repo.Root(), reviewID)
	if err != nil {
		return VerificationResult{}, err
	}
	defer unlock()
	review, diagnostics, err := LoadReview(repo.Root(), reviewID)
	if err != nil {
		return VerificationResult{}, err
	}
	if len(diagnostics) != 0 {
		return VerificationResult{}, validationError("proposal has %d error(s)", len(diagnostics))
	}
	status, _, err := ReviewStatus(repo.Root(), reviewID)
	if err != nil {
		return VerificationResult{}, err
	}
	if !status.Applied {
		return VerificationResult{}, preconditionError("verification requires a completed application event")
	}
	if changesRules(review) && !status.Rendered {
		return VerificationResult{}, preconditionError("rule changes require a separate rerender event before verification")
	}
	if status.Verified {
		return VerificationResult{}, preconditionError("review already has a verification event")
	}
	approval, err := approvalFor(review)
	if err != nil {
		return VerificationResult{}, err
	}
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	required := make(map[string]CheckRequirement)
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok {
			continue
		}
		for _, check := range action.RequiredVerification {
			if !stableIDPattern.MatchString(check.ID) || check.Command == "" {
				return VerificationResult{}, validationError("action %s has invalid verification requirement", action.ID)
			}
			if prior, duplicate := required[check.ID]; duplicate && prior.Command != check.Command {
				return VerificationResult{}, validationError("check id %s has conflicting commands", check.ID)
			}
			required[check.ID] = check
		}
	}
	if len(required) == 0 {
		return VerificationResult{}, validationError("review has no required external checks and cannot enter verified state")
	}
	result := VerificationResult{Receipts: make(map[string]string, len(required))}
	ids := make([]string, 0, len(required))
	for id := range required {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		filePath := filepath.Join(receiptsDir, id+".yaml")
		data, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return VerificationResult{}, preconditionError("required receipt %s is missing from %s", id, receiptsDir)
			}
			return VerificationResult{}, fmt.Errorf("read receipt %s: %w", id, err)
		}
		var receipt CheckReceipt
		if err := yaml.Load(data, &receipt, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
			return VerificationResult{}, validationError("parse receipt %s: %v", id, err)
		}
		if receipt.Schema != CheckSchema || receipt.ReviewID != reviewID ||
			receipt.ProposalDigest != review.ProposalDigest || receipt.CheckID != id ||
			receipt.Command != required[id].Command || receipt.Status != "passed" {
			return VerificationResult{}, validationError("receipt %s does not match the required passing check", id)
		}
		if len(receipt.Evidence) == 0 {
			return VerificationResult{}, validationError("receipt %s requires content-addressed evidence", id)
		}
		for _, evidence := range receipt.Evidence {
			if !safeRelativePath(evidence.Path) || !validDigest(evidence.SHA256) {
				return VerificationResult{}, validationError("receipt %s has invalid evidence reference", id)
			}
			evidencePath := filepath.Join(receiptsDir, filepath.FromSlash(evidence.Path))
			if err := requireRegularBundleFile(receiptsDir, evidencePath); err != nil {
				return VerificationResult{}, validationError("receipt %s evidence %s is unsafe: %v", id, evidence.Path, err)
			}
			content, err := os.ReadFile(evidencePath)
			if err != nil || digestBytes(content) != evidence.SHA256 {
				return VerificationResult{}, validationError("receipt %s evidence %s is missing or has changed", id, evidence.Path)
			}
		}
		result.Receipts[id] = digestBytes(data)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return VerificationResult{}, err
	}
	if now == nil {
		now = time.Now
	}
	event := newEvent(review, EventVerified, data, now().UTC())
	if err := appendEvent(review, event); err != nil {
		return VerificationResult{}, err
	}
	return result, nil
}

func changesRules(review Review) bool {
	approval, err := approvalFor(review)
	if err != nil {
		return false
	}
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok || action.Disposition == DispositionKeep {
			continue
		}
		for _, source := range action.Sources {
			if source.Kind == ArtifactRule {
				return true
			}
		}
		if action.Target != nil && action.Target.Kind == ArtifactRule {
			return true
		}
	}
	return false
}

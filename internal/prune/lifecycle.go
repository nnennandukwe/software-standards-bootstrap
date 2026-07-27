package prune

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

const (
	EventApproved = "approved"
	EventApplied  = "applied"
	EventRendered = "rendered"
	EventADR      = "adr-recorded"
	EventVerified = "verified"
)

var writeEventLogAtomically = atomicWrite

// CreateResult identifies the immutable context written for a new review.
type CreateResult struct {
	Context     Context
	ContextPath string
}

// Review is a validated context/proposal pair and its exact proposal digest.
type Review struct {
	Root           string
	Context        Context
	Proposal       Proposal
	ProposalDigest string
	Events         []Event
}

// Event is one digest-chained review lifecycle transition.
type Event struct {
	Schema              string          `json:"schema"`
	ID                  string          `json:"id"`
	ReviewID            string          `json:"review_id"`
	Kind                string          `json:"kind"`
	RecordedAt          string          `json:"recorded_at"`
	BaselineCommit      string          `json:"baseline_commit"`
	ContextDigest       string          `json:"context_digest"`
	ProposalDigest      string          `json:"proposal_digest"`
	PreviousEventDigest string          `json:"previous_event_digest,omitempty"`
	Payload             json.RawMessage `json:"payload"`
	EventDigest         string          `json:"event_digest"`
}

// ApprovalPayload records one explicit disposition for every proposed action.
type ApprovalPayload struct {
	Approved []string `json:"approved"`
	Rejected []string `json:"rejected"`
}

// ApprovalOptions supplies the complete human decision set.
type ApprovalOptions struct {
	ReviewID string
	Approved []string
	Rejected []string
	Now      func() time.Time
}

type renderedEventPayload struct {
	Path          string `json:"path"`
	Changed       bool   `json:"changed"`
	DryRun        bool   `json:"dry_run"`
	SourceDigest  string `json:"source_digest"`
	ContentDigest string `json:"content_digest"`
}

type adrEventPayload struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
	DryRun  bool   `json:"dry_run"`
}

// CreateReview writes only a deterministic context. Proposal generation and
// approval remain separate host-agent and human states.
func CreateReview(
	ctx context.Context,
	repo *workspace.Repository,
	options ContextOptions,
) (CreateResult, error) {
	reviewContext, err := BuildContext(ctx, repo, options)
	if err != nil {
		if errors.Is(err, ErrIncompleteInventory) || errors.Is(err, workspace.ErrPrecondition) {
			return CreateResult{}, err
		}
		return CreateResult{}, validationError("%v", err)
	}
	if err := repo.VerifyPruneSnapshot(ctx); err != nil {
		return CreateResult{}, err
	}
	root, err := reviewRoot(repo.Root(), options.ReviewID)
	if err != nil {
		return CreateResult{}, err
	}
	if _, err := os.Lstat(root); err == nil {
		return CreateResult{}, preconditionError("review %q already exists", options.ReviewID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return CreateResult{}, fmt.Errorf("inspect review path: %w", err)
	}
	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return CreateResult{}, fmt.Errorf("create reviews directory: %w", err)
	}
	staging, err := os.MkdirTemp(parent, "."+options.ReviewID+"-")
	if err != nil {
		return CreateResult{}, fmt.Errorf("create review staging directory: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := snapshotReviewInputs(staging, options, reviewContext); err != nil {
		return CreateResult{}, err
	}
	data, err := json.MarshalIndent(reviewContext, "", "  ")
	if err != nil {
		return CreateResult{}, fmt.Errorf("encode prune context: %w", err)
	}
	data = append(data, '\n')
	contextFile := filepath.Join(staging, "context.json")
	if err := writeExclusive(contextFile, data, 0o644); err != nil {
		return CreateResult{}, err
	}
	if err := os.Rename(staging, root); err != nil {
		return CreateResult{}, fmt.Errorf("publish review bundle: %w", err)
	}
	complete = true
	return CreateResult{
		Context:     reviewContext,
		ContextPath: filepath.ToSlash(mustRelative(repo.Root(), filepath.Join(root, "context.json"))),
	}, nil
}

func snapshotReviewInputs(staging string, options ContextOptions, reviewContext Context) error {
	if err := snapshotFile(
		options.Capabilities,
		filepath.Join(staging, filepath.FromSlash(reviewContext.CapabilityProfilePath)),
		reviewContext.CapabilityProfileDigest,
	); err != nil {
		return fmt.Errorf("snapshot capability profile: %w", err)
	}
	profileBase := filepath.Dir(options.Capabilities)
	for _, evidence := range reviewContext.Capabilities.Evidence {
		if evidence.Path == filepath.Base(options.Capabilities) {
			return fmt.Errorf("capability evidence path collides with the profile snapshot: %s", evidence.Path)
		}
		if err := snapshotFile(
			filepath.Join(profileBase, filepath.FromSlash(evidence.Path)),
			filepath.Join(staging, "inputs", "capability", filepath.FromSlash(evidence.Path)),
			evidence.SHA256,
		); err != nil {
			return fmt.Errorf("snapshot capability evidence %s: %w", evidence.ID, err)
		}
	}
	if options.Provenance != "" {
		if err := snapshotFile(
			options.Provenance,
			filepath.Join(staging, filepath.FromSlash(reviewContext.ProvenanceManifestPath)),
			reviewContext.ProvenanceManifestDigest,
		); err != nil {
			return fmt.Errorf("snapshot provenance manifest: %w", err)
		}
	}
	return nil
}

func snapshotFile(source, target, expectedDigest string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if digestBytes(data) != expectedDigest {
		return fmt.Errorf("source changed after context construction")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return writeExclusive(target, data, 0o644)
}

// LoadReview strictly loads and validates the immutable context and current
// proposal. Diagnostics are returned separately from I/O/contract errors.
func LoadReview(repoRoot, reviewID string) (Review, []Diagnostic, error) {
	root, err := reviewRoot(repoRoot, reviewID)
	if err != nil {
		return Review{}, nil, err
	}
	contextData, err := os.ReadFile(filepath.Join(root, "context.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Review{}, nil, preconditionError("review %s has no context.json", reviewID)
		}
		return Review{}, nil, fmt.Errorf("read review context: %w", err)
	}
	var reviewContext Context
	if err := decodeStrictJSON(contextData, &reviewContext); err != nil {
		return Review{}, nil, validationError("parse review context: %v", err)
	}
	if err := validateContext(reviewContext); err != nil {
		return Review{}, nil, validationError("%v", err)
	}
	if err := validateReviewSnapshots(root, reviewContext); err != nil {
		return Review{}, nil, validationError("%v", err)
	}
	proposalData, err := os.ReadFile(filepath.Join(root, "proposal.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Review{}, nil, preconditionError("review %s has no proposal.yaml", reviewID)
		}
		return Review{}, nil, fmt.Errorf("read review proposal: %w", err)
	}
	var proposal Proposal
	if err := yaml.Load(proposalData, &proposal, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return Review{}, nil, validationError("parse review proposal: %v", err)
	}
	events, err := loadEvents(filepath.Join(root, "events.jsonl"))
	if err != nil {
		return Review{}, nil, validationError("%v", err)
	}
	review := Review{
		Root:           root,
		Context:        reviewContext,
		Proposal:       proposal,
		ProposalDigest: digestBytes(proposalData),
		Events:         events,
	}
	if err := validateReviewEvents(review); err != nil {
		return Review{}, nil, validationError("%v", err)
	}
	return review, ValidateProposal(reviewContext, proposal, root), nil
}

func validateReviewEvents(review Review) error {
	seen := make(map[string]bool)
	applied := false
	rendered := false
	var approval ApprovalPayload
	for index, event := range review.Events {
		if event.ReviewID != review.Context.ReviewID ||
			event.BaselineCommit != review.Context.BaselineCommit ||
			event.ContextDigest != review.Context.ContextDigest ||
			event.ProposalDigest != review.ProposalDigest {
			return fmt.Errorf("event %d is bound to a different review identity", index+1)
		}
		if event.ID != fmt.Sprintf("%s-%03d", event.Kind, index+1) {
			return fmt.Errorf("event %d has an invalid id", index+1)
		}
		if _, err := time.Parse(time.RFC3339, event.RecordedAt); err != nil {
			return fmt.Errorf("event %d has an invalid recorded_at", index+1)
		}
		if seen[event.Kind] {
			return fmt.Errorf("event kind %s appears more than once", event.Kind)
		}
		switch event.Kind {
		case EventApproved:
			if index != 0 {
				return fmt.Errorf("approval must be the first event")
			}
			var payload ApprovalPayload
			if err := decodeStrictJSON(event.Payload, &payload); err != nil {
				return fmt.Errorf("approval payload is invalid: %w", err)
			}
			if err := validateRecordedApproval(review.Proposal, payload); err != nil {
				return fmt.Errorf("approval payload is invalid: %w", err)
			}
			approval = payload
		case EventApplied:
			if !seen[EventApproved] {
				return fmt.Errorf("application event precedes approval")
			}
			var payload ApplyResult
			if err := decodeStrictJSON(event.Payload, &payload); err != nil || payload.DryRun {
				return fmt.Errorf("application payload is invalid")
			}
			expected := expectedRecordedChanges(review, approval)
			actualDigest, _ := canonicalDigest(payload.Changes)
			expectedDigest, _ := canonicalDigest(expected)
			if actualDigest != expectedDigest {
				return fmt.Errorf("application payload does not match approved changes")
			}
			applied = true
		case EventRendered:
			if !applied {
				return fmt.Errorf("rerender event precedes application")
			}
			var payload renderedEventPayload
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				payload.Path != "AGENTS.md" || payload.DryRun ||
				!validDigest(payload.SourceDigest) || !validDigest(payload.ContentDigest) {
				return fmt.Errorf("rerender payload is invalid")
			}
			rendered = true
		case EventADR:
			if !applied {
				return fmt.Errorf("ADR event precedes application")
			}
			var payload adrEventPayload
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				!payload.Created || payload.DryRun || !safeRelativePath(payload.Path) ||
				!strings.HasSuffix(payload.Path, ".md") {
				return fmt.Errorf("ADR payload is invalid")
			}
		case EventVerified:
			if !applied {
				return fmt.Errorf("verification event precedes application")
			}
			var payload VerificationResult
			if err := decodeStrictJSON(event.Payload, &payload); err != nil || len(payload.Receipts) == 0 {
				return fmt.Errorf("verification payload is invalid")
			}
			required := requiredRecordedChecks(review.Proposal, approval)
			if len(payload.Receipts) != len(required) {
				return fmt.Errorf("verification payload does not cover every required check")
			}
			for id := range required {
				if !validDigest(payload.Receipts[id]) {
					return fmt.Errorf("verification payload lacks a valid receipt for %s", id)
				}
			}
			if changesRules(review) && !rendered {
				return fmt.Errorf("verification event precedes required rerendering")
			}
		default:
			return fmt.Errorf("event %d has unsupported kind %q", index+1, event.Kind)
		}
		seen[event.Kind] = true
	}
	return nil
}

func expectedRecordedChanges(review Review, approval ApprovalPayload) []Change {
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	byPath := make(map[string]Change)
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok || action.Disposition == DispositionKeep {
			continue
		}
		for _, source := range action.Sources {
			byPath[source.Path] = Change{ActionID: action.ID, Path: source.Path, Kind: "remove"}
			if actionRemovesSupportingFiles(action, source) {
				for _, supporting := range supportingFilesFor(review, source.Path) {
					byPath[supporting.Path] = Change{
						ActionID: action.ID,
						Path:     supporting.Path,
						Kind:     "remove",
					}
				}
			}
		}
		if action.Target != nil {
			byPath[action.Target.TargetPath] = Change{
				ActionID: action.ID,
				Path:     action.Target.TargetPath,
				Kind:     "write",
				SHA256:   action.Target.SHA256,
			}
			for _, supporting := range action.Target.SupportingFiles {
				byPath[supporting.TargetPath] = Change{
					ActionID: action.ID,
					Path:     supporting.TargetPath,
					Kind:     "write",
					SHA256:   supporting.SHA256,
				}
			}
		}
	}
	paths := make([]string, 0, len(byPath))
	for itemPath := range byPath {
		paths = append(paths, itemPath)
	}
	sort.Strings(paths)
	result := make([]Change, 0, len(paths))
	for _, itemPath := range paths {
		result = append(result, byPath[itemPath])
	}
	return result
}

func requiredRecordedChecks(proposal Proposal, approval ApprovalPayload) map[string]struct{} {
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	required := make(map[string]struct{})
	for _, action := range proposal.Actions {
		if _, ok := approved[action.ID]; !ok {
			continue
		}
		for _, check := range action.RequiredVerification {
			required[check.ID] = struct{}{}
		}
	}
	return required
}

func validateRecordedApproval(proposal Proposal, payload ApprovalPayload) error {
	approved, err := uniqueDecisionSet(payload.Approved, "approved")
	if err != nil {
		return err
	}
	rejected, err := uniqueDecisionSet(payload.Rejected, "rejected")
	if err != nil {
		return err
	}
	for id := range approved {
		if _, conflict := rejected[id]; conflict {
			return fmt.Errorf("action %s is both approved and rejected", id)
		}
		if !hasAction(proposal, id) {
			return fmt.Errorf("approved action %s does not exist", id)
		}
	}
	for id := range rejected {
		if !hasAction(proposal, id) {
			return fmt.Errorf("rejected action %s does not exist", id)
		}
	}
	for _, action := range proposal.Actions {
		_, isApproved := approved[action.ID]
		_, isRejected := rejected[action.ID]
		if !isApproved && !isRejected {
			return fmt.Errorf("action %s has no decision", action.ID)
		}
		if action.Disposition == DispositionUnableToDetermine && isApproved {
			return fmt.Errorf("unable-to-determine action %s is approved", action.ID)
		}
		if isApproved {
			for _, dependency := range action.Dependencies {
				if _, ok := approved[dependency]; !ok {
					return fmt.Errorf("approved action %s lacks dependency %s", action.ID, dependency)
				}
			}
		}
	}
	return nil
}

// Approve records exactly one digest-bound human decision event.
func Approve(ctx context.Context, repoPath string, options ApprovalOptions) (Event, error) {
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return Event{}, err
	}
	unlock, err := acquireReviewLock(repo.Root(), options.ReviewID)
	if err != nil {
		return Event{}, err
	}
	defer unlock()
	review, diagnostics, err := LoadReview(repo.Root(), options.ReviewID)
	if err != nil {
		return Event{}, err
	}
	if len(diagnostics) != 0 {
		return Event{}, validationError("proposal has %d error(s); run ssb prune validate", len(diagnostics))
	}
	if repo.Baseline() != review.Context.BaselineCommit {
		return Event{}, preconditionError("review baseline is stale: expected %s, found %s", review.Context.BaselineCommit, repo.Baseline())
	}
	for _, event := range review.Events {
		if event.Kind == EventApproved {
			return Event{}, preconditionError("review already has an approval event")
		}
	}
	approved, err := uniqueDecisionSet(options.Approved, "approved")
	if err != nil {
		return Event{}, preconditionError("%v", err)
	}
	rejected, err := uniqueDecisionSet(options.Rejected, "rejected")
	if err != nil {
		return Event{}, preconditionError("%v", err)
	}
	for id := range approved {
		if _, conflict := rejected[id]; conflict {
			return Event{}, preconditionError("action %s is both approved and rejected", id)
		}
	}
	for _, action := range review.Proposal.Actions {
		_, isApproved := approved[action.ID]
		_, isRejected := rejected[action.ID]
		if action.Disposition == DispositionUnableToDetermine && isApproved {
			return Event{}, preconditionError("unable-to-determine action %s cannot be approved", action.ID)
		}
		if !isApproved && !isRejected {
			return Event{}, preconditionError("action %s has no explicit approve or reject decision", action.ID)
		}
	}
	for id := range approved {
		if !hasAction(review.Proposal, id) {
			return Event{}, preconditionError("approved action %s does not exist", id)
		}
	}
	for id := range rejected {
		if !hasAction(review.Proposal, id) {
			return Event{}, preconditionError("rejected action %s does not exist", id)
		}
	}
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok {
			continue
		}
		for _, dependency := range action.Dependencies {
			if _, ok := approved[dependency]; !ok {
				return Event{}, preconditionError("approved action %s requires approved dependency %s", action.ID, dependency)
			}
		}
	}
	approvedIDs := sortedKeys(approved)
	rejectedIDs := sortedKeys(rejected)
	payload, err := json.Marshal(ApprovalPayload{Approved: approvedIDs, Rejected: rejectedIDs})
	if err != nil {
		return Event{}, fmt.Errorf("encode approval: %w", err)
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	event := newEvent(review, EventApproved, payload, now().UTC())
	if err := appendEvent(review, event); err != nil {
		return Event{}, err
	}
	return event, nil
}

func acquireReviewLock(repoRoot, reviewID string) (func(), error) {
	root, err := reviewRoot(repoRoot, reviewID)
	if err != nil {
		return nil, err
	}
	lockPath := filepath.Join(root, ".transition.lock")
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, preconditionError("review %s does not exist", reviewID)
		}
		if errors.Is(err, os.ErrExist) {
			return nil, preconditionError("another review transition is active; if it crashed, confirm no process is active and remove %s", lockPath)
		}
		return nil, fmt.Errorf("create review transition lock: %w", err)
	}
	return func() {
		_ = file.Close()
		_ = os.Remove(lockPath)
	}, nil
}

func validateContext(reviewContext Context) error {
	if reviewContext.Schema != ContextSchema {
		return fmt.Errorf("review context schema must be %s", ContextSchema)
	}
	if !stableIDPattern.MatchString(reviewContext.ReviewID) {
		return fmt.Errorf("review context has invalid review_id")
	}
	if !validDigest(reviewContext.ContextDigest) {
		return fmt.Errorf("review context has invalid digest")
	}
	expected := reviewContext.ContextDigest
	reviewContext.ContextDigest = ""
	actual, err := canonicalDigest(reviewContext)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("review context digest mismatch")
	}
	if reviewContext.Inventory.Truncated {
		return fmt.Errorf("%w: stored context is truncated", ErrIncompleteInventory)
	}
	for _, artifact := range reviewContext.Artifacts {
		if artifact.Mode != "100644" && artifact.Mode != "100755" {
			return fmt.Errorf("review context artifact %s has an invalid mode", artifact.Path)
		}
		for _, supporting := range artifact.SupportingFiles {
			if supporting.Mode != "100644" && supporting.Mode != "100755" {
				return fmt.Errorf("review context supporting file %s has an invalid mode", supporting.Path)
			}
		}
	}
	if !validDigest(reviewContext.CapabilityProfileDigest) ||
		!safeRelativePath(reviewContext.CapabilityProfilePath) ||
		!strings.HasPrefix(reviewContext.CapabilityProfilePath, "inputs/capability/") {
		return fmt.Errorf("review context has an invalid capability snapshot reference")
	}
	if reviewContext.ProvenanceManifestDigest != "" &&
		(!validDigest(reviewContext.ProvenanceManifestDigest) ||
			!safeRelativePath(reviewContext.ProvenanceManifestPath) ||
			!strings.HasPrefix(reviewContext.ProvenanceManifestPath, "inputs/provenance/")) {
		return fmt.Errorf("review context has an invalid provenance snapshot reference")
	}
	return nil
}

func validateReviewSnapshots(root string, reviewContext Context) error {
	profilePath := filepath.Join(root, filepath.FromSlash(reviewContext.CapabilityProfilePath))
	if err := verifySnapshotFile(profilePath, reviewContext.CapabilityProfileDigest); err != nil {
		return fmt.Errorf("capability profile snapshot: %w", err)
	}
	profileRoot := filepath.Dir(profilePath)
	for _, evidence := range reviewContext.Capabilities.Evidence {
		target := filepath.Join(profileRoot, filepath.FromSlash(evidence.Path))
		if err := requireRegularBundleFile(root, target); err != nil {
			return fmt.Errorf("capability evidence snapshot %s: %w", evidence.ID, err)
		}
		if err := verifySnapshotFile(target, evidence.SHA256); err != nil {
			return fmt.Errorf("capability evidence snapshot %s: %w", evidence.ID, err)
		}
	}
	if reviewContext.ProvenanceManifestDigest != "" {
		target := filepath.Join(root, filepath.FromSlash(reviewContext.ProvenanceManifestPath))
		if err := verifySnapshotFile(target, reviewContext.ProvenanceManifestDigest); err != nil {
			return fmt.Errorf("provenance snapshot: %w", err)
		}
	}
	return nil
}

func verifySnapshotFile(filePath, expectedDigest string) error {
	info, err := os.Lstat(filePath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("snapshot is not a regular file")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if digestBytes(data) != expectedDigest {
		return fmt.Errorf("snapshot digest mismatch")
	}
	return nil
}

func reviewRoot(repoRoot, reviewID string) (string, error) {
	if !stableIDPattern.MatchString(reviewID) {
		return "", fmt.Errorf("review id must be lower-case kebab-case")
	}
	return filepath.Join(repoRoot, ".software-standards", "reviews", reviewID), nil
}

func writeExclusive(filePath string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", filePath, err)
	}
	defer file.Close()
	if written, err := file.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	} else if written != len(data) {
		return fmt.Errorf("write %s: %w", filePath, io.ErrShortWrite)
	}
	return file.Sync()
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("unexpected trailing JSON")
	}
	return nil
}

func loadEvents(filePath string) ([]Event, error) {
	file, err := os.Open(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return []Event{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read review events: %w", err)
	}
	defer file.Close()
	events := make([]Event, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), 64<<20)
	previous := ""
	for scanner.Scan() {
		var event Event
		if err := decodeStrictJSON(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("parse review event %d: %w", len(events)+1, err)
		}
		if event.Schema != EventSchema || event.PreviousEventDigest != previous {
			return nil, fmt.Errorf("review event chain is invalid at event %d", len(events)+1)
		}
		expected := event.EventDigest
		event.EventDigest = ""
		actual, err := canonicalDigest(event)
		if err != nil || expected != actual {
			return nil, fmt.Errorf("review event digest is invalid at event %d", len(events)+1)
		}
		event.EventDigest = expected
		previous = expected
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read review events: %w", err)
	}
	return events, nil
}

func newEvent(review Review, kind string, payload []byte, recordedAt time.Time) Event {
	previous := ""
	if len(review.Events) != 0 {
		previous = review.Events[len(review.Events)-1].EventDigest
	}
	event := Event{
		Schema:              EventSchema,
		ID:                  fmt.Sprintf("%s-%03d", kind, len(review.Events)+1),
		ReviewID:            review.Context.ReviewID,
		Kind:                kind,
		RecordedAt:          recordedAt.Format(time.RFC3339),
		BaselineCommit:      review.Context.BaselineCommit,
		ContextDigest:       review.Context.ContextDigest,
		ProposalDigest:      review.ProposalDigest,
		PreviousEventDigest: previous,
		Payload:             payload,
	}
	event.EventDigest, _ = canonicalDigest(event)
	return event
}

func appendEvent(review Review, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode review event: %w", err)
	}
	data = append(data, '\n')
	eventPath := filepath.Join(review.Root, "events.jsonl")
	existing, err := os.ReadFile(eventPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read review event log before append: %w", err)
	}
	if len(existing) != 0 && existing[len(existing)-1] != '\n' {
		return fmt.Errorf("review event log does not end at a complete record")
	}
	next := make([]byte, 0, len(existing)+len(data))
	next = append(next, existing...)
	next = append(next, data...)
	if err := writeEventLogAtomically(eventPath, next, 0o644); err != nil {
		return fmt.Errorf("atomically replace review event log: %w", err)
	}
	return nil
}

func uniqueDecisionSet(ids []string, label string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if !stableIDPattern.MatchString(id) {
			return nil, fmt.Errorf("%s action id %q is invalid", label, id)
		}
		if _, duplicate := result[id]; duplicate {
			return nil, fmt.Errorf("%s action %s appears more than once", label, id)
		}
		result[id] = struct{}{}
	}
	return result, nil
}

func hasAction(proposal Proposal, id string) bool {
	for _, action := range proposal.Actions {
		if action.ID == id {
			return true
		}
	}
	return false
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func mustRelative(root, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(relative, "..") {
		panic("review path escaped repository root")
	}
	return relative
}

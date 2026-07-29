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
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

var removeReviewTransitionLock = func(store *reviewStore) error {
	return store.Remove(".transition.lock")
}

type durableExclusiveFile interface {
	Write([]byte) (int, error)
	Sync() error
	Chmod(os.FileMode) error
	Close() error
}

var (
	openExclusiveFile = func(name string, flag int, perm os.FileMode) (durableExclusiveFile, error) {
		return os.OpenFile(name, flag, perm)
	}
	removeIncompleteExclusiveFile = os.Remove
)

const (
	EventApproved = "approved"
	EventApplied  = "applied"
	EventRendered = "rendered"
	EventADR      = "adr-recorded"
	EventVerified = "verified"
)

var writeEventLogAtomically = func(store *reviewStore, data []byte, mode os.FileMode) error {
	return store.AtomicWrite("events.jsonl", data, mode)
}

// CreateResult identifies the immutable context written for a new review.
type CreateResult struct {
	Context     Context
	ContextPath string
}

// Review is a validated context/proposal pair and its exact proposal digest.
type Review struct {
	RepoRoot       string
	Root           string
	Context        Context
	Proposal       Proposal
	ProposalDigest string
	Events         []Event
	store          *reviewStore
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
	OutputDigest  string `json:"output_digest"`
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
	staging, target, err := createReviewStagingStore(repo.Root(), options.ReviewID)
	if err != nil {
		return CreateResult{}, err
	}
	complete := false
	defer func() {
		if !complete {
			_ = staging.RemoveStaging()
		}
		_ = staging.Close()
	}()
	if err := snapshotReviewInputs(staging, options, reviewContext); err != nil {
		return CreateResult{}, err
	}
	data, err := json.MarshalIndent(reviewContext, "", "  ")
	if err != nil {
		return CreateResult{}, fmt.Errorf("encode prune context: %w", err)
	}
	data = append(data, '\n')
	if err := staging.WriteExclusive("context.json", data, 0o644); err != nil {
		return CreateResult{}, err
	}
	if err := staging.Publish(target); err != nil {
		return CreateResult{}, fmt.Errorf("publish review bundle: %w", err)
	}
	complete = true
	return CreateResult{
		Context:     reviewContext,
		ContextPath: filepath.ToSlash(filepath.Join(target, "context.json")),
	}, nil
}

func snapshotReviewInputs(staging *reviewStore, options ContextOptions, reviewContext Context) error {
	if err := snapshotFile(
		options.Capabilities,
		staging,
		reviewContext.CapabilityProfilePath,
		reviewContext.CapabilityProfileDigest,
	); err != nil {
		return fmt.Errorf("snapshot capability profile: %w", err)
	}
	profileBase := filepath.Dir(options.Capabilities)
	for _, evidence := range reviewContext.Capabilities.Evidence {
		if evidence.Path == filepath.Base(options.Capabilities) {
			return fmt.Errorf("capability evidence path collides with the profile snapshot: %s", evidence.Path)
		}
		source, err := resolvePortablePath(profileBase, evidence.Path)
		if err != nil {
			return fmt.Errorf("resolve capability evidence %s: %w", evidence.ID, err)
		}
		target := path.Join("inputs/capability", evidence.Path)
		if err := snapshotFile(source, staging, target, evidence.SHA256); err != nil {
			return fmt.Errorf("snapshot capability evidence %s: %w", evidence.ID, err)
		}
	}
	if options.Provenance != "" {
		if err := snapshotFile(
			options.Provenance,
			staging,
			reviewContext.ProvenanceManifestPath,
			reviewContext.ProvenanceManifestDigest,
		); err != nil {
			return fmt.Errorf("snapshot provenance manifest: %w", err)
		}
	}
	return nil
}

func snapshotFile(source string, staging *reviewStore, target, expectedDigest string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if digestBytes(data) != expectedDigest {
		return fmt.Errorf("source changed after context construction")
	}
	if err := staging.MkdirAll(path.Dir(target), 0o755); err != nil {
		return err
	}
	return staging.WriteExclusive(target, data, 0o644)
}

// LoadReview strictly loads and validates the immutable context and current
// proposal. Diagnostics are returned separately from I/O/contract errors.
func LoadReview(repoRoot, reviewID string) (Review, []Diagnostic, error) {
	store, err := openReviewStore(repoRoot, reviewID)
	if err != nil {
		return Review{}, nil, err
	}
	defer store.Close()
	review, diagnostics, err := loadReviewFromStore(store)
	review.store = nil
	return review, diagnostics, err
}

func loadReviewFromStore(store *reviewStore) (Review, []Diagnostic, error) {
	repoRoot := store.repoRoot
	reviewID := store.reviewID
	contextData, _, err := store.ReadRegular("context.json")
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
	if err := validateReviewSnapshots(store, reviewContext); err != nil {
		return Review{}, nil, validationError("%v", err)
	}
	proposalData, _, err := store.ReadRegular("proposal.yaml")
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
	events := []Event{}
	eventData, _, eventErr := store.ReadRegular("events.jsonl")
	if eventErr == nil {
		events, err = loadEvents(bytes.NewReader(eventData))
		if err != nil {
			return Review{}, nil, validationError("%v", err)
		}
	} else if !errors.Is(eventErr, os.ErrNotExist) {
		return Review{}, nil, validationError("read review events: %v", eventErr)
	}
	review := Review{
		RepoRoot:       repoRoot,
		Root:           store.absolute,
		Context:        reviewContext,
		Proposal:       proposal,
		ProposalDigest: digestBytes(proposalData),
		Events:         events,
		store:          store,
	}
	if err := validateReviewEvents(review); err != nil {
		return Review{}, nil, validationError("%v", err)
	}
	return review, validateProposal(reviewContext, proposal, store.ReadRegular), nil
}

// ValidateReview extends structural loading with candidate contracts and the
// resulting rule-skill graph without recording approval or mutating files.
func ValidateReview(
	ctx context.Context,
	repo *workspace.Repository,
	reviewID string,
) (Review, []Diagnostic, error) {
	review, diagnostics, err := LoadReview(repo.Root(), reviewID)
	if err != nil || len(diagnostics) != 0 {
		return review, diagnostics, err
	}
	approval := ApprovalPayload{}
	for _, action := range review.Proposal.Actions {
		if action.Disposition == DispositionUnableToDetermine {
			approval.Rejected = append(approval.Rejected, action.ID)
		} else {
			approval.Approved = append(approval.Approved, action.ID)
		}
	}
	sort.Strings(approval.Approved)
	sort.Strings(approval.Rejected)
	plan, err := buildApplicationPlan(review, approval, "")
	if err == nil {
		err = validateBuiltApplicationPlan(ctx, repo, review, plan, approval)
	}
	if err == nil {
		return review, diagnostics, nil
	}
	if errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, workspace.ErrGitOperation) {
		return Review{}, nil, err
	}
	diagnostics = append(diagnostics, Diagnostic{
		Path:     "proposal.yaml",
		Field:    "application_plan",
		Message:  err.Error(),
		Recovery: "correct the candidate or lifecycle actions, then rerun ssb prune validate",
	})
	return review, diagnostics, nil
}

func validateReviewEvents(review Review) error {
	seen := make(map[string]bool)
	applied := false
	rendered := false
	var approval ApprovalPayload
	var appliedResult ApplyResult
	applicationEventDigest := ""
	renderEventDigest := ""
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
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				payload.DryRun || payload.NoChangesApproved || !validDigest(payload.PlanDigest) ||
				len(payload.Changes) == 0 {
				return fmt.Errorf("application payload is invalid")
			}
			expected, err := canonicalApplicationPlan(review, approval)
			if err != nil {
				return fmt.Errorf("derive application plan: %w", err)
			}
			actualDigest, _ := canonicalDigest(payload.Changes)
			expectedDigest, _ := canonicalDigest(expected.Changes)
			if payload.PlanDigest != expected.PlanDigest || actualDigest != expectedDigest {
				return fmt.Errorf("application payload does not match approved changes")
			}
			applied = true
			appliedResult = payload
			applicationEventDigest = event.EventDigest
		case EventRendered:
			if !applied {
				return fmt.Errorf("rerender event precedes application")
			}
			if seen[EventVerified] {
				return fmt.Errorf("rerender event follows verification")
			}
			var payload renderedEventPayload
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				payload.Path != "AGENTS.md" || payload.DryRun ||
				!validDigest(payload.SourceDigest) || !validDigest(payload.ContentDigest) ||
				!validDigest(payload.OutputDigest) {
				return fmt.Errorf("rerender payload is invalid")
			}
			rendered = true
			renderEventDigest = event.EventDigest
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
			if err := decodeStrictJSON(event.Payload, &payload); err != nil ||
				len(payload.Receipts) == 0 ||
				payload.ApplicationEventDigest != applicationEventDigest ||
				payload.PlanDigest != appliedResult.PlanDigest ||
				payload.RenderEventDigest != renderEventDigest {
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
func Approve(ctx context.Context, repoPath string, options ApprovalOptions) (result Event, returnErr error) {
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return Event{}, err
	}
	store, unlock, err := acquireReviewLock(repo.Root(), options.ReviewID)
	if err != nil {
		return Event{}, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, unlock())
	}()
	review, diagnostics, err := loadReviewFromStore(store)
	if err != nil {
		return Event{}, err
	}
	if len(diagnostics) != 0 {
		return Event{}, validationError(
			"proposal is invalid: %s; run ssb prune validate",
			diagnostics[0].Message,
		)
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
	approval := ApprovalPayload{Approved: approvedIDs, Rejected: rejectedIDs}
	plan, err := buildApplicationPlan(review, approval, "")
	if err != nil {
		return Event{}, preconditionError(
			"approval cannot produce a safe application plan: %v",
			err,
		)
	}
	if err := validateBuiltApplicationPlan(ctx, repo, review, plan, approval); err != nil {
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, workspace.ErrGitOperation) {
			return Event{}, err
		}
		return Event{}, validationError(
			"approval candidate or resulting graph is invalid: %v",
			err,
		)
	}
	payload, err := json.Marshal(approval)
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

func acquireReviewLock(repoRoot, reviewID string) (*reviewStore, func() error, error) {
	store, err := openReviewStore(repoRoot, reviewID)
	if err != nil {
		return nil, nil, err
	}
	lockPath := store.display(".transition.lock")
	err = store.WriteExclusive(".transition.lock", []byte(reviewID+"\n"), 0o600)
	if err != nil {
		_ = store.Close()
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, preconditionError("review %s does not exist", reviewID)
		}
		if errors.Is(err, os.ErrExist) {
			return nil, nil, preconditionError("another review transition is active; if it crashed, confirm no process is active and remove %s", lockPath)
		}
		return nil, nil, fmt.Errorf("create review transition lock: %w", err)
	}
	return store, func() error {
		removeErr := removeReviewTransitionLock(store)
		closeErr := store.Close()
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(
				fmt.Errorf("remove review transition lock %s: %w", lockPath, removeErr),
				closeErr,
			)
		}
		return closeErr
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
	if len(reviewContext.Artifacts) == 0 {
		return fmt.Errorf("review context has no governed artifacts")
	}
	governedPaths := make([]string, 0, len(reviewContext.Artifacts))
	for _, artifact := range reviewContext.Artifacts {
		kind, id, canonical := artifactIdentity(artifact.Path)
		if !canonical || kind != artifact.Kind || id != artifact.ID ||
			!validDigest(artifact.SHA256) ||
			(artifact.Mode != "100644" && artifact.Mode != "100755") {
			return fmt.Errorf("review context artifact %s is invalid", artifact.Path)
		}
		switch artifact.Origin {
		case OriginGenerated, OriginUserAuthored, OriginMixed, OriginUnknown:
		default:
			return fmt.Errorf("review context artifact %s has invalid provenance", artifact.Path)
		}
		if artifact.Kind != ArtifactSkill && len(artifact.SupportingFiles) != 0 {
			return fmt.Errorf("review context rule %s has supporting files", artifact.Path)
		}
		governedPaths = append(governedPaths, artifact.Path)
		skillRoot := path.Dir(artifact.Path) + "/"
		for _, supporting := range artifact.SupportingFiles {
			if artifact.Kind != ArtifactSkill ||
				!strings.HasPrefix(supporting.Path, skillRoot) ||
				supporting.Path == artifact.Path ||
				!validDigest(supporting.SHA256) ||
				(supporting.Mode != "100644" && supporting.Mode != "100755") {
				return fmt.Errorf("review context supporting file %s is invalid", supporting.Path)
			}
			governedPaths = append(governedPaths, supporting.Path)
		}
	}
	if err := validateGovernedTreePaths(governedPaths); err != nil {
		return fmt.Errorf("review context governed paths are invalid: %w", err)
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

func validateReviewSnapshots(store *reviewStore, reviewContext Context) error {
	profileData, _, err := store.ReadRegular(reviewContext.CapabilityProfilePath)
	if err != nil {
		return fmt.Errorf("capability profile snapshot: %w", err)
	}
	if digestBytes(profileData) != reviewContext.CapabilityProfileDigest {
		return fmt.Errorf("capability profile snapshot: snapshot digest mismatch")
	}
	profileRoot := path.Dir(reviewContext.CapabilityProfilePath)
	for _, evidence := range reviewContext.Capabilities.Evidence {
		target := path.Join(profileRoot, evidence.Path)
		data, _, err := store.ReadRegular(target)
		if err != nil {
			return fmt.Errorf("capability evidence snapshot %s: %w", evidence.ID, err)
		}
		if digestBytes(data) != evidence.SHA256 {
			return fmt.Errorf("capability evidence snapshot %s: snapshot digest mismatch", evidence.ID)
		}
	}
	if reviewContext.ProvenanceManifestDigest != "" {
		data, _, err := store.ReadRegular(reviewContext.ProvenanceManifestPath)
		if err != nil {
			return fmt.Errorf("provenance snapshot: %w", err)
		}
		if digestBytes(data) != reviewContext.ProvenanceManifestDigest {
			return fmt.Errorf("provenance snapshot: snapshot digest mismatch")
		}
	}
	return nil
}

func reviewRoot(repoRoot, reviewID string) (string, error) {
	if !stableIDPattern.MatchString(reviewID) {
		return "", fmt.Errorf("review id must be lower-case kebab-case")
	}
	root := filepath.Join(repoRoot, ".software-standards", "reviews", reviewID)
	current := repoRoot
	for _, component := range []string{".software-standards", "reviews", reviewID} {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return root, nil
		}
		if err != nil {
			return "", fmt.Errorf("inspect review path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("review path %s contains a symlink", current)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("review path %s is not a directory", current)
		}
	}
	return root, nil
}

func writeExclusive(filePath string, data []byte, mode os.FileMode) (returnErr error) {
	file, err := openExclusiveFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", filePath, err)
	}
	return writeDurableExclusive(filePath, file, data, mode)
}

func writeDurableExclusive(
	filePath string,
	file durableExclusiveFile,
	data []byte,
	mode os.FileMode,
) (returnErr error) {
	return writeDurableExclusiveWithCleanup(
		filePath,
		file,
		data,
		mode,
		func() error { return removeIncompleteExclusiveFile(filePath) },
	)
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

func loadEvents(reader io.Reader) ([]Event, error) {
	events := make([]Event, 0)
	scanner := bufio.NewScanner(reader)
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
	store := review.store
	owned := false
	if store == nil {
		var err error
		store, err = openReviewStore(review.RepoRoot, review.Context.ReviewID)
		if err != nil {
			return fmt.Errorf("open anchored review event path: %w", err)
		}
		owned = true
	}
	if owned {
		defer store.Close()
	}
	if store.absolute != review.Root {
		return fmt.Errorf("review event path changed after loading")
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode review event: %w", err)
	}
	data = append(data, '\n')
	existing, _, err := store.ReadRegular("events.jsonl")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read review event log before append: %w", err)
	}
	if len(existing) != 0 && existing[len(existing)-1] != '\n' {
		return fmt.Errorf("review event log does not end at a complete record")
	}
	next := make([]byte, 0, len(existing)+len(data))
	next = append(next, existing...)
	next = append(next, data...)
	if err := writeEventLogAtomically(store, next, 0o644); err != nil {
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

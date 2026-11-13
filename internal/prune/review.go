package prune

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

// CreateResult identifies the immutable context written for a new review.
type CreateResult struct {
	Context     ReviewContext
	ContextPath string
}

// Review is a validated context/proposal pair and its exact proposal digest.
type Review struct {
	RepoRoot       string
	Root           string
	Context        ReviewContext
	Proposal       Proposal
	ProposalDigest string
	Events         []Event
	store          *reviewStore
}

// CreateReview writes only a deterministic context. Proposal generation and
// approval remain separate host-agent and human states.
func CreateReview(
	ctx context.Context,
	repo *workspace.Repository,
	options CreateReviewOptions,
) (CreateResult, error) {
	reviewContext, err := BuildReviewContext(ctx, repo, options)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, ErrIncompleteInventory) || errors.Is(err, workspace.ErrPrecondition) {
			return CreateResult{}, err
		}
		return CreateResult{}, validationError("%v", err)
	}
	if err := repo.VerifyPruneSnapshot(ctx); err != nil {
		return CreateResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return CreateResult{}, err
	}
	// Review publication is bounded; finish it or remove its owned staging tree.
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

func snapshotReviewInputs(staging *reviewStore, options CreateReviewOptions, reviewContext ReviewContext) error {
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
		if err := snapshotFile(
			source,
			staging,
			target,
			evidence.SHA256,
		); err != nil {
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
func LoadReview(ctx context.Context, repoRoot, reviewID string) (Review, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return Review{}, nil, err
	}
	store, err := openReviewStore(repoRoot, reviewID)
	if err != nil {
		return Review{}, nil, err
	}
	defer store.Close()
	review, diagnostics, err := loadReviewFromStore(ctx, store)
	review.store = nil
	return review, diagnostics, err
}

func loadReviewFromStore(ctx context.Context, store *reviewStore) (Review, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return Review{}, nil, err
	}
	repoRoot := store.repoRoot
	reviewID := store.reviewID
	contextData, _, err := store.ReadRegular("context.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Review{}, nil, preconditionError("review %s has no context.json", reviewID)
		}
		return Review{}, nil, fmt.Errorf("read review context: %w", err)
	}
	var reviewContext ReviewContext
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
	if err := yaml.Load(
		proposalData,
		&proposal,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
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
	if err := validateReviewEvents(ctx, review); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Review{}, nil, err
		}
		return Review{}, nil, validationError("%v", err)
	}
	diagnostics := validateProposal(reviewContext, proposal, func(name string) ([]byte, os.FileInfo, error) {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		return store.ReadRegular(name)
	})
	if err := ctx.Err(); err != nil {
		return Review{}, nil, err
	}
	return review, diagnostics, nil
}

// ValidateReview extends structural loading with candidate contracts and the
// resulting rule-skill graph without recording approval or mutating files.
func ValidateReview(
	ctx context.Context,
	repo *workspace.Repository,
	reviewID string,
) (Review, []Diagnostic, error) {
	review, diagnostics, err := LoadReview(ctx, repo.Root(), reviewID)
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
	plan, err := buildApplicationPlan(
		ctx,
		review,
		approval,
		"",
	)
	if err == nil {
		err = validateBuiltApplicationPlan(
			ctx,
			repo,
			review,
			plan,
			approval,
		)
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

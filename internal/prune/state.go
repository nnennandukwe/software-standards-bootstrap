package prune

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// Status preserves each lifecycle state instead of collapsing completion.
type Status struct {
	ReviewID          string `json:"review_id"`
	Inspected         bool   `json:"inspected"`
	Proposed          bool   `json:"proposed"`
	ProposalValid     bool   `json:"proposal_valid"`
	Approved          bool   `json:"approved"`
	NoChangesApproved bool   `json:"no_changes_approved"`
	Applied           bool   `json:"applied"`
	Rendered          bool   `json:"rendered"`
	ADRRecorded       bool   `json:"adr_recorded"`
	Verified          bool   `json:"verified"`
}

// ReviewStatus reads the bundle and derives every independent state.
func ReviewStatus(ctx context.Context, repoRoot, reviewID string) (Status, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return Status{}, nil, err
	}
	store, err := openReviewStore(repoRoot, reviewID)
	if err != nil {
		return Status{}, nil, err
	}
	defer store.Close()
	status := Status{ReviewID: reviewID}
	if _, _, err := store.ReadRegular("context.json"); err != nil {
		return Status{}, nil, fmt.Errorf("read review context: %w", err)
	}
	status.Inspected = true
	if _, _, err := store.ReadRegular("proposal.yaml"); err != nil {
		if os.IsNotExist(err) {
			return status, nil, nil
		}
		return Status{}, nil, err
	}
	status.Proposed = true
	review, diagnostics, err := loadReviewFromStore(ctx, store)
	if err != nil {
		return Status{}, nil, err
	}
	return deriveReviewStatus(ctx, review, diagnostics)
}

func deriveReviewStatus(
	ctx context.Context,
	review Review,
	diagnostics []Diagnostic,
) (Status, []Diagnostic, error) {
	status := Status{
		ReviewID:      review.Context.ReviewID,
		Inspected:     true,
		Proposed:      true,
		ProposalValid: len(diagnostics) == 0,
	}
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
	if status.Approved && !status.Applied {
		approval, approvalErr := approvalFor(review)
		if approvalErr != nil {
			return Status{}, nil, approvalErr
		}
		plan, planErr := canonicalApplicationPlan(ctx, review, approval)
		if planErr != nil {
			if errors.Is(planErr, context.Canceled) || errors.Is(planErr, context.DeadlineExceeded) {
				return Status{}, nil, planErr
			}
			diagnostics = append(diagnostics, Diagnostic{
				Path:     "events.jsonl",
				Field:    "application_plan",
				Message:  "approved decisions cannot produce a safe application plan: " + planErr.Error(),
				Recovery: "create a new review whose approved decisions preserve the actionable report contract",
			})
			return status, diagnostics, nil
		}
		status.NoChangesApproved = len(plan.Changes) == 0
	}
	return status, diagnostics, nil
}

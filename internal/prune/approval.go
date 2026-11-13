package prune

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

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
	store, unlock, err := acquireReviewLock(ctx, repo.Root(), options.ReviewID)
	if err != nil {
		return Event{}, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, unlock())
	}()
	review, diagnostics, err := loadReviewFromStore(ctx, store)
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
	plan, err := buildApplicationPlan(
		ctx,
		review,
		approval,
		"",
	)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Event{}, err
		}
		return Event{}, preconditionError(
			"approval cannot produce a safe application plan: %v",
			err,
		)
	}
	if err := validateBuiltApplicationPlan(
		ctx,
		repo,
		review,
		plan,
		approval,
	); err != nil {
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
	event := newEvent(
		review,
		EventApproved,
		payload,
		now().UTC(),
	)
	if err := appendEvent(ctx, review, event); err != nil {
		return Event{}, err
	}
	return event, nil
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

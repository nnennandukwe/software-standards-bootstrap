package prune

import (
	"fmt"
	"sort"
	"strings"
)

const applicationPlanSchema = "ssb.dev/prune-application-plan/v1"

// FileState identifies exact governed bytes and mode, or explicit absence.
type FileState struct {
	Exists bool   `json:"exists"`
	SHA256 string `json:"sha256,omitempty"`
	Mode   string `json:"mode,omitempty"`
}

type plannedFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   string `json:"mode"`
}

type applicationPlan struct {
	Schema              string        `json:"schema"`
	ReviewID            string        `json:"review_id"`
	BaselineCommit      string        `json:"baseline_commit"`
	ContextDigest       string        `json:"context_digest"`
	ProposalDigest      string        `json:"proposal_digest"`
	ApprovalEventDigest string        `json:"approval_event_digest"`
	Changes             []Change      `json:"changes"`
	Poststate           []plannedFile `json:"poststate"`
	PlanDigest          string        `json:"plan_digest"`
}

func canonicalApplicationPlan(review Review, approval ApprovalPayload) (applicationPlan, error) {
	approvalEvent, err := recordedApprovalEvent(review)
	if err != nil {
		return applicationPlan{}, err
	}
	return buildApplicationPlan(review, approval, approvalEvent.EventDigest)
}

func buildApplicationPlan(
	review Review,
	approval ApprovalPayload,
	approvalEventDigest string,
) (applicationPlan, error) {
	initial := make(map[string]FileState)
	for _, artifact := range review.Context.Artifacts {
		initial[artifact.Path] = FileState{
			Exists: true,
			SHA256: artifact.SHA256,
			Mode:   artifact.Mode,
		}
		for _, supporting := range artifact.SupportingFiles {
			initial[supporting.Path] = FileState{
				Exists: true,
				SHA256: supporting.SHA256,
				Mode:   supporting.Mode,
			}
		}
	}
	poststate := make(map[string]FileState, len(initial))
	for itemPath, state := range initial {
		poststate[itemPath] = state
	}

	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	changes := make(map[string]Change)
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok || action.Disposition == DispositionKeep {
			continue
		}
		switch action.Disposition {
		case DispositionRemove, DispositionUpdate, DispositionConsolidate:
		default:
			return applicationPlan{}, fmt.Errorf(
				"approved action %s has non-applicable disposition %s",
				action.ID,
				action.Disposition,
			)
		}
		for _, source := range action.Sources {
			before, exists := initial[source.Path]
			if !exists {
				return applicationPlan{}, fmt.Errorf("approved source %s is absent from context", source.Path)
			}
			changes[source.Path] = Change{
				ActionID: action.ID,
				Path:     source.Path,
				Kind:     "remove",
				Prestate: before,
				Poststate: FileState{
					Exists: false,
				},
			}
			delete(poststate, source.Path)
			if actionRemovesSupportingFiles(action, source) {
				for _, supporting := range supportingFilesFor(review, source.Path) {
					before := initial[supporting.Path]
					changes[supporting.Path] = Change{
						ActionID: action.ID,
						Path:     supporting.Path,
						Kind:     "remove",
						Prestate: before,
						Poststate: FileState{
							Exists: false,
						},
					}
					delete(poststate, supporting.Path)
				}
			}
		}
		if action.Target == nil {
			continue
		}
		write := func(targetPath, digest, mode string) {
			after := FileState{Exists: true, SHA256: digest, Mode: mode}
			changes[targetPath] = Change{
				ActionID:  action.ID,
				Path:      targetPath,
				Kind:      "write",
				Prestate:  initial[targetPath],
				Poststate: after,
			}
			poststate[targetPath] = after
		}
		write(action.Target.TargetPath, action.Target.SHA256, action.Target.Mode)
		for _, supporting := range action.Target.SupportingFiles {
			write(supporting.TargetPath, supporting.SHA256, supporting.Mode)
		}
	}

	paths := make([]string, 0, len(changes))
	for itemPath := range changes {
		paths = append(paths, itemPath)
	}
	sort.Strings(paths)
	orderedChanges := make([]Change, 0, len(paths))
	for _, itemPath := range paths {
		orderedChanges = append(orderedChanges, changes[itemPath])
	}

	poststatePaths := make([]string, 0, len(poststate))
	for itemPath := range poststate {
		poststatePaths = append(poststatePaths, itemPath)
	}
	sort.Strings(poststatePaths)
	folded := make(map[string]string, len(poststate))
	rules := 0
	for _, itemPath := range poststatePaths {
		foldedPath := strings.ToLower(itemPath)
		if prior, collision := folded[foldedPath]; collision && prior != itemPath {
			return applicationPlan{}, fmt.Errorf(
				"configuration paths %s and %s collide on case-insensitive filesystems",
				prior,
				itemPath,
			)
		}
		folded[foldedPath] = itemPath
		if kind, _, ok := artifactIdentity(itemPath); ok && kind == ArtifactRule {
			rules++
		}
	}
	if rules == 0 {
		return applicationPlan{}, fmt.Errorf("application would remove every rule; retain or replace at least one rule")
	}
	orderedPoststate := make([]plannedFile, 0, len(poststatePaths))
	for _, itemPath := range poststatePaths {
		state := poststate[itemPath]
		orderedPoststate = append(orderedPoststate, plannedFile{
			Path: itemPath, SHA256: state.SHA256, Mode: state.Mode,
		})
	}

	plan := applicationPlan{
		Schema:              applicationPlanSchema,
		ReviewID:            review.Context.ReviewID,
		BaselineCommit:      review.Context.BaselineCommit,
		ContextDigest:       review.Context.ContextDigest,
		ProposalDigest:      review.ProposalDigest,
		ApprovalEventDigest: approvalEventDigest,
		Changes:             orderedChanges,
		Poststate:           orderedPoststate,
	}
	planDigest, err := canonicalDigest(plan)
	if err != nil {
		return applicationPlan{}, fmt.Errorf("digest application plan: %w", err)
	}
	plan.PlanDigest = planDigest
	return plan, nil
}

func recordedApprovalEvent(review Review) (Event, error) {
	for index := len(review.Events) - 1; index >= 0; index-- {
		if review.Events[index].Kind == EventApproved {
			return review.Events[index], nil
		}
	}
	return Event{}, fmt.Errorf("review has no approval event")
}

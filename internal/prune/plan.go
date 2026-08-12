package prune

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const applicationPlanSchema = "ssb.dev/prune-application-plan/v1"
const actionableReportPath = ".software-standards/report.md"
const actionableManifestPath = ".software-standards/manifest.yaml"

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
	reportChange, _, err := actionableMetadataChange(review, approval)
	if err != nil {
		return applicationPlan{}, err
	}
	if reportChange != nil {
		changes[reportChange.Path] = *reportChange
		poststate[reportChange.Path] = reportChange.Poststate
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

func actionableMetadataChange(
	review Review,
	approval ApprovalPayload,
) (*Change, []byte, error) {
	if review.RepoRoot == "" {
		return nil, nil, nil
	}
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	removedIDs := make(map[string]struct{})
	updatedDigests := make(map[string]string)
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok || action.Disposition == DispositionKeep {
			continue
		}
		for _, source := range action.Sources {
			removedIDs[source.ID] = struct{}{}
		}
		if action.Target == nil {
			continue
		}
		retainsManifestEntry := false
		for _, source := range action.Sources {
			if source.Kind == action.Target.Kind &&
				source.ID == action.Target.ID &&
				source.Path == action.Target.TargetPath {
				retainsManifestEntry = true
				break
			}
		}
		if !retainsManifestEntry {
			return nil, nil, fmt.Errorf(
				"action %s changes the canonical artifact id or path; create a new actionable pack so its manifest can record fresh provenance, confidence, and utility",
				action.ID,
			)
		}
		delete(removedIDs, action.Target.ID)
		updatedDigests[action.Target.ID] = action.Target.SHA256
	}
	if len(removedIDs) == 0 && len(updatedDigests) == 0 {
		return nil, nil, nil
	}

	repo, err := workspace.Open(context.Background(), review.RepoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("open repository for pack metadata update: %w", err)
	}
	reviewBaseline, err := repo.AtCommit(context.Background(), review.Context.BaselineCommit)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"open actionable pack at review baseline %s: %w",
			review.Context.BaselineCommit,
			err,
		)
	}
	metadataPath := actionableReportPath
	manifestEntry, manifestExists, err := reviewBaseline.EntryAtBaseline(context.Background(), actionableManifestPath)
	if err != nil {
		return nil, nil, err
	}
	if manifestExists {
		if manifestEntry.Kind != "blob" || (manifestEntry.Mode != "100644" && manifestEntry.Mode != "100755") {
			return nil, nil, fmt.Errorf("actionable manifest is not a tracked regular file")
		}
		metadataPath = actionableManifestPath
	}
	metadataData, err := reviewBaseline.ReadBaselineFile(context.Background(), metadataPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read actionable pack metadata at review baseline: %w", err)
	}
	metadataEntry, exists, err := reviewBaseline.EntryAtBaseline(context.Background(), metadataPath)
	if err != nil {
		return nil, nil, err
	}
	if !exists || metadataEntry.Kind != "blob" ||
		(metadataEntry.Mode != "100644" && metadataEntry.Mode != "100755") {
		return nil, nil, fmt.Errorf("actionable pack metadata is not a tracked regular file")
	}
	var updated []byte
	if metadataPath == actionableManifestPath {
		updated, err = rulepack.UpdateManifestArtifacts(metadataData, removedIDs, updatedDigests)
		if err != nil {
			return nil, nil, fmt.Errorf("update actionable manifest: %w", err)
		}
	} else {
		if len(removedIDs) == 0 {
			return nil, nil, nil
		}
		updated, err = rulepack.RemoveReportArtifacts(metadataData, removedIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("update actionable report manifest: %w", err)
		}
	}
	if bytes.Equal(updated, metadataData) {
		return nil, nil, fmt.Errorf(
			"pack metadata does not reflect the approved artifact change; refresh the review from a valid actionable pack",
		)
	}
	change := &Change{
		ActionID: "pack-manifest",
		Path:     metadataPath,
		Kind:     "write",
		Prestate: FileState{
			Exists: true,
			SHA256: digestBytes(metadataData),
			Mode:   metadataEntry.Mode,
		},
		Poststate: FileState{
			Exists: true,
			SHA256: digestBytes(updated),
			Mode:   metadataEntry.Mode,
		},
	}
	return change, updated, nil
}

func recordedApprovalEvent(review Review) (Event, error) {
	for index := len(review.Events) - 1; index >= 0; index-- {
		if review.Events[index].Kind == EventApproved {
			return review.Events[index], nil
		}
	}
	return Event{}, fmt.Errorf("review has no approval event")
}

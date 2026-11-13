package prune

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func approvalFor(review Review) (ApprovalPayload, error) {
	for index := len(review.Events) - 1; index >= 0; index-- {
		if review.Events[index].Kind != EventApproved {
			continue
		}
		if review.Events[index].ProposalDigest != review.ProposalDigest {
			return ApprovalPayload{}, fmt.Errorf("approval is bound to a different proposal digest")
		}
		var payload ApprovalPayload
		if err := decodeStrictJSON(review.Events[index].Payload, &payload); err != nil {
			return ApprovalPayload{}, fmt.Errorf("parse approval payload: %w", err)
		}
		return payload, nil
	}
	return ApprovalPayload{}, fmt.Errorf("review has no approval event")
}

func planOperations(
	ctx context.Context,
	repo *workspace.Repository,
	review Review,
	approval ApprovalPayload,
) (applicationPlan, []operation, error) {
	plan, err := canonicalApplicationPlan(ctx, review, approval)
	if err != nil {
		return applicationPlan{}, nil, err
	}
	if err := validateApplicationPlanModes(plan, runtime.GOOS); err != nil {
		return applicationPlan{}, nil, err
	}

	for _, change := range plan.Changes {
		target, err := resolvePortablePath(repo.Root(), change.Path)
		if err != nil {
			return applicationPlan{}, nil, err
		}
		if change.Prestate.Exists {
			if err := requireCurrentDigest(target, change.Prestate.SHA256); err != nil {
				return applicationPlan{}, nil, fmt.Errorf(
					"approved source %s no longer matches its context digest: %w",
					change.Path,
					err,
				)
			}
		} else if _, err := os.Lstat(target); err == nil {
			return applicationPlan{}, nil, fmt.Errorf(
				"candidate target %s would overwrite an unreviewed worktree file",
				change.Path,
			)
		} else if !errors.Is(err, os.ErrNotExist) {
			return applicationPlan{}, nil, fmt.Errorf("inspect candidate target %s: %w", change.Path, err)
		}
	}

	byPath, err := buildCandidateOperations(
		ctx,
		repo,
		review,
		plan,
		approval,
	)
	if err != nil {
		return applicationPlan{}, nil, err
	}
	operations := make([]operation, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		item := byPath[change.Path]
		item.ExpectedAbsent = !change.Prestate.Exists
		item.ExpectedSHA256 = change.Prestate.SHA256
		byPath[change.Path] = item
		operations = append(operations, item)
	}
	return plan, operations, nil
}

func candidateInputsForApproval(
	review Review,
	approval ApprovalPayload,
) map[string]candidateInput {
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	candidates := make(map[string]candidateInput)
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok || action.Target == nil {
			continue
		}
		targetCopy := *action.Target
		candidates[action.Target.TargetPath] = candidateInput{
			sourcePath: action.Target.SourcePath,
			target:     &targetCopy,
			mode:       action.Target.Mode,
		}
		for _, supporting := range action.Target.SupportingFiles {
			candidates[supporting.TargetPath] = candidateInput{
				sourcePath: supporting.SourcePath,
				mode:       supporting.Mode,
			}
		}
	}
	return candidates
}

func validateApplicationPlanModes(plan applicationPlan, goos string) error {
	if goos != "windows" {
		return nil
	}
	for _, change := range plan.Changes {
		for _, state := range []struct {
			name string
			file FileState
		}{
			{name: "prestate", file: change.Prestate},
			{name: "poststate", file: change.Poststate},
		} {
			if state.file.Exists && state.file.Mode == "100755" {
				return preconditionError(
					"application plan %s for %s uses Git executable mode 100755, which cannot be safely materialized on Windows; apply this review from a POSIX host because ssb will not stage an index-only chmod",
					state.name,
					change.Path,
				)
			}
		}
	}
	return nil
}

func supportingFilesFor(review Review, skillPath string) []ArtifactFile {
	for _, artifact := range review.Context.Artifacts {
		if artifact.Path == skillPath {
			return artifact.SupportingFiles
		}
	}
	return nil
}

func actionRemovesSupportingFiles(action Action, source ArtifactRef) bool {
	return source.Kind == ArtifactSkill &&
		(action.Disposition == DispositionRemove ||
			action.Disposition == DispositionConsolidate ||
			action.Disposition == DispositionUpdate)
}

func readCandidateFile(review Review, sourcePath string) ([]byte, error) {
	store := review.store
	owned := false
	if store == nil {
		var err error
		store, err = openReviewStore(review.RepoRoot, review.Context.ReviewID)
		if err != nil {
			return nil, err
		}
		owned = true
	}
	if owned {
		defer store.Close()
	}
	data, info, err := store.ReadRegular(sourcePath)
	if err != nil {
		return nil, err
	}
	limit := review.Context.Inventory.Limits.MaxFileBytes
	if limit <= 0 || info.Size() > limit {
		return nil, fmt.Errorf("candidate exceeds max_file_bytes=%d", limit)
	}
	return data, nil
}

func candidateMode(mode string) os.FileMode {
	if mode == "100755" {
		return 0o755
	}
	return 0o644
}

func validateCandidate(
	ctx context.Context,
	repo *workspace.Repository,
	target CandidateRef,
	content []byte,
	packLayout rulepack.Layout,
	manifestByID map[string]rulepack.AcceptedArtifact,
) error {
	manifest, manifestExists := manifestByID[target.ID]
	switch target.Kind {
	case ArtifactRule:
		var diagnostics []rulepack.Diagnostic
		if packLayout == rulepack.LayoutManifest {
			if !manifestExists || manifest.Kind != ArtifactRule {
				return fmt.Errorf("candidate %s has no matching rule entry in manifest.yaml", target.SourcePath)
			}
			_, diagnostics = rulepack.ValidateManifestCandidateRule(target.TargetPath, manifest, content)
		} else {
			_, diagnostics = rulepack.ValidateCandidateRule(
				ctx,
				repo,
				target.TargetPath,
				content,
			)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(diagnostics) != 0 {
			return fmt.Errorf("candidate %s violates the rule contract: %s", target.SourcePath, diagnostics[0].Message)
		}
	case ArtifactSkill:
		var skill rulepack.Skill
		var diagnostics []rulepack.Diagnostic
		if packLayout == rulepack.LayoutManifest {
			if !manifestExists || manifest.Kind != ArtifactSkill {
				return fmt.Errorf("candidate %s has no matching skill entry in manifest.yaml", target.SourcePath)
			}
			skill, diagnostics = rulepack.ValidateManifestCandidateSkill(target.TargetPath, manifest, content)
		} else {
			skill, diagnostics = rulepack.ValidateCandidateSkill(target.TargetPath, target.ID, content)
		}
		if len(diagnostics) != 0 {
			return fmt.Errorf("candidate %s violates the Agent Skill contract: %s", target.SourcePath, diagnostics[0].Message)
		}
		if !manifestExists || manifest.Kind != ArtifactSkill {
			return fmt.Errorf("candidate %s has no matching skill entry in the pack manifest", target.SourcePath)
		}
		if packLayout == rulepack.LayoutEmbedded && skill.Category != manifest.Category {
			return fmt.Errorf(
				"candidate %s changes metadata.category from %s to %s; create a new actionable pack so report.md can record fresh skill provenance",
				target.SourcePath,
				manifest.Category,
				skill.Category,
			)
		}
	default:
		return fmt.Errorf("candidate %s has unsupported kind %s", target.SourcePath, target.Kind)
	}
	return nil
}

func buildCandidateOperations(
	ctx context.Context,
	repo *workspace.Repository,
	review Review,
	plan applicationPlan,
	approval ApprovalPayload,
) (map[string]operation, error) {
	candidates := candidateInputsForApproval(review, approval)
	_, reportContent, err := actionableMetadataChange(ctx, review, approval)
	if err != nil {
		return nil, err
	}
	manifestByID := make(map[string]rulepack.AcceptedArtifact)
	packLayout := rulepack.LayoutEmbedded
	needsPackManifest := false
	for _, candidate := range candidates {
		if candidate.target != nil {
			needsPackManifest = true
			break
		}
	}
	if needsPackManifest {
		pack, diagnostics, err := rulepack.ValidateRetainedPack(ctx, repo)
		if err != nil {
			return nil, fmt.Errorf("validate retained actionable pack: %w", err)
		}
		if len(diagnostics) != 0 {
			return nil, fmt.Errorf("retained actionable pack is invalid: %s", diagnostics[0].Message)
		}
		packLayout = pack.Layout
		for _, artifact := range pack.Manifest.Artifacts {
			manifestByID[artifact.ID] = artifact
		}
	}
	operations := make(map[string]operation, len(plan.Changes))
	for _, change := range plan.Changes {
		item := operation{Change: change}
		if change.Poststate.Exists {
			if change.Path == actionableReportPath || change.Path == actionableManifestPath {
				if len(reportContent) == 0 ||
					digestBytes(reportContent) != change.Poststate.SHA256 {
					return nil, fmt.Errorf("application plan pack metadata content does not match its poststate")
				}
				item.Content = reportContent
				item.Mode = candidateMode(change.Poststate.Mode)
				operations[change.Path] = item
				continue
			}
			candidate, exists := candidates[change.Path]
			if !exists {
				return nil, fmt.Errorf("application plan lacks candidate input for %s", change.Path)
			}
			content, err := readCandidateFile(review, candidate.sourcePath)
			if err != nil {
				return nil, fmt.Errorf("read candidate %s: %w", candidate.sourcePath, err)
			}
			if digestBytes(content) != change.Poststate.SHA256 {
				return nil, fmt.Errorf(
					"candidate %s no longer matches its proposal digest",
					candidate.sourcePath,
				)
			}
			if candidate.target != nil {
				if err := validateCandidate(
					ctx,
					repo,
					*candidate.target,
					content,
					packLayout,
					manifestByID,
				); err != nil {
					return nil, err
				}
			}
			item.Content = content
			item.Mode = candidateMode(candidate.mode)
		}
		operations[change.Path] = item
	}
	if err := validateResultingGraph(
		ctx,
		repo,
		review,
		operations,
	); err != nil {
		return nil, err
	}
	return operations, nil
}

func validateBuiltApplicationPlan(
	ctx context.Context,
	repo *workspace.Repository,
	review Review,
	plan applicationPlan,
	approval ApprovalPayload,
) error {
	if err := validateApplicationPlanModes(plan, runtime.GOOS); err != nil {
		return err
	}
	_, err := buildCandidateOperations(
		ctx,
		repo,
		review,
		plan,
		approval,
	)
	return err
}

func validateResultingGraph(
	ctx context.Context,
	repo *workspace.Repository,
	review Review,
	operations map[string]operation,
) error {
	repoRoot := repo.Root()
	pack, packDiagnostics, err := rulepack.ValidateRetainedPack(ctx, repo)
	if err != nil {
		return fmt.Errorf("validate retained actionable pack: %w", err)
	}
	if len(packDiagnostics) != 0 {
		return fmt.Errorf("retained actionable pack is invalid: %s", packDiagnostics[0].Message)
	}
	manifestByID := make(map[string]rulepack.AcceptedArtifact, len(pack.Manifest.Artifacts))
	for _, artifact := range pack.Manifest.Artifacts {
		manifestByID[artifact.ID] = artifact
	}
	type finalArtifact struct {
		kind    string
		content []byte
	}
	final := make(map[string]finalArtifact, len(review.Context.Artifacts))
	contextArtifacts := make(map[string]struct{}, len(review.Context.Artifacts))
	for _, artifact := range review.Context.Artifacts {
		contextArtifacts[artifact.Path] = struct{}{}
		if operation, changed := operations[artifact.Path]; changed {
			if operation.Kind == "remove" {
				continue
			}
			final[artifact.Path] = finalArtifact{
				kind:    artifact.Kind,
				content: operation.Content,
			}
			continue
		}
		target, err := resolvePortablePath(repoRoot, artifact.Path)
		if err != nil {
			return fmt.Errorf("resolve resulting-graph source %s: %w", artifact.Path, err)
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("read resulting-graph source %s: %w", artifact.Path, err)
		}
		final[artifact.Path] = finalArtifact{kind: artifact.Kind, content: content}
	}
	for itemPath, operation := range operations {
		if _, existed := contextArtifacts[itemPath]; existed {
			continue
		}
		if operation.Kind == "remove" {
			delete(final, itemPath)
			continue
		}
		kind, _, ok := artifactIdentity(itemPath)
		if !ok {
			if validApplicationPath(itemPath) {
				continue
			}
			return fmt.Errorf("resulting graph contains unsafe target %s", itemPath)
		}
		final[itemPath] = finalArtifact{kind: kind, content: operation.Content}
	}
	if pack.Orientation != nil {
		for _, relatedID := range pack.Orientation.RelatedArtifactIDs {
			related := manifestByID[relatedID]
			if related.Kind != ArtifactSkill {
				continue
			}
			if _, retained := final[related.Path]; !retained {
				return fmt.Errorf(
					"resulting graph removes Agent Skill %s while orientation still references it; create a new reviewed pack that revises orientation before removing the skill",
					relatedID,
				)
			}
		}
	}
	for itemPath, artifact := range final {
		if artifact.kind != ArtifactSkill {
			continue
		}
		_, id, _ := artifactIdentity(itemPath)
		var diagnostics []rulepack.Diagnostic
		if pack.Layout == rulepack.LayoutManifest {
			_, diagnostics = rulepack.ValidateManifestCandidateSkill(itemPath, manifestByID[id], artifact.content)
		} else {
			_, diagnostics = rulepack.ValidateCandidateSkill(itemPath, id, artifact.content)
		}
		if len(diagnostics) != 0 {
			return fmt.Errorf("resulting skill %s violates the Agent Skill contract: %s", itemPath, diagnostics[0].Message)
		}
	}
	for itemPath, artifact := range final {
		if artifact.kind != ArtifactRule {
			continue
		}
		var rule rulepack.Rule
		var diagnostics []rulepack.Diagnostic
		if pack.Layout == rulepack.LayoutManifest {
			_, id, _ := artifactIdentity(itemPath)
			rule, diagnostics = rulepack.ValidateManifestCandidateRule(itemPath, manifestByID[id], artifact.content)
		} else {
			var validateErr error
			rule, diagnostics, validateErr = rulepack.ValidateRetainedRule(
				ctx,
				repo,
				itemPath,
				artifact.content,
			)
			if validateErr != nil {
				return fmt.Errorf("validate resulting rule %s: %w", itemPath, validateErr)
			}
		}
		if len(diagnostics) != 0 {
			return fmt.Errorf("resulting rule %s violates the rule contract: %s", itemPath, diagnostics[0].Message)
		}
		_, pathID, _ := artifactIdentity(itemPath)
		if rule.ID != pathID {
			return fmt.Errorf("resulting rule %s declares id %s", itemPath, rule.ID)
		}
	}
	return nil
}

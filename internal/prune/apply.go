package prune

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const journalSchema = "ssb.dev/prune-application-journal/v1"

var (
	removeApplicationJournal = func(store *reviewStore) error {
		return store.Remove("application-journal.json")
	}
	removePruneMutationLock = func(root *os.Root, packIdentity os.FileInfo) error {
		pack, err := openPinnedPack(root, packIdentity)
		if err != nil {
			return err
		}
		defer pack.Close()
		return pack.Remove(".prune-mutation.lock")
	}
	publishApplicationFile           = publishApplicationFileDurably
	removeApplicationPublicationFile = removeApplicationPublicationFileDurably
)

const (
	prestateClaim    = "pre"
	poststateClaim   = "post"
	publicationClaim = "publication"
)

// ApplyOptions selects a review and requires Write for mutation.
type ApplyOptions struct {
	ReviewID string
	Write    bool
	Now      func() time.Time
}

// Change is one complete-file operation in an application plan.
type Change struct {
	ActionID  string    `json:"action_id"`
	Path      string    `json:"path"`
	Kind      string    `json:"kind"`
	Prestate  FileState `json:"prestate"`
	Poststate FileState `json:"poststate"`
}

// ApplyResult reports the bounded plan or completed application.
type ApplyResult struct {
	DryRun            bool     `json:"dry_run"`
	NoChangesApproved bool     `json:"no_changes_approved"`
	PlanDigest        string   `json:"plan_digest"`
	Changes           []Change `json:"changes"`
}

type operation struct {
	Change
	Content        []byte
	ExpectedAbsent bool
	ExpectedSHA256 string
	Mode           os.FileMode
}

type candidateInput struct {
	sourcePath string
	target     *CandidateRef
	mode       string
}

type journalEntry struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"`
	Mode    uint32 `json:"mode,omitempty"`
	Content []byte `json:"content,omitempty"`
}

type applicationJournal struct {
	Schema             string         `json:"schema"`
	ReviewID           string         `json:"review_id"`
	ProposalDigest     string         `json:"proposal_digest"`
	PlanDigest         string         `json:"plan_digest"`
	Entries            []journalEntry `json:"entries"`
	CreatedDirectories []string       `json:"created_directories,omitempty"`
}

// Apply computes a dry run by default and mutates only when Write is true.
func Apply(
	ctx context.Context,
	repoPath string,
	options ApplyOptions,
) (returned ApplyResult, returnErr error) {
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return ApplyResult{}, err
	}
	var lockedReviewStore *reviewStore
	var unlockReview func() error
	var unlockMutation func() error
	defer func() {
		if unlockMutation != nil {
			returnErr = errors.Join(returnErr, unlockMutation())
		}
		if unlockReview != nil {
			returnErr = errors.Join(returnErr, unlockReview())
		}
	}()
	if options.Write {
		lockedReviewStore, unlockReview, err = acquireReviewLock(repo.Root(), options.ReviewID)
		if err != nil {
			return ApplyResult{}, err
		}
		unlockMutation, err = acquireMutationLock(repo.Root(), options.ReviewID)
		if err != nil {
			return ApplyResult{}, err
		}
	}
	var review Review
	var diagnostics []Diagnostic
	if lockedReviewStore != nil {
		review, diagnostics, err = loadReviewFromStore(lockedReviewStore)
	} else {
		review, diagnostics, err = LoadReview(repo.Root(), options.ReviewID)
	}
	if err != nil {
		return ApplyResult{}, err
	}
	if len(diagnostics) != 0 {
		return ApplyResult{}, validationError(
			"proposal is invalid: %s; run ssb prune validate",
			diagnostics[0].Message,
		)
	}
	approval, err := approvalFor(review)
	if err != nil {
		return ApplyResult{}, preconditionError("%v", err)
	}
	for _, event := range review.Events {
		if event.Kind == EventApplied {
			return ApplyResult{}, preconditionError("review already has an application event")
		}
	}
	if repo.Baseline() != review.Context.BaselineCommit {
		return ApplyResult{}, preconditionError("review baseline is stale: expected %s, found %s", review.Context.BaselineCommit, repo.Baseline())
	}
	status, err := repo.Git(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=no", "--ignore-submodules=untracked")
	if err != nil {
		return ApplyResult{}, fmt.Errorf("read tracked worktree state: %w", err)
	}
	if len(status) != 0 {
		return ApplyResult{}, preconditionError("tracked or staged changes block application; commit, stash, or restore them")
	}
	if err := repo.RejectUntrackedConfigurations(ctx); err != nil {
		return ApplyResult{}, err
	}
	plan, operations, err := planOperations(ctx, repo, review, approval)
	if err != nil {
		return ApplyResult{}, classifyApplicationPlanError(err)
	}
	result := ApplyResult{
		DryRun:     !options.Write,
		PlanDigest: plan.PlanDigest,
		Changes:    make([]Change, len(plan.Changes)),
	}
	copy(result.Changes, plan.Changes)
	if len(result.Changes) == 0 {
		result.NoChangesApproved = true
		return result, nil
	}
	if !options.Write {
		return result, nil
	}
	journal, err := captureJournal(repo.Root(), review, plan, operations)
	if err != nil {
		return ApplyResult{}, err
	}
	journalStore := lockedReviewStore
	journalData, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return ApplyResult{}, fmt.Errorf("encode application journal: %w", err)
	}
	if err := journalStore.WriteExclusive("application-journal.json", append(journalData, '\n'), 0o600); err != nil {
		return ApplyResult{}, fmt.Errorf("application recovery journal already exists or cannot be created: %w", err)
	}
	completed, executeErr := executeOperations(repo.Root(), operations)
	if executeErr != nil {
		rollbackErr := restoreJournalPaths(repo.Root(), journal, completed, operationPoststates(operations))
		if rollbackErr == nil {
			if removeErr := removeApplicationJournal(journalStore); removeErr != nil {
				return ApplyResult{}, fmt.Errorf(
					"apply review: %w; changed files were restored but recovery journal cleanup failed: %v; run ssb prune recover --review %s",
					executeErr,
					removeErr,
					options.ReviewID,
				)
			}
			return ApplyResult{}, fmt.Errorf("apply review: %w; all changed files were restored", executeErr)
		}
		return ApplyResult{}, fmt.Errorf(
			"apply review: %w; rollback also failed: %v; run ssb prune recover --review %s",
			executeErr,
			rollbackErr,
			options.ReviewID,
		)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		rollbackErr := restoreJournal(repo.Root(), journal, operationPoststates(operations))
		if rollbackErr == nil {
			if removeErr := removeApplicationJournal(journalStore); removeErr != nil {
				return ApplyResult{}, fmt.Errorf(
					"encode application event: %w; changed files were restored but recovery journal cleanup failed: %v; run ssb prune recover --review %s",
					err,
					removeErr,
					options.ReviewID,
				)
			}
			return ApplyResult{}, fmt.Errorf("encode application event: %w; all changed files were restored", err)
		}
		return ApplyResult{}, fmt.Errorf(
			"encode application event: %w; rollback failed: %v; run ssb prune recover --review %s",
			err,
			rollbackErr,
			options.ReviewID,
		)
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	event := newEvent(review, EventApplied, payload, now().UTC())
	if err := appendEvent(review, event); err != nil {
		rollbackErr := restoreJournal(repo.Root(), journal, operationPoststates(operations))
		if rollbackErr == nil {
			if removeErr := removeApplicationJournal(journalStore); removeErr != nil {
				return ApplyResult{}, fmt.Errorf(
					"record application event: %w; changed files were restored but recovery journal cleanup failed: %v; run ssb prune recover --review %s",
					err,
					removeErr,
					options.ReviewID,
				)
			}
			return ApplyResult{}, fmt.Errorf("record application event: %w; all changed files were restored", err)
		}
		return ApplyResult{}, fmt.Errorf(
			"record application event: %w; rollback failed: %v; run ssb prune recover --review %s",
			err,
			rollbackErr,
			options.ReviewID,
		)
	}
	if err := unlockMutation(); err != nil {
		return ApplyResult{}, fmt.Errorf(
			"application succeeded but lock cleanup failed: %w; run ssb prune recover --review %s --clear-stale-lock",
			err,
			options.ReviewID,
		)
	}
	unlockMutation = nil
	if err := removeReviewTransitionLock(lockedReviewStore); err != nil {
		return ApplyResult{}, fmt.Errorf(
			"application succeeded but lock cleanup failed: %w; run ssb prune recover --review %s --clear-stale-lock",
			err,
			options.ReviewID,
		)
	}
	// The transition lock is gone, but the identity-pinned descriptor remains
	// open until the journal is removed. This avoids reopening a replaced path.
	unlockReview = lockedReviewStore.Close
	if err := cleanupApplicationJournal(journalStore, options.ReviewID); err != nil {
		return ApplyResult{}, err
	}
	closeReviewStore := unlockReview
	unlockReview = nil
	if err := closeReviewStore(); err != nil {
		return ApplyResult{}, fmt.Errorf(
			"application succeeded but review descriptor cleanup failed: %w",
			err,
		)
	}
	return result, nil
}

func cleanupApplicationJournal(store *reviewStore, reviewID string) error {
	if err := removeApplicationJournal(store); err != nil {
		return fmt.Errorf(
			"application succeeded but recovery journal cleanup failed: %w; run ssb prune recover --review %s",
			err,
			reviewID,
		)
	}
	return nil
}

func classifyApplicationPlanError(err error) error {
	if errors.Is(err, ErrPrecondition) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, workspace.ErrGitOperation) {
		return err
	}
	return validationError("%v", err)
}

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
	plan, err := canonicalApplicationPlan(review, approval)
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

	byPath, err := buildCandidateOperations(ctx, repo, review, plan, approval)
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
			_, diagnostics = rulepack.ValidateCandidateRule(ctx, repo, target.TargetPath, content)
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
	_, reportContent, err := actionableMetadataChange(review, approval)
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
				if err := validateCandidate(ctx, repo, *candidate.target, content, packLayout, manifestByID); err != nil {
					return nil, err
				}
			}
			item.Content = content
			item.Mode = candidateMode(candidate.mode)
		}
		operations[change.Path] = item
	}
	if err := validateResultingGraph(ctx, repo, review, operations); err != nil {
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
	_, err := buildCandidateOperations(ctx, repo, review, plan, approval)
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
			rule, diagnostics, validateErr = rulepack.ValidateRetainedRule(ctx, repo, itemPath, artifact.content)
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

func captureJournal(
	repoRoot string,
	review Review,
	plan applicationPlan,
	operations []operation,
) (applicationJournal, error) {
	journal := applicationJournal{
		Schema:         journalSchema,
		ReviewID:       review.Context.ReviewID,
		ProposalDigest: review.ProposalDigest,
		PlanDigest:     plan.PlanDigest,
		Entries:        make([]journalEntry, 0, len(operations)),
	}
	createdDirectories, err := missingApplicationDirectories(repoRoot, operations)
	if err != nil {
		return applicationJournal{}, err
	}
	journal.CreatedDirectories = createdDirectories
	for _, operation := range operations {
		if err := safeApplicationTarget(repoRoot, operation.Path); err != nil {
			return applicationJournal{}, err
		}
		target, err := resolvePortablePath(repoRoot, operation.Path)
		if err != nil {
			return applicationJournal{}, err
		}
		if operation.ExpectedAbsent {
			if _, err := os.Lstat(target); err == nil {
				return applicationJournal{}, fmt.Errorf("new target %s appeared after preflight", operation.Path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return applicationJournal{}, fmt.Errorf("inspect expected-absent target %s: %w", operation.Path, err)
			}
			journal.Entries = append(journal.Entries, journalEntry{Path: operation.Path})
			continue
		}
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			return applicationJournal{}, fmt.Errorf(
				"application target %s disappeared after preflight",
				operation.Path,
			)
		}
		if err != nil {
			return applicationJournal{}, fmt.Errorf("inspect application target %s: %w", operation.Path, err)
		}
		if !info.Mode().IsRegular() {
			return applicationJournal{}, fmt.Errorf("application target %s is not a regular file", operation.Path)
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return applicationJournal{}, fmt.Errorf("capture application target %s: %w", operation.Path, err)
		}
		if digestBytes(content) != operation.Prestate.SHA256 ||
			materializedMode(info.Mode()) != operation.Prestate.Mode {
			return applicationJournal{}, fmt.Errorf(
				"application target %s changed after preflight",
				operation.Path,
			)
		}
		journal.Entries = append(journal.Entries, journalEntry{
			Path:    operation.Path,
			Existed: true,
			Mode:    uint32(info.Mode().Perm()),
			Content: content,
		})
	}
	return journal, nil
}

func executeOperations(repoRoot string, operations []operation) (map[string]struct{}, error) {
	completed := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if err := safeApplicationTarget(repoRoot, operation.Path); err != nil {
			return completed, err
		}
		target, err := resolvePortablePath(repoRoot, operation.Path)
		if err != nil {
			return completed, err
		}
		completed[operation.Path] = struct{}{}
		switch operation.Kind {
		case "remove":
			claim, err := claimExpectedFile(target, operation.ExpectedSHA256, prestateClaim)
			if err != nil {
				return completed, fmt.Errorf("claim %s before removal: %w", operation.Path, err)
			}
			if err := os.Remove(claim); err != nil {
				return completed, fmt.Errorf("remove claimed %s: %w", operation.Path, err)
			}
		case "write":
			var err error
			if operation.ExpectedAbsent {
				err = writeNewExclusive(target, operation.Content, operation.Mode)
			} else {
				var claim string
				claim, err = claimExpectedFile(target, operation.ExpectedSHA256, prestateClaim)
				if err == nil {
					err = writeNewExclusive(target, operation.Content, operation.Mode)
				}
				if err == nil {
					err = os.Remove(claim)
				} else if claim != "" {
					if restoreErr := restoreClaimedFile(claim, target); restoreErr != nil {
						err = fmt.Errorf("%w; preserve claimed source: %v", err, restoreErr)
					}
				}
			}
			if err != nil {
				return completed, fmt.Errorf("write %s: %w", operation.Path, err)
			}
		default:
			return completed, fmt.Errorf("unsupported application operation %s", operation.Kind)
		}
	}
	for _, operation := range operations {
		target, err := resolvePortablePath(repoRoot, operation.Path)
		if err != nil {
			return completed, err
		}
		if operation.Kind == "remove" {
			if _, err := os.Lstat(target); err == nil {
				return completed, fmt.Errorf("target %s reappeared during application", operation.Path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return completed, fmt.Errorf("verify removed target %s: %w", operation.Path, err)
			}
			continue
		}
		if err := requireCurrentDigest(target, operation.Poststate.SHA256); err != nil {
			return completed, fmt.Errorf("verify written target %s: %w", operation.Path, err)
		}
	}
	return completed, nil
}

func claimExpectedFile(target, expectedDigest, phase string) (string, error) {
	claim := claimPathForTarget(target, phase)
	if _, err := os.Lstat(claim); err == nil {
		return "", fmt.Errorf("claim path already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Rename(target, claim); err != nil {
		return "", err
	}
	if err := requireCurrentDigest(claim, expectedDigest); err != nil {
		if restoreErr := restoreClaimedFile(claim, target); restoreErr != nil {
			return claim, fmt.Errorf("%w; restore claimed file: %v", err, restoreErr)
		}
		return "", err
	}
	return claim, nil
}

func claimPathForTarget(target, phase string) string {
	return filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".ssb-prune-"+phase+"-claim")
}

func restoreClaimedFile(claim, target string) error {
	if err := os.Link(claim, target); err != nil {
		return err
	}
	return os.Remove(claim)
}

func requireCurrentDigest(filePath, expected string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if digestBytes(data) != expected {
		return fmt.Errorf("current bytes no longer match the approved source digest")
	}
	return nil
}

func restoreJournal(repoRoot string, journal applicationJournal, poststates map[string]string) error {
	paths := make(map[string]struct{}, len(journal.Entries))
	for _, entry := range journal.Entries {
		paths[entry.Path] = struct{}{}
	}
	return restoreJournalPaths(repoRoot, journal, paths, poststates)
}

func restoreJournalPaths(
	repoRoot string,
	journal applicationJournal,
	paths map[string]struct{},
	poststates map[string]string,
) error {
	for _, entry := range journal.Entries {
		if _, restore := paths[entry.Path]; !restore {
			continue
		}
		if err := safeApplicationTarget(repoRoot, entry.Path); err != nil {
			return err
		}
		target, err := resolvePortablePath(repoRoot, entry.Path)
		if err != nil {
			return err
		}
		expectedPoststate, hasPoststate := poststates[entry.Path]
		if !hasPoststate {
			return fmt.Errorf("missing approved poststate for %s", entry.Path)
		}
		if err := restoreJournalEntry(target, entry, expectedPoststate); err != nil {
			return err
		}
	}
	return removeApplicationDirectories(repoRoot, journal.CreatedDirectories)
}

func restoreJournalEntry(target string, entry journalEntry, expectedPoststate string) error {
	if err := reconcilePublicationClaim(target, entry, expectedPoststate); err != nil {
		return err
	}
	preClaim := claimPathForTarget(target, prestateClaim)
	postClaim := claimPathForTarget(target, poststateClaim)
	preExists, err := regularFileExists(preClaim)
	if err != nil {
		return err
	}
	postExists, err := regularFileExists(postClaim)
	if err != nil {
		return err
	}
	if preExists && postExists {
		if !entry.Existed {
			return fmt.Errorf("unexpected prestate claim for new target %s", entry.Path)
		}
		if digest, digestErr := fileDigest(preClaim); digestErr != nil || digest != digestBytes(entry.Content) {
			return fmt.Errorf("prestate claim for %s is invalid", entry.Path)
		}
		if digest, digestErr := fileDigest(postClaim); digestErr != nil || digest != expectedPoststate {
			return fmt.Errorf("poststate claim for %s is invalid", entry.Path)
		}
		targetDigest, absent, digestErr := currentFileDigest(target)
		if digestErr != nil {
			return digestErr
		}
		switch {
		case absent:
			if err := restoreClaimedFile(preClaim, target); err != nil {
				return err
			}
		case targetDigest == digestBytes(entry.Content):
			if err := os.Remove(preClaim); err != nil {
				return err
			}
		default:
			return fmt.Errorf("recovery target %s appeared while prestate and poststate were claimed", entry.Path)
		}
		return os.Remove(postClaim)
	}
	if postExists {
		if digest, digestErr := fileDigest(postClaim); digestErr != nil || digest != expectedPoststate {
			return fmt.Errorf("poststate claim for %s is invalid", entry.Path)
		}
		return finishPoststateClaim(target, postClaim, entry)
	}
	if preExists {
		if !entry.Existed {
			return fmt.Errorf("unexpected prestate claim for new target %s", entry.Path)
		}
		if digest, digestErr := fileDigest(preClaim); digestErr != nil || digest != digestBytes(entry.Content) {
			return fmt.Errorf("prestate claim for %s is invalid", entry.Path)
		}
		targetDigest, absent, err := currentFileDigest(target)
		if err != nil {
			return err
		}
		switch {
		case absent:
			return restoreClaimedFile(preClaim, target)
		case targetDigest == digestBytes(entry.Content):
			return os.Remove(preClaim)
		case targetDigest == expectedPoststate:
			if err := os.Rename(target, postClaim); err != nil {
				return err
			}
			if err := restoreClaimedFile(preClaim, target); err != nil {
				return err
			}
			return os.Remove(postClaim)
		default:
			return fmt.Errorf("recovery target %s contains neither approved prestate nor poststate bytes", entry.Path)
		}
	}

	currentDigest, absent, err := currentFileDigest(target)
	if err != nil {
		return err
	}
	prestateDigest := ""
	if entry.Existed {
		prestateDigest = digestBytes(entry.Content)
	}
	if (entry.Existed && !absent && currentDigest == prestateDigest) || (!entry.Existed && absent) {
		return nil
	}
	poststateMatches := (expectedPoststate == "" && absent) ||
		(expectedPoststate != "" && !absent && currentDigest == expectedPoststate)
	if !poststateMatches {
		return fmt.Errorf("recovery target %s contains neither approved prestate nor poststate bytes", entry.Path)
	}
	if absent {
		return writeNewExclusive(target, entry.Content, os.FileMode(entry.Mode))
	}
	claim, err := claimExpectedFile(target, expectedPoststate, poststateClaim)
	if err != nil {
		return err
	}
	return finishPoststateClaim(target, claim, entry)
}

func reconcilePublicationClaim(
	target string,
	entry journalEntry,
	expectedPoststate string,
) error {
	claim := claimPathForTarget(target, publicationClaim)
	exists, err := regularFileExists(claim)
	if err != nil || !exists {
		return err
	}
	claimDigest, err := fileDigest(claim)
	if err != nil {
		return err
	}
	prestateDigest := ""
	if entry.Existed {
		prestateDigest = digestBytes(entry.Content)
	}
	isPrestate := entry.Existed && claimDigest == prestateDigest
	isPoststate := expectedPoststate != "" && claimDigest == expectedPoststate
	targetDigest, absent, err := currentFileDigest(target)
	if err != nil {
		return err
	}
	if !isPrestate && !isPoststate {
		targetIsApproved := !absent &&
			((entry.Existed && targetDigest == prestateDigest) ||
				(expectedPoststate != "" && targetDigest == expectedPoststate))
		if absent || targetIsApproved {
			return os.Remove(claim)
		}
		return fmt.Errorf("publication claim for %s is incomplete while the recovery target has unapproved bytes", entry.Path)
	}
	switch {
	case absent && isPrestate:
		return restoreClaimedFile(claim, target)
	case absent:
		return os.Remove(claim)
	case targetDigest == claimDigest:
		return os.Remove(claim)
	default:
		return fmt.Errorf("recovery target %s changed while a publication claim was present", entry.Path)
	}
}

func regularFileExists(filePath string) (bool, error) {
	info, err := os.Lstat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("claim path %s is not a regular file", filePath)
	}
	return true, nil
}

func finishPoststateClaim(target, postClaim string, entry journalEntry) error {
	currentDigest, absent, err := currentFileDigest(target)
	if err != nil {
		return err
	}
	if !absent {
		if entry.Existed && currentDigest == digestBytes(entry.Content) {
			return os.Remove(postClaim)
		}
		return fmt.Errorf("recovery target %s appeared while its poststate was claimed", entry.Path)
	}
	if entry.Existed {
		if err := writeNewExclusive(target, entry.Content, os.FileMode(entry.Mode)); err != nil {
			return err
		}
	}
	return os.Remove(postClaim)
}

func currentFileDigest(target string) (string, bool, error) {
	digest, err := fileDigest(target)
	if errors.Is(err, os.ErrNotExist) {
		return "", true, nil
	}
	return digest, false, err
}

func missingApplicationDirectories(repoRoot string, operations []operation) ([]string, error) {
	missing := make(map[string]struct{})
	for _, operation := range operations {
		target, err := resolvePortablePath(repoRoot, operation.Path)
		if err != nil {
			return nil, err
		}
		parent := filepath.Dir(target)
		for parent != repoRoot {
			info, err := os.Lstat(parent)
			if err == nil {
				if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					return nil, fmt.Errorf("application parent %s is unsafe", parent)
				}
				break
			}
			if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			relative, err := filepath.Rel(repoRoot, parent)
			if err != nil {
				return nil, err
			}
			missing[filepath.ToSlash(relative)] = struct{}{}
			parent = filepath.Dir(parent)
		}
	}
	result := make([]string, 0, len(missing))
	for relative := range missing {
		result = append(result, relative)
	}
	sort.Strings(result)
	return result, nil
}

func removeApplicationDirectories(repoRoot string, directories []string) error {
	ordered := append([]string(nil), directories...)
	sort.Slice(ordered, func(i, j int) bool {
		return strings.Count(ordered[i], "/") > strings.Count(ordered[j], "/")
	})
	for _, relative := range ordered {
		target, err := resolvePortablePath(repoRoot, relative)
		if err != nil {
			return fmt.Errorf("unsafe application directory %s: %w", relative, err)
		}
		if err := os.Remove(target); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func fileDigest(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func operationPoststates(operations []operation) map[string]string {
	result := make(map[string]string, len(operations))
	for _, operation := range operations {
		if operation.Kind == "write" {
			result[operation.Path] = operation.Poststate.SHA256
		} else {
			result[operation.Path] = ""
		}
	}
	return result
}

func writeNewExclusive(
	target string,
	content []byte,
	mode os.FileMode,
) (returnErr error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	publication := claimPathForTarget(target, publicationClaim)
	file, err := openExclusiveFile(
		publication,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		mode,
	)
	if err != nil {
		return fmt.Errorf("create application publication claim: %w", err)
	}
	defer func() {
		if err := removeApplicationPublicationFile(publication); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("remove application publication claim %s: %w", publication, err),
			)
		}
	}()
	if err := writeDurableExclusiveWithCleanup(
		publication,
		file,
		content,
		mode,
		func() error { return removeIncompleteExclusiveFile(publication) },
	); err != nil {
		return err
	}
	if err := publishApplicationFile(publication, target); err != nil {
		return fmt.Errorf("publish application file without overwrite: %w", err)
	}
	return nil
}

// Recover restores an interrupted application journal, or finalizes journal
// cleanup when the application event was already recorded.
func Recover(
	ctx context.Context,
	repoPath, reviewID string,
	clearStaleLock bool,
) (returnErr error) {
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return err
	}
	if clearStaleLock {
		store, storeErr := openReviewStore(repo.Root(), reviewID)
		if storeErr != nil {
			return storeErr
		}
		if _, readErr := store.Lstat(".transition.lock"); readErr == nil {
			if removeErr := store.Remove(".transition.lock"); removeErr != nil &&
				!errors.Is(removeErr, os.ErrNotExist) {
				_ = store.Close()
				return fmt.Errorf("clear stale review transition lock: %w", removeErr)
			}
		} else if !errors.Is(readErr, os.ErrNotExist) {
			_ = store.Close()
			return fmt.Errorf("read stale review transition lock: %w", readErr)
		}
		if err := clearMutationLock(repo.Root(), reviewID); err != nil {
			_ = store.Close()
			return err
		}
		_, journalErr := store.Lstat("application-journal.json")
		closeErr := store.Close()
		if errors.Is(journalErr, os.ErrNotExist) {
			if closeErr != nil {
				return closeErr
			}
			return nil
		} else if journalErr != nil {
			return fmt.Errorf("inspect application recovery journal: %w", journalErr)
		} else if closeErr != nil {
			return closeErr
		}
	}
	lockedReviewStore, unlock, err := acquireReviewLock(repo.Root(), reviewID)
	if err != nil {
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, unlock())
	}()
	unlockMutation, err := acquireMutationLock(repo.Root(), reviewID)
	if err != nil {
		return err
	}
	defer func() {
		returnErr = errors.Join(returnErr, unlockMutation())
	}()
	review, diagnostics, err := loadReviewFromStore(lockedReviewStore)
	if err != nil {
		return err
	}
	if len(diagnostics) != 0 {
		return validationError("proposal has %d error(s)", len(diagnostics))
	}
	if repo.Baseline() != review.Context.BaselineCommit {
		return preconditionError("recovery baseline is stale: expected %s, found %s", review.Context.BaselineCommit, repo.Baseline())
	}
	if _, err := approvalFor(review); err != nil {
		return preconditionError("%v", err)
	}
	journalStore := lockedReviewStore
	data, _, err := journalStore.ReadRegular("application-journal.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return preconditionError("review has no application recovery journal")
		}
		return fmt.Errorf("read application journal: %w", err)
	}
	var journal applicationJournal
	if err := decodeStrictJSON(data, &journal); err != nil {
		return fmt.Errorf("parse application journal: %w", err)
	}
	if journal.Schema != journalSchema || journal.ReviewID != reviewID {
		return fmt.Errorf("application journal identity is invalid")
	}
	if !validDigest(journal.ProposalDigest) || !validDigest(journal.PlanDigest) {
		return fmt.Errorf("application journal proposal or plan digest is invalid")
	}
	if journal.ProposalDigest != review.ProposalDigest {
		return fmt.Errorf("application journal is bound to a different proposal digest")
	}
	seen := make(map[string]struct{}, len(journal.Entries))
	for _, entry := range journal.Entries {
		if !validApplicationPath(entry.Path) {
			return fmt.Errorf("application journal contains non-canonical target %s", entry.Path)
		}
		if _, duplicate := seen[entry.Path]; duplicate {
			return fmt.Errorf("application journal repeats target %s", entry.Path)
		}
		seen[entry.Path] = struct{}{}
	}
	if err := validateJournalPlan(ctx, repo, review, journal); err != nil {
		return validationError("%v", err)
	}
	for _, event := range review.Events {
		if event.Kind == EventApplied && event.ProposalDigest == journal.ProposalDigest {
			var applied ApplyResult
			if err := decodeStrictJSON(event.Payload, &applied); err != nil {
				return validationError("application event payload is invalid: %v", err)
			}
			if applied.PlanDigest != journal.PlanDigest {
				return validationError("application journal is bound to a different application plan")
			}
			return removeApplicationJournal(journalStore)
		}
	}
	restorePaths, poststates, err := recoverableJournalPaths(repo.Root(), review, journal)
	if err != nil {
		return validationError("%v", err)
	}
	if err := restoreJournalPaths(repo.Root(), journal, restorePaths, poststates); err != nil {
		return fmt.Errorf("restore application journal: %w", err)
	}
	return removeApplicationJournal(journalStore)
}

func acquireMutationLock(repoRoot, reviewID string) (func() error, error) {
	lockPath := filepath.Join(repoRoot, ".software-standards", ".prune-mutation.lock")
	root, packIdentity, err := openPruneRepositoryRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	pack, err := openPinnedPack(root, packIdentity)
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("pin .software-standards for mutation lock: %w", err)
	}
	file, err := pack.OpenFile(".prune-mutation.lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		err = writeDurableExclusiveWithCleanup(
			lockPath,
			file,
			[]byte(reviewID+"\n"),
			0o600,
			func() error { return pack.Remove(".prune-mutation.lock") },
		)
	}
	closePackErr := pack.Close()
	if err != nil {
		_ = root.Close()
		if errors.Is(err, os.ErrExist) {
			return nil, preconditionError("another prune application or recovery owns the repository mutation lock")
		}
		return nil, fmt.Errorf("create repository prune mutation lock: %w", err)
	}
	if closePackErr != nil {
		_ = root.Close()
		return nil, fmt.Errorf("close pinned standards pack after mutation lock creation: %w", closePackErr)
	}
	return func() error {
		removeErr := removePruneMutationLock(root, packIdentity)
		closeErr := root.Close()
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(
				fmt.Errorf("remove repository prune mutation lock %s: %w", lockPath, removeErr),
				closeErr,
			)
		}
		return closeErr
	}, nil
}

func clearMutationLock(repoRoot, reviewID string) error {
	root, packIdentity, err := openPruneRepositoryRoot(repoRoot)
	if err != nil {
		return err
	}
	defer root.Close()
	pack, err := openPinnedPack(root, packIdentity)
	if err != nil {
		return fmt.Errorf("pin .software-standards to clear mutation lock: %w", err)
	}
	defer pack.Close()
	data, err := pack.ReadFile(".prune-mutation.lock")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read repository prune mutation lock: %w", err)
	}
	if strings.TrimSpace(string(data)) != reviewID {
		return preconditionError("repository mutation lock belongs to another review and cannot be cleared")
	}
	if err := pack.Remove(".prune-mutation.lock"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear repository prune mutation lock: %w", err)
	}
	return nil
}

func recoverableJournalPaths(
	repoRoot string,
	review Review,
	journal applicationJournal,
) (map[string]struct{}, map[string]string, error) {
	approval, err := approvalFor(review)
	if err != nil {
		return nil, nil, err
	}
	plan, err := canonicalApplicationPlan(review, approval)
	if err != nil {
		return nil, nil, err
	}
	if journal.PlanDigest != plan.PlanDigest {
		return nil, nil, fmt.Errorf("application journal is bound to a different application plan")
	}
	poststate := planOperationPoststates(plan)
	restore := make(map[string]struct{})
	for _, entry := range journal.Entries {
		target, err := resolvePortablePath(repoRoot, entry.Path)
		if err != nil {
			return nil, nil, err
		}
		data, readErr := os.ReadFile(target)
		absent := errors.Is(readErr, os.ErrNotExist)
		if readErr != nil && !absent {
			return nil, nil, fmt.Errorf("inspect recovery state for %s: %w", entry.Path, readErr)
		}
		currentDigest := ""
		if !absent {
			currentDigest = digestBytes(data)
		}
		prestateMatches := entry.Existed && !absent && currentDigest == digestBytes(entry.Content)
		if !entry.Existed && absent {
			prestateMatches = true
		}
		expectedPoststate, exists := poststate[entry.Path]
		if !exists {
			return nil, nil, fmt.Errorf("journal target %s has no approved poststate", entry.Path)
		}
		poststateMatches := expectedPoststate == "" && absent
		if expectedPoststate != "" && !absent && currentDigest == expectedPoststate {
			poststateMatches = true
		}
		_, preClaimErr := os.Lstat(claimPathForTarget(target, prestateClaim))
		_, postClaimErr := os.Lstat(claimPathForTarget(target, poststateClaim))
		if preClaimErr == nil || postClaimErr == nil {
			restore[entry.Path] = struct{}{}
			continue
		} else if !errors.Is(preClaimErr, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("inspect recovery prestate claim for %s: %w", entry.Path, preClaimErr)
		} else if !errors.Is(postClaimErr, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("inspect recovery poststate claim for %s: %w", entry.Path, postClaimErr)
		}
		switch {
		case prestateMatches:
			continue
		case poststateMatches:
			restore[entry.Path] = struct{}{}
		default:
			return nil, nil, fmt.Errorf("recovery target %s contains neither approved prestate nor poststate bytes", entry.Path)
		}
	}
	return restore, poststate, nil
}

func validateJournalPlan(
	ctx context.Context,
	repo *workspace.Repository,
	review Review,
	journal applicationJournal,
) error {
	approval, err := approvalFor(review)
	if err != nil {
		return err
	}
	plan, err := canonicalApplicationPlan(review, approval)
	if err != nil {
		return err
	}
	if journal.PlanDigest != plan.PlanDigest {
		return fmt.Errorf("application journal is bound to a different application plan")
	}
	expected := make(map[string]Change, len(plan.Changes))
	for _, change := range plan.Changes {
		expected[change.Path] = change
	}
	if len(journal.Entries) != len(expected) {
		return fmt.Errorf("application journal does not match the approved operation count")
	}
	seenDirectories := make(map[string]struct{}, len(journal.CreatedDirectories))
	for _, directory := range journal.CreatedDirectories {
		if !safeRelativePath(directory) {
			return fmt.Errorf("application journal contains unsafe created directory %s", directory)
		}
		if _, duplicate := seenDirectories[directory]; duplicate {
			return fmt.Errorf("application journal repeats created directory %s", directory)
		}
		seenDirectories[directory] = struct{}{}
		ownsDirectory := false
		for itemPath := range expected {
			ownsDirectory = ownsDirectory || strings.HasPrefix(itemPath, directory+"/")
		}
		if !ownsDirectory {
			return fmt.Errorf("application journal directory %s is outside the approved operation plan", directory)
		}
	}
	for _, entry := range journal.Entries {
		change, ok := expected[entry.Path]
		if !ok {
			return fmt.Errorf("application journal target %s is not in the approved operation plan", entry.Path)
		}
		if !change.Prestate.Exists {
			if entry.Existed || len(entry.Content) != 0 || entry.Mode != 0 {
				return fmt.Errorf("application journal invents prestate for new target %s", entry.Path)
			}
			continue
		}
		if !entry.Existed || digestBytes(entry.Content) != change.Prestate.SHA256 {
			return fmt.Errorf("application journal prestate does not match %s", entry.Path)
		}
		baselineEntry, exists, err := repo.EntryAtBaseline(ctx, entry.Path)
		if err != nil || !exists {
			return fmt.Errorf("read baseline mode for %s", entry.Path)
		}
		if entry.Mode > 0o777 ||
			materializedMode(os.FileMode(entry.Mode)) != baselineEntry.Mode {
			return fmt.Errorf("application journal mode does not match %s", entry.Path)
		}
	}
	return nil
}

func planOperationPoststates(plan applicationPlan) map[string]string {
	result := make(map[string]string, len(plan.Changes))
	for _, change := range plan.Changes {
		if change.Poststate.Exists {
			result[change.Path] = change.Poststate.SHA256
		} else {
			result[change.Path] = ""
		}
	}
	return result
}

func safeApplicationTarget(repoRoot, relative string) error {
	if !validApplicationPath(relative) {
		return fmt.Errorf("application target %s is not a canonical actionable artifact or report path", relative)
	}
	target, err := resolvePortablePath(repoRoot, relative)
	if err != nil {
		return err
	}
	current := repoRoot
	parent := filepath.Dir(target)
	relativeParent, err := filepath.Rel(repoRoot, parent)
	if err != nil {
		return fmt.Errorf("resolve application target parent %s: %w", relative, err)
	}
	for _, component := range strings.Split(filepath.ToSlash(relativeParent), "/") {
		if component == "." {
			continue
		}
		current = filepath.Join(current, filepath.FromSlash(component))
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect application target %s: %w", relative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("application target %s has an unsafe parent component", relative)
		}
	}
	return nil
}

func validApplicationPath(relative string) bool {
	if relative == actionableReportPath || relative == actionableManifestPath {
		return true
	}
	if _, _, ok := artifactIdentity(relative); ok {
		return true
	}
	if !safeRelativePath(relative) || !strings.HasPrefix(relative, ".agents/skills/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(relative, ".agents/skills/"), "/")
	return len(parts) >= 2 && stableIDPattern.MatchString(parts[0])
}

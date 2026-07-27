package prune

import (
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

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

const journalSchema = "ssb.dev/prune-application-journal/v1"

var removeApplicationJournal = os.Remove

const (
	prestateClaim  = "pre"
	poststateClaim = "post"
)

// ApplyOptions selects a review and requires Write for mutation.
type ApplyOptions struct {
	ReviewID string
	Write    bool
	Now      func() time.Time
}

// Change is one complete-file operation in an application plan.
type Change struct {
	ActionID string `json:"action_id"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	SHA256   string `json:"sha256,omitempty"`
}

// ApplyResult reports the bounded plan or completed application.
type ApplyResult struct {
	DryRun  bool     `json:"dry_run"`
	Changes []Change `json:"changes"`
}

type operation struct {
	Change
	Content        []byte
	ExpectedAbsent bool
	ExpectedSHA256 string
	Mode           os.FileMode
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
	Entries            []journalEntry `json:"entries"`
	CreatedDirectories []string       `json:"created_directories,omitempty"`
}

// Apply computes a dry run by default and mutates only when Write is true.
func Apply(ctx context.Context, repoPath string, options ApplyOptions) (ApplyResult, error) {
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return ApplyResult{}, err
	}
	if options.Write {
		unlock, err := acquireReviewLock(repo.Root(), options.ReviewID)
		if err != nil {
			return ApplyResult{}, err
		}
		defer unlock()
		unlockMutation, err := acquireMutationLock(repo.Root(), options.ReviewID)
		if err != nil {
			return ApplyResult{}, err
		}
		defer unlockMutation()
	}
	review, diagnostics, err := LoadReview(repo.Root(), options.ReviewID)
	if err != nil {
		return ApplyResult{}, err
	}
	if len(diagnostics) != 0 {
		return ApplyResult{}, validationError("proposal has %d error(s); run ssb prune validate", len(diagnostics))
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
	operations, err := planOperations(ctx, repo, review, approval)
	if err != nil {
		return ApplyResult{}, validationError("%v", err)
	}
	result := ApplyResult{DryRun: !options.Write, Changes: make([]Change, len(operations))}
	for index, operation := range operations {
		result.Changes[index] = operation.Change
	}
	if !options.Write {
		return result, nil
	}
	journal, err := captureJournal(repo.Root(), review, operations)
	if err != nil {
		return ApplyResult{}, err
	}
	journalPath := filepath.Join(review.Root, "application-journal.json")
	journalData, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return ApplyResult{}, fmt.Errorf("encode application journal: %w", err)
	}
	if err := writeExclusive(journalPath, append(journalData, '\n'), 0o600); err != nil {
		return ApplyResult{}, fmt.Errorf("application recovery journal already exists or cannot be created: %w", err)
	}
	completed, executeErr := executeOperations(repo.Root(), operations)
	if executeErr != nil {
		rollbackErr := restoreJournalPaths(repo.Root(), journal, completed, operationPoststates(operations))
		if rollbackErr == nil {
			_ = os.Remove(journalPath)
			return ApplyResult{}, fmt.Errorf("apply review: %w; all changed files were restored", executeErr)
		}
		return ApplyResult{}, fmt.Errorf("apply review: %w; rollback also failed: %v; run ssb prune recover", executeErr, rollbackErr)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		rollbackErr := restoreJournal(repo.Root(), journal, operationPoststates(operations))
		if rollbackErr == nil {
			_ = os.Remove(journalPath)
			return ApplyResult{}, fmt.Errorf("encode application event: %w; all changed files were restored", err)
		}
		return ApplyResult{}, fmt.Errorf("encode application event: %w; rollback failed: %v; run ssb prune recover", err, rollbackErr)
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	event := newEvent(review, EventApplied, payload, now().UTC())
	if err := appendEvent(review, event); err != nil {
		rollbackErr := restoreJournal(repo.Root(), journal, operationPoststates(operations))
		if rollbackErr == nil {
			_ = os.Remove(journalPath)
			return ApplyResult{}, fmt.Errorf("record application event: %w; all changed files were restored", err)
		}
		return ApplyResult{}, fmt.Errorf("record application event: %w; rollback failed: %v; run ssb prune recover", err, rollbackErr)
	}
	if err := cleanupApplicationJournal(journalPath, options.ReviewID); err != nil {
		return ApplyResult{}, err
	}
	return result, nil
}

func cleanupApplicationJournal(journalPath, reviewID string) error {
	if err := removeApplicationJournal(journalPath); err != nil {
		return fmt.Errorf(
			"application succeeded but recovery journal cleanup failed: %w; run ssb prune recover --review %s",
			err,
			reviewID,
		)
	}
	return nil
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
) ([]operation, error) {
	repoRoot := repo.Root()
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	finalRules := make(map[string]struct{})
	for _, artifact := range review.Context.Artifacts {
		if artifact.Kind == ArtifactRule {
			finalRules[artifact.Path] = struct{}{}
		}
	}
	byPath := make(map[string]operation)
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok {
			continue
		}
		switch action.Disposition {
		case DispositionKeep:
			continue
		case DispositionRemove, DispositionUpdate, DispositionConsolidate:
			for _, source := range action.Sources {
				currentPath := filepath.Join(repoRoot, filepath.FromSlash(source.Path))
				content, err := os.ReadFile(currentPath)
				if err != nil {
					return nil, fmt.Errorf("read approved source %s: %w", source.Path, err)
				}
				if digestBytes(content) != source.SHA256 {
					return nil, fmt.Errorf("approved source %s no longer matches its context digest", source.Path)
				}
				byPath[source.Path] = operation{Change: Change{
					ActionID: action.ID,
					Path:     source.Path,
					Kind:     "remove",
				}, ExpectedSHA256: source.SHA256}
				if source.Kind == ArtifactRule {
					delete(finalRules, source.Path)
				}
				if actionRemovesSupportingFiles(action, source) {
					for _, supporting := range supportingFilesFor(review, source.Path) {
						content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(supporting.Path)))
						if err != nil {
							return nil, fmt.Errorf("read skill supporting file %s: %w", supporting.Path, err)
						}
						if digestBytes(content) != supporting.SHA256 {
							return nil, fmt.Errorf("skill supporting file %s no longer matches its context digest", supporting.Path)
						}
						byPath[supporting.Path] = operation{
							Change:         Change{ActionID: action.ID, Path: supporting.Path, Kind: "remove"},
							ExpectedSHA256: supporting.SHA256,
						}
					}
				}
			}
			if action.Target != nil {
				content, err := readCandidateFile(review, action.Target.SourcePath)
				if err != nil {
					return nil, fmt.Errorf("read candidate %s: %w", action.Target.SourcePath, err)
				}
				if digestBytes(content) != action.Target.SHA256 {
					return nil, fmt.Errorf("candidate %s no longer matches its proposal digest", action.Target.SourcePath)
				}
				if err := validateCandidate(ctx, repo, *action.Target, content); err != nil {
					return nil, err
				}
				ownedTarget := false
				expectedTargetDigest := ""
				for _, source := range action.Sources {
					ownedTarget = ownedTarget || source.Path == action.Target.TargetPath
					if source.Path == action.Target.TargetPath {
						expectedTargetDigest = source.SHA256
					}
					for _, supporting := range supportingFilesFor(review, source.Path) {
						if supporting.Path == action.Target.TargetPath {
							ownedTarget = true
							expectedTargetDigest = supporting.SHA256
						}
					}
				}
				if !ownedTarget {
					if _, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(action.Target.TargetPath))); err == nil {
						return nil, fmt.Errorf("candidate target %s would overwrite an unreviewed worktree file", action.Target.TargetPath)
					} else if !errors.Is(err, os.ErrNotExist) {
						return nil, fmt.Errorf("inspect candidate target %s: %w", action.Target.TargetPath, err)
					}
				}
				if existing, collision := byPath[action.Target.TargetPath]; collision &&
					existing.ActionID != action.ID {
					return nil, fmt.Errorf("actions %s and %s both change %s", existing.ActionID, action.ID, action.Target.TargetPath)
				}
				byPath[action.Target.TargetPath] = operation{
					Change: Change{
						ActionID: action.ID,
						Path:     action.Target.TargetPath,
						Kind:     "write",
						SHA256:   action.Target.SHA256,
					},
					Content:        content,
					ExpectedAbsent: !ownedTarget,
					ExpectedSHA256: expectedTargetDigest,
					Mode:           candidateMode(action.Target.Mode),
				}
				if action.Target.Kind == ArtifactRule {
					finalRules[action.Target.TargetPath] = struct{}{}
				}
				for _, supporting := range action.Target.SupportingFiles {
					supportingContent, err := readCandidateFile(review, supporting.SourcePath)
					if err != nil {
						return nil, fmt.Errorf("read candidate %s: %w", supporting.SourcePath, err)
					}
					if digestBytes(supportingContent) != supporting.SHA256 {
						return nil, fmt.Errorf("candidate %s no longer matches its proposal digest", supporting.SourcePath)
					}
					ownedSupporting := false
					expectedSupportingDigest := ""
					for _, source := range action.Sources {
						for _, existing := range supportingFilesFor(review, source.Path) {
							if existing.Path == supporting.TargetPath {
								ownedSupporting = true
								expectedSupportingDigest = existing.SHA256
							}
						}
					}
					if !ownedSupporting {
						if _, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(supporting.TargetPath))); err == nil {
							return nil, fmt.Errorf("candidate target %s would overwrite an unreviewed worktree file", supporting.TargetPath)
						} else if !errors.Is(err, os.ErrNotExist) {
							return nil, fmt.Errorf("inspect candidate target %s: %w", supporting.TargetPath, err)
						}
					}
					if existing, collision := byPath[supporting.TargetPath]; collision &&
						existing.ActionID != action.ID {
						return nil, fmt.Errorf("actions %s and %s both change %s", existing.ActionID, action.ID, supporting.TargetPath)
					}
					byPath[supporting.TargetPath] = operation{
						Change: Change{
							ActionID: action.ID,
							Path:     supporting.TargetPath,
							Kind:     "write",
							SHA256:   supporting.SHA256,
						},
						Content:        supportingContent,
						ExpectedAbsent: !ownedSupporting,
						ExpectedSHA256: expectedSupportingDigest,
						Mode:           candidateMode(supporting.Mode),
					}
				}
			}
		default:
			return nil, fmt.Errorf("approved action %s has non-applicable disposition %s", action.ID, action.Disposition)
		}
	}
	if len(finalRules) == 0 {
		return nil, fmt.Errorf("application would remove every rule; retain or replace at least one rule")
	}
	if err := validateResultingGraph(ctx, repo, review, byPath); err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(byPath))
	for itemPath := range byPath {
		paths = append(paths, itemPath)
	}
	sort.Strings(paths)
	result := make([]operation, 0, len(paths))
	for _, itemPath := range paths {
		result = append(result, byPath[itemPath])
	}
	return result, nil
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
	target := filepath.Join(review.Root, filepath.FromSlash(sourcePath))
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	limit := review.Context.Inventory.Limits.MaxFileBytes
	if limit <= 0 || info.Size() > limit {
		return nil, fmt.Errorf("candidate exceeds max_file_bytes=%d", limit)
	}
	return os.ReadFile(target)
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
) error {
	switch target.Kind {
	case ArtifactRule:
		_, diagnostics := rulepack.ValidateCandidateRule(ctx, repo, target.TargetPath, content)
		if len(diagnostics) != 0 {
			return fmt.Errorf("candidate %s violates the rule contract: %s", target.SourcePath, diagnostics[0].Message)
		}
	case ArtifactSkill:
		_, diagnostics := rulepack.ValidateCandidateSkill(target.TargetPath, target.ID, content)
		if len(diagnostics) != 0 {
			return fmt.Errorf("candidate %s violates the Agent Skill contract: %s", target.SourcePath, diagnostics[0].Message)
		}
	default:
		return fmt.Errorf("candidate %s has unsupported kind %s", target.SourcePath, target.Kind)
	}
	return nil
}

func validateResultingGraph(
	ctx context.Context,
	repo *workspace.Repository,
	review Review,
	operations map[string]operation,
) error {
	repoRoot := repo.Root()
	type finalArtifact struct {
		kind    string
		content []byte
	}
	final := make(map[string]finalArtifact, len(review.Context.Artifacts))
	for _, artifact := range review.Context.Artifacts {
		content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(artifact.Path)))
		if err != nil {
			return fmt.Errorf("read resulting-graph source %s: %w", artifact.Path, err)
		}
		final[artifact.Path] = finalArtifact{kind: artifact.Kind, content: content}
	}
	for itemPath, operation := range operations {
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
	skills := make(map[string]struct{})
	for itemPath, artifact := range final {
		if artifact.kind != ArtifactSkill {
			continue
		}
		_, id, _ := artifactIdentity(itemPath)
		if _, diagnostics := rulepack.ValidateCandidateSkill(itemPath, id, artifact.content); len(diagnostics) != 0 {
			return fmt.Errorf("resulting skill %s violates the Agent Skill contract: %s", itemPath, diagnostics[0].Message)
		}
		skills[id] = struct{}{}
	}
	for itemPath, artifact := range final {
		if artifact.kind != ArtifactRule {
			continue
		}
		rule, diagnostics := rulepack.ValidateRetainedRule(ctx, repo, itemPath, artifact.content)
		if len(diagnostics) != 0 {
			return fmt.Errorf("resulting rule %s violates the rule contract: %s", itemPath, diagnostics[0].Message)
		}
		_, pathID, _ := artifactIdentity(itemPath)
		if rule.ID != pathID {
			return fmt.Errorf("resulting rule %s declares id %s", itemPath, rule.ID)
		}
		for _, skillID := range rule.RelatedSkillIDs {
			if _, exists := skills[skillID]; !exists {
				return fmt.Errorf("resulting rule %s references missing skill %s", itemPath, skillID)
			}
		}
	}
	return nil
}

func captureJournal(repoRoot string, review Review, operations []operation) (applicationJournal, error) {
	journal := applicationJournal{
		Schema:         journalSchema,
		ReviewID:       review.Context.ReviewID,
		ProposalDigest: review.ProposalDigest,
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
		target := filepath.Join(repoRoot, filepath.FromSlash(operation.Path))
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
			journal.Entries = append(journal.Entries, journalEntry{Path: operation.Path})
			continue
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
		target := filepath.Join(repoRoot, filepath.FromSlash(operation.Path))
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
		target := filepath.Join(repoRoot, filepath.FromSlash(operation.Path))
		if operation.Kind == "remove" {
			if _, err := os.Lstat(target); err == nil {
				return completed, fmt.Errorf("target %s reappeared during application", operation.Path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return completed, fmt.Errorf("verify removed target %s: %w", operation.Path, err)
			}
			continue
		}
		if err := requireCurrentDigest(target, operation.SHA256); err != nil {
			return completed, fmt.Errorf("verify written target %s: %w", operation.Path, err)
		}
	}
	return completed, nil
}

func claimExpectedFile(target, expectedDigest, phase string) (string, error) {
	claim := claimPathForTarget(target, phase)
	if _, err := os.Lstat(claim); err == nil {
		return "", fmt.Errorf("claim path already exists; run ssb prune recover")
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
		target := filepath.Join(repoRoot, filepath.FromSlash(entry.Path))
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
		parent := filepath.Dir(filepath.Join(repoRoot, filepath.FromSlash(operation.Path)))
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
		if !safeRelativePath(relative) {
			return fmt.Errorf("unsafe application directory %s", relative)
		}
		if err := os.Remove(filepath.Join(repoRoot, filepath.FromSlash(relative))); err != nil &&
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
			result[operation.Path] = operation.SHA256
		} else {
			result[operation.Path] = ""
		}
	}
	return result
}

func writeNewExclusive(target string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(target)
		}
	}()
	written, err := file.Write(content)
	if err != nil {
		return err
	}
	if written != len(content) {
		return io.ErrShortWrite
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}

func atomicWrite(target string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".ssb-prune-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if written, err := temp.Write(content); err != nil {
		temp.Close()
		return err
	} else if written != len(content) {
		temp.Close()
		return fmt.Errorf("write temporary application file: %w", io.ErrShortWrite)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, target)
}

// Recover restores an interrupted application journal, or finalizes journal
// cleanup when the application event was already recorded.
func Recover(ctx context.Context, repoPath, reviewID string, clearStaleLock bool) error {
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return err
	}
	if clearStaleLock {
		root, rootErr := reviewRoot(repo.Root(), reviewID)
		if rootErr != nil {
			return rootErr
		}
		if _, journalErr := os.Stat(filepath.Join(root, "application-journal.json")); journalErr != nil {
			return preconditionError("cannot clear a transition lock without an application recovery journal")
		}
		if removeErr := os.Remove(filepath.Join(root, ".transition.lock")); removeErr != nil &&
			!errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("clear stale review transition lock: %w", removeErr)
		}
		if err := clearMutationLock(repo.Root(), reviewID); err != nil {
			return err
		}
	}
	unlock, err := acquireReviewLock(repo.Root(), reviewID)
	if err != nil {
		return err
	}
	defer unlock()
	unlockMutation, err := acquireMutationLock(repo.Root(), reviewID)
	if err != nil {
		return err
	}
	defer unlockMutation()
	review, diagnostics, err := LoadReview(repo.Root(), reviewID)
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
	root, err := reviewRoot(repo.Root(), reviewID)
	if err != nil {
		return err
	}
	journalPath := filepath.Join(root, "application-journal.json")
	data, err := os.ReadFile(journalPath)
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
	if !validDigest(journal.ProposalDigest) {
		return fmt.Errorf("application journal proposal digest is invalid")
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
			return os.Remove(journalPath)
		}
	}
	restorePaths, poststates, err := recoverableJournalPaths(repo.Root(), review, journal)
	if err != nil {
		return validationError("%v", err)
	}
	if err := restoreJournalPaths(repo.Root(), journal, restorePaths, poststates); err != nil {
		return fmt.Errorf("restore application journal: %w", err)
	}
	return os.Remove(journalPath)
}

func acquireMutationLock(repoRoot, reviewID string) (func(), error) {
	lockPath := filepath.Join(repoRoot, ".software-standards", ".prune-mutation.lock")
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, preconditionError("another prune application or recovery owns the repository mutation lock")
		}
		return nil, fmt.Errorf("create repository prune mutation lock: %w", err)
	}
	if _, err := file.WriteString(reviewID + "\n"); err != nil {
		_ = file.Close()
		_ = os.Remove(lockPath)
		return nil, fmt.Errorf("write repository prune mutation lock: %w", err)
	}
	return func() {
		_ = file.Close()
		_ = os.Remove(lockPath)
	}, nil
}

func clearMutationLock(repoRoot, reviewID string) error {
	lockPath := filepath.Join(repoRoot, ".software-standards", ".prune-mutation.lock")
	data, err := os.ReadFile(lockPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read repository prune mutation lock: %w", err)
	}
	if strings.TrimSpace(string(data)) != reviewID {
		return preconditionError("repository mutation lock belongs to another review and cannot be cleared")
	}
	if err := os.Remove(lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
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
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	poststate := make(map[string]string)
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok || action.Disposition == DispositionKeep {
			continue
		}
		for _, source := range action.Sources {
			poststate[source.Path] = ""
			if actionRemovesSupportingFiles(action, source) {
				for _, supporting := range supportingFilesFor(review, source.Path) {
					poststate[supporting.Path] = ""
				}
			}
		}
		if action.Target != nil {
			poststate[action.Target.TargetPath] = action.Target.SHA256
			for _, supporting := range action.Target.SupportingFiles {
				poststate[supporting.TargetPath] = supporting.SHA256
			}
		}
	}
	restore := make(map[string]struct{})
	for _, entry := range journal.Entries {
		target := filepath.Join(repoRoot, filepath.FromSlash(entry.Path))
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
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	expected := make(map[string]struct{})
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok || action.Disposition == DispositionKeep {
			continue
		}
		for _, source := range action.Sources {
			expected[source.Path] = struct{}{}
			if actionRemovesSupportingFiles(action, source) {
				for _, supporting := range supportingFilesFor(review, source.Path) {
					expected[supporting.Path] = struct{}{}
				}
			}
		}
		if action.Target != nil {
			expected[action.Target.TargetPath] = struct{}{}
			for _, supporting := range action.Target.SupportingFiles {
				expected[supporting.TargetPath] = struct{}{}
			}
		}
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
	baselineDigests := make(map[string]string, len(review.Context.Artifacts))
	for _, artifact := range review.Context.Artifacts {
		baselineDigests[artifact.Path] = artifact.SHA256
		for _, supporting := range artifact.SupportingFiles {
			baselineDigests[supporting.Path] = supporting.SHA256
		}
	}
	for _, entry := range journal.Entries {
		if _, ok := expected[entry.Path]; !ok {
			return fmt.Errorf("application journal target %s is not in the approved operation plan", entry.Path)
		}
		baselineDigest, existedAtBaseline := baselineDigests[entry.Path]
		if !existedAtBaseline {
			if entry.Existed || len(entry.Content) != 0 || entry.Mode != 0 {
				return fmt.Errorf("application journal invents prestate for new target %s", entry.Path)
			}
			continue
		}
		if !entry.Existed || digestBytes(entry.Content) != baselineDigest {
			return fmt.Errorf("application journal prestate does not match %s", entry.Path)
		}
		baselineEntry, exists, err := repo.EntryAtBaseline(ctx, entry.Path)
		if err != nil || !exists {
			return fmt.Errorf("read baseline mode for %s", entry.Path)
		}
		expectedMode := uint32(0o644)
		if baselineEntry.Mode == "100755" {
			expectedMode = 0o755
		}
		if entry.Mode != expectedMode {
			return fmt.Errorf("application journal mode does not match %s", entry.Path)
		}
	}
	return nil
}

func safeApplicationTarget(repoRoot, relative string) error {
	if !validApplicationPath(relative) {
		return fmt.Errorf("application target %s is not a canonical rule or skill path", relative)
	}
	current := repoRoot
	components := strings.Split(relative, "/")
	for _, component := range components[:len(components)-1] {
		current = filepath.Join(current, component)
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
	if _, _, ok := artifactIdentity(relative); ok {
		return true
	}
	if !safeRelativePath(relative) || !strings.HasPrefix(relative, ".agents/skills/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(relative, ".agents/skills/"), "/")
	return len(parts) >= 2 && stableIDPattern.MatchString(parts[0])
}

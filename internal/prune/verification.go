package prune

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

// ReceiptEvidence is one content-addressed external check artifact.
type ReceiptEvidence struct {
	Path   string `yaml:"path" json:"path"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// CheckReceipt records externally executed proof. SSB verifies the receipt
// and evidence bytes but never executes its command.
type CheckReceipt struct {
	Schema                 string            `yaml:"schema" json:"schema"`
	ReviewID               string            `yaml:"review_id" json:"review_id"`
	ProposalDigest         string            `yaml:"proposal_digest" json:"proposal_digest"`
	ApplicationEventDigest string            `yaml:"application_event_digest" json:"application_event_digest"`
	PlanDigest             string            `yaml:"plan_digest" json:"plan_digest"`
	RenderEventDigest      string            `yaml:"render_event_digest,omitempty" json:"render_event_digest,omitempty"`
	CheckID                string            `yaml:"check_id" json:"check_id"`
	Command                string            `yaml:"command" json:"command"`
	Status                 string            `yaml:"status" json:"status"`
	ObservedAt             Timestamp         `yaml:"observed_at" json:"observed_at"`
	Evidence               []ReceiptEvidence `yaml:"evidence" json:"evidence"`
}

// VerificationResult identifies the exact receipts accepted for a review.
type VerificationResult struct {
	ApplicationEventDigest string            `json:"application_event_digest"`
	PlanDigest             string            `json:"plan_digest"`
	RenderEventDigest      string            `json:"render_event_digest,omitempty"`
	Receipts               map[string]string `json:"receipts"`
}

// VerificationOptions identifies the review, receipt directory, and optional clock.
type VerificationOptions struct {
	ReviewID    string
	ReceiptsDir string
	Now         func() time.Time
}

// Verify accepts exact passing receipts and records a separate verification event.
func Verify(
	ctx context.Context,
	repoPath string,
	options VerificationOptions,
) (result VerificationResult, returnErr error) {
	reviewID, receiptsDir, now := options.ReviewID, options.ReceiptsDir, options.Now
	repo, err := workspace.Open(ctx, repoPath)
	if err != nil {
		return VerificationResult{}, err
	}
	store, unlock, err := acquireReviewLock(ctx, repo.Root(), reviewID)
	if err != nil {
		return VerificationResult{}, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, unlock())
	}()
	review, diagnostics, err := loadReviewFromStore(ctx, store)
	if err != nil {
		return VerificationResult{}, err
	}
	if len(diagnostics) != 0 {
		return VerificationResult{}, validationError("proposal has %d error(s)", len(diagnostics))
	}
	status, _, err := deriveReviewStatus(ctx, review, diagnostics)
	if err != nil {
		return VerificationResult{}, err
	}
	if status.NoChangesApproved {
		return VerificationResult{}, preconditionError("verification is not applicable because no changes were approved")
	}
	if !status.Applied {
		return VerificationResult{}, preconditionError("verification requires a completed application event")
	}
	if changesRules(review) && !status.Rendered {
		return VerificationResult{}, preconditionError("rule changes require a separate rerender event before verification")
	}
	if status.Verified {
		return VerificationResult{}, preconditionError("review already has a verification event")
	}
	approval, err := approvalFor(review)
	if err != nil {
		return VerificationResult{}, err
	}
	plan, err := canonicalApplicationPlan(ctx, review, approval)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return VerificationResult{}, err
		}
		return VerificationResult{}, validationError("derive application plan: %v", err)
	}
	applicationEvent, applied, err := appliedEventFor(review)
	if err != nil {
		return VerificationResult{}, validationError("%v", err)
	}
	if applied.PlanDigest != plan.PlanDigest {
		return VerificationResult{}, validationError("application event is bound to a different plan")
	}
	if err := verifyPlanPoststate(repo.Root(), plan); err != nil {
		return VerificationResult{}, preconditionError("application poststate drift: %v", err)
	}
	renderEvent := Event{}
	if changesRules(review) || status.Rendered {
		renderEvent, err = recordedEvent(review, EventRendered)
		if err != nil {
			return VerificationResult{}, preconditionError("rule changes require a separate rerender event before verification")
		}
		if err := verifyRenderedPoststate(repo.Root(), renderEvent); err != nil {
			return VerificationResult{}, preconditionError("rendered poststate drift: %v", err)
		}
	}
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	required := make(map[string]CheckRequirement)
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok {
			continue
		}
		for _, check := range action.RequiredVerification {
			if !stableIDPattern.MatchString(check.ID) || check.Command == "" {
				return VerificationResult{}, validationError("action %s has invalid verification requirement", action.ID)
			}
			if prior, duplicate := required[check.ID]; duplicate && prior.Command != check.Command {
				return VerificationResult{}, validationError("check id %s has conflicting commands", check.ID)
			}
			required[check.ID] = check
		}
	}
	if len(required) == 0 {
		return VerificationResult{}, validationError("review has no required external checks and cannot enter verified state")
	}
	result = VerificationResult{
		ApplicationEventDigest: applicationEvent.EventDigest,
		PlanDigest:             plan.PlanDigest,
		RenderEventDigest:      renderEvent.EventDigest,
		Receipts:               make(map[string]string, len(required)),
	}
	ids := make([]string, 0, len(required))
	for id := range required {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		filePath := filepath.Join(receiptsDir, id+".yaml")
		data, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return VerificationResult{}, preconditionError("required receipt %s is missing from %s", id, receiptsDir)
			}
			return VerificationResult{}, fmt.Errorf("read receipt %s: %w", id, err)
		}
		var receipt CheckReceipt
		if err := yaml.Load(
			data,
			&receipt,
			yaml.WithKnownFields(),
			yaml.WithUniqueKeys(),
		); err != nil {
			return VerificationResult{}, validationError("parse receipt %s: %v", id, err)
		}
		if receipt.Schema != CheckSchema || receipt.ReviewID != reviewID ||
			receipt.ProposalDigest != review.ProposalDigest ||
			receipt.ApplicationEventDigest != applicationEvent.EventDigest ||
			receipt.PlanDigest != plan.PlanDigest ||
			receipt.RenderEventDigest != renderEvent.EventDigest ||
			receipt.CheckID != id ||
			receipt.Command != required[id].Command || receipt.Status != "passed" {
			return VerificationResult{}, validationError("receipt %s does not match the required passing check", id)
		}
		observedAt, err := time.Parse(time.RFC3339, string(receipt.ObservedAt))
		if err != nil {
			return VerificationResult{}, validationError("receipt %s has invalid observed_at", id)
		}
		appliedAt, _ := time.Parse(time.RFC3339, applicationEvent.RecordedAt)
		if observedAt.Before(appliedAt) {
			return VerificationResult{}, validationError("receipt %s predates the application event", id)
		}
		if len(receipt.Evidence) == 0 {
			return VerificationResult{}, validationError("receipt %s requires content-addressed evidence", id)
		}
		for _, evidence := range receipt.Evidence {
			if !safeRelativePath(evidence.Path) || !validDigest(evidence.SHA256) {
				return VerificationResult{}, validationError("receipt %s has invalid evidence reference", id)
			}
			evidencePath, err := resolvePortablePath(receiptsDir, evidence.Path)
			if err != nil {
				return VerificationResult{}, validationError(
					"receipt %s evidence %s is unsafe: %v",
					id,
					evidence.Path,
					err,
				)
			}
			if err := requireRegularBundleFile(receiptsDir, evidencePath); err != nil {
				return VerificationResult{}, validationError(
					"receipt %s evidence %s is unsafe: %v",
					id,
					evidence.Path,
					err,
				)
			}
			content, err := os.ReadFile(evidencePath)
			if err != nil || digestBytes(content) != evidence.SHA256 {
				return VerificationResult{}, validationError("receipt %s evidence %s is missing or has changed", id, evidence.Path)
			}
		}
		result.Receipts[id] = digestBytes(data)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return VerificationResult{}, err
	}
	if now == nil {
		now = time.Now
	}
	event := newEvent(
		review,
		EventVerified,
		data,
		now().UTC(),
	)
	if err := appendEvent(ctx, review, event); err != nil {
		return VerificationResult{}, err
	}
	return result, nil
}

func appliedEventFor(review Review) (Event, ApplyResult, error) {
	event, err := recordedEvent(review, EventApplied)
	if err != nil {
		return Event{}, ApplyResult{}, err
	}
	var result ApplyResult
	if err := decodeStrictJSON(event.Payload, &result); err != nil {
		return Event{}, ApplyResult{}, fmt.Errorf("application event payload is invalid: %w", err)
	}
	return event, result, nil
}

func recordedEvent(review Review, kind string) (Event, error) {
	for _, event := range review.Events {
		if event.Kind == kind {
			return event, nil
		}
	}
	return Event{}, fmt.Errorf("review has no %s event", kind)
}

func verifyPlanPoststate(repoRoot string, plan applicationPlan) error {
	expected := make(map[string]plannedFile, len(plan.Poststate))
	for _, file := range plan.Poststate {
		expected[file.Path] = file
		target, err := resolvePortablePath(repoRoot, file.Path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(target)
		if err != nil {
			return fmt.Errorf("%s is missing: %w", file.Path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", file.Path)
		}
		if err := requireCurrentDigest(target, file.SHA256); err != nil {
			return fmt.Errorf("%s digest changed: %w", file.Path, err)
		}
		if runtime.GOOS != "windows" && materializedMode(info.Mode()) != file.Mode {
			return fmt.Errorf(
				"%s mode is %s, expected %s",
				file.Path,
				materializedMode(info.Mode()),
				file.Mode,
			)
		}
	}
	for _, change := range plan.Changes {
		if change.Poststate.Exists {
			continue
		}
		target, err := resolvePortablePath(repoRoot, change.Path)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(target); err == nil {
			return fmt.Errorf("removed path %s reappeared", change.Path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect removed path %s: %w", change.Path, err)
		}
	}
	current, err := currentGovernedFiles(repoRoot)
	if err != nil {
		return err
	}
	expectedGoverned := make(map[string]plannedFile, len(expected))
	for itemPath, file := range expected {
		if itemPath == actionableReportPath || itemPath == actionableManifestPath {
			continue
		}
		expectedGoverned[itemPath] = file
	}
	for itemPath := range current {
		if _, exists := expectedGoverned[itemPath]; !exists {
			return fmt.Errorf("unapproved governed path %s appeared", itemPath)
		}
	}
	if len(current) != len(expectedGoverned) {
		return fmt.Errorf("governed path count is %d, expected %d", len(current), len(expectedGoverned))
	}
	return nil
}

func currentGovernedFiles(repoRoot string) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	for _, relativeRoot := range []string{".software-standards/rules", ".agents/skills"} {
		root, err := resolvePortablePath(repoRoot, relativeRoot)
		if err != nil {
			return nil, err
		}
		if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		err = filepath.WalkDir(root, func(itemPath string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if itemPath == root {
				return nil
			}
			relative, err := filepath.Rel(repoRoot, itemPath)
			if err != nil {
				return err
			}
			portable := filepath.ToSlash(relative)
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("governed path %s is a symlink", portable)
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("governed path %s is not a regular file", portable)
			}
			result[portable] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func materializedMode(mode os.FileMode) string {
	if mode.Perm()&0o111 != 0 {
		return "100755"
	}
	return "100644"
}

func verifyRenderedPoststate(repoRoot string, event Event) error {
	var payload renderedEventPayload
	if err := decodeStrictJSON(event.Payload, &payload); err != nil {
		return err
	}
	target, err := resolvePortablePath(repoRoot, payload.Path)
	if err != nil {
		return err
	}
	if !payload.Exists {
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			if err == nil {
				return fmt.Errorf("%s exists but the recorded render left no file", payload.Path)
			}
			return fmt.Errorf("inspect %s: %w", payload.Path, err)
		}
		if payload.OutputDigest != digestBytes(nil) {
			return fmt.Errorf("%s recorded an invalid absent-file render digest", payload.Path)
		}
	} else {
		if err := requireRegularBundleFile(repoRoot, target); err != nil {
			return fmt.Errorf("%s is not a safe regular repository file: %w", payload.Path, err)
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		if digestBytes(content) != payload.OutputDigest {
			return fmt.Errorf("%s no longer matches the recorded render digest", payload.Path)
		}
	}
	return verifyRenderedRoutingPoststate(repoRoot, payload.Routing)
}

func verifyRenderedRoutingPoststate(repoRoot string, payload *renderedRoutingPayload) error {
	if payload == nil {
		return nil
	}
	target, err := resolvePortablePath(repoRoot, payload.Path)
	if err != nil {
		return err
	}
	if !payload.Exists {
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			if err == nil {
				return fmt.Errorf("routing tree exists but the recorded render removed it")
			}
			return fmt.Errorf("inspect routing tree: %w", err)
		}
		emptyDigest, err := canonicalDigest(make([]routingFileIdentity, 0))
		if err != nil {
			return err
		}
		if payload.TreeDigest != emptyDigest {
			return fmt.Errorf("routing tree recorded an invalid absent-tree digest")
		}
		return nil
	}
	info, err := os.Lstat(target)
	if err != nil {
		return fmt.Errorf("inspect routing tree: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("routing tree is not a safe regular directory")
	}
	identities := make([]routingFileIdentity, 0)
	hasCatalog := false
	err = filepath.WalkDir(target, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == target {
			return nil
		}
		relative, err := filepath.Rel(target, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("routing tree path %s is a symlink", relative)
		}
		if entry.IsDir() {
			if relative != "bundles" {
				return fmt.Errorf("routing tree contains unexpected directory %s", relative)
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() ||
			relative != "catalog.md" && !(strings.HasPrefix(relative, "bundles/") && strings.HasSuffix(relative, ".md")) {
			return fmt.Errorf("routing tree contains unsafe file %s", relative)
		}
		if relative == "catalog.md" {
			hasCatalog = true
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		identities = append(identities, routingFileIdentity{
			Path: payload.Path + "/" + relative, SHA256: digestBytes(content),
		})
		return nil
	})
	if err != nil {
		return err
	}
	if !hasCatalog {
		return fmt.Errorf("routing tree catalog is missing")
	}
	sort.Slice(identities, func(i, j int) bool { return identities[i].Path < identities[j].Path })
	actual, err := canonicalDigest(identities)
	if err != nil {
		return err
	}
	if actual != payload.TreeDigest {
		return fmt.Errorf("routing tree no longer matches the recorded digest")
	}
	return nil
}

type routingFileIdentity struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func changesRules(review Review) bool {
	approval, err := approvalFor(review)
	if err != nil {
		return false
	}
	approved := make(map[string]struct{}, len(approval.Approved))
	for _, id := range approval.Approved {
		approved[id] = struct{}{}
	}
	for _, action := range review.Proposal.Actions {
		if _, ok := approved[action.ID]; !ok || action.Disposition == DispositionKeep {
			continue
		}
		for _, source := range action.Sources {
			if source.Kind == ArtifactRule {
				return true
			}
		}
		if action.Target != nil && action.Target.Kind == ArtifactRule {
			return true
		}
	}
	return false
}

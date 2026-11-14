package rulepack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

// Continuation is a read-only, point-in-time assessment. File identities identify
// retained bytes, not approval or permission to overwrite them.
type Continuation struct {
	SchemaVersion     int             `json:"schema_version"`
	State             string          `json:"state"`
	BaselineCommit    string          `json:"baseline_commit"`
	InventoryReusable bool            `json:"inventory_reusable"`
	ProposalReusable  bool            `json:"proposal_reusable"`
	Files             []FileReference `json:"retained_files"`
	Repeat            []string        `json:"repeat"`
	Guidance          []string        `json:"guidance"`
	Diagnostics       []Diagnostic    `json:"diagnostics"`
}

// inventoryVerification is created only after the complete pinned inventory
// check. It avoids a second Git scan within this assessment, while validation
// still checks raw identity, metadata, and manifest binding. Public validation
// never receives this proof, and the assessment rechecks retained bytes.
type inventoryVerification struct {
	baselineCommit string
	rawSHA256      string
}

// AssessContinuation verifies retained generation output without creating,
// repairing, or adopting anything. The final manifest is the existing completion
// boundary for semantic generation; incomplete sources remain review material.
func AssessContinuation(ctx context.Context, repo *workspace.Repository) (result Continuation, pack Pack, returnErr error) {
	result = Continuation{
		SchemaVersion: 1, State: "blocked", BaselineCommit: repo.Baseline(),
		Files: []FileReference{}, Repeat: []string{"inspection", "semantic_generation", "validation", "rendering", "developer_review"},
		Guidance: []string{}, Diagnostics: []Diagnostic{},
	}
	const manifestPath = ".software-standards/manifest.yaml"
	const inventoryPath = ".software-standards/inventory.json"
	for _, relative := range []string{manifestPath, ".software-standards/report.md"} {
		_, exists, err := repo.EntryAtBaseline(ctx, relative)
		if err != nil {
			return result, pack, err
		}
		if exists {
			result.State = "existing_pack"
			result.Repeat = []string{}
			result.Guidance = []string{"A pack is already recorded in HEAD. Use existing-pack consumption or explicitly requested maintenance; generation continuation can't establish adoption."}
			return result, pack, nil
		}
	}
	if err := continuationBaseline(ctx, repo); err != nil {
		return result, pack, err
	}
	// Capture before validation and recheck afterward. These observations aren't a
	// write lease: validate and render must still enforce their own contracts.
	manifestMissing := false
	defer func() {
		if returnErr != nil {
			return
		}
		if err := continuationBaseline(ctx, repo); err != nil {
			returnErr = err
			return
		}
		for _, reference := range result.Files {
			_, diagnostics, err := readDigestBoundFile(repo.Root(), reference, "retained file", inventory.DefaultLimits().MaxCandidateBytes)
			if err != nil {
				returnErr = err
				return
			}
			if len(diagnostics) != 0 {
				returnErr = fmt.Errorf("%w: retained files changed during continuation assessment; preserve edits and rerun ssb resume", workspace.ErrPrecondition)
				return
			}
		}
		if manifestMissing {
			if _, err := os.Lstat(filepath.Join(repo.Root(), manifestPath)); !errors.Is(err, os.ErrNotExist) {
				returnErr = fmt.Errorf("%w: manifest changed during continuation assessment; rerun ssb resume", workspace.ErrPrecondition)
			}
		}
	}()
	inventoryBytes, diagnostics, err := continuationFile(repo.Root(), inventoryPath, inventory.DefaultLimits().MaxCandidateBytes)
	if err != nil {
		return result, pack, err
	}
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	if len(diagnostics) != 0 {
		result.Guidance = []string{"Preserve the interrupted pack. A complete inventory is required before generation work can be reused; see SSB's generation recovery guide."}
		return result, pack, nil
	}
	result.Files = append(result.Files, FileReference{Path: inventoryPath, SHA256: digest(inventoryBytes)})
	var recorded ReportInventory
	if err := decodeStrictJSON(inventoryBytes, &recorded); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic(inventoryPath, "inventory", err.Error(), "preserve this file and recover a complete unedited inspection response for the pinned baseline"))
	} else {
		result.Diagnostics = append(result.Diagnostics, validateInventoryMetadata(inventoryPath, repo.Baseline(), recorded)...)
		if len(result.Diagnostics) == 0 {
			diagnostics, err = validateReportInventory(ctx, repo, recorded, inventoryPath)
			if err != nil {
				return result, pack, err
			}
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
		}
	}
	if len(result.Diagnostics) != 0 {
		result.Guidance = []string{"Inventory reuse is blocked. Preserve the pack and recover against its original baseline, or deliberately move it aside before a fresh inspection and semantic review."}
		return result, pack, nil
	}
	result.InventoryReusable = true
	result.Repeat = []string{"proposal_review", "validation", "rendering", "developer_review"}
	if _, err := os.Lstat(filepath.Join(repo.Root(), manifestPath)); errors.Is(err, os.ErrNotExist) {
		manifestMissing = true
		result.State = "needs_proposal"
		result.Guidance = []string{"Reuse the verified inventory. Preserve partial proposal files and manual edits; review retained candidates against pinned blobs and finish only missing work. Semantic completion can't be established without the final manifest. Write manifest.yaml last, then rerun ssb resume."}
		return result, pack, nil
	} else if err != nil {
		return result, pack, fmt.Errorf("inspect continuation manifest: %w", err)
	}
	manifestBytes, diagnostics, err := continuationFile(repo.Root(), manifestPath, inventory.DefaultLimits().MaxFileBytes)
	if err != nil {
		return result, pack, err
	}
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	result.State = "needs_correction"
	result.Guidance = []string{"Reuse the verified inventory and review retained proposal sources. Correct only affected canonical sources or manifest fields, deliberately update matching raw-byte SHA-256 values, then rerun ssb resume and ssb validate before rendering. Repeat semantic judgments only where a correction changes them."}
	if len(diagnostics) != 0 {
		return result, pack, nil
	}
	result.Files = append(result.Files, FileReference{Path: manifestPath, SHA256: digest(manifestBytes)})
	var manifest Manifest
	if err := yaml.Load(manifestBytes, &manifest, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		result.Diagnostics = append(result.Diagnostics, yamlDiagnostic(manifestPath, err, "correct the retained manifest explicitly; preserve canonical sources and rerun ssb resume"))
		return result, pack, nil
	}
	if manifest.BaselineCommit != repo.Baseline() {
		result.State = "blocked"
		result.Repeat = []string{"semantic_generation", "validation", "rendering", "developer_review"}
		result.Diagnostics = append(result.Diagnostics, diagnostic(manifestPath, "baseline_commit", "proposal baseline differs from the verified inventory and current HEAD", "preserve the old proposal; review against the pinned baseline before reusing semantic work"))
		result.Guidance = []string{"The retained proposal belongs to another baseline. Its semantic work can't be reused without a new review; preserve every source."}
		return result, pack, nil
	}
	// Capture only canonical paths; malformed paths are diagnosed by Validate.
	references := []FileReference{}
	if manifest.Report.Path == ".software-standards/report.md" {
		references = append(references, manifest.Report)
	}
	if manifest.Orientation.Path == orientationPath {
		references = append(references, manifest.Orientation)
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.Path != "" && artifact.Path == canonicalArtifactPath(artifact.Kind, artifact.ID) {
			references = append(references, FileReference{Path: artifact.Path, SHA256: artifact.SHA256})
		}
	}
	for _, reference := range references {
		content, fileDiagnostics, err := continuationFile(repo.Root(), reference.Path, inventory.DefaultLimits().MaxFileBytes)
		if err != nil {
			return result, pack, err
		}
		if len(fileDiagnostics) == 0 {
			result.Files = append(result.Files, FileReference{Path: reference.Path, SHA256: digest(content)})
		}
		result.Diagnostics = append(result.Diagnostics, fileDiagnostics...)
	}
	if len(result.Diagnostics) != 0 {
		return result, pack, nil
	}
	pack, result.Diagnostics, err = validateActionablePack(ctx, repo, false, &inventoryVerification{
		baselineCommit: repo.Baseline(),
		rawSHA256:      digest(inventoryBytes),
	})
	if err != nil {
		return result, pack, err
	}
	if len(result.Diagnostics) != 0 {
		return result, pack, nil
	}
	result.State = "ready_to_render"
	result.ProposalReusable = true
	result.Repeat = []string{"rendering", "developer_review"}
	result.Guidance = []string{"Reuse the validated proposal and verified inventory. Run ssb render --repo PATH --dry-run, then ssb render --repo PATH. Review the complete uncommitted result before the developer's adoption decision."}
	return result, pack, nil
}

func continuationFile(root, relative string, maxBytes int64) ([]byte, []Diagnostic, error) {
	if component, found, err := findSymlinkComponent(root, relative); err != nil {
		return nil, nil, err
	} else if found {
		return nil, []Diagnostic{diagnostic(relative, "file", "retained path contains symlink component "+component, "preserve the target and restore a regular repository-local file before continuation")}, nil
	}
	return readRequiredRegularFileLimited(filepath.Join(root, filepath.FromSlash(relative)), relative, maxBytes)
}

func continuationBaseline(ctx context.Context, repo *workspace.Repository) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := workspace.Open(ctx, repo.Root())
	if err != nil {
		return err
	}
	if current.Baseline() != repo.Baseline() {
		return fmt.Errorf("%w: HEAD changed during continuation assessment; preserve the pack and rerun ssb resume", workspace.ErrPrecondition)
	}
	changed, err := repo.Git(ctx, "diff", "--no-ext-diff", "--no-textconv", "--ignore-submodules=untracked", "--no-renames", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		return err
	}
	for _, relative := range bytes.Split(changed, []byte{0}) {
		if len(relative) == 0 || string(relative) == "AGENTS.md" {
			continue
		}
		_, exists, err := repo.EntryAtBaseline(ctx, string(relative))
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%w: tracked baseline file %q has edits; preserve them and restore a stable baseline before ssb resume", workspace.ErrPrecondition, relative)
		}
	}
	return nil
}

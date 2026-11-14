package rulepack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

// continuationSnapshot binds one assessment to its captured manifest and
// already-verified inventory bytes. Public validation never receives a
// snapshot and always performs its own complete checks.
type continuationSnapshot struct {
	baselineCommit  string
	inventorySHA256 string
	manifestSHA256  string
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
		if err := verifyContinuation(ctx, repo, result, pack); err != nil {
			returnErr = err
			return
		}
		if manifestMissing {
			if _, err := os.Lstat(filepath.Join(repo.Root(), manifestPath)); !errors.Is(err, os.ErrNotExist) {
				returnErr = fmt.Errorf("%w: manifest changed during continuation assessment; rerun ssb resume", workspace.ErrPrecondition)
			}
		}
	}()
	if _, err := os.Lstat(filepath.Join(repo.Root(), manifestPath)); errors.Is(err, os.ErrNotExist) {
		manifestMissing = true
		const reportPath = ".software-standards/report.md"
		content, diagnostics, err := continuationFile(repo.Root(), reportPath, inventory.DefaultLimits().MaxFileBytes, true)
		if err != nil {
			return result, pack, err
		}
		if len(diagnostics) == 0 {
			header, _, splitErr := splitFrontmatter(content)
			var metadata struct {
				Schema string `yaml:"schema"`
			}
			if splitErr == nil && yaml.Load(header, &metadata, yaml.WithUniqueKeys()) == nil && metadata.Schema == ReportSchema {
				result.Files = append(result.Files, FileReference{Path: reportPath, SHA256: digest(content)})
				result.Repeat = []string{"validation", "rendering", "developer_review"}
				result.Guidance = []string{"This retained report declares an embedded-layout pack. Preserve its sources and use its existing maintenance workflow: ssb validate --repo PATH, then ssb render --repo PATH after review. Generation continuation doesn't migrate this layout or require a new manifest."}
				result.Diagnostics = []Diagnostic{diagnostic(reportPath, "layout", "embedded-layout generation continuation is outside the manifest workflow", "preserve the pack and use its existing validation and rendering workflow; don't create a manifest as recovery")}
				return result, pack, nil
			}
		}
	}
	inventoryBytes, diagnostics, err := continuationFile(repo.Root(), inventoryPath, inventory.DefaultLimits().MaxCandidateBytes, false)
	if err != nil {
		return result, pack, err
	}
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	if len(diagnostics) != 0 {
		result.Guidance = []string{"Preserve the interrupted pack. A complete inventory is required before generation work can be reused; see SSB's generation recovery guide."}
		return result, pack, nil
	}
	inventoryIdentity := FileReference{Path: inventoryPath, SHA256: digest(inventoryBytes)}
	result.Files = append(result.Files, inventoryIdentity)
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
		partialFiles, partialDiagnostics, err := partialContinuationFiles(ctx, repo.Root())
		if err != nil {
			return result, pack, err
		}
		result.Files = append(result.Files, partialFiles...)
		result.Diagnostics = append(result.Diagnostics, partialDiagnostics...)
		if len(partialDiagnostics) != 0 {
			result.State = "needs_correction"
		}
		return result, pack, nil
	} else if err != nil {
		return result, pack, fmt.Errorf("inspect continuation manifest: %w", err)
	}
	manifestBytes, diagnostics, err := continuationFile(repo.Root(), manifestPath, inventory.DefaultLimits().MaxFileBytes, false)
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
	if manifest.Inventory != inventoryIdentity {
		result.Diagnostics = append(result.Diagnostics, diagnostic(manifestPath, "inventory", "manifest inventory reference differs from the captured verified inventory", "bind the canonical inventory path and its raw-byte SHA-256 through explicit pack review; preserve the unedited inventory"))
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
		content, fileDiagnostics, err := readDigestBoundFile(repo.Root(), reference, "retained source", inventory.DefaultLimits().MaxFileBytes)
		if err != nil {
			return result, pack, err
		}
		if len(content) != 0 {
			result.Files = append(result.Files, FileReference{Path: reference.Path, SHA256: digest(content)})
		}
		result.Diagnostics = append(result.Diagnostics, fileDiagnostics...)
	}
	if len(result.Diagnostics) != 0 {
		return result, pack, nil
	}
	pack, result.Diagnostics, err = validateActionablePack(ctx, repo, false, &continuationSnapshot{
		baselineCommit:  repo.Baseline(),
		inventorySHA256: inventoryIdentity.SHA256,
		manifestSHA256:  digest(manifestBytes),
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

// RecheckContinuation checks captured inputs after projection assessment and
// identifies shared repository skills that still need explicit review. It
// neither validates unindexed skills nor grants permission to write them.
func RecheckContinuation(ctx context.Context, repo *workspace.Repository, result Continuation, pack Pack) (Continuation, error) {
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := verifyContinuation(ctx, repo, result, pack); err != nil {
		return result, err
	}
	paths, err := unindexedRepositorySkills(ctx, repo.Root(), pack.Manifest, inventory.DefaultLimits())
	if err != nil {
		return result, err
	}
	if len(paths) != 0 {
		for index := range paths {
			paths[index] = fmt.Sprintf("%q", paths[index])
		}
		result.Guidance = append(result.Guidance, "Unindexed repository skill paths need explicit review of ownership and completion before reuse: "+strings.Join(paths, ", ")+".")
	}
	return result, ctx.Err()
}

func verifyContinuation(ctx context.Context, repo *workspace.Repository, result Continuation, pack Pack) error {
	if err := continuationBaseline(ctx, repo); err != nil {
		return err
	}
	limits := inventory.DefaultLimits()
	for _, reference := range result.Files {
		maxBytes := limits.MaxFileBytes
		if reference.Path == ".software-standards/inventory.json" {
			maxBytes = limits.MaxCandidateBytes
		}
		content, diagnostics, err := continuationFile(repo.Root(), reference.Path, maxBytes, true)
		if err != nil {
			return err
		}
		if len(diagnostics) != 0 || digest(content) != reference.SHA256 {
			return fmt.Errorf("%w: retained files changed during continuation assessment; preserve edits and rerun ssb resume", workspace.ErrPrecondition)
		}
	}
	if result.ProposalReusable {
		return verifyContinuationArtifactSet(repo.Root(), pack.Manifest)
	}
	if result.State == "needs_proposal" {
		paths, diagnostics, err := partialContinuationPaths(repo.Root())
		if err != nil {
			return err
		}
		known := make(map[string]bool, len(result.Files))
		for _, reference := range result.Files {
			known[reference.Path] = true
		}
		changed := len(diagnostics) != 0
		for _, relative := range paths {
			changed = changed || !known[relative]
		}
		_, err = os.Lstat(filepath.Join(repo.Root(), ".software-standards/manifest.yaml"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect continuation manifest: %w", err)
		}
		if changed || err == nil {
			return fmt.Errorf("%w: partial proposal artifact set changed during continuation assessment; preserve manual work and rerun ssb resume", workspace.ErrPrecondition)
		}
	}
	return nil
}

func unindexedRepositorySkills(ctx context.Context, root string, manifest Manifest, limits inventory.Limits) ([]string, error) {
	const directory = ".agents/skills"
	for _, relative := range []string{".agents", directory} {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("inspect repository skills: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return []string{relative}, nil
		}
	}
	store, err := os.Open(filepath.Join(root, filepath.FromSlash(directory)))
	if err != nil {
		return nil, fmt.Errorf("read repository skills: %w", err)
	}
	defer store.Close()
	listed := make(map[string]bool)
	for _, artifact := range manifest.Artifacts {
		if artifact.Kind == "skill" {
			listed[artifact.Path] = true
		}
	}
	paths := []string{}
	var entriesSeen int
	var pathBytes int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entries, readErr := store.ReadDir(256)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, fmt.Errorf("read repository skills: %w", readErr)
		}
		entriesSeen += len(entries)
		if entriesSeen > limits.MaxCandidateFiles {
			return nil, fmt.Errorf("%w: shared skill review exceeds the entry limit; preserve workflows and review the namespace before retrying", workspace.ErrPrecondition)
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			relative := directory + "/" + entry.Name()
			if entry.Type()&os.ModeSymlink == 0 && entry.IsDir() {
				if listed[relative+"/SKILL.md"] {
					support, err := repositorySkillHasSupport(filepath.Join(root, filepath.FromSlash(relative)))
					if err != nil {
						return nil, fmt.Errorf("inspect skill support paths %s: %w", relative, err)
					}
					if !support {
						continue
					}
				} else if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative), "SKILL.md")); err == nil {
					relative += "/SKILL.md"
				} else if !errors.Is(err, os.ErrNotExist) {
					return nil, fmt.Errorf("inspect repository skill %s: %w", relative, err)
				}
			}
			pathBytes += int64(len(fmt.Sprintf("%q", relative)))
			if pathBytes > limits.MaxFileBytes {
				return nil, fmt.Errorf("%w: shared skill review exceeds the path-name byte limit; preserve workflows and review the namespace before retrying", workspace.ErrPrecondition)
			}
			paths = append(paths, relative)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func repositorySkillHasSupport(directory string) (bool, error) {
	store, err := os.Open(directory)
	if err != nil {
		return false, err
	}
	defer store.Close()
	entries, err := store.ReadDir(2)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	for _, entry := range entries {
		if entry.Name() != "SKILL.md" {
			return true, nil
		}
	}
	return false, nil
}

func verifyContinuationArtifactSet(root string, manifest Manifest) error {
	listed := make(map[string]AcceptedArtifact, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		listed[artifact.Path] = artifact
	}
	unlisted, diagnostics, err := unlistedNativeArtifacts(root, listed)
	if err != nil {
		return err
	}
	changed := len(unlisted) != 0 || len(diagnostics) != 0
	if manifest.Orientation.IsZero() {
		_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(orientationPath)))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect optional orientation during continuation: %w", err)
		}
		changed = changed || err == nil
	}
	if changed {
		return fmt.Errorf("%w: proposal artifact set changed during continuation assessment; preserve manual work and rerun ssb resume", workspace.ErrPrecondition)
	}
	return nil
}

func partialContinuationPaths(root string) ([]string, []Diagnostic, error) {
	paths, diagnostics, err := unlistedNativeArtifacts(root, map[string]AcceptedArtifact{})
	if err != nil {
		return nil, nil, err
	}
	for _, relative := range []string{".software-standards/report.md", orientationPath} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative))); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, nil, fmt.Errorf("inspect partial source %s: %w", relative, err)
		}
		paths = append(paths, relative)
	}
	return paths, diagnostics, nil
}

func partialContinuationFiles(ctx context.Context, root string) ([]FileReference, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	paths, diagnostics, err := partialContinuationPaths(root)
	if err != nil {
		return nil, nil, err
	}
	limits := inventory.DefaultLimits()
	if len(paths) > limits.MaxCandidateFiles {
		return nil, append(diagnostics, diagnostic(".software-standards", "files", "partial sources exceed the candidate-file limit", "preserve the partial pack and review its contents before continuation")), nil
	}
	files := make([]FileReference, 0, len(paths))
	var totalBytes int64
	for _, relative := range paths {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		content, fileDiagnostics, err := continuationFile(root, relative, limits.MaxFileBytes, true)
		if err != nil {
			return nil, nil, err
		}
		diagnostics = append(diagnostics, fileDiagnostics...)
		if len(fileDiagnostics) != 0 {
			continue
		}
		totalBytes += int64(len(content))
		if totalBytes > limits.MaxCandidateBytes {
			diagnostics = append(diagnostics, diagnostic(relative, "files", "partial sources exceed the candidate-byte limit", "preserve the partial pack and review its contents before continuation"))
			break
		}
		files = append(files, FileReference{Path: relative, SHA256: digest(content)})
	}
	return files, diagnostics, nil
}

func continuationFile(root, relative string, maxBytes int64, allowEmpty bool) ([]byte, []Diagnostic, error) {
	if component, found, err := findSymlinkComponent(root, relative); err != nil {
		return nil, nil, err
	} else if found {
		return nil, []Diagnostic{diagnostic(relative, "file", "retained path contains symlink component "+component, "preserve the target and restore a regular repository-local file before continuation")}, nil
	}
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	if allowEmpty {
		return readRegularFileLimited(absolute, relative, maxBytes)
	}
	return readRequiredRegularFileLimited(absolute, relative, maxBytes)
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
	rootEntry, rootExists, err := repo.EntryAtBaseline(ctx, "AGENTS.md")
	if err != nil {
		return err
	}
	rootBaselineRegular := rootExists && rootEntry.Kind == "blob" && (rootEntry.Mode == "100644" || rootEntry.Mode == "100755")
	visibility, err := repo.Git(ctx, "ls-files", "-v", "--stage", "-z", "--")
	if err != nil {
		return err
	}
	rootRegularInGit := false
	for _, record := range bytes.Split(visibility, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		if len(record) < 3 || record[1] != ' ' {
			return errors.New("read index visibility: malformed Git file record")
		}
		metadata, pathBytes, hasPath := bytes.Cut(record[2:], []byte{'\t'})
		fields := bytes.Fields(metadata)
		if !hasPath || len(fields) != 3 {
			return errors.New("read index visibility: malformed staged Git file record")
		}
		relative := string(pathBytes)
		if relative == "AGENTS.md" {
			cached := record[0] == 'H' || record[0] == 'h' || record[0] == 'S' || record[0] == 's'
			regular := string(fields[0]) == "100644" || string(fields[0]) == "100755"
			rootRegularInGit = rootBaselineRegular && cached && regular && string(fields[2]) == "0"
		}
		// Lowercase tags indicate assume-unchanged; S indicates skip-worktree.
		if record[0] != 'S' && (record[0] < 'a' || record[0] > 'z') {
			continue
		}
		_, exists, err := repo.EntryAtBaseline(ctx, relative)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%w: index visibility flags hide tracked baseline file %q; preserve edits and review assume-unchanged or skip-worktree flags before ssb resume", workspace.ErrPrecondition, relative)
		}
	}
	if rootExists {
		info, err := os.Lstat(filepath.Join(repo.Root(), "AGENTS.md"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect root instructions: %w", err)
		}
		if err == nil && info.Mode().IsRegular() && !rootRegularInGit {
			return fmt.Errorf("%w: tracked baseline file AGENTS.md lacks regular, conflict-free entries across HEAD and the index; preserve manual work and resolve the root entry before ssb resume", workspace.ErrPrecondition)
		}
	}
	seen := make(map[string]bool)
	for _, cached := range []bool{false, true} {
		args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-renames", "--ignore-submodules=untracked", "--name-status", "-z"}
		if cached {
			args = append(args, "--cached")
		}
		args = append(args, "HEAD", "--")
		changed, err := repo.Git(ctx, args...)
		if err != nil {
			return err
		}
		if len(changed) == 0 {
			continue
		}
		records := bytes.Split(bytes.TrimSuffix(changed, []byte{0}), []byte{0})
		if len(records)%2 != 0 {
			return errors.New("read baseline changes: malformed Git name-status record")
		}
		for index := 0; index < len(records); index += 2 {
			kind, relative := string(records[index]), string(records[index+1])
			if kind == "" || relative == "" {
				return errors.New("read baseline changes: empty Git name-status field")
			}
			key := kind + "\x00" + relative
			if seen[key] {
				continue
			}
			seen[key] = true
			if kind == "M" && editableRootInstructions(repo.Root(), relative, rootRegularInGit) {
				continue
			}
			_, exists, err := repo.EntryAtBaseline(ctx, relative)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("%w: tracked baseline file %q has edits; preserve them and restore a stable baseline before ssb resume", workspace.ErrPrecondition, relative)
			}
		}
	}
	return nil
}

// Existing regular root instructions can retain developer edits around the
// managed projection when baseline, index, and worktree all hold regular files.
func editableRootInstructions(root, relative string, rootRegularInGit bool) bool {
	if relative != "AGENTS.md" || !rootRegularInGit {
		return false
	}
	info, err := os.Lstat(filepath.Join(root, "AGENTS.md"))
	return err == nil && info.Mode().IsRegular()
}

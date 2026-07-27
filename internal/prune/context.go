package prune

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

var stableIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ErrIncompleteInventory means the current repository evidence set was not
// fully observed and no semantic proposal may be produced.
var ErrIncompleteInventory = errors.New("inventory coverage incomplete")

// ContextOptions identifies the exact local evidence inputs for one review.
type ContextOptions struct {
	ReviewID        string
	Capabilities    string
	Provenance      string
	InventoryLimits inventory.Limits
}

// BuildContext captures current repository evidence and exact local
// capability/provenance inputs without writing lifecycle artifacts.
func BuildContext(
	ctx context.Context,
	repo *workspace.Repository,
	options ContextOptions,
) (Context, error) {
	if !stableIDPattern.MatchString(options.ReviewID) {
		return Context{}, fmt.Errorf("review id must be lower-case kebab-case")
	}
	if options.Capabilities == "" {
		return Context{}, fmt.Errorf("capability profile path is required")
	}
	profile, profileDigest, err := loadCapabilityProfile(options.Capabilities)
	if err != nil {
		return Context{}, err
	}
	provenance, provenanceDigest, err := loadProvenance(options.Provenance)
	if err != nil {
		return Context{}, err
	}
	report, err := inventory.ScanForPrune(ctx, repo, options.InventoryLimits)
	if err != nil {
		return Context{}, err
	}
	if report.Truncated {
		return Context{}, fmt.Errorf("%w: %s", ErrIncompleteInventory, report.TruncationReason)
	}
	artifacts, err := loadArtifacts(ctx, repo, provenance, report)
	if err != nil {
		return Context{}, err
	}
	result := Context{
		Schema:                   ContextSchema,
		ReviewID:                 options.ReviewID,
		BaselineCommit:           repo.Baseline(),
		Inventory:                report,
		Artifacts:                artifacts,
		Capabilities:             profile,
		CapabilityProfileDigest:  profileDigest,
		CapabilityProfilePath:    path.Join("inputs/capability", filepath.Base(options.Capabilities)),
		ProvenanceManifestDigest: provenanceDigest,
	}
	if options.Provenance != "" {
		result.ProvenanceManifestPath = path.Join("inputs/provenance", filepath.Base(options.Provenance))
	}
	digest, err := canonicalDigest(result)
	if err != nil {
		return Context{}, fmt.Errorf("digest prune context: %w", err)
	}
	result.ContextDigest = digest
	return result, nil
}

func loadCapabilityProfile(filePath string) (CapabilityProfile, string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return CapabilityProfile{}, "", fmt.Errorf("read capability profile: %w", err)
	}
	var profile CapabilityProfile
	if err := yaml.Load(data, &profile, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return CapabilityProfile{}, "", fmt.Errorf("parse capability profile: %w", err)
	}
	if profile.Schema != CapabilitySchema {
		return CapabilityProfile{}, "", fmt.Errorf("capability profile schema must be %s", CapabilitySchema)
	}
	if !stableIDPattern.MatchString(profile.ID) {
		return CapabilityProfile{}, "", fmt.Errorf("capability profile id must be lower-case kebab-case")
	}
	if strings.TrimSpace(profile.Host.Name) == "" || strings.TrimSpace(profile.Host.Version) == "" {
		return CapabilityProfile{}, "", fmt.Errorf("capability profile requires exact host name and version")
	}
	if strings.TrimSpace(profile.Model.Provider) == "" || strings.TrimSpace(profile.Model.ID) == "" {
		return CapabilityProfile{}, "", fmt.Errorf("capability profile requires exact model provider and id")
	}
	if _, err := time.Parse(time.RFC3339, string(profile.ObservedAt)); err != nil {
		return CapabilityProfile{}, "", fmt.Errorf("capability profile observed_at must be RFC3339: %w", err)
	}

	base := filepath.Dir(filePath)
	evidenceByID := make(map[string]CapabilityEvidence)
	conformance := make(map[string]bool)
	for _, item := range profile.Evidence {
		if !stableIDPattern.MatchString(item.ID) {
			return CapabilityProfile{}, "", fmt.Errorf("capability evidence id %q must be lower-case kebab-case", item.ID)
		}
		if _, duplicate := evidenceByID[item.ID]; duplicate {
			return CapabilityProfile{}, "", fmt.Errorf("duplicate capability evidence id %q", item.ID)
		}
		if !safeRelativePath(item.Path) {
			return CapabilityProfile{}, "", fmt.Errorf("unsafe capability evidence path %q", item.Path)
		}
		evidencePath := filepath.Join(base, filepath.FromSlash(item.Path))
		content, err := os.ReadFile(evidencePath)
		if err != nil {
			return CapabilityProfile{}, "", fmt.Errorf("read capability evidence %s: %w", item.Path, err)
		}
		if digestBytes(content) != item.SHA256 {
			return CapabilityProfile{}, "", fmt.Errorf("capability evidence digest mismatch for %s", item.Path)
		}
		evidenceByID[item.ID] = item
		conformance[item.ID] = item.Kind == "conformance"
	}
	if len(profile.Capabilities) == 0 {
		return CapabilityProfile{}, "", fmt.Errorf("capability profile must contain at least one capability")
	}
	seenCapabilities := make(map[string]struct{})
	for _, capability := range profile.Capabilities {
		if !stableIDPattern.MatchString(capability.ID) {
			return CapabilityProfile{}, "", fmt.Errorf("capability id %q must be lower-case kebab-case", capability.ID)
		}
		if _, duplicate := seenCapabilities[capability.ID]; duplicate {
			return CapabilityProfile{}, "", fmt.Errorf("duplicate capability id %q", capability.ID)
		}
		seenCapabilities[capability.ID] = struct{}{}
		switch capability.Status {
		case CapabilitySupported, CapabilityUnsupported:
			hasConformance := false
			for _, evidenceID := range capability.EvidenceIDs {
				if _, exists := evidenceByID[evidenceID]; !exists {
					return CapabilityProfile{}, "", fmt.Errorf("capability %s references unknown evidence %s", capability.ID, evidenceID)
				}
				hasConformance = hasConformance || conformance[evidenceID]
			}
			if !hasConformance {
				return CapabilityProfile{}, "", fmt.Errorf("capability %s requires content-addressed conformance evidence", capability.ID)
			}
		case CapabilityUnknown:
			for _, evidenceID := range capability.EvidenceIDs {
				if _, exists := evidenceByID[evidenceID]; !exists {
					return CapabilityProfile{}, "", fmt.Errorf("capability %s references unknown evidence %s", capability.ID, evidenceID)
				}
			}
		default:
			return CapabilityProfile{}, "", fmt.Errorf("capability %s has unsupported status %q", capability.ID, capability.Status)
		}
	}
	sort.Slice(profile.Evidence, func(i, j int) bool { return profile.Evidence[i].ID < profile.Evidence[j].ID })
	sort.Slice(profile.Capabilities, func(i, j int) bool {
		return profile.Capabilities[i].ID < profile.Capabilities[j].ID
	})
	return profile, digestBytes(data), nil
}

func loadProvenance(filePath string) (ProvenanceManifest, string, error) {
	if filePath == "" {
		return ProvenanceManifest{Schema: ProvenanceSchema, Artifacts: []ProvenanceEntry{}}, "", nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ProvenanceManifest{}, "", fmt.Errorf("read provenance manifest: %w", err)
	}
	var manifest ProvenanceManifest
	if err := yaml.Load(data, &manifest, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return ProvenanceManifest{}, "", fmt.Errorf("parse provenance manifest: %w", err)
	}
	if manifest.Schema != ProvenanceSchema {
		return ProvenanceManifest{}, "", fmt.Errorf("provenance schema must be %s", ProvenanceSchema)
	}
	seen := make(map[string]struct{})
	for _, item := range manifest.Artifacts {
		if !safeRelativePath(item.Path) {
			return ProvenanceManifest{}, "", fmt.Errorf("unsafe provenance path %q", item.Path)
		}
		if !validDigest(item.SHA256) {
			return ProvenanceManifest{}, "", fmt.Errorf("invalid provenance digest for %s", item.Path)
		}
		switch item.Origin {
		case OriginGenerated, OriginUserAuthored, OriginMixed, OriginUnknown:
		default:
			return ProvenanceManifest{}, "", fmt.Errorf("unsupported provenance origin %q", item.Origin)
		}
		if strings.TrimSpace(item.Declaration) == "" {
			return ProvenanceManifest{}, "", fmt.Errorf("provenance declaration is required for %s", item.Path)
		}
		if _, duplicate := seen[item.Path]; duplicate {
			return ProvenanceManifest{}, "", fmt.Errorf("duplicate provenance path %s", item.Path)
		}
		seen[item.Path] = struct{}{}
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path
	})
	return manifest, digestBytes(data), nil
}

func loadArtifacts(
	ctx context.Context,
	repo *workspace.Repository,
	provenance ProvenanceManifest,
	report inventory.Report,
) ([]Artifact, error) {
	raw, err := repo.Git(
		ctx,
		"ls-tree", "-r", "-z", "--name-only", repo.Baseline(), "--",
		".software-standards/rules", ".agents/skills",
	)
	if err != nil {
		return nil, fmt.Errorf("list prune artifacts: %w", err)
	}
	provenanceByPath := make(map[string]ProvenanceEntry)
	for _, item := range provenance.Artifacts {
		provenanceByPath[item.Path] = item
	}
	treePaths := make([]string, 0)
	for _, record := range bytes.Split(raw, []byte{0}) {
		if len(record) != 0 {
			treePaths = append(treePaths, string(record))
		}
	}
	inventoryFiles := make(map[string]inventory.File, len(report.Files))
	for _, file := range report.Files {
		inventoryFiles[file.Path] = file
	}
	artifacts := make([]Artifact, 0)
	for _, relative := range treePaths {
		kind, id, ok := artifactIdentity(relative)
		if !ok {
			continue
		}
		inventoried, exists := inventoryFiles[relative]
		if !exists {
			return nil, fmt.Errorf("configuration %s is excluded from the bounded inventory", relative)
		}
		mainEntry, exists, err := repo.EntryAtBaseline(ctx, relative)
		if err != nil {
			return nil, err
		}
		if !exists || mainEntry.Kind != "blob" || (mainEntry.Mode != "100644" && mainEntry.Mode != "100755") {
			return nil, fmt.Errorf("configuration %s is not a tracked regular file", relative)
		}
		digest := inventoried.SHA256
		origin := OriginUnknown
		if declared, exists := provenanceByPath[relative]; exists {
			if declared.SHA256 != digest {
				return nil, fmt.Errorf("provenance digest mismatch for %s", relative)
			}
			origin = declared.Origin
			delete(provenanceByPath, relative)
		}
		artifact := Artifact{
			Kind:   kind,
			ID:     id,
			Path:   relative,
			SHA256: digest,
			Mode:   mainEntry.Mode,
			Origin: origin,
		}
		if kind == ArtifactSkill {
			prefix := path.Dir(relative) + "/"
			for _, supportingPath := range treePaths {
				if supportingPath == relative || !strings.HasPrefix(supportingPath, prefix) {
					continue
				}
				entry, exists, err := repo.EntryAtBaseline(ctx, supportingPath)
				if err != nil {
					return nil, err
				}
				if !exists || entry.Kind != "blob" || (entry.Mode != "100644" && entry.Mode != "100755") {
					return nil, fmt.Errorf("skill supporting path %s is not a tracked regular file", supportingPath)
				}
				supportingInventory, inventoried := inventoryFiles[supportingPath]
				if !inventoried {
					return nil, fmt.Errorf("skill supporting path %s is excluded from the bounded inventory", supportingPath)
				}
				artifact.SupportingFiles = append(artifact.SupportingFiles, ArtifactFile{
					Path: supportingPath, SHA256: supportingInventory.SHA256, Mode: entry.Mode,
				})
			}
		}
		artifacts = append(artifacts, artifact)
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Path < artifacts[j].Path
	})
	if len(provenanceByPath) != 0 {
		paths := make([]string, 0, len(provenanceByPath))
		for declaredPath := range provenanceByPath {
			paths = append(paths, declaredPath)
		}
		sort.Strings(paths)
		return nil, fmt.Errorf("provenance declares non-canonical or missing artifact %s", paths[0])
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("adopted pack has no lifecycle candidate rules or repository skills")
	}
	return artifacts, nil
}

func artifactIdentity(relative string) (string, string, bool) {
	if strings.HasPrefix(relative, ".software-standards/rules/") &&
		path.Dir(relative) == ".software-standards/rules" &&
		strings.HasSuffix(relative, ".md") {
		id := strings.TrimSuffix(path.Base(relative), ".md")
		return ArtifactRule, id, stableIDPattern.MatchString(id)
	}
	const skillPrefix = ".agents/skills/"
	if strings.HasPrefix(relative, skillPrefix) && path.Base(relative) == "SKILL.md" {
		remainder := strings.TrimPrefix(relative, skillPrefix)
		parts := strings.Split(remainder, "/")
		if len(parts) == 2 && parts[1] == "SKILL.md" && stableIDPattern.MatchString(parts[0]) {
			return ArtifactSkill, parts[0], true
		}
	}
	return "", "", false
}

func canonicalDigest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && value == strings.ToLower(value)
}

func safeRelativePath(value string) bool {
	return value != "" && !path.IsAbs(value) && path.Clean(value) == value &&
		value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

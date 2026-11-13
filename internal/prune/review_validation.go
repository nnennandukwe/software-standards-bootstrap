package prune

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func validateContext(reviewContext ReviewContext) error {
	if reviewContext.Schema != ContextSchema {
		return fmt.Errorf("review context schema must be %s", ContextSchema)
	}
	if !stableIDPattern.MatchString(reviewContext.ReviewID) {
		return fmt.Errorf("review context has invalid review_id")
	}
	if !validDigest(reviewContext.ContextDigest) {
		return fmt.Errorf("review context has invalid digest")
	}
	expected := reviewContext.ContextDigest
	reviewContext.ContextDigest = ""
	actual, err := canonicalDigest(reviewContext)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("review context digest mismatch")
	}
	if reviewContext.Inventory.Truncated {
		return fmt.Errorf("%w: stored context is truncated", ErrIncompleteInventory)
	}
	if len(reviewContext.Artifacts) == 0 {
		return fmt.Errorf("review context has no governed artifacts")
	}
	governedPaths := make([]string, 0, len(reviewContext.Artifacts))
	for _, artifact := range reviewContext.Artifacts {
		kind, id, canonical := artifactIdentity(artifact.Path)
		if !canonical || kind != artifact.Kind || id != artifact.ID ||
			!validDigest(artifact.SHA256) ||
			(artifact.Mode != "100644" && artifact.Mode != "100755") {
			return fmt.Errorf("review context artifact %s is invalid", artifact.Path)
		}
		switch artifact.Origin {
		case OriginGenerated, OriginUserAuthored, OriginMixed, OriginUnknown:
		default:
			return fmt.Errorf("review context artifact %s has invalid provenance", artifact.Path)
		}
		if artifact.Kind != ArtifactSkill && len(artifact.SupportingFiles) != 0 {
			return fmt.Errorf("review context rule %s has supporting files", artifact.Path)
		}
		governedPaths = append(governedPaths, artifact.Path)
		skillRoot := path.Dir(artifact.Path) + "/"
		for _, supporting := range artifact.SupportingFiles {
			if artifact.Kind != ArtifactSkill ||
				!strings.HasPrefix(supporting.Path, skillRoot) ||
				supporting.Path == artifact.Path ||
				!validDigest(supporting.SHA256) ||
				(supporting.Mode != "100644" && supporting.Mode != "100755") {
				return fmt.Errorf("review context supporting file %s is invalid", supporting.Path)
			}
			governedPaths = append(governedPaths, supporting.Path)
		}
	}
	if err := validateGovernedTreePaths(governedPaths); err != nil {
		return fmt.Errorf("review context governed paths are invalid: %w", err)
	}
	if !validDigest(reviewContext.CapabilityProfileDigest) ||
		!safeRelativePath(reviewContext.CapabilityProfilePath) ||
		!strings.HasPrefix(reviewContext.CapabilityProfilePath, "inputs/capability/") {
		return fmt.Errorf("review context has an invalid capability snapshot reference")
	}
	if reviewContext.ProvenanceManifestDigest != "" &&
		(!validDigest(reviewContext.ProvenanceManifestDigest) ||
			!safeRelativePath(reviewContext.ProvenanceManifestPath) ||
			!strings.HasPrefix(reviewContext.ProvenanceManifestPath, "inputs/provenance/")) {
		return fmt.Errorf("review context has an invalid provenance snapshot reference")
	}
	return nil
}

func validateReviewSnapshots(store *reviewStore, reviewContext ReviewContext) error {
	profileData, _, err := store.ReadRegular(reviewContext.CapabilityProfilePath)
	if err != nil {
		return fmt.Errorf("capability profile snapshot: %w", err)
	}
	if digestBytes(profileData) != reviewContext.CapabilityProfileDigest {
		return fmt.Errorf("capability profile snapshot: snapshot digest mismatch")
	}
	profileRoot := path.Dir(reviewContext.CapabilityProfilePath)
	for _, evidence := range reviewContext.Capabilities.Evidence {
		target := path.Join(profileRoot, evidence.Path)
		data, _, err := store.ReadRegular(target)
		if err != nil {
			return fmt.Errorf("capability evidence snapshot %s: %w", evidence.ID, err)
		}
		if digestBytes(data) != evidence.SHA256 {
			return fmt.Errorf("capability evidence snapshot %s: snapshot digest mismatch", evidence.ID)
		}
	}
	if reviewContext.ProvenanceManifestDigest != "" {
		data, _, err := store.ReadRegular(reviewContext.ProvenanceManifestPath)
		if err != nil {
			return fmt.Errorf("provenance snapshot: %w", err)
		}
		if digestBytes(data) != reviewContext.ProvenanceManifestDigest {
			return fmt.Errorf("provenance snapshot: snapshot digest mismatch")
		}
	}
	return nil
}

func reviewRoot(repoRoot, reviewID string) (string, error) {
	if !stableIDPattern.MatchString(reviewID) {
		return "", fmt.Errorf("review id must be lower-case kebab-case")
	}
	root := filepath.Join(
		repoRoot,
		".software-standards",
		"reviews",
		reviewID,
	)
	current := repoRoot
	for _, component := range []string{".software-standards", "reviews", reviewID} {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return root, nil
		}
		if err != nil {
			return "", fmt.Errorf("inspect review path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("review path %s contains a symlink", current)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("review path %s is not a directory", current)
		}
	}
	return root, nil
}

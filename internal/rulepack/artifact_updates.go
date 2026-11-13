package rulepack

import (
	"fmt"
	"sort"

	"go.yaml.in/yaml/v4"
)

// RemoveReportArtifacts removes accepted artifacts and relationships to
// them while preserving the report inventory and narrative. Lifecycle
// mutation uses this to keep report.md in the same atomic write set as the
// governed artifacts it removes.
func RemoveReportArtifacts(data []byte, removedIDs map[string]struct{}) ([]byte, error) {
	if len(removedIDs) == 0 {
		return data, nil
	}
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}
	var report Report
	if err := yaml.Load(
		frontmatter,
		&report,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return nil, fmt.Errorf("parse report frontmatter: %w", err)
	}
	found := make(map[string]struct{}, len(removedIDs))
	artifacts := make([]AcceptedArtifact, 0, len(report.Artifacts))
	for _, artifact := range report.Artifacts {
		if _, removed := removedIDs[artifact.ID]; removed {
			found[artifact.ID] = struct{}{}
			continue
		}
		relationships := artifact.RelatedArtifactIDs[:0]
		for _, relatedID := range artifact.RelatedArtifactIDs {
			if _, removed := removedIDs[relatedID]; !removed {
				relationships = append(relationships, relatedID)
			}
		}
		artifact.RelatedArtifactIDs = relationships
		artifacts = append(artifacts, artifact)
	}
	if len(found) != len(removedIDs) {
		missing := make([]string, 0, len(removedIDs)-len(found))
		for id := range removedIDs {
			if _, exists := found[id]; !exists {
				missing = append(missing, id)
			}
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("report does not list artifact %s", missing[0])
	}
	report.Artifacts = artifacts
	encoded, err := yaml.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode report frontmatter: %w", err)
	}
	result := make([]byte, 0, len(encoded)+len(body)+10)
	result = append(result, "---\n"...)
	result = append(result, encoded...)
	result = append(result, "---\n"...)
	result = append(result, body...)
	return result, nil
}

// UpdateManifestArtifacts removes accepted artifacts, clears dangling
// relationships, and refreshes primary-file digests without changing semantic
// metadata. Governed prune includes the returned bytes in its atomic write set.
func UpdateManifestArtifacts(
	data []byte,
	removedIDs map[string]struct{},
	updatedDigests map[string]string,
) ([]byte, error) {
	if len(removedIDs) == 0 && len(updatedDigests) == 0 {
		return data, nil
	}
	var manifest Manifest
	if err := yaml.Load(
		data,
		&manifest,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if manifest.Schema != ManifestSchema {
		return nil, fmt.Errorf("manifest schema must be %s", ManifestSchema)
	}
	foundRemoved := make(map[string]struct{}, len(removedIDs))
	foundUpdated := make(map[string]struct{}, len(updatedDigests))
	artifacts := make([]AcceptedArtifact, 0, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		if _, removed := removedIDs[artifact.ID]; removed {
			if _, alsoUpdated := updatedDigests[artifact.ID]; alsoUpdated {
				return nil, fmt.Errorf("artifact %s cannot be removed and digest-updated", artifact.ID)
			}
			foundRemoved[artifact.ID] = struct{}{}
			continue
		}
		if updated, exists := updatedDigests[artifact.ID]; exists {
			if !digestPattern.MatchString(updated) {
				return nil, fmt.Errorf("updated digest for artifact %s is invalid", artifact.ID)
			}
			artifact.SHA256 = updated
			foundUpdated[artifact.ID] = struct{}{}
		}
		relationships := artifact.RelatedArtifactIDs[:0]
		for _, relatedID := range artifact.RelatedArtifactIDs {
			if _, removed := removedIDs[relatedID]; !removed {
				relationships = append(relationships, relatedID)
			}
		}
		artifact.RelatedArtifactIDs = relationships
		artifacts = append(artifacts, artifact)
	}
	if missing := missingManifestIDs(removedIDs, foundRemoved); len(missing) != 0 {
		return nil, fmt.Errorf("manifest does not list artifact %s", missing[0])
	}
	if missing := missingManifestIDs(updatedDigests, foundUpdated); len(missing) != 0 {
		return nil, fmt.Errorf("manifest does not list updated artifact %s", missing[0])
	}
	rootCore := manifest.RootCore[:0]
	for _, ruleID := range manifest.RootCore {
		if _, removed := removedIDs[ruleID]; !removed {
			rootCore = append(rootCore, ruleID)
		}
	}
	manifest.RootCore = rootCore
	manifest.Artifacts = artifacts
	encoded, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode manifest: %w", err)
	}
	return encoded, nil
}

func missingManifestIDs[T any](expected map[string]T, found map[string]struct{}) []string {
	missing := make([]string, 0)
	for id := range expected {
		if _, exists := found[id]; !exists {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	return missing
}

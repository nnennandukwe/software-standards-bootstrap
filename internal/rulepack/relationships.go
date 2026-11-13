package rulepack

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
)

func validateRelationships(sourcePath string, artifacts []AcceptedArtifact, entriesByID map[string]AcceptedArtifact) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for _, artifact := range artifacts {
		seenRelated := make(map[string]struct{}, len(artifact.RelatedArtifactIDs))
		for _, relatedID := range artifact.RelatedArtifactIDs {
			if relatedID == artifact.ID {
				diagnostics = append(diagnostics, diagnostic(
					sourcePath,
					"related_artifacts",
					fmt.Sprintf("artifact %s cannot relate to itself", artifact.ID),
					"remove the self relationship",
				))
			}
			if _, duplicate := seenRelated[relatedID]; duplicate {
				diagnostics = append(diagnostics, diagnostic(
					sourcePath,
					"related_artifacts",
					fmt.Sprintf("artifact %s repeats relationship %s", artifact.ID, relatedID),
					"list each related artifact once",
				))
			}
			seenRelated[relatedID] = struct{}{}
			if _, exists := entriesByID[relatedID]; !exists {
				diagnostics = append(diagnostics, diagnostic(
					sourcePath,
					"related_artifacts",
					fmt.Sprintf("artifact %s references missing related artifact %s", artifact.ID, relatedID),
					"restore the related artifact or remove the dangling relationship",
				))
			}
		}
	}
	return diagnostics
}

func validateRootCore(
	sourcePath string,
	ruleIDs []string,
	entriesByID map[string]AcceptedArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field,
			message,
			recovery,
		))
	}
	if len(ruleIDs) > RootCoreMaxRules {
		add(
			"root_core",
			fmt.Sprintf("root_core has %d rules; maximum is %d", len(ruleIDs), RootCoreMaxRules),
			"keep only repository-wide rules that must be read before routing and move the rest to on-demand bundles",
		)
	}
	seen := make(map[string]struct{}, len(ruleIDs))
	for index, ruleID := range ruleIDs {
		field := fmt.Sprintf("root_core[%d]", index)
		if !stableIDPattern.MatchString(ruleID) {
			add(field, fmt.Sprintf("root core rule id %q must be lower-case kebab-case", ruleID), "use the stable id of a retained repository-wide semantic rule")
			continue
		}
		if _, duplicate := seen[ruleID]; duplicate {
			add(field, fmt.Sprintf("duplicate root core rule %q", ruleID), "list each root core rule once")
			continue
		}
		seen[ruleID] = struct{}{}
		artifact, exists := entriesByID[ruleID]
		if !exists || artifact.Kind != "rule" {
			add(field, fmt.Sprintf("root core id %q must resolve to a retained semantic rule", ruleID), "reference a retained repository-wide semantic rule or remove the id")
			continue
		}
		if len(artifact.Lenses) != 1 || artifact.Lenses[0] != (Lens{Kind: "base"}) {
			add(field, fmt.Sprintf("root core rule %q must use the sole base lens", ruleID), "route contextual rules through the generated catalog instead")
		}
		if len(artifact.Scopes) != 1 || artifact.Scopes[0] != "**/*" {
			add(field, fmt.Sprintf("root core rule %q must use the sole repository-wide scope **/*", ruleID), "route path-specific rules through the generated catalog instead")
		}
	}
	return diagnostics
}

func buildRoutingCatalog(artifacts []AcceptedArtifact) *RoutingCatalog {
	type selector struct {
		Lenses []Lens   `json:"lenses"`
		Scopes []string `json:"scopes"`
	}
	bundlesByKey := make(map[string]*RoutingBundle)
	for _, artifact := range artifacts {
		if artifact.Kind == "automation" {
			continue
		}
		lenses := append([]Lens(nil), artifact.Lenses...)
		sort.Slice(lenses, func(i, j int) bool {
			if lenses[i].Kind != lenses[j].Kind {
				return lenses[i].Kind < lenses[j].Kind
			}
			return lenses[i].Value < lenses[j].Value
		})
		scopes := append([]string(nil), artifact.Scopes...)
		sort.Strings(scopes)
		encoded, err := json.Marshal(selector{Lenses: lenses, Scopes: scopes})
		if err != nil {
			panic(fmt.Sprintf("encode validated routing selector: %v", err))
		}
		key := string(encoded)
		bundle := bundlesByKey[key]
		if bundle == nil {
			sum := sha256.Sum256(encoded)
			id := "route-" + hex.EncodeToString(sum[:])
			bundle = &RoutingBundle{
				ID: id, Path: path.Join(RoutingBundleDirectory, id+".md"),
				Lenses: lenses, Scopes: scopes, ArtifactIDs: make([]string, 0),
			}
			bundlesByKey[key] = bundle
		}
		bundle.ArtifactIDs = append(bundle.ArtifactIDs, artifact.ID)
	}
	keys := make([]string, 0, len(bundlesByKey))
	for key := range bundlesByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	bundles := make([]RoutingBundle, 0, len(keys))
	for _, key := range keys {
		bundle := bundlesByKey[key]
		sort.Strings(bundle.ArtifactIDs)
		bundles = append(bundles, *bundle)
	}
	return &RoutingCatalog{CatalogPath: RoutingCatalogPath, Bundles: bundles}
}

func sortPackArtifacts(pack *Pack) {
	sort.Slice(pack.Rules, func(i, j int) bool { return pack.Rules[i].ID < pack.Rules[j].ID })
	sort.Slice(pack.Recipes, func(i, j int) bool { return pack.Recipes[i].ID < pack.Recipes[j].ID })
	sort.Slice(pack.Skills, func(i, j int) bool { return pack.Skills[i].ID < pack.Skills[j].ID })
	sort.Slice(pack.Automations, func(i, j int) bool { return pack.Automations[i].ID < pack.Automations[j].ID })
}

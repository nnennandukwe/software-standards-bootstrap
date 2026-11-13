package rulepack

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func validateManifest(repo *workspace.Repository, manifest Manifest, retained bool) []Diagnostic {
	const sourcePath = ".software-standards/manifest.yaml"
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field,
			message,
			recovery,
		))
	}
	if manifest.Schema != ManifestSchema {
		add("schema", "schema must be "+ManifestSchema, "update the manifest schema value")
	}
	if !commitPattern.MatchString(manifest.BaselineCommit) {
		add("baseline_commit", "baseline_commit must be a 40-character lowercase Git object id", "copy the exact commit from ssb inspect")
	} else if !retained && manifest.BaselineCommit != repo.Baseline() {
		add(
			"baseline_commit",
			fmt.Sprintf("baseline_commit must equal current HEAD %s", repo.Baseline()),
			"reinspect the new commit and refresh the manifest, inventory, and every evidence hash",
		)
	}
	if manifest.Inventory.Path != ".software-standards/inventory.json" {
		add("inventory.path", "inventory path must be .software-standards/inventory.json", "use the canonical manifest-layout inventory path")
	}
	if !digestPattern.MatchString(manifest.Inventory.SHA256) {
		add("inventory.sha256", "inventory sha256 must use sha256:<64 lowercase hex characters>", "hash the exact inventory.json bytes")
	}
	if manifest.Report.Path != ".software-standards/report.md" {
		add("report.path", "report path must be .software-standards/report.md", "use the canonical manifest-layout report path")
	}
	if !digestPattern.MatchString(manifest.Report.SHA256) {
		add("report.sha256", "report sha256 must use sha256:<64 lowercase hex characters>", "hash the exact report.md bytes")
	}
	if !manifest.Orientation.IsZero() {
		if manifest.Orientation.Path != orientationPath {
			add("orientation.path", "orientation path must be "+orientationPath, "use the canonical manifest-layout orientation path")
		}
		if !digestPattern.MatchString(manifest.Orientation.SHA256) {
			add("orientation.sha256", "orientation sha256 must use sha256:<64 lowercase hex characters>", "hash the exact orientation.yaml bytes")
		}
	}
	return diagnostics
}

func validateManifestArtifact(sourcePath, field string, artifact AcceptedArtifact) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field+suffix,
			message,
			recovery,
		))
	}
	if !stableIDPattern.MatchString(artifact.ID) {
		add(".id", "artifact id must be lower-case kebab-case", "choose a stable id such as verify-test-migrations")
	}
	expectedPath := canonicalArtifactPath(artifact.Kind, artifact.ID)
	switch artifact.Kind {
	case "rule", "verification", "skill", "automation":
	default:
		add(".kind", fmt.Sprintf("artifact kind %q is not supported", artifact.Kind), "use rule, verification, skill, or automation")
	}
	if expectedPath != "" && artifact.Path != expectedPath {
		add(
			".path",
			fmt.Sprintf("%s artifact path must be %s", artifact.Kind, expectedPath),
			"move the artifact to its canonical path or correct the manifest entry",
		)
	}
	if !digestPattern.MatchString(artifact.SHA256) {
		add(".sha256", "artifact sha256 must use sha256:<64 lowercase hex characters>", "hash the exact primary artifact bytes")
	}
	if artifact.Confidence != "medium" && artifact.Confidence != "high" {
		add(".confidence", "accepted artifact confidence must be medium or high", "remove the candidate instead of preserving a low-confidence artifact")
	}
	diagnostics = append(diagnostics, validateUtility(sourcePath, field+".utility", artifact.Utility)...)
	if artifact.Kind != "rule" && artifact.Directive != "" {
		add(".directive", "directive applies only to semantic rules", "remove directive from this manifest entry")
	}
	return diagnostics
}

func validateManifestOwnedMetadata(
	ctx context.Context,
	repo *workspace.Repository,
	sourcePath string,
	field string,
	artifact AcceptedArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field+suffix,
			message,
			recovery,
		))
	}
	if _, supported := supportedCategories[artifact.Category]; artifact.Category == "" || !supported {
		add(".category", fmt.Sprintf("category %q is not supported", artifact.Category), categoryRecovery)
	}
	diagnostics = append(diagnostics, validateActionableLenses(sourcePath, field+".lenses", artifact.Lenses)...)
	diagnostics = append(diagnostics, prefixDiagnosticFields(validateScopes(sourcePath, artifact.Scopes), field+".")...)
	if artifact.Kind == "rule" {
		if _, supported := supportedDirectives[artifact.Directive]; !supported {
			add(".directive", fmt.Sprintf("directive %q is not supported", artifact.Directive), "use always, ask-first, never, or prefer")
		}
	}
	diagnostics = append(diagnostics, prefixDiagnosticFields(validateDerivationEvidence(
		ctx,
		repo,
		sourcePath,
		artifact.Derivation,
		artifact.Evidence,
	), field+".")...)
	return diagnostics
}

func validatePortableManifestSkill(sourcePath string, metadata skillFrontmatter, body []byte) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if _, exists := metadata.Metadata["category"]; exists {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			"metadata.category",
			"manifest-layout Agent Skills must not repeat SSB-owned metadata.category",
			"remove metadata.category; the manifest owns category and provenance",
		))
	}
	if strings.TrimSpace(metadata.License) == "" && strings.TrimSpace(metadata.Compatibility) == "" {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			"frontmatter",
			"manifest-layout Agent Skills require a meaningful license or compatibility field",
			"add the applicable SPDX license or host compatibility statement",
		))
	}
	if !bytes.HasPrefix(body, []byte("# ")) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			"body",
			"skill body must begin with an H1 title",
			"begin the portable skill body with # followed by its task title",
		))
	}
	normalized := strings.ReplaceAll(string(body), "\r\n", "\n")
	procedureParts := strings.SplitN(normalized, "\n## Procedure\n", 2)
	if len(procedureParts) != 2 || strings.TrimSpace(procedureParts[1]) == "" {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			"body",
			"skill body must include a nonempty Procedure section",
			"add ## Procedure followed by the actionable workflow",
		))
	}
	return diagnostics
}

func validateNativeMetadataBinding(
	sourcePath string,
	id string,
	category string,
	lenses []Lens,
	scopes []string,
	derivation string,
	evidence []Evidence,
	manifest AcceptedArtifact,
) []Diagnostic {
	if id == manifest.ID && category == manifest.Category &&
		reflect.DeepEqual(lenses, manifest.Lenses) &&
		reflect.DeepEqual(scopes, manifest.Scopes) &&
		derivation == manifest.Derivation &&
		reflect.DeepEqual(evidence, manifest.Evidence) {
		return nil
	}
	return []Diagnostic{diagnostic(
		sourcePath,
		"manifest",
		"native artifact metadata must exactly match its manifest entry",
		"align the native schema fields with the manifest-owned ID, selection metadata, and provenance",
	)}
}

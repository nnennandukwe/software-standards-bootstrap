package rulepack

import (
	"context"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func loadManifestSkill(
	root string,
	manifest AcceptedArtifact,
) (Skill, []Diagnostic, error) {
	data, diagnostics, err := readManifestArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return Skill{}, diagnostics, err
	}
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Skill{}, append(diagnostics, diagnostic(
			manifest.Path,
			"frontmatter",
			splitErr.Error(),
			"add portable Agent Skill YAML frontmatter",
		)), nil
	}
	var metadata skillFrontmatter
	if err := yaml.Load(
		frontmatter,
		&metadata,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return Skill{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"use only Agent Skills core specification fields",
		)), nil
	}
	diagnostics = append(diagnostics, validatePortableManifestSkill(manifest.Path, metadata, body)...)
	if metadata.Name != manifest.ID {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"name",
			fmt.Sprintf("skill name %q must match manifest id %q", metadata.Name, manifest.ID),
			"align the skill name, directory, and manifest entry",
		))
	}
	if len(metadata.Name) > 64 {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"name",
			"skill name must be at most 64 characters",
			"shorten the portable skill name",
		))
	}
	if strings.TrimSpace(metadata.Description) == "" || len(metadata.Description) > 1024 {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"description",
			"skill description must contain 1-1024 characters",
			"describe what the skill does and when to use it",
		))
	}
	return Skill{
		ID:          manifest.ID,
		Description: metadata.Description,
		Category:    manifest.Category,
		SourcePath:  manifest.Path,
		Body:        string(body),
	}, diagnostics, nil
}

func loadActionableSkill(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	reportPath string,
	manifestField string,
	manifest AcceptedArtifact,
) (Skill, []Diagnostic, error) {
	data, diagnostics, err := readEmbeddedArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return Skill{}, diagnostics, err
	}
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Skill{}, append(diagnostics, diagnostic(
			manifest.Path,
			"frontmatter",
			splitErr.Error(),
			"add portable Agent Skill YAML frontmatter",
		)), nil
	}
	var metadata skillFrontmatter
	if err := yaml.Load(
		frontmatter,
		&metadata,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return Skill{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"use only Agent Skills core specification fields",
		)), nil
	}
	diagnostics = append(diagnostics, validateDeprecatedSkillMetadata(manifest.Path, metadata)...)
	skill := Skill{
		ID:          manifest.ID,
		Description: metadata.Description,
		Category:    metadata.Metadata["category"],
		SourcePath:  manifest.Path,
		Body:        string(body),
	}
	if metadata.Name != manifest.ID {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"name",
			fmt.Sprintf("skill name %q must match manifest id %q", metadata.Name, manifest.ID),
			"align the skill name, directory, and report entry",
		))
	}
	if len(metadata.Name) > 64 {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"name",
			"skill name must be at most 64 characters",
			"shorten the portable skill name",
		))
	}
	if strings.TrimSpace(metadata.Description) == "" || len(metadata.Description) > 1024 {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"description",
			"skill description must contain 1-1024 characters",
			"describe what the skill does and when to use it",
		))
	}
	if skill.Category != manifest.Category {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"metadata.category",
			fmt.Sprintf("skill metadata.category %q must match manifest category %q", skill.Category, manifest.Category),
			"align the portable metadata and report manifest",
		))
	}
	if strings.TrimSpace(skill.Body) == "" {
		diagnostics = append(diagnostics, diagnostic(
			manifest.Path,
			"body",
			"skill body is required",
			"document the procedural workflow",
		))
	}
	diagnostics = append(
		diagnostics,
		prefixDiagnosticFields(validateDerivationEvidence(
			ctx,
			evidenceRepo,
			reportPath,
			manifest.Derivation,
			manifest.Evidence,
		), manifestField+".")...,
	)
	return skill, diagnostics, nil
}

// ValidateManifestCandidateSkill validates portable Agent Skill bytes while
// retaining SSB-owned metadata from the existing manifest entry.
func ValidateManifestCandidateSkill(
	relative string,
	manifest AcceptedArtifact,
	data []byte,
) (Skill, []Diagnostic) {
	diagnostics := make([]Diagnostic, 0)
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Skill{}, append(diagnostics, diagnostic(
			relative,
			"frontmatter",
			splitErr.Error(),
			"add portable Agent Skill YAML frontmatter",
		))
	}
	var metadata skillFrontmatter
	if err := yaml.Load(
		frontmatter,
		&metadata,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return Skill{}, append(diagnostics, yamlDiagnostic(relative, err, "use only Agent Skills core specification fields"))
	}
	diagnostics = append(diagnostics, validatePortableManifestSkill(relative, metadata, body)...)
	if manifest.Kind != "skill" || manifest.Path != relative || metadata.Name != manifest.ID {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"manifest",
			"candidate skill name and path must match its manifest entry",
			"keep the existing canonical skill ID and path",
		))
	}
	if len(metadata.Name) > 64 {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"name",
			"skill name must be at most 64 characters",
			"shorten the portable skill name",
		))
	}
	if strings.TrimSpace(metadata.Description) == "" || len(metadata.Description) > 1024 {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"description",
			"skill description must contain 1-1024 characters",
			"describe what the skill does and when to use it",
		))
	}
	return Skill{
		ID:          manifest.ID,
		Description: metadata.Description,
		Category:    manifest.Category,
		SourcePath:  relative,
		Body:        string(body),
	}, diagnostics
}

// ValidateCandidateSkill validates complete portable Agent Skill bytes without
// requiring the candidate to exist in the worktree. SSB provenance remains in
// report.md and is validated after the skill is accepted.
func ValidateCandidateSkill(relative, skillID string, data []byte) (Skill, []Diagnostic) {
	diagnostics := make([]Diagnostic, 0)
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Skill{}, append(diagnostics, diagnostic(
			relative,
			"frontmatter",
			splitErr.Error(),
			"add portable Agent Skill YAML frontmatter",
		))
	}
	var metadata skillFrontmatter
	if err := yaml.Load(
		frontmatter,
		&metadata,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return Skill{}, append(diagnostics, yamlDiagnostic(relative, err, "use only Agent Skills core specification fields"))
	}
	diagnostics = append(diagnostics, validateDeprecatedSkillMetadata(relative, metadata)...)
	if metadata.Name != skillID {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"name",
			fmt.Sprintf("skill name %q must match directory %q", metadata.Name, skillID),
			"align the skill name and directory",
		))
	}
	if len(metadata.Name) > 64 {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"name",
			"skill name must be at most 64 characters",
			"shorten the portable skill name",
		))
	}
	if strings.TrimSpace(metadata.Description) == "" || len(metadata.Description) > 1024 {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"description",
			"skill description must contain 1-1024 characters",
			"describe what the skill does and when to use it",
		))
	}
	category := metadata.Metadata["category"]
	if _, supported := supportedCategories[category]; category == "" || !supported {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"metadata.category",
			fmt.Sprintf("category %q is not supported", category),
			categoryRecovery,
		))
	}
	if strings.TrimSpace(string(body)) == "" {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"body",
			"skill body is required",
			"document the procedural workflow",
		))
	}
	return Skill{
		ID:          skillID,
		Description: metadata.Description,
		Category:    category,
		SourcePath:  relative,
		Body:        string(body),
	}, diagnostics
}

func validateDeprecatedSkillMetadata(relative string, metadata skillFrontmatter) []Diagnostic {
	if _, exists := metadata.Metadata["topic"]; !exists {
		return nil
	}
	return []Diagnostic{diagnostic(
		relative,
		"metadata.topic",
		"metadata.topic is not supported",
		"remove metadata.topic and use metadata.category",
	)}
}

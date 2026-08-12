package rulepack

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

var windowsVolumePattern = regexp.MustCompile(`^[A-Za-z]:`)

type verificationStepV1 struct {
	Run            string `yaml:"run"`
	SourceEvidence string `yaml:"source_evidence"`
	ExpectedResult string `yaml:"expected_result"`
}

type verificationRecipeV1 struct {
	Schema     string               `yaml:"schema"`
	ID         string               `yaml:"id"`
	Title      string               `yaml:"title"`
	Category   string               `yaml:"category"`
	Lenses     []Lens               `yaml:"lenses"`
	Scopes     []string             `yaml:"scopes"`
	Derivation string               `yaml:"derivation"`
	Evidence   []Evidence           `yaml:"evidence"`
	When       string               `yaml:"when"`
	Steps      []verificationStepV1 `yaml:"steps"`
}

type verificationStepV2 struct {
	Run              string `yaml:"run"`
	WorkingDirectory string `yaml:"working_directory"`
	SourceEvidence   string `yaml:"source_evidence"`
	ExpectedResult   string `yaml:"expected_result"`
}

type verificationRecipeV2 struct {
	Schema     string               `yaml:"schema"`
	ID         string               `yaml:"id"`
	Title      string               `yaml:"title"`
	Category   string               `yaml:"category"`
	Lenses     []Lens               `yaml:"lenses"`
	Scopes     []string             `yaml:"scopes"`
	Derivation string               `yaml:"derivation"`
	Evidence   []Evidence           `yaml:"evidence"`
	When       string               `yaml:"when"`
	Steps      []verificationStepV2 `yaml:"steps"`
}

func decodeVerificationRecipe(sourcePath string, data []byte) (VerificationRecipe, []Diagnostic) {
	var version struct {
		Schema string `yaml:"schema"`
	}
	if err := yaml.Load(data, &version, yaml.WithUniqueKeys()); err != nil {
		return VerificationRecipe{SourcePath: sourcePath}, []Diagnostic{yamlDiagnostic(
			sourcePath,
			err,
			"remove duplicate fields and use a supported verification schema",
		)}
	}

	switch version.Schema {
	case VerificationSchemaV1:
		var raw verificationRecipeV1
		if err := yaml.Load(data, &raw, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
			return VerificationRecipe{Schema: version.Schema, SourcePath: sourcePath}, []Diagnostic{yamlDiagnostic(
				sourcePath,
				err,
				"use only fields from the ssb.dev/verification/v1 schema",
			)}
		}
		steps := make([]VerificationStep, len(raw.Steps))
		for index, step := range raw.Steps {
			steps[index] = VerificationStep{
				Run:              step.Run,
				WorkingDirectory: ".",
				SourceEvidence:   step.SourceEvidence,
				ExpectedResult:   step.ExpectedResult,
			}
		}
		return VerificationRecipe{
			Schema: raw.Schema, ID: raw.ID, Title: raw.Title, Category: raw.Category,
			Lenses: raw.Lenses, Scopes: raw.Scopes, Derivation: raw.Derivation,
			Evidence: raw.Evidence, When: raw.When, Steps: steps, SourcePath: sourcePath,
		}, nil
	case VerificationSchemaV2:
		var raw verificationRecipeV2
		if err := yaml.Load(data, &raw, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
			return VerificationRecipe{Schema: version.Schema, SourcePath: sourcePath}, []Diagnostic{yamlDiagnostic(
				sourcePath,
				err,
				"use only fields from the ssb.dev/verification/v2 schema",
			)}
		}
		steps := make([]VerificationStep, len(raw.Steps))
		for index, step := range raw.Steps {
			steps[index] = VerificationStep{
				Run: step.Run, WorkingDirectory: step.WorkingDirectory,
				SourceEvidence: step.SourceEvidence, ExpectedResult: step.ExpectedResult,
			}
		}
		return VerificationRecipe{
			Schema: raw.Schema, ID: raw.ID, Title: raw.Title, Category: raw.Category,
			Lenses: raw.Lenses, Scopes: raw.Scopes, Derivation: raw.Derivation,
			Evidence: raw.Evidence, When: raw.When, Steps: steps, SourcePath: sourcePath,
		}, nil
	default:
		return VerificationRecipe{Schema: version.Schema, SourcePath: sourcePath}, []Diagnostic{diagnostic(
			sourcePath,
			"schema",
			fmt.Sprintf("schema must be %s or %s", VerificationSchemaV1, VerificationSchemaV2),
			"use verification/v1 for an existing recipe or verification/v2 for newly generated recipes",
		)}
	}
}

func validateVerificationRecipe(
	ctx context.Context,
	repo *workspace.Repository,
	recipe VerificationRecipe,
	manifest AcceptedArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(recipe.SourcePath, field, message, recovery))
	}
	if recipe.Schema != VerificationSchemaV1 && recipe.Schema != VerificationSchemaV2 {
		return diagnostics
	}
	if recipe.ID != manifest.ID {
		add("id", fmt.Sprintf("recipe id %q must match manifest id %q", recipe.ID, manifest.ID), "align the recipe id, filename, and manifest entry")
	}
	if strings.TrimSpace(recipe.Title) == "" {
		add("title", "title is required", "add a concise developer-facing title")
	}
	if _, supported := supportedCategories[recipe.Category]; recipe.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", recipe.Category), categoryRecovery)
	}
	diagnostics = append(diagnostics, validateActionableLenses(recipe.SourcePath, "lenses", recipe.Lenses)...)
	diagnostics = append(diagnostics, validateScopes(recipe.SourcePath, recipe.Scopes)...)
	diagnostics = append(diagnostics, validateDerivationEvidence(ctx, repo, recipe.SourcePath, recipe.Derivation, recipe.Evidence)...)
	if strings.TrimSpace(recipe.When) == "" {
		add("when", "when is required", "state the exact handoff context in which the recipe applies")
	}
	enforcesByRef := make(map[string]struct{})
	seenRefs := make(map[string]struct{})
	for index, evidence := range recipe.Evidence {
		field := fmt.Sprintf("evidence[%d].ref", index)
		if !stableIDPattern.MatchString(evidence.Ref) {
			add(field, "recipe evidence ref must be lower-case kebab-case", "give every evidence citation a stable ref")
		}
		if _, duplicate := seenRefs[evidence.Ref]; duplicate {
			add(field, fmt.Sprintf("duplicate evidence ref %q", evidence.Ref), "give every recipe evidence citation a unique ref")
		}
		seenRefs[evidence.Ref] = struct{}{}
		if evidence.Role == "enforces" {
			enforcesByRef[evidence.Ref] = struct{}{}
		}
	}
	if len(recipe.Steps) == 0 {
		add("steps", "verification recipe requires at least one ordered command", "record an existing deliberately invoked command")
	}
	for index, step := range recipe.Steps {
		field := fmt.Sprintf("steps[%d]", index)
		if strings.TrimSpace(step.Run) == "" {
			add(field+".run", "run is required", "record the exact existing repository command")
		}
		if _, exists := enforcesByRef[step.SourceEvidence]; !exists {
			add(field+".source_evidence", fmt.Sprintf("step references missing enforces evidence %q", step.SourceEvidence), "reference an evidence ref whose role is enforces")
		}
		if strings.TrimSpace(step.ExpectedResult) == "" {
			add(field+".expected_result", "expected_result is required", "state the observable successful result")
		}
		if recipe.Schema == VerificationSchemaV2 {
			diagnostics = append(diagnostics, validateWorkingDirectory(ctx, repo, recipe.SourcePath, field+".working_directory", step.WorkingDirectory)...)
		}
	}
	return diagnostics
}

func validateWorkingDirectory(
	ctx context.Context,
	repo *workspace.Repository,
	sourcePath string,
	field string,
	relative string,
) []Diagnostic {
	add := func(message, recovery string) []Diagnostic {
		return []Diagnostic{diagnostic(sourcePath, field, message, recovery)}
	}
	if relative == "" {
		return add("working_directory is required by verification/v2", "use . or a canonical tracked repository directory")
	}
	if relative == "." {
		return nil
	}
	if len([]byte(relative)) > 1024 || path.IsAbs(relative) || path.Clean(relative) != relative ||
		relative == ".." || strings.HasPrefix(relative, "../") ||
		strings.Contains(relative, `\`) || strings.ContainsRune(relative, '\x00') ||
		windowsVolumePattern.MatchString(relative) {
		return add(fmt.Sprintf("unsafe working_directory %q", relative), "use . or a canonical repository-relative path with / separators and no traversal")
	}
	hasSubmodule, err := repo.HasSubmodulePrefix(ctx, relative)
	if err != nil {
		return add(err.Error(), "use . or a canonical tracked directory")
	}
	if hasSubmodule {
		return add(fmt.Sprintf("working_directory %q passes through a submodule", relative), "choose a tracked directory outside submodules")
	}
	entry, exists, err := repo.EntryAtBaseline(ctx, relative)
	if err != nil {
		return add(err.Error(), "use . or a canonical tracked directory")
	}
	if !exists {
		return add(fmt.Sprintf("working_directory %q does not exist at the pinned baseline", relative), "use . or a tracked directory from the pinned baseline")
	}
	if entry.Kind != "tree" || entry.Mode != "040000" {
		return add(fmt.Sprintf("working_directory %q is not a directory at the pinned baseline", relative), "use . or a tracked directory from the pinned baseline")
	}
	return nil
}

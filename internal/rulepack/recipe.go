package rulepack

import (
	"context"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func loadManifestVerificationRecipe(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	manifest AcceptedArtifact,
) (VerificationRecipe, []Diagnostic, error) {
	data, diagnostics, err := readManifestArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return VerificationRecipe{}, diagnostics, err
	}
	recipe, recipeDiagnostics := decodeVerificationRecipe(manifest.Path, data)
	diagnostics = append(diagnostics, recipeDiagnostics...)
	if len(recipeDiagnostics) != 0 {
		return VerificationRecipe{}, diagnostics, nil
	}
	diagnostics = append(diagnostics, validateVerificationRecipe(
		ctx,
		evidenceRepo,
		recipe,
		manifest,
	)...)
	diagnostics = append(diagnostics, validateNativeMetadataBinding(
		manifest.Path,
		recipe.ID,
		recipe.Category,
		recipe.Lenses,
		recipe.Scopes,
		recipe.Derivation,
		recipe.Evidence,
		manifest,
	)...)
	return recipe, diagnostics, nil
}

func loadVerificationRecipe(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	manifest AcceptedArtifact,
) (VerificationRecipe, []Diagnostic, error) {
	data, diagnostics, err := readEmbeddedArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return VerificationRecipe{}, diagnostics, err
	}
	recipe, recipeDiagnostics := decodeVerificationRecipe(manifest.Path, data)
	diagnostics = append(diagnostics, recipeDiagnostics...)
	if len(recipeDiagnostics) != 0 {
		return VerificationRecipe{}, diagnostics, nil
	}
	diagnostics = append(diagnostics, validateVerificationRecipe(
		ctx,
		evidenceRepo,
		recipe,
		manifest,
	)...)
	return recipe, diagnostics, nil
}

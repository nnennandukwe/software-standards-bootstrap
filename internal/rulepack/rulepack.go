// Package rulepack validates the editable, evidence-backed standards pack.
package rulepack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

// Validate parses the current editable pack and verifies all evidence against
// the repository's pinned HEAD commit. Validation never writes files.
func Validate(ctx context.Context, repo *workspace.Repository) (Pack, []Diagnostic, error) {
	return validateActionablePack(ctx, repo, false)
}

// ValidateRetainedPack parses an adopted pack and verifies artifact evidence
// against the historical baseline recorded in report.md. It is intended for
// review-aware post-application rendering and ADR creation; ordinary editable
// pack validation remains pinned to the repository's current HEAD.
func ValidateRetainedPack(ctx context.Context, repo *workspace.Repository) (Pack, []Diagnostic, error) {
	return validateActionablePack(ctx, repo, true)
}

func validateActionablePack(
	ctx context.Context,
	repo *workspace.Repository,
	retained bool,
) (pack Pack, diagnostics []Diagnostic, returnErr error) {
	if err := ctx.Err(); err != nil {
		return Pack{}, nil, err
	}
	defer func() {
		if err := ctx.Err(); err != nil && returnErr == nil {
			pack, diagnostics, returnErr = Pack{}, nil, err
		}
	}()
	manifestPath := filepath.Join(repo.Root(), ".software-standards", "manifest.yaml")
	_, err := os.Lstat(manifestPath)
	switch {
	case err == nil:
		return validateManifestLayoutPack(ctx, repo, retained)
	case errors.Is(err, os.ErrNotExist):
		return validateEmbeddedLayoutPack(ctx, repo, retained)
	default:
		return Pack{}, nil, fmt.Errorf("inspect .software-standards/manifest.yaml: %w", err)
	}
}

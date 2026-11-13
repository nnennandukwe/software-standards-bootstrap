package rulepack

import (
	"context"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func loadManifestAutomationProposal(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	manifest AcceptedArtifact,
) (AutomationProposal, []Diagnostic, error) {
	data, diagnostics, err := readManifestArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return AutomationProposal{}, diagnostics, err
	}
	var proposal AutomationProposal
	if err := yaml.Load(
		data,
		&proposal,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return AutomationProposal{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"use only fields from the ssb.dev/automation/v1 schema",
		)), nil
	}
	proposal.SourcePath = manifest.Path
	diagnostics = append(diagnostics, validateAutomationProposal(
		ctx,
		evidenceRepo,
		proposal,
		manifest,
	)...)
	diagnostics = append(diagnostics, validateNativeMetadataBinding(
		manifest.Path,
		proposal.ID,
		proposal.Category,
		proposal.Lenses,
		proposal.Scopes,
		proposal.Derivation,
		proposal.Evidence,
		manifest,
	)...)
	return proposal, diagnostics, nil
}

func loadAutomationProposal(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	manifest AcceptedArtifact,
) (AutomationProposal, []Diagnostic, error) {
	data, diagnostics, err := readEmbeddedArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return AutomationProposal{}, diagnostics, err
	}
	var proposal AutomationProposal
	if err := yaml.Load(
		data,
		&proposal,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return AutomationProposal{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"use only fields from the ssb.dev/automation/v1 schema",
		)), nil
	}
	proposal.SourcePath = manifest.Path
	diagnostics = append(diagnostics, validateAutomationProposal(
		ctx,
		evidenceRepo,
		proposal,
		manifest,
	)...)
	return proposal, diagnostics, nil
}

func validateAutomationProposal(
	ctx context.Context,
	repo *workspace.Repository,
	proposal AutomationProposal,
	manifest AcceptedArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			proposal.SourcePath,
			field,
			message,
			recovery,
		))
	}
	if proposal.Schema != AutomationSchema {
		add("schema", "schema must be "+AutomationSchema, "update the automation proposal schema value")
	}
	if proposal.ID != manifest.ID {
		add("id", fmt.Sprintf("automation id %q must match manifest id %q", proposal.ID, manifest.ID), "align the automation id, filename, and report entry")
	}
	if strings.TrimSpace(proposal.Title) == "" {
		add("title", "title is required", "add a concise developer-facing title")
	}
	if _, supported := supportedCategories[proposal.Category]; proposal.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", proposal.Category), categoryRecovery)
	}
	diagnostics = append(diagnostics, validateActionableLenses(proposal.SourcePath, "lenses", proposal.Lenses)...)
	diagnostics = append(diagnostics, validateScopes(proposal.SourcePath, proposal.Scopes)...)
	diagnostics = append(diagnostics, validateDerivationEvidence(
		ctx,
		repo,
		proposal.SourcePath,
		proposal.Derivation,
		proposal.Evidence,
	)...)
	required := []struct {
		field string
		value string
	}{
		{"condition", proposal.Condition},
		{"suggested_check", proposal.SuggestedCheck},
		{"trigger", proposal.Trigger},
		{"expected_success", proposal.ExpectedSuccess},
		{"expected_failure", proposal.ExpectedFailure},
	}
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			add(item.field, item.field+" is required", "complete the reviewable automation design")
		}
	}
	return diagnostics
}

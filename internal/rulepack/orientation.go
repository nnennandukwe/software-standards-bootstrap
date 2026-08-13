package rulepack

import (
	"context"
	"fmt"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

const (
	OrientationSchema              = "ssb.dev/orientation/v1"
	OrientationMaxFileBytes  int64 = 1 << 20
	OrientationMaxEntries          = 32
	OrientationMaxEvidence         = 16
	OrientationMaxTextRunes        = 1024
	OrientationMaxLabelRunes       = 160
	OrientationMaxPathBytes        = 1024
	orientationPath                = ".software-standards/orientation.yaml"
)

// Orientation is reviewed repository context for a manifest-layout pack. It
// is not an actionable artifact and is never counted as one.
type Orientation struct {
	Schema             string                    `yaml:"schema" json:"schema"`
	Summary            *OrientationStatement     `yaml:"summary,omitempty" json:"summary,omitempty"`
	Areas              []OrientationArea         `yaml:"areas,omitempty" json:"areas,omitempty"`
	Prerequisites      []OrientationPrerequisite `yaml:"prerequisites,omitempty" json:"prerequisites,omitempty"`
	Documents          []OrientationDocument     `yaml:"documents,omitempty" json:"documents,omitempty"`
	RelatedArtifactIDs []string                  `yaml:"related_artifacts,omitempty" json:"related_artifacts,omitempty"`
	Guidance           []OrientationGuidance     `yaml:"guidance,omitempty" json:"guidance,omitempty"`
}

type OrientationStatement struct {
	Text     string     `yaml:"text" json:"text"`
	Evidence []Evidence `yaml:"evidence" json:"evidence"`
}

type OrientationArea struct {
	Path     string     `yaml:"path" json:"path"`
	Purpose  string     `yaml:"purpose" json:"purpose"`
	Evidence []Evidence `yaml:"evidence" json:"evidence"`
}

type OrientationPrerequisite struct {
	Requirement string     `yaml:"requirement" json:"requirement"`
	Evidence    []Evidence `yaml:"evidence" json:"evidence"`
}

type OrientationDocument struct {
	Label    string     `yaml:"label" json:"label"`
	Path     string     `yaml:"path" json:"path"`
	Evidence []Evidence `yaml:"evidence" json:"evidence"`
}

type OrientationGuidance struct {
	Kind     string     `yaml:"kind" json:"kind"`
	Text     string     `yaml:"text" json:"text"`
	Evidence []Evidence `yaml:"evidence" json:"evidence"`
}

func loadOrientation(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	reference FileReference,
) (*Orientation, []Diagnostic, error) {
	data, diagnostics, err := readDigestBoundFile(root, reference, "orientation.yaml", OrientationMaxFileBytes)
	if err != nil || len(data) == 0 {
		for index := range diagnostics {
			if strings.Contains(diagnostics[index].Message, "does not exist") {
				diagnostics[index].Recovery = "restore the reviewed orientation bytes or remove the manifest reference"
			}
		}
		return nil, diagnostics, err
	}
	var orientation Orientation
	if err := yaml.Load(data, &orientation, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		return nil, append(diagnostics, yamlDiagnostic(
			orientationPath,
			err,
			"remove unknown or duplicate fields and use the ssb.dev/orientation/v1 schema",
		)), nil
	}
	diagnostics = append(diagnostics, validateOrientation(ctx, evidenceRepo, orientationPath, &orientation)...)
	return &orientation, diagnostics, nil
}

func validateOrientation(
	ctx context.Context,
	repo *workspace.Repository,
	sourcePath string,
	orientation *Orientation,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(sourcePath, field, message, recovery))
	}
	if orientation.Schema != OrientationSchema {
		add("schema", "schema must be "+OrientationSchema, "update the orientation schema value")
	}
	if orientation.Summary != nil {
		diagnostics = append(diagnostics, validateOrientationText(sourcePath, "summary.text", orientation.Summary.Text, OrientationMaxTextRunes, "summary")...)
		diagnostics = append(diagnostics, validateOrientationEvidence(ctx, repo, sourcePath, "summary.evidence", orientation.Summary.Evidence)...)
	}

	diagnostics = append(diagnostics, validateOrientationCollectionLimits(sourcePath, orientation)...)

	seenAreas := make(map[string]struct{}, len(orientation.Areas))
	for index, area := range orientation.Areas {
		field := fmt.Sprintf("areas[%d]", index)
		diagnostics = append(diagnostics, validateOrientationPath(sourcePath, field+".path", area.Path)...)
		diagnostics = append(diagnostics, validateOrientationAreaPath(ctx, repo, sourcePath, field+".path", area.Path)...)
		diagnostics = append(diagnostics, validateOrientationText(sourcePath, field+".purpose", area.Purpose, OrientationMaxTextRunes, "purpose")...)
		diagnostics = append(diagnostics, validateOrientationEvidence(ctx, repo, sourcePath, field+".evidence", area.Evidence)...)
		if _, duplicate := seenAreas[area.Path]; duplicate {
			add(field+".path", fmt.Sprintf("duplicate area path %q", area.Path), "list each repository area once")
		}
		seenAreas[area.Path] = struct{}{}
	}

	seenRequirements := make(map[string]struct{}, len(orientation.Prerequisites))
	for index, prerequisite := range orientation.Prerequisites {
		field := fmt.Sprintf("prerequisites[%d]", index)
		diagnostics = append(diagnostics, validateOrientationText(sourcePath, field+".requirement", prerequisite.Requirement, OrientationMaxTextRunes, "requirement")...)
		diagnostics = append(diagnostics, validateOrientationEvidence(ctx, repo, sourcePath, field+".evidence", prerequisite.Evidence)...)
		if _, duplicate := seenRequirements[prerequisite.Requirement]; duplicate {
			add(field+".requirement", fmt.Sprintf("duplicate prerequisite %q", prerequisite.Requirement), "list each prerequisite once")
		}
		seenRequirements[prerequisite.Requirement] = struct{}{}
	}

	seenDocuments := make(map[string]struct{}, len(orientation.Documents))
	for index, document := range orientation.Documents {
		field := fmt.Sprintf("documents[%d]", index)
		diagnostics = append(diagnostics, validateOrientationText(sourcePath, field+".label", document.Label, OrientationMaxLabelRunes, "document label")...)
		diagnostics = append(diagnostics, validateOrientationPath(sourcePath, field+".path", document.Path)...)
		if len(validateOrientationPath(sourcePath, field+".path", document.Path)) == 0 {
			if _, err := inventory.ReadEvidence(ctx, repo, document.Path); err != nil {
				add(field+".path", err.Error(), "choose an eligible tracked regular document from the pinned baseline")
			}
		}
		diagnostics = append(diagnostics, validateOrientationEvidence(ctx, repo, sourcePath, field+".evidence", document.Evidence)...)
		if _, duplicate := seenDocuments[document.Path]; duplicate {
			add(field+".path", fmt.Sprintf("duplicate document path %q", document.Path), "list each canonical document once")
		}
		seenDocuments[document.Path] = struct{}{}
	}

	seenRelationships := make(map[string]struct{}, len(orientation.RelatedArtifactIDs))
	for index, relatedID := range orientation.RelatedArtifactIDs {
		field := fmt.Sprintf("related_artifacts[%d]", index)
		if !stableIDPattern.MatchString(relatedID) {
			add(field, fmt.Sprintf("related artifact id %q must be lower-case kebab-case", relatedID), "use the stable ID of a retained verification recipe or Agent Skill")
		}
		if _, duplicate := seenRelationships[relatedID]; duplicate {
			add(field, fmt.Sprintf("duplicate related artifact %q", relatedID), "list each related artifact once")
		}
		seenRelationships[relatedID] = struct{}{}
	}

	seenGuidance := make(map[string]struct{}, len(orientation.Guidance))
	for index, guidance := range orientation.Guidance {
		field := fmt.Sprintf("guidance[%d]", index)
		switch guidance.Kind {
		case "planning", "implementation", "verification", "handoff":
		default:
			add(field+".kind", fmt.Sprintf("guidance kind %q is not supported", guidance.Kind), "use planning, implementation, verification, or handoff")
		}
		diagnostics = append(diagnostics, validateOrientationText(sourcePath, field+".text", guidance.Text, OrientationMaxTextRunes, "guidance text")...)
		diagnostics = append(diagnostics, validateOrientationEvidence(ctx, repo, sourcePath, field+".evidence", guidance.Evidence)...)
		key := guidance.Kind + "\x00" + guidance.Text
		if _, duplicate := seenGuidance[key]; duplicate {
			add(field, fmt.Sprintf("duplicate %s guidance %q", guidance.Kind, guidance.Text), "list each guidance statement once")
		}
		seenGuidance[key] = struct{}{}
	}
	return diagnostics
}

func validateOrientationCollectionLimits(sourcePath string, orientation *Orientation) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	collections := []struct {
		field string
		count int
	}{
		{field: "areas", count: len(orientation.Areas)},
		{field: "prerequisites", count: len(orientation.Prerequisites)},
		{field: "documents", count: len(orientation.Documents)},
		{field: "related_artifacts", count: len(orientation.RelatedArtifactIDs)},
		{field: "guidance", count: len(orientation.Guidance)},
	}
	for _, collection := range collections {
		if collection.count > OrientationMaxEntries {
			diagnostics = append(diagnostics, diagnostic(
				sourcePath,
				collection.field,
				fmt.Sprintf("%s has %d entries; maximum is %d", collection.field, collection.count, OrientationMaxEntries),
				"reduce the collection to the documented limit",
			))
		}
	}
	return diagnostics
}

func validateOrientationText(sourcePath, field, value string, maxRunes int, label string) []Diagnostic {
	if value == "" || value != strings.TrimSpace(value) {
		return []Diagnostic{diagnostic(sourcePath, field, label+" must be trimmed and nonempty", "write one concise reviewed statement")}
	}
	if !utf8.ValidString(value) {
		return []Diagnostic{diagnostic(sourcePath, field, label+" must be valid UTF-8", "replace invalid text bytes")}
	}
	count := utf8.RuneCountInString(value)
	if count > maxRunes {
		return []Diagnostic{diagnostic(sourcePath, field, fmt.Sprintf("%s has %d Unicode code points; maximum is %d", label, count, maxRunes), "shorten the reviewed statement")}
	}
	for _, character := range value {
		if unicode.IsControl(character) || unicode.In(character, unicode.Cf, unicode.Zl, unicode.Zp) {
			return []Diagnostic{diagnostic(sourcePath, field, label+" must be single-paragraph text without control or format characters", "replace line breaks, bidirectional overrides, and other invisible formatting with ordinary text")}
		}
	}
	return nil
}

func validateOrientationEvidence(
	ctx context.Context,
	repo *workspace.Repository,
	sourcePath string,
	field string,
	evidence []Evidence,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(sourcePath, field+suffix, message, recovery))
	}
	diagnostics = append(diagnostics, validateOrientationEvidenceLimit(sourcePath, field, len(evidence))...)
	seenLocations := make(map[string]struct{}, len(evidence))
	for index, item := range evidence {
		suffix := fmt.Sprintf("[%d]", index)
		if item.Ref != "" {
			add(suffix+".ref", "orientation evidence does not use ref", "remove ref from the orientation citation")
		}
		if item.Role != "declares" && item.Role != "enforces" {
			add(suffix+".role", fmt.Sprintf("orientation evidence role %q is not supported", item.Role), "use declares or enforces")
		}
		diagnostics = append(diagnostics, validateOrientationPath(sourcePath, field+suffix+".path", item.Path)...)
		diagnostics = append(diagnostics, validateEvidence(ctx, repo, sourcePath, field+suffix, item)...)
		location := item.Path + "\x00" + item.Lines
		if _, duplicate := seenLocations[location]; duplicate {
			add("", fmt.Sprintf("duplicate orientation evidence location %s:%s", item.Path, item.Lines), "cite each distinct location once per statement")
		}
		seenLocations[location] = struct{}{}
	}
	return diagnostics
}

func validateOrientationEvidenceLimit(sourcePath, field string, count int) []Diagnostic {
	if count >= 1 && count <= OrientationMaxEvidence {
		return nil
	}
	return []Diagnostic{diagnostic(
		sourcePath,
		field,
		fmt.Sprintf("orientation evidence requires 1-%d citations", OrientationMaxEvidence),
		"cite exact authoritative baseline lines for this statement",
	)}
}

func validateOrientationPath(sourcePath, field, relative string) []Diagnostic {
	if relative == "" || len([]byte(relative)) > OrientationMaxPathBytes || !utf8.ValidString(relative) ||
		path.IsAbs(relative) || path.Clean(relative) != relative || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, "../") || strings.Contains(relative, `\`) ||
		strings.ContainsRune(relative, '\x00') || windowsVolumePattern.MatchString(relative) {
		return []Diagnostic{diagnostic(
			sourcePath,
			field,
			fmt.Sprintf("unsafe or empty repository-relative path %q", relative),
			"use a canonical repository-relative path with / separators and no traversal",
		)}
	}
	return nil
}

func validateOrientationAreaPath(
	ctx context.Context,
	repo *workspace.Repository,
	sourcePath string,
	field string,
	relative string,
) []Diagnostic {
	if len(validateOrientationPath(sourcePath, field, relative)) != 0 {
		return nil
	}
	hasSubmodule, err := repo.HasSubmodulePrefix(ctx, relative)
	if err != nil {
		return []Diagnostic{diagnostic(sourcePath, field, err.Error(), "choose a tracked regular file or repository tree from the pinned baseline")}
	}
	if hasSubmodule {
		return []Diagnostic{diagnostic(sourcePath, field, fmt.Sprintf("area path %q passes through a submodule", relative), "choose a tracked regular file or repository tree outside submodules")}
	}
	entry, exists, err := repo.EntryAtBaseline(ctx, relative)
	if err != nil {
		return []Diagnostic{diagnostic(sourcePath, field, err.Error(), "choose a tracked regular file or repository tree from the pinned baseline")}
	}
	if !exists {
		return []Diagnostic{diagnostic(sourcePath, field, fmt.Sprintf("area path %q does not exist at the pinned baseline", relative), "choose a tracked regular file or repository tree from the pinned baseline")}
	}
	regular := entry.Kind == "blob" && (entry.Mode == "100644" || entry.Mode == "100755")
	if !regular && !(entry.Kind == "tree" && entry.Mode == "040000") {
		return []Diagnostic{diagnostic(sourcePath, field, fmt.Sprintf("area path %q is not a regular file or tree at the pinned baseline", relative), "choose a tracked regular file or repository tree from the pinned baseline")}
	}
	return nil
}

func validateOrientationRelationships(
	sourcePath string,
	orientation *Orientation,
	entriesByID map[string]AcceptedArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for index, relatedID := range orientation.RelatedArtifactIDs {
		artifact, exists := entriesByID[relatedID]
		if !exists || (artifact.Kind != "verification" && artifact.Kind != "skill") {
			diagnostics = append(diagnostics, diagnostic(
				sourcePath,
				fmt.Sprintf("related_artifacts[%d]", index),
				fmt.Sprintf("orientation relationship %q must resolve to a verification recipe or Agent Skill", relatedID),
				"reference a retained verification recipe or Agent Skill, or remove the relationship",
			))
		}
	}
	return diagnostics
}

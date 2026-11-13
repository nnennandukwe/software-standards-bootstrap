package rulepack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func parseHumanRule(sourcePath string, data []byte) (string, string, []Diagnostic) {
	newline := bytes.IndexByte(data, '\n')
	if newline < 0 {
		return "", "", []Diagnostic{diagnostic(
			sourcePath,
			"body",
			"semantic rule must begin with one H1 title",
			"begin with # followed by a concise rule title",
		)}
	}
	titleLine := strings.TrimSuffix(string(data[:newline]), "\r")
	if !strings.HasPrefix(titleLine, "# ") || strings.TrimSpace(strings.TrimPrefix(titleLine, "# ")) == "" {
		return "", "", []Diagnostic{diagnostic(
			sourcePath,
			"body",
			"semantic rule must begin with one H1 title",
			"remove frontmatter and begin with # followed by a concise rule title",
		)}
	}
	title := strings.TrimSpace(strings.TrimPrefix(titleLine, "# "))
	bodyBytes := bodyAfterOpeningHeading(data)
	if strings.TrimSpace(string(bodyBytes)) == "" {
		return title, "", []Diagnostic{diagnostic(
			sourcePath,
			"body",
			"semantic rule actionable text must not be empty",
			"write the actionable obligation immediately after the H1 title",
		)}
	}
	if containsUnicodeFormatCharacter(string(bodyBytes)) {
		return title, string(bodyBytes), []Diagnostic{diagnostic(
			sourcePath,
			"body",
			"semantic rule body must not contain Unicode format characters",
			"remove bidirectional overrides and other invisible format characters",
		)}
	}
	firstContent := true
	normalizedBody := strings.ReplaceAll(string(bodyBytes), "\r\n", "\n")
	normalizedBody = strings.ReplaceAll(normalizedBody, "\r", "\n")
	for _, line := range strings.Split(normalizedBody, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			return title, string(bodyBytes), []Diagnostic{diagnostic(
				sourcePath,
				"body",
				"semantic rule must contain exactly one H1 title",
				"use lower-level headings after the opening title",
			)}
		}
		if firstContent && strings.HasPrefix(trimmed, "#") {
			return title, string(bodyBytes), []Diagnostic{diagnostic(
				sourcePath,
				"body",
				"semantic rule actionable text must immediately follow the H1 title",
				"move headings after the opening actionable obligation",
			)}
		}
		firstContent = false
	}
	return title, string(bodyBytes), nil
}

func loadManifestRule(
	root string,
	manifest AcceptedArtifact,
) (Rule, []Diagnostic, error) {
	data, diagnostics, err := readManifestArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		return Rule{}, diagnostics, err
	}
	title, body, presentationDiagnostics := parseHumanRule(manifest.Path, data)
	diagnostics = append(diagnostics, presentationDiagnostics...)
	return Rule{
		Schema:     RuleSchema,
		ID:         manifest.ID,
		Title:      title,
		Category:   manifest.Category,
		Lenses:     manifest.Lenses,
		Directive:  manifest.Directive,
		Scopes:     manifest.Scopes,
		Derivation: manifest.Derivation,
		Evidence:   manifest.Evidence,
		SourcePath: manifest.Path,
		Body:       body,
	}, diagnostics, nil
}

func loadActionableRule(
	ctx context.Context,
	evidenceRepo *workspace.Repository,
	root string,
	manifest AcceptedArtifact,
) (Rule, []Diagnostic, error) {
	data, diagnostics, err := readEmbeddedArtifact(root, manifest)
	if err != nil || len(data) == 0 {
		if len(data) == 0 && len(diagnostics) != 0 {
			for index := range diagnostics {
				diagnostics[index].Recovery = "remove its manifest entry or restore the artifact at the canonical path"
			}
		}
		return Rule{}, diagnostics, err
	}
	frontmatter, body, splitErr := splitFrontmatter(data)
	if splitErr != nil {
		return Rule{}, append(diagnostics, diagnostic(
			manifest.Path,
			"frontmatter",
			splitErr.Error(),
			"add strict YAML frontmatter between --- markers",
		)), nil
	}
	var source semanticRuleSource
	if err := yaml.Load(
		frontmatter,
		&source,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		return Rule{}, append(diagnostics, yamlDiagnostic(
			manifest.Path,
			err,
			"remove old proof-oriented fields and use the current ssb.dev/rule/v2 schema",
		)), nil
	}
	rule := Rule{
		Schema:     source.Schema,
		ID:         source.ID,
		Title:      source.Title,
		Category:   source.Category,
		Lenses:     source.Lenses,
		Directive:  source.Directive,
		Scopes:     source.Scopes,
		Derivation: source.Derivation,
		Evidence:   source.Evidence,
		SourcePath: manifest.Path,
		Body:       string(body),
	}
	diagnostics = append(diagnostics, validateActionableRule(
		ctx,
		evidenceRepo,
		rule,
		manifest,
	)...)
	return rule, diagnostics, nil
}

func validateActionableRule(
	ctx context.Context,
	repo *workspace.Repository,
	rule Rule,
	manifest AcceptedArtifact,
) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(field, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			rule.SourcePath,
			field,
			message,
			recovery,
		))
	}
	if rule.Schema != RuleSchema {
		add("schema", "schema must be "+RuleSchema, "rewrite this pre-release rule using the current semantic rule contract")
	}
	if rule.ID != manifest.ID {
		add("id", fmt.Sprintf("rule id %q must match manifest id %q", rule.ID, manifest.ID), "align the rule id, filename, and report entry")
	}
	if strings.TrimSpace(rule.Title) == "" {
		add("title", "title is required", "add a concise developer-facing title")
	}
	if _, supported := supportedCategories[rule.Category]; rule.Category == "" || !supported {
		add("category", fmt.Sprintf("category %q is not supported", rule.Category), categoryRecovery)
	}
	if strings.TrimSpace(rule.Body) == "" {
		add("body", "rule body is required", "write the actionable semantic obligation")
	}
	if containsUnicodeFormatCharacter(rule.Body) {
		add("body", "rule body must not contain Unicode format characters", "remove bidirectional overrides and other invisible format characters")
	}
	diagnostics = append(diagnostics, validateActionableLenses(rule.SourcePath, "lenses", rule.Lenses)...)
	if _, supported := supportedDirectives[rule.Directive]; !supported {
		add("directive", fmt.Sprintf("directive %q is not supported", rule.Directive), "use always, ask-first, never, or prefer")
	}
	diagnostics = append(diagnostics, validateScopes(rule.SourcePath, rule.Scopes)...)
	diagnostics = append(diagnostics, validateDerivationEvidence(
		ctx,
		repo,
		rule.SourcePath,
		rule.Derivation,
		rule.Evidence,
	)...)
	return diagnostics
}

func containsUnicodeFormatCharacter(value string) bool {
	for _, character := range value {
		if unicode.In(character, unicode.Cf) {
			return true
		}
	}
	return false
}

func validateActionableLenses(sourcePath, field string, lenses []Lens) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	add := func(suffix, message, recovery string) {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			field+suffix,
			message,
			recovery,
		))
	}
	if len(lenses) == 0 {
		add("", "at least one activation lens is required", "add one base lens or one or more language, framework, and task lenses")
		return diagnostics
	}
	baseCount := 0
	seen := make(map[string]struct{}, len(lenses))
	for index, lens := range lenses {
		suffix := fmt.Sprintf("[%d]", index)
		switch lens.Kind {
		case "base":
			baseCount++
			if lens.Value != "" {
				add(suffix+".value", "base lens must not have a value", "remove the value from the base lens")
			}
		case "language", "framework":
			if !stableIDPattern.MatchString(lens.Value) {
				add(suffix+".value", lens.Kind+" lens requires a lower-case kebab-case value", "add a value such as go, python, cobra, or django")
			}
		case "task":
			switch lens.Value {
			case "planning", "implementation", "verification":
			default:
				add(suffix+".value", fmt.Sprintf("task lens value %q is not supported", lens.Value), "use planning, implementation, or verification")
			}
		default:
			add(suffix+".kind", fmt.Sprintf("lens kind %q is not supported", lens.Kind), "use base, language, framework, or task")
		}
		key := lens.Kind + ":" + lens.Value
		if _, duplicate := seen[key]; duplicate {
			add("", "duplicate activation lens "+key, "list each lens once")
		}
		seen[key] = struct{}{}
	}
	if baseCount != 0 && len(lenses) != 1 {
		add("", "base must be the sole activation lens", "remove contextual lenses or remove the base lens")
	}
	return diagnostics
}

func validateScopes(sourcePath string, scopes []string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	if len(scopes) == 0 {
		diagnostics = append(diagnostics, diagnostic(
			sourcePath,
			"scopes",
			"at least one path scope is required",
			"add one or more repository-relative glob scopes",
		))
	}
	for index, scope := range scopes {
		if unsafeScope(scope) {
			diagnostics = append(diagnostics, diagnostic(
				sourcePath,
				fmt.Sprintf("scopes[%d]", index),
				fmt.Sprintf("unsafe or empty scope %q", scope),
				"use a repository-relative glob without path traversal",
			))
		}
	}
	return diagnostics
}

func parseActionableRuleBytes(relative string, data []byte) (Rule, *Diagnostic) {
	frontmatter, body, err := splitFrontmatter(data)
	if err != nil {
		item := diagnostic(
			relative,
			"frontmatter",
			err.Error(),
			"add strict YAML frontmatter between --- markers",
		)
		return Rule{}, &item
	}
	var source semanticRuleSource
	if err := yaml.Load(
		frontmatter,
		&source,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		item := yamlDiagnostic(
			relative,
			err,
			"remove old proof-oriented fields and use the current ssb.dev/rule/v2 schema",
		)
		return Rule{}, &item
	}
	return Rule{
		Schema:     source.Schema,
		ID:         source.ID,
		Title:      source.Title,
		Category:   source.Category,
		Lenses:     source.Lenses,
		Directive:  source.Directive,
		Scopes:     source.Scopes,
		Derivation: source.Derivation,
		Evidence:   source.Evidence,
		SourcePath: relative,
		Body:       string(body),
	}, nil
}

// ValidateCandidateRule validates complete proposed semantic-rule bytes against
// the current commit without requiring the candidate to exist in the worktree.
func ValidateCandidateRule(
	ctx context.Context,
	repo *workspace.Repository,
	relative string,
	data []byte,
) (Rule, []Diagnostic) {
	rule, parseDiagnostic := parseActionableRuleBytes(relative, data)
	if parseDiagnostic != nil {
		return Rule{}, []Diagnostic{*parseDiagnostic}
	}
	manifestID := strings.TrimSuffix(path.Base(relative), path.Ext(relative))
	manifest := AcceptedArtifact{ID: manifestID, Kind: "rule", Path: relative}
	return rule, validateActionableRule(
		ctx,
		repo,
		rule,
		manifest,
	)
}

// ValidateManifestCandidateRule validates human-first rule bytes while
// retaining immutable semantic metadata from the existing manifest entry.
func ValidateManifestCandidateRule(
	relative string,
	manifest AcceptedArtifact,
	data []byte,
) (Rule, []Diagnostic) {
	title, body, diagnostics := parseHumanRule(relative, data)
	if manifest.Kind != "rule" || manifest.Path != relative {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"manifest",
			"candidate rule does not match its manifest entry",
			"keep the existing canonical rule ID and path",
		))
	}
	return Rule{
		Schema:     RuleSchema,
		ID:         manifest.ID,
		Title:      title,
		Category:   manifest.Category,
		Lenses:     manifest.Lenses,
		Directive:  manifest.Directive,
		Scopes:     manifest.Scopes,
		Derivation: manifest.Derivation,
		Evidence:   manifest.Evidence,
		SourcePath: relative,
		Body:       body,
	}, diagnostics
}

// ValidateRetainedRule validates a proposed update against the historical
// baseline owned by report.md and the rule's existing manifest entry.
func ValidateRetainedRule(
	ctx context.Context,
	repo *workspace.Repository,
	relative string,
	data []byte,
) (Rule, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return Rule{}, nil, err
	}
	rule, parseDiagnostic := parseActionableRuleBytes(relative, data)
	if parseDiagnostic != nil {
		return Rule{}, []Diagnostic{*parseDiagnostic}, nil
	}
	report, diagnostics, err := loadReport(repo)
	if err != nil || len(diagnostics) != 0 {
		return rule, diagnostics, err
	}
	manifest := AcceptedArtifact{}
	for _, candidate := range report.Artifacts {
		if candidate.Kind == "rule" && candidate.Path == relative {
			manifest = candidate
			break
		}
	}
	if manifest.ID == "" {
		return rule, []Diagnostic{diagnostic(
			relative,
			"manifest",
			relative+" is not listed as a rule in .software-standards/report.md",
			"restore its report entry before proposing a retained-rule update",
		)}, nil
	}
	historical, historicalErr := repo.AtCommit(ctx, report.BaselineCommit)
	if historicalErr != nil {
		if !errors.Is(historicalErr, workspace.ErrHistoricalCommit) {
			return rule, nil, historicalErr
		}
		return rule, []Diagnostic{diagnostic(
			".software-standards/report.md",
			"baseline_commit",
			fmt.Sprintf(
				"recorded baseline_commit %q is not a reachable ancestor; retained-rule evidence cannot be verified: %v",
				report.BaselineCommit,
				historicalErr,
			),
			"restore the recorded baseline to current repository history or refresh the accepted pack through review",
		)}, nil
	}
	diagnostics = validateActionableRule(
		ctx,
		historical,
		rule,
		manifest,
	)
	if err := ctx.Err(); err != nil {
		return Rule{}, nil, err
	}
	return rule, diagnostics, nil
}

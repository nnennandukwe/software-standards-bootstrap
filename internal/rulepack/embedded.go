package rulepack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func validateEmbeddedLayoutPack(
	ctx context.Context,
	repo *workspace.Repository,
	retained bool,
) (Pack, []Diagnostic, error) {
	const reportPath = ".software-standards/report.md"
	pack := Pack{
		Layout:         LayoutEmbedded,
		BaselineCommit: repo.Baseline(),
		ReportPath:     reportPath,
		Rules:          make([]Rule, 0),
		Recipes:        make([]VerificationRecipe, 0),
		Skills:         make([]Skill, 0),
		Automations:    make([]AutomationProposal, 0),
	}
	diagnostics := make([]Diagnostic, 0)

	packRootPath := filepath.Join(repo.Root(), ".software-standards")
	info, err := os.Lstat(packRootPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			diagnostics = append(diagnostics, diagnostic(
				".software-standards",
				"file",
				".software-standards does not exist",
				"run ssb inspect --repo PATH --format json, then use the software-standards-bootstrap skill "+
					"to generate the manifest-layout pack (manifest.yaml, inventory.json, and report.md); "+
					"rerun ssb validate --repo PATH",
			))
			return pack, diagnostics, nil
		}
		return Pack{}, nil, fmt.Errorf("inspect .software-standards: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		diagnostics = append(diagnostics, diagnostic(
			".software-standards",
			"file",
			".software-standards must be a real directory, not a symlink",
			"move the editable pack inside the repository",
		))
		return pack, diagnostics, nil
	}

	reportBytes, reportDiagnostics, err := readRequiredRegularFile(
		filepath.Join(repo.Root(), filepath.FromSlash(reportPath)),
		reportPath,
	)
	if err != nil {
		return Pack{}, nil, err
	}
	diagnostics = append(diagnostics, reportDiagnostics...)
	if len(reportBytes) == 0 {
		return pack, diagnostics, nil
	}
	frontmatter, body, splitErr := splitFrontmatter(reportBytes)
	if splitErr != nil {
		diagnostics = append(diagnostics, diagnostic(
			reportPath,
			"frontmatter",
			splitErr.Error(),
			"add strict YAML frontmatter between --- markers",
		))
		return pack, diagnostics, nil
	}
	if err := yaml.Load(
		frontmatter,
		&pack.Report,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		diagnostics = append(diagnostics, yamlDiagnostic(
			reportPath,
			err,
			"remove unknown or duplicate fields and use the ssb.dev/report/v1 schema",
		))
		return pack, diagnostics, nil
	}
	pack.Report.Body = string(body)
	pack.BaselineCommit = pack.Report.BaselineCommit
	pack.Manifest = Manifest{
		Schema:         pack.Report.Schema,
		BaselineCommit: pack.Report.BaselineCommit,
		Artifacts:      pack.Report.Artifacts,
	}
	pack.Inventory = pack.Report.Inventory
	pack.HumanReport = HumanReport{Body: pack.Report.Body}
	diagnostics = append(diagnostics, validateReport(repo, pack.Report, retained)...)
	if strings.TrimSpace(pack.Report.Body) == "" {
		diagnostics = append(diagnostics, diagnostic(
			reportPath,
			"body",
			"report narrative must not be empty",
			"record run-wide limitations and accepted-output summaries",
		))
	}

	evidenceRepo := repo
	if retained {
		historical, historicalErr := repo.AtCommit(ctx, pack.Report.BaselineCommit)
		if historicalErr != nil {
			if !errors.Is(historicalErr, workspace.ErrHistoricalCommit) {
				return Pack{}, nil, historicalErr
			}
			diagnostics = append(diagnostics, diagnostic(
				reportPath,
				"baseline_commit",
				fmt.Sprintf(
					"recorded baseline_commit %q is not a reachable ancestor; artifact evidence cannot be verified: %v",
					pack.Report.BaselineCommit,
					historicalErr,
				),
				"restore the recorded baseline to current repository history or update this pack through a new approved review",
			))
			return pack, diagnostics, nil
		}
		evidenceRepo = historical
	}
	inventoryDiagnostics, inventoryErr := validateReportInventory(
		ctx,
		evidenceRepo,
		pack.Report.Inventory,
		reportPath,
	)
	if inventoryErr != nil {
		return Pack{}, nil, inventoryErr
	}
	diagnostics = append(diagnostics, inventoryDiagnostics...)

	entriesByID := make(map[string]AcceptedArtifact, len(pack.Report.Artifacts))
	entriesByPath := make(map[string]AcceptedArtifact, len(pack.Report.Artifacts))
	for index, artifact := range pack.Report.Artifacts {
		field := fmt.Sprintf("artifacts[%d]", index)
		diagnostics = append(diagnostics, validateEmbeddedArtifact(reportPath, field, artifact)...)
		if prior, exists := entriesByID[artifact.ID]; exists {
			diagnostics = append(diagnostics, diagnostic(
				reportPath,
				field+".id",
				fmt.Sprintf("duplicate artifact id %q also used by %s", artifact.ID, prior.Path),
				"give every accepted artifact a globally unique stable id",
			))
		} else {
			entriesByID[artifact.ID] = artifact
		}
		if prior, exists := entriesByPath[artifact.Path]; exists {
			diagnostics = append(diagnostics, diagnostic(
				reportPath,
				field+".path",
				fmt.Sprintf("duplicate artifact path %q also used by %s", artifact.Path, prior.ID),
				"list each canonical artifact path once",
			))
		} else {
			entriesByPath[artifact.Path] = artifact
		}
	}

	for index, artifact := range pack.Report.Artifacts {
		if canonicalArtifactPath(artifact.Kind, artifact.ID) != artifact.Path {
			continue
		}
		switch artifact.Kind {
		case "rule":
			rule, artifactDiagnostics, loadErr := loadActionableRule(
				ctx,
				evidenceRepo,
				repo.Root(),
				artifact,
			)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if rule.ID != "" {
				pack.Rules = append(pack.Rules, rule)
			}
		case "verification":
			recipe, artifactDiagnostics, loadErr := loadVerificationRecipe(
				ctx,
				evidenceRepo,
				repo.Root(),
				artifact,
			)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if recipe.ID != "" {
				pack.Recipes = append(pack.Recipes, recipe)
			}
		case "skill":
			skill, artifactDiagnostics, loadErr := loadActionableSkill(
				ctx,
				evidenceRepo,
				repo.Root(),
				reportPath,
				fmt.Sprintf("artifacts[%d]", index),
				artifact,
			)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if skill.ID != "" {
				pack.Skills = append(pack.Skills, skill)
			}
		case "automation":
			automation, artifactDiagnostics, loadErr := loadAutomationProposal(
				ctx,
				evidenceRepo,
				repo.Root(),
				artifact,
			)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if automation.ID != "" {
				pack.Automations = append(pack.Automations, automation)
			}
		}
	}

	diagnostics = append(diagnostics, validateRelationships(reportPath, pack.Report.Artifacts, entriesByID)...)

	unlisted, scanDiagnostics, scanErr := unlistedNativeArtifacts(repo.Root(), entriesByPath)
	if scanErr != nil {
		return Pack{}, nil, scanErr
	}
	diagnostics = append(diagnostics, scanDiagnostics...)
	for _, relative := range unlisted {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"file",
			relative+" is not listed in .software-standards/report.md",
			"add the accepted artifact to the report or remove the unaccepted file",
		))
	}
	sort.Slice(pack.Rules, func(i, j int) bool { return pack.Rules[i].ID < pack.Rules[j].ID })
	sort.Slice(pack.Recipes, func(i, j int) bool { return pack.Recipes[i].ID < pack.Recipes[j].ID })
	sort.Slice(pack.Skills, func(i, j int) bool { return pack.Skills[i].ID < pack.Skills[j].ID })
	sort.Slice(pack.Automations, func(i, j int) bool { return pack.Automations[i].ID < pack.Automations[j].ID })
	return pack, diagnostics, nil
}

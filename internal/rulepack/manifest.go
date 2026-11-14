package rulepack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v4"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func validateManifestLayoutPack(
	ctx context.Context,
	repo *workspace.Repository,
	retained bool,
	snapshot *continuationSnapshot,
) (Pack, []Diagnostic, error) {
	const (
		manifestPath  = ".software-standards/manifest.yaml"
		inventoryPath = ".software-standards/inventory.json"
		reportPath    = ".software-standards/report.md"
	)
	pack := Pack{
		Layout:         LayoutManifest,
		BaselineCommit: repo.Baseline(),
		ManifestPath:   manifestPath,
		InventoryPath:  inventoryPath,
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
			return pack, []Diagnostic{diagnostic(
				".software-standards",
				"file",
				".software-standards does not exist",
				"create the manifest-layout pack and rerun ssb validate",
			)}, nil
		}
		return Pack{}, nil, fmt.Errorf("inspect .software-standards: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return pack, []Diagnostic{diagnostic(
			".software-standards",
			"file",
			".software-standards must be a real directory, not a symlink",
			"move the editable pack inside the repository",
		)}, nil
	}

	manifestBytes, manifestDiagnostics, err := readRequiredRegularFileLimited(
		filepath.Join(repo.Root(), filepath.FromSlash(manifestPath)),
		manifestPath,
		inventory.DefaultLimits().MaxFileBytes,
	)
	if err != nil {
		return Pack{}, nil, err
	}
	diagnostics = append(diagnostics, manifestDiagnostics...)
	if len(manifestBytes) == 0 {
		return pack, diagnostics, nil
	}
	if snapshot != nil && snapshot.manifestSHA256 != digest(manifestBytes) {
		return Pack{}, nil, fmt.Errorf("%w: manifest changed during continuation validation; preserve edits and rerun ssb resume", workspace.ErrPrecondition)
	}
	if err := yaml.Load(
		manifestBytes,
		&pack.Manifest,
		yaml.WithKnownFields(),
		yaml.WithUniqueKeys(),
	); err != nil {
		diagnostics = append(diagnostics, yamlDiagnostic(
			manifestPath,
			err,
			"remove unknown or duplicate fields and use the ssb.dev/manifest/v1 schema",
		))
		return pack, diagnostics, nil
	}
	pack.BaselineCommit = pack.Manifest.BaselineCommit
	diagnostics = append(diagnostics, validateManifest(repo, pack.Manifest, retained)...)
	if pack.Manifest.Orientation.Path == orientationPath {
		pack.OrientationPath = orientationPath
	}
	if pack.Manifest.Orientation.IsZero() {
		orientationInfo, orientationErr := os.Lstat(filepath.Join(repo.Root(), filepath.FromSlash(orientationPath)))
		switch {
		case orientationErr == nil:
			message := orientationPath + " is present without a manifest reference"
			recovery := "bind it in the manifest or remove it"
			if orientationInfo.Mode()&os.ModeSymlink != 0 {
				message = orientationPath + " is a symlink present without a manifest reference"
				recovery = "replace it with reviewed regular-file bytes and bind those bytes in the manifest, or remove the symlink"
			} else if !orientationInfo.Mode().IsRegular() {
				message = orientationPath + " is a non-regular entry present without a manifest reference"
				recovery = "replace it with reviewed regular-file bytes and bind those bytes in the manifest, or remove the entry"
			}
			diagnostics = append(diagnostics, diagnostic(
				orientationPath,
				"file",
				message,
				recovery,
			))
		case !errors.Is(orientationErr, os.ErrNotExist):
			return Pack{}, nil, fmt.Errorf("inspect %s: %w", orientationPath, orientationErr)
		}
	}

	var inventoryBytes []byte
	if pack.Manifest.Inventory.Path == inventoryPath {
		inventoryBytes, manifestDiagnostics, err = readDigestBoundFile(
			repo.Root(),
			pack.Manifest.Inventory,
			"inventory.json",
			inventory.DefaultLimits().MaxCandidateBytes,
		)
		if err != nil {
			return Pack{}, nil, err
		}
		diagnostics = append(diagnostics, manifestDiagnostics...)
	}
	if len(inventoryBytes) != 0 {
		if err := decodeStrictJSON(inventoryBytes, &pack.Inventory); err != nil {
			diagnostics = append(diagnostics, diagnostic(
				inventoryPath,
				"inventory",
				err.Error(),
				"copy the complete unedited ssb inspect --format json response",
			))
		}
	}

	var reportBytes []byte
	if pack.Manifest.Report.Path == reportPath {
		reportBytes, manifestDiagnostics, err = readDigestBoundFile(
			repo.Root(),
			pack.Manifest.Report,
			"report.md",
			inventory.DefaultLimits().MaxFileBytes,
		)
		if err != nil {
			return Pack{}, nil, err
		}
		diagnostics = append(diagnostics, manifestDiagnostics...)
	}
	if len(reportBytes) != 0 {
		diagnostics = append(diagnostics, validateHumanReport(reportPath, reportBytes)...)
		pack.HumanReport = HumanReport{Body: string(reportBytes)}
	}

	pack.Report = Report{
		Schema:         pack.Manifest.Schema,
		BaselineCommit: pack.Manifest.BaselineCommit,
		Inventory:      pack.Inventory,
		Artifacts:      pack.Manifest.Artifacts,
		Body:           pack.HumanReport.Body,
	}

	evidenceRepo := repo
	if retained && commitPattern.MatchString(pack.Manifest.BaselineCommit) {
		historical, historicalErr := repo.AtCommit(ctx, pack.Manifest.BaselineCommit)
		if historicalErr != nil {
			if !errors.Is(historicalErr, workspace.ErrHistoricalCommit) {
				return Pack{}, nil, historicalErr
			}
			diagnostics = append(diagnostics, diagnostic(
				manifestPath,
				"baseline_commit",
				fmt.Sprintf(
					"recorded baseline_commit %q is not a reachable ancestor; artifact evidence cannot be verified: %v",
					pack.Manifest.BaselineCommit,
					historicalErr,
				),
				"restore the recorded baseline to current repository history or update this pack through a new approved review",
			))
			return pack, diagnostics, nil
		}
		evidenceRepo = historical
	}
	if len(inventoryBytes) != 0 && pack.Inventory.BaselineCommit != "" {
		diagnostics = append(diagnostics, validateInventoryMetadata(
			inventoryPath,
			pack.Manifest.BaselineCommit,
			pack.Inventory,
		)...)
		if snapshot == nil ||
			snapshot.baselineCommit != evidenceRepo.Baseline() ||
			snapshot.inventorySHA256 != digest(inventoryBytes) {
			inventoryDiagnostics, inventoryErr := validateReportInventory(
				ctx,
				evidenceRepo,
				pack.Inventory,
				inventoryPath,
			)
			if inventoryErr != nil {
				return Pack{}, nil, inventoryErr
			}
			diagnostics = append(diagnostics, inventoryDiagnostics...)
		}
	}
	if pack.Manifest.Orientation.Path == orientationPath {
		orientation, orientationDiagnostics, orientationErr := loadOrientation(
			ctx,
			evidenceRepo,
			repo.Root(),
			pack.Manifest.Orientation,
		)
		if orientationErr != nil {
			return Pack{}, nil, orientationErr
		}
		diagnostics = append(diagnostics, orientationDiagnostics...)
		pack.Orientation = orientation
	}

	entriesByID := make(map[string]AcceptedArtifact, len(pack.Manifest.Artifacts))
	entriesByPath := make(map[string]AcceptedArtifact, len(pack.Manifest.Artifacts))
	for index, artifact := range pack.Manifest.Artifacts {
		field := fmt.Sprintf("artifacts[%d]", index)
		diagnostics = append(diagnostics, validateManifestArtifact(manifestPath, field, artifact)...)
		diagnostics = append(diagnostics, validateManifestOwnedMetadata(
			ctx,
			evidenceRepo,
			manifestPath,
			field,
			artifact,
		)...)
		if prior, exists := entriesByID[artifact.ID]; exists {
			diagnostics = append(diagnostics, diagnostic(
				manifestPath,
				field+".id",
				fmt.Sprintf("duplicate artifact id %q also used by %s", artifact.ID, prior.Path),
				"give every accepted artifact a globally unique stable id",
			))
		} else {
			entriesByID[artifact.ID] = artifact
		}
		if prior, exists := entriesByPath[artifact.Path]; exists {
			diagnostics = append(diagnostics, diagnostic(
				manifestPath,
				field+".path",
				fmt.Sprintf("duplicate artifact path %q also used by %s", artifact.Path, prior.ID),
				"list each canonical artifact path once",
			))
		} else {
			entriesByPath[artifact.Path] = artifact
		}
	}

	for _, artifact := range pack.Manifest.Artifacts {
		if canonicalArtifactPath(artifact.Kind, artifact.ID) != artifact.Path {
			continue
		}
		switch artifact.Kind {
		case "rule":
			rule, artifactDiagnostics, loadErr := loadManifestRule(repo.Root(), artifact)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if rule.ID != "" {
				pack.Rules = append(pack.Rules, rule)
			}
		case "verification":
			recipe, artifactDiagnostics, loadErr := loadManifestVerificationRecipe(
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
			skill, artifactDiagnostics, loadErr := loadManifestSkill(repo.Root(), artifact)
			if loadErr != nil {
				return Pack{}, nil, loadErr
			}
			diagnostics = append(diagnostics, artifactDiagnostics...)
			if skill.ID != "" {
				pack.Skills = append(pack.Skills, skill)
			}
		case "automation":
			automation, artifactDiagnostics, loadErr := loadManifestAutomationProposal(
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

	diagnostics = append(diagnostics, validateRelationships(manifestPath, pack.Manifest.Artifacts, entriesByID)...)
	if pack.Orientation != nil {
		diagnostics = append(diagnostics, validateOrientationRelationships(orientationPath, pack.Orientation, entriesByID)...)
	}
	diagnostics = append(diagnostics, validateRootCore(manifestPath, pack.Manifest.RootCore, entriesByID)...)
	unlisted, scanDiagnostics, scanErr := unlistedNativeArtifacts(ctx, repo.Root(), entriesByPath)
	if scanErr != nil {
		return Pack{}, nil, scanErr
	}
	diagnostics = append(diagnostics, scanDiagnostics...)
	for _, relative := range unlisted {
		diagnostics = append(diagnostics, diagnostic(
			relative,
			"file",
			relative+" is not listed in .software-standards/manifest.yaml",
			"add the accepted artifact to the manifest or remove the unaccepted file",
		))
	}
	sortPackArtifacts(&pack)
	sort.Strings(pack.Manifest.RootCore)
	pack.Routing = buildRoutingCatalog(pack.Manifest.Artifacts)
	return pack, diagnostics, nil
}

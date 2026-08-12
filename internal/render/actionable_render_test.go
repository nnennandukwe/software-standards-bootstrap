package render_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestApplyProjectsRulesRecipesAndSkillsWithoutAutomationOrCommands(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableProjectionPack(ws.Baseline())

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, required := range []string{
		"### Standing orders",
		"Keep public APIs compatible.",
		"- Category: `compatibility`",
		"- Evidence: `README.md:1-1`",
		"### Contextual semantic rules",
		"[Review command changes](.software-standards/rules/review-command-changes.md)",
		"### Verification recipes",
		"[Verify change](.software-standards/verification/verify-change.yaml)",
		"### Agent Skills",
		"[Review change](.agents/skills/review-change/SKILL.md)",
		"description: Review a change using repository evidence.",
		"Related recipe: [Verify change](.software-standards/verification/verify-change.yaml)",
		"Related skill: [Review change](.agents/skills/review-change/SKILL.md)",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("projection missing %q:\n%s", required, content)
		}
	}
	for _, forbidden := range []string{
		"Contextual body must stay canonical.",
		"automate-check",
		"go test ./...",
		"proof",
		"coverage",
		"classification",
		"topic",
	} {
		if strings.Contains(strings.ToLower(content), strings.ToLower(forbidden)) {
			t.Errorf("projection contains forbidden %q:\n%s", forbidden, content)
		}
	}
}

func TestApplyBindsManifestSources(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	embedded := actionableProjectionPack(ws.Baseline())
	embedded.Layout = rulepack.LayoutEmbedded
	embeddedResult, err := render.Apply(ws, embedded, true)
	if err != nil {
		t.Fatal(err)
	}

	manifestLayout := actionableProjectionPack(ws.Baseline())
	manifestLayout.Layout = rulepack.LayoutManifest
	manifestLayout.ManifestPath = ".software-standards/manifest.yaml"
	manifestLayout.InventoryPath = ".software-standards/inventory.json"
	manifestLayout.Manifest = rulepack.Manifest{
		Schema:         rulepack.ManifestSchema,
		BaselineCommit: ws.Baseline(),
		Inventory: rulepack.FileReference{
			Path: ".software-standards/inventory.json", SHA256: "sha256:" + strings.Repeat("1", 64),
		},
		Report: rulepack.FileReference{
			Path: ".software-standards/report.md", SHA256: "sha256:" + strings.Repeat("2", 64),
		},
		Artifacts: manifestLayout.Report.Artifacts,
	}
	manifestResult, err := render.Apply(ws, manifestLayout, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(manifestResult.Content)
	for _, source := range []string{
		"`.software-standards/manifest.yaml`",
		"`.software-standards/inventory.json`",
		"`.software-standards/report.md`",
	} {
		if !strings.Contains(content, source) {
			t.Errorf("manifest-layout projection missing source %s:\n%s", source, content)
		}
	}
	if manifestResult.SourceDigest == embeddedResult.SourceDigest {
		t.Fatal("manifest-layout source digest did not bind manifest file references")
	}
}

func TestApplyPreservesEmbeddedBytes(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableProjectionPack(strings.Repeat("a", 40))
	pack.Layout = rulepack.LayoutEmbedded
	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(result.Content)
	got := "sha256:" + hex.EncodeToString(sum[:])
	const want = "sha256:043364ea803d1689596baa6fb10d76d42b72d0edba63ef4bd7335f5d4c159ae2"
	if got != want {
		t.Fatalf("embedded projection digest = %s, want %s", got, want)
	}
}

func TestApplyDoesNotWriteEmptyOrAutomationOnlyProjection(t *testing.T) {
	tests := []struct {
		name string
		pack func(string) rulepack.Pack
	}{
		{
			name: "empty",
			pack: func(baseline string) rulepack.Pack {
				return rulepack.Pack{BaselineCommit: baseline}
			},
		},
		{
			name: "automation only",
			pack: func(baseline string) rulepack.Pack {
				return rulepack.Pack{
					BaselineCommit: baseline,
					Automations: []rulepack.AutomationProposal{{
						ID:         "automate-check",
						SourcePath: ".software-standards/automation/automate-check.yaml",
					}},
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := committedRepository(t)
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			result, err := render.Apply(ws, test.pack(ws.Baseline()), false)
			if err != nil {
				t.Fatal(err)
			}
			if result.Changed {
				t.Fatalf("non-renderable pack reported a change: %#v", result)
			}
			if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("non-renderable pack wrote AGENTS.md: %v", err)
			}
		})
	}
}

func actionableProjectionPack(baseline string) rulepack.Pack {
	return rulepack.Pack{
		BaselineCommit: baseline,
		ReportPath:     ".software-standards/report.md",
		Report: rulepack.Report{
			Schema:         rulepack.ReportSchema,
			BaselineCommit: baseline,
			Artifacts: []rulepack.AcceptedArtifact{
				{
					ID:                 "keep-public-apis-compatible",
					Kind:               "rule",
					Path:               ".software-standards/rules/keep-public-apis-compatible.md",
					Confidence:         "high",
					Utility:            projectionUtility(80),
					RelatedArtifactIDs: []string{"verify-change", "review-change", "automate-check"},
				},
				{
					ID:         "review-command-changes",
					Kind:       "rule",
					Path:       ".software-standards/rules/review-command-changes.md",
					Confidence: "medium",
					Utility:    projectionUtility(60),
				},
				{
					ID:         "verify-change",
					Kind:       "verification",
					Path:       ".software-standards/verification/verify-change.yaml",
					Confidence: "high",
					Utility:    projectionUtility(70),
				},
				{
					ID:         "review-change",
					Kind:       "skill",
					Path:       ".agents/skills/review-change/SKILL.md",
					Confidence: "medium",
					Utility:    projectionUtility(60),
					Category:   "correctness",
					Lenses:     []rulepack.Lens{{Kind: "task", Value: "verification"}},
					Scopes:     []string{"**/*.go"},
					Derivation: "extracted",
					Evidence: []rulepack.Evidence{{
						Role: "enforces", Path: "Makefile", Lines: "1-2",
					}},
				},
				{
					ID:         "automate-check",
					Kind:       "automation",
					Path:       ".software-standards/automation/automate-check.yaml",
					Confidence: "medium",
					Utility:    projectionUtility(45),
				},
			},
		},
		Rules: []rulepack.Rule{
			{
				Schema: rulepack.RuleSchema, ID: "keep-public-apis-compatible",
				Title: "Keep public APIs compatible", Category: "compatibility",
				Lenses: []rulepack.Lens{{Kind: "base"}}, Directive: "always",
				Scopes: []string{"**/*.go"}, Derivation: "extracted",
				Evidence: []rulepack.Evidence{{
					Role: "declares", Path: "README.md", Lines: "1-1",
				}},
				SourcePath: ".software-standards/rules/keep-public-apis-compatible.md",
				Body:       "Keep public APIs compatible.\n",
			},
			{
				Schema: rulepack.RuleSchema, ID: "review-command-changes",
				Title: "Review command changes", Category: "maintainability",
				Lenses: []rulepack.Lens{{Kind: "task", Value: "verification"}}, Directive: "prefer",
				Scopes: []string{"cmd/**"}, Derivation: "inferred",
				Evidence: []rulepack.Evidence{{
					Role: "demonstrates", Path: "README.md", Lines: "1-1",
				}},
				SourcePath: ".software-standards/rules/review-command-changes.md",
				Body:       "Contextual body must stay canonical.\n",
			},
		},
		Recipes: []rulepack.VerificationRecipe{{
			Schema: rulepack.VerificationSchema, ID: "verify-change",
			Title: "Verify change", Category: "correctness",
			Lenses: []rulepack.Lens{{Kind: "task", Value: "verification"}},
			Scopes: []string{"**/*.go"}, Derivation: "extracted",
			Evidence: []rulepack.Evidence{{
				Ref: "make-verify", Role: "enforces", Path: "Makefile", Lines: "1-2",
			}},
			When: "Before handoff.",
			Steps: []rulepack.VerificationStep{{
				Run: "go test ./...", SourceEvidence: "make-verify",
				ExpectedResult: "The command exits successfully.",
			}},
			SourcePath: ".software-standards/verification/verify-change.yaml",
		}},
		Skills: []rulepack.Skill{{
			ID: "review-change", Description: "Review a change using repository evidence.",
			Category: "correctness", SourcePath: ".agents/skills/review-change/SKILL.md",
		}},
		Automations: []rulepack.AutomationProposal{{
			ID: "automate-check", Title: "Add a checker",
			SourcePath: ".software-standards/automation/automate-check.yaml",
		}},
	}
}

func projectionUtility(total int) rulepack.Utility {
	return rulepack.Utility{Method: rulepack.UtilityMethod, Total: total}
}

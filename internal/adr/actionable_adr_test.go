package adr_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/adr"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestCreateRecordsAdoptableArtifactsAndExcludesAutomation(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableADRPack(ws.Baseline())

	result, err := adr.Create(context.Background(), ws, pack, adr.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, required := range []string{
		"Status: Proposed",
		"Report: `.software-standards/report.md`",
		"## Semantic rules",
		"Keep public APIs compatible.",
		"## Verification recipes",
		"Verify change",
		"## Agent Skills",
		"Review change",
		"Category: `compatibility`",
		"Derivation: `extracted`",
		"Confidence: `high`",
		"Utility: `very-high` (80/100, `ssb-utility-v1`)",
		"Evidence: `README.md:1-1` (`declares`)",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("ADR missing %q:\n%s", required, content)
		}
	}
	for _, forbidden := range []string{"automate-check", "proof gap", "coverage", "classification"} {
		if strings.Contains(strings.ToLower(content), forbidden) {
			t.Errorf("ADR contains forbidden %q:\n%s", forbidden, content)
		}
	}
}

func TestCreateManifestLayoutRecordsManifestInventoryAndReportPaths(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableADRPack(ws.Baseline())
	pack.Layout = rulepack.LayoutManifest
	pack.ManifestPath = ".software-standards/manifest.yaml"
	pack.InventoryPath = ".software-standards/inventory.json"

	result, err := adr.Create(context.Background(), ws, pack, adr.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, required := range []string{
		"Manifest: `.software-standards/manifest.yaml`",
		"Inventory: `.software-standards/inventory.json`",
		"Report: `.software-standards/report.md`",
		"the manifest, inventory, human report, and canonical artifact source files remain editable",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("manifest-layout ADR missing %q:\n%s", required, content)
		}
	}
}

func TestCreateEmbeddedADRBytesRemainStable(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableADRPack(strings.Repeat("a", 40))
	pack.Layout = rulepack.LayoutEmbedded
	result, err := adr.Create(context.Background(), ws, pack, adr.Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(result.Content)
	got := "sha256:" + hex.EncodeToString(sum[:])
	const want = "sha256:7e70dfdddd33860f5269c7395c69f6c7dc9d70cd0e12a6e57a889bef0578b1cd"
	if got != want {
		t.Fatalf("embedded ADR digest = %s, want %s", got, want)
	}
}

func TestCreateFailsSafelyWhenNothingIsAdoptable(t *testing.T) {
	tests := []struct {
		name string
		pack rulepack.Pack
	}{
		{name: "empty", pack: rulepack.Pack{}},
		{name: "automation only", pack: rulepack.Pack{
			Automations: []rulepack.AutomationProposal{{ID: "automate-check"}},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := committedRepository(t)
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			_, err = adr.Create(context.Background(), ws, test.pack, adr.Options{})
			if !errors.Is(err, adr.ErrNoAdoptableArtifacts) {
				t.Fatalf("error = %v, want ErrNoAdoptableArtifacts", err)
			}
			if _, statErr := os.Lstat(filepath.Join(repo, "docs")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("failed ADR created directories: %v", statErr)
			}
		})
	}
}

func actionableADRPack(baseline string) rulepack.Pack {
	utility := func(total int) rulepack.Utility {
		return rulepack.Utility{Method: rulepack.UtilityMethod, Total: total}
	}
	return rulepack.Pack{
		BaselineCommit: baseline,
		ReportPath:     ".software-standards/report.md",
		Report: rulepack.Report{
			Schema:         rulepack.ReportSchema,
			BaselineCommit: baseline,
			Artifacts: []rulepack.AcceptedArtifact{
				{
					ID: "keep-public-apis-compatible", Kind: "rule",
					Path:       ".software-standards/rules/keep-public-apis-compatible.md",
					Confidence: "high", Utility: utility(80),
				},
				{
					ID: "verify-change", Kind: "verification",
					Path:       ".software-standards/verification/verify-change.yaml",
					Confidence: "high", Utility: utility(70),
				},
				{
					ID: "review-change", Kind: "skill",
					Path:       ".agents/skills/review-change/SKILL.md",
					Confidence: "medium", Utility: utility(60),
					Category: "correctness", Lenses: []rulepack.Lens{{Kind: "task", Value: "verification"}},
					Scopes: []string{"**/*.go"}, Derivation: "extracted",
					Evidence: []rulepack.Evidence{{
						Role: "enforces", Path: "Makefile", Lines: "1-2",
					}},
				},
				{
					ID: "automate-check", Kind: "automation",
					Path:       ".software-standards/automation/automate-check.yaml",
					Confidence: "medium", Utility: utility(45),
				},
			},
		},
		Rules: []rulepack.Rule{{
			Schema: rulepack.RuleSchema, ID: "keep-public-apis-compatible",
			Title: "Keep public APIs compatible", Category: "compatibility",
			Lenses: []rulepack.Lens{{Kind: "base"}}, Directive: "always",
			Scopes: []string{"**/*.go"}, Derivation: "extracted",
			Evidence: []rulepack.Evidence{{
				Role: "declares", Path: "README.md", Lines: "1-1",
			}},
			SourcePath: ".software-standards/rules/keep-public-apis-compatible.md",
			Body:       "Keep public APIs compatible.\n",
		}},
		Recipes: []rulepack.VerificationRecipe{{
			ID: "verify-change", Title: "Verify change", Category: "correctness",
			Lenses: []rulepack.Lens{{Kind: "task", Value: "verification"}},
			Scopes: []string{"**/*.go"}, Derivation: "extracted",
			Evidence: []rulepack.Evidence{{
				Ref: "make-verify", Role: "enforces", Path: "Makefile", Lines: "1-2",
			}},
			When: "Before handoff.", SourcePath: ".software-standards/verification/verify-change.yaml",
		}},
		Skills: []rulepack.Skill{{
			ID: "review-change", Description: "Review a change using repository evidence.",
			Category: "correctness", SourcePath: ".agents/skills/review-change/SKILL.md",
			Body: "# Review change\n",
		}},
		Automations: []rulepack.AutomationProposal{{
			ID: "automate-check", Title: "Add a checker",
			SourcePath: ".software-standards/automation/automate-check.yaml",
		}},
	}
}

package render_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/render"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestApplyProjectsActionFirstRulesInertCommandsAndScannableSkills(t *testing.T) {
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
		"This managed section is derived",
		"An unmerged generated change is a proposal",
		"review and merge are the adoption decision",
		"File presence alone does not prove adoption",
		"did not stage, commit, push, open a pull request, execute any displayed recipe command, or activate another system",
		"Recipe presence and expected results are not execution evidence",
		"### How routing works",
		"### Standing orders",
		"Keep public APIs compatible.",
		"- Category: `compatibility`",
		"- Evidence: `README.md:1-1`",
		"### Contextual semantic rules",
		"[Review command changes](.software-standards/rules/review-command-changes.md)",
		"### Verification commands",
		"[Verify change](.software-standards/verification/verify-change.yaml)",
		"go test ./...",
		"printf '```'",
		"touch SHOULD_NOT_EXIST",
		"Working directory: `tools`",
		"Expected result: The command exits successfully.",
		"### Agent Skills",
		"[Review change](.agents/skills/review-change/SKILL.md)",
		"Review a change using repository evidence.",
		"Related recipe: [Verify change](.software-standards/verification/verify-change.yaml)",
		"Related skill: [Review change](.agents/skills/review-change/SKILL.md)",
		"Related rule: [Review command changes](.software-standards/rules/review-command-changes.md)",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("projection missing %q:\n%s", required, content)
		}
	}
	for _, forbidden := range []string{
		"Contextual body must stay canonical.",
		"execute any command",
		"automate-check",
		"coverage",
		"classification",
		"topic",
	} {
		if strings.Contains(strings.ToLower(content), strings.ToLower(forbidden)) {
			t.Errorf("projection contains forbidden %q:\n%s", forbidden, content)
		}
	}
	if _, err := os.Lstat(filepath.Join(repo, "SHOULD_NOT_EXIST")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("render executed a displayed command: %v", err)
	}
	assertOrdered(t, content,
		"## Software Standards Bootstrap",
		"### How routing works",
		"### Standing orders",
		"### Contextual semantic rules",
		"### Verification commands",
		"### Agent Skills",
	)
	assertOrdered(t, content, "go test ./...\nprintf '```'", "touch SHOULD_NOT_EXIST")
	if !strings.Contains(content, "````\ngo test ./...\nprintf '```'\n````") {
		t.Fatalf("command with backticks did not use a safe fence:\n%s", content)
	}
	if strings.Contains(content, "Working directory: `.`") {
		t.Fatalf("root working directory should be omitted:\n%s", content)
	}
	body := strings.Index(content, "Keep public APIs compatible.\n")
	metadata := strings.Index(content, "- Applies to: `**/*.go`")
	if body < 0 || metadata < 0 || body > metadata {
		t.Fatalf("base rule is not action-first:\n%s", content)
	}
}

func TestApplyKeepsOrientationDocumentLinksRepositoryRelative(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableProjectionPack(ws.Baseline())
	pack.Layout = rulepack.LayoutManifest
	pack.Orientation = projectionOrientation()
	pack.Orientation.Documents[0].Path = "javascript:alert(1)"

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	if strings.Contains(content, "](javascript:") ||
		!strings.Contains(content, "](./javascript:alert%281%29)") {
		t.Fatalf("orientation document link is not explicitly repository-relative:\n%s", content)
	}
}

func TestApplyOrdersEveryDirectiveAndUtilityDeterministically(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := testPack(ws.Baseline(),
		"never-rule", "Never body.",
		"ask-rule", "Ask body.",
		"always-low", "Always low body.",
		"always-high", "Always high body.",
		"prefer-rule", "Prefer body.",
	)
	directives := map[string]string{
		"never-rule": "never", "ask-rule": "ask-first", "always-low": "always",
		"always-high": "always", "prefer-rule": "prefer",
	}
	utilities := map[string]int{"always-low": 40, "always-high": 80}
	for index := range pack.Rules {
		pack.Rules[index].Directive = directives[pack.Rules[index].ID]
	}
	for index := range pack.Report.Artifacts {
		if total, exists := utilities[pack.Report.Artifacts[index].ID]; exists {
			pack.Report.Artifacts[index].Utility.Total = total
		}
	}
	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	assertOrdered(t, content, "#### Never", "#### Ask first", "#### Always", "#### Prefer")
	assertOrdered(t, content, "Always high body.", "Always low body.")
}

func TestApplyEscapesSchemaScalarsWithoutChangingCanonicalRuleBodies(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableProjectionPack(ws.Baseline())
	pack.Rules[0].Title = "Keep [public] *APIs* compatible"
	pack.Rules[0].Body = "Raw *canonical* Markdown stays active.\n"
	pack.Recipes[0].When = "Before *handoff* [now]."
	pack.Recipes[0].Steps[0].ExpectedResult = "A [literal] *result*."
	pack.Skills[0].Description = "Use *literal* [routing] text."
	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, escaped := range []string{
		"Keep \\[public\\] \\*APIs\\* compatible",
		"Before \\*handoff\\* \\[now\\].",
		"A \\[literal\\] \\*result\\*.",
		"Use \\*literal\\* \\[routing\\] text.",
	} {
		if !strings.Contains(content, escaped) {
			t.Errorf("projection did not escape %q:\n%s", escaped, content)
		}
	}
	if !strings.Contains(content, "Raw *canonical* Markdown stays active.") {
		t.Fatalf("canonical base-rule Markdown was escaped:\n%s", content)
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

func TestApplyPreservesEmbeddedLayoutBehavior(t *testing.T) {
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
	content := string(result.Content)
	for _, required := range []string{
		"Generated from `.software-standards/report.md`",
		"Edit canonical sources and the report index together",
		"### How routing works",
		"### Standing orders",
		"### Verification commands",
		"go test ./...",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("embedded projection missing %q:\n%s", required, content)
		}
	}
	if strings.Contains(content, "### Repository orientation") {
		t.Fatalf("embedded projection rendered orientation:\n%s", content)
	}
	if strings.Contains(content, "the manifest together") {
		t.Fatalf("embedded projection gave manifest-layout recovery guidance:\n%s", content)
	}
}

func TestApplyProjectsOrientationAndBindsItToSourceIdentity(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableProjectionPack(ws.Baseline())
	pack.Layout = rulepack.LayoutManifest
	pack.ManifestPath = ".software-standards/manifest.yaml"
	pack.InventoryPath = ".software-standards/inventory.json"
	pack.ReportPath = ".software-standards/report.md"
	pack.OrientationPath = ".software-standards/orientation.yaml"
	pack.Manifest = rulepack.Manifest{
		Schema:      rulepack.ManifestSchema,
		Inventory:   rulepack.FileReference{Path: pack.InventoryPath, SHA256: "sha256:" + strings.Repeat("1", 64)},
		Report:      rulepack.FileReference{Path: pack.ReportPath, SHA256: "sha256:" + strings.Repeat("2", 64)},
		Orientation: rulepack.FileReference{Path: pack.OrientationPath, SHA256: "sha256:" + strings.Repeat("3", 64)},
		Artifacts:   pack.Report.Artifacts,
	}
	pack.Orientation = projectionOrientation()

	first, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(first.Content)
	for _, required := range []string{
		"### Repository orientation",
		"A compact \\*reviewed\\* summary.",
		"#### Important areas",
		"`internal/render`",
		"#### Prerequisites",
		"#### Canonical documents",
		"[Contributor guide](CONTRIBUTING.md)",
		"#### Related standards",
		"#### Related standards\n\n- Related recipe:",
		"Related recipe: [Verify change](.software-standards/verification/verify-change.yaml)",
		"#### Task guidance",
		"**Handoff:** Report the result.",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("orientation projection missing %q:\n%s", required, content)
		}
	}
	assertOrdered(t, content, "This managed section is derived", "### Repository orientation", "### How routing works")

	pack.Orientation.Summary.Text = "A changed reviewed summary."
	second, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	if first.SourceDigest == second.SourceDigest || first.ContentDigest == second.ContentDigest {
		t.Fatal("orientation change did not affect source and content digests")
	}
}

func TestApplyRendersNonCommandControlCharactersAsVisibleText(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableProjectionPack(ws.Baseline())
	pack.Recipes[0].Title = "Verify\n### injected heading"
	pack.Recipes[0].When = "Before\n- injected list"
	pack.Recipes[0].Scopes = []string{"tools\n- injected scope"}
	pack.Recipes[0].Steps[0].ExpectedResult = "Success\n### injected result"
	pack.Skills[0].Description = "Review\n### injected skill heading"

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, injected := range []string{
		"\n### injected heading",
		"\n- injected list",
		"\n- injected scope",
		"\n### injected result",
		"\n### injected skill heading",
	} {
		if strings.Contains(content, injected) {
			t.Fatalf("non-command scalar injected Markdown structure %q:\n%s", injected, content)
		}
	}
	for _, visible := range []string{
		`Verify\n\#\#\# injected heading`,
		`Before\n- injected list`,
		`tools\n- injected scope`,
		`Success\n\#\#\# injected result`,
		`Review\n\#\#\# injected skill heading`,
	} {
		if !strings.Contains(content, visible) {
			t.Errorf("projection did not render control characters visibly as %q:\n%s", visible, content)
		}
	}
}

func TestApplyOmitsEmptyOrientationAndRejectsMarkerInjection(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableProjectionPack(ws.Baseline())
	pack.Layout = rulepack.LayoutManifest
	pack.OrientationPath = ".software-standards/orientation.yaml"
	pack.Manifest.Orientation = rulepack.FileReference{Path: pack.OrientationPath, SHA256: "sha256:" + strings.Repeat("3", 64)}
	pack.Orientation = &rulepack.Orientation{Schema: rulepack.OrientationSchema}
	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(result.Content), "### Repository orientation") {
		t.Fatalf("schema-only orientation produced an empty heading:\n%s", result.Content)
	}

	target := filepath.Join(repo, "AGENTS.md")
	before := "# Human guidance\n"
	writeFile(t, target, before)
	pack.Orientation = projectionOrientation()
	pack.Orientation.Summary.Text = "unsafe " + render.StartMarker
	_, err = render.Apply(ws, pack, false)
	if !errors.Is(err, render.ErrMarkers) {
		t.Fatalf("expected marker error, got %v", err)
	}
	after, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != before {
		t.Fatal("marker rejection changed AGENTS.md")
	}
}

func TestApplyRejectsReservedMarkersFromCanonicalInputs(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*rulepack.Pack)
	}{
		{name: "rule title", mutate: func(pack *rulepack.Pack) { pack.Rules[0].Title = render.StartMarker }},
		{name: "rule body", mutate: func(pack *rulepack.Pack) { pack.Rules[0].Body = render.EndMarker }},
		{name: "recipe command", mutate: func(pack *rulepack.Pack) { pack.Recipes[0].Steps[0].Run = render.StartMarker }},
		{name: "skill description", mutate: func(pack *rulepack.Pack) { pack.Skills[0].Description = render.EndMarker }},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := committedRepository(t)
			target := filepath.Join(repo, "AGENTS.md")
			before := "# Human guidance\n"
			writeFile(t, target, before)
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			pack := actionableProjectionPack(ws.Baseline())
			test.mutate(&pack)
			_, err = render.Apply(ws, pack, false)
			if !errors.Is(err, render.ErrMarkers) {
				t.Fatalf("expected marker error, got %v", err)
			}
			after, readErr := os.ReadFile(target)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(after) != before {
				t.Fatal("marker rejection changed AGENTS.md")
			}
		})
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
		{
			name: "orientation only",
			pack: func(baseline string) rulepack.Pack {
				return rulepack.Pack{BaselineCommit: baseline, Orientation: projectionOrientation()}
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

func TestApplyRemovesStaleSectionForNonActionablePacks(t *testing.T) {
	for _, test := range []struct {
		name string
		pack func(string) rulepack.Pack
	}{
		{name: "empty", pack: func(baseline string) rulepack.Pack { return rulepack.Pack{BaselineCommit: baseline} }},
		{name: "orientation only", pack: func(baseline string) rulepack.Pack {
			return rulepack.Pack{BaselineCommit: baseline, Orientation: projectionOrientation()}
		}},
		{name: "automation only", pack: func(baseline string) rulepack.Pack {
			return rulepack.Pack{BaselineCommit: baseline, Automations: []rulepack.AutomationProposal{{ID: "automate-check"}}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := committedRepository(t)
			prefix := "# Human guidance\n\n"
			writeFile(t, filepath.Join(repo, "AGENTS.md"), prefix)
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := render.Apply(ws, actionableProjectionPack(ws.Baseline()), false); err != nil {
				t.Fatal(err)
			}
			result, err := render.Apply(ws, test.pack(ws.Baseline()), false)
			if err != nil {
				t.Fatal(err)
			}
			if !result.Changed {
				t.Fatal("stale managed section was not removed")
			}
			content, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != prefix || strings.Contains(string(content), render.StartMarker) {
				t.Fatalf("stale removal changed human bytes or retained markers: %q", content)
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
					ID:                 "verify-change",
					Kind:               "verification",
					Path:               ".software-standards/verification/verify-change.yaml",
					Confidence:         "high",
					Utility:            projectionUtility(70),
					RelatedArtifactIDs: []string{"review-command-changes", "review-change"},
				},
				{
					ID:                 "review-change",
					Kind:               "skill",
					Path:               ".agents/skills/review-change/SKILL.md",
					Confidence:         "medium",
					Utility:            projectionUtility(60),
					RelatedArtifactIDs: []string{"verify-change"},
					Category:           "correctness",
					Lenses:             []rulepack.Lens{{Kind: "task", Value: "verification"}},
					Scopes:             []string{"**/*.go"},
					Derivation:         "extracted",
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
			Steps: []rulepack.VerificationStep{
				{
					Run: "go test ./...\nprintf '```'", WorkingDirectory: ".", SourceEvidence: "make-verify",
					ExpectedResult: "The command exits successfully.",
				},
				{
					Run: "touch SHOULD_NOT_EXIST", WorkingDirectory: "tools", SourceEvidence: "make-verify",
					ExpectedResult: "The sentinel command would exit successfully if deliberately run.",
				},
			},
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

func projectionOrientation() *rulepack.Orientation {
	evidence := []rulepack.Evidence{{Role: "declares", Path: "README.md", Lines: "1-1"}}
	return &rulepack.Orientation{
		Schema:             rulepack.OrientationSchema,
		Summary:            &rulepack.OrientationStatement{Text: "A compact *reviewed* summary.", Evidence: evidence},
		Areas:              []rulepack.OrientationArea{{Path: "internal/render", Purpose: "Projects validated guidance.", Evidence: evidence}},
		Prerequisites:      []rulepack.OrientationPrerequisite{{Requirement: "Go 1.26.5", Evidence: evidence}},
		Documents:          []rulepack.OrientationDocument{{Label: "Contributor guide", Path: "CONTRIBUTING.md", Evidence: evidence}},
		RelatedArtifactIDs: []string{"verify-change"},
		Guidance:           []rulepack.OrientationGuidance{{Kind: "handoff", Text: "Report the result.", Evidence: evidence}},
	}
}

func assertOrdered(t *testing.T, content string, values ...string) {
	t.Helper()
	position := -1
	for _, value := range values {
		next := strings.Index(content, value)
		if next < 0 || next <= position {
			t.Fatalf("%q is missing or out of order:\n%s", value, content)
		}
		position = next
	}
}

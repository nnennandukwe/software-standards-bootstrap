package render_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
		"report-listed artifacts",
		"did not stage, commit, push, open a pull request, execute any displayed recipe command, or activate another system",
		"Recipe presence and expected results are not execution evidence",
		"### How routing works",
		"**Host-specific:** `AGENTS.md` discovery, directory placement, and nested-file precedence " +
			"depend on the active host; consult that host's documented behavior.",
		"**SSB generator-defined:** Scopes and lenses are SSB routing metadata, " +
			"not native `AGENTS.md` glob activation.",
		"**SSB generator-defined:** Directives mean:",
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
		"accepted artifacts",
		"Directory placement and nearest-file precedence are host-level",
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

func TestApplyProjectsManifestRootAndPortableRoutingWriteSet(t *testing.T) {
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
	pack.Manifest = rulepack.Manifest{
		Schema: rulepack.ManifestSchema, BaselineCommit: ws.Baseline(),
		RootCore: []string{"keep-public-apis-compatible"}, Artifacts: pack.Report.Artifacts,
	}
	pack.Rules[0].Scopes = []string{"**/*"}
	pack.Orientation = projectionOrientation()
	pack.Orientation.Guidance = append(pack.Orientation.Guidance,
		rulepack.OrientationGuidance{Kind: "planning", Text: "Plan the change.", Evidence: pack.Orientation.Guidance[0].Evidence},
		rulepack.OrientationGuidance{Kind: "implementation", Text: "Implement the change.", Evidence: pack.Orientation.Guidance[0].Evidence},
		rulepack.OrientationGuidance{Kind: "verification", Text: "Verify the change.", Evidence: pack.Orientation.Guidance[0].Evidence},
	)
	pack.Routing = &rulepack.RoutingCatalog{
		CatalogPath: rulepack.RoutingCatalogPath,
		Bundles: []rulepack.RoutingBundle{
			{ID: "route-base", Path: rulepack.RoutingBundleDirectory + "/route-base.md", Lenses: []rulepack.Lens{{Kind: "base"}}, Scopes: []string{"**/*"}, ArtifactIDs: []string{"keep-public-apis-compatible"}},
			{ID: "route-command", Path: rulepack.RoutingBundleDirectory + "/route-command.md", Lenses: []rulepack.Lens{{Kind: "task", Value: "verification"}}, Scopes: []string{"cmd/**"}, ArtifactIDs: []string{"review-command-changes"}},
			{ID: "route-verification", Path: rulepack.RoutingBundleDirectory + "/route-verification.md", Lenses: []rulepack.Lens{{Kind: "task", Value: "verification"}}, Scopes: []string{"**/*.go"}, ArtifactIDs: []string{"review-change", "verify-change"}},
		},
	}

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	root := string(result.Content)
	for _, required := range []string{
		"Keep public APIs compatible.",
		"[routing catalog](.software-standards/routing/catalog.md)",
		"Verify the change.",
		"Report the result.",
	} {
		if !strings.Contains(root, required) {
			t.Errorf("finite root missing %q:\n%s", required, root)
		}
	}
	for _, forbidden := range []string{
		"### Contextual semantic rules",
		"### Verification commands",
		"### Agent Skills",
		"Plan the change.",
		"Implement the change.",
		"#### Related standards",
	} {
		if strings.Contains(root, forbidden) {
			t.Errorf("finite root contains routed content %q:\n%s", forbidden, root)
		}
	}
	if result.Routing == nil || result.Routing.Path != ".software-standards/routing" || len(result.Routing.Files) != 4 {
		t.Fatalf("routing result = %#v, want catalog plus three bundles", result.Routing)
	}
	catalog := routingFileContent(t, result.Routing.Files, rulepack.RoutingCatalogPath)
	for _, required := range []string{"route-base", "route-command", "route-verification", "keep-public-apis-compatible", "review-change", "verify-change"} {
		if !strings.Contains(catalog, required) {
			t.Errorf("catalog missing %q:\n%s", required, catalog)
		}
	}
	commandBundle := routingFileContent(t, result.Routing.Files, rulepack.RoutingBundleDirectory+"/route-command.md")
	if !strings.Contains(commandBundle, "Contextual body must stay canonical.") {
		t.Fatalf("rule bundle lacks operational body:\n%s", commandBundle)
	}
	verificationBundle := routingFileContent(t, result.Routing.Files, rulepack.RoutingBundleDirectory+"/route-verification.md")
	for _, required := range []string{"go test ./...", "Expected result:", "[Review change](../../../.agents/skills/review-change/SKILL.md)"} {
		if !strings.Contains(verificationBundle, required) {
			t.Errorf("verification bundle missing %q:\n%s", required, verificationBundle)
		}
	}
	if baseBundle := routingFileContent(t, result.Routing.Files, rulepack.RoutingBundleDirectory+"/route-base.md"); strings.Contains(baseBundle, "Keep public APIs compatible.\n") || !strings.Contains(baseBundle, "projected in root") {
		t.Fatalf("root-core body was duplicated in its bundle:\n%s", baseBundle)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".software-standards", "routing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry run wrote routing files: %v", err)
	}
}

func TestApplyPublishesRoutingTreeAndBlocksDriftWithoutPartialOutput(t *testing.T) {
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
	pack.Manifest = rulepack.Manifest{Schema: rulepack.ManifestSchema, Artifacts: pack.Report.Artifacts}
	attachManifestRouting(&pack)
	if err := os.MkdirAll(filepath.Join(repo, ".software-standards"), 0o755); err != nil {
		t.Fatal(err)
	}

	first, err := render.Apply(ws, pack, false)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || first.Routing == nil || !first.Routing.Changed {
		t.Fatalf("first render did not publish the complete write set: %#v", first)
	}
	for _, file := range first.Routing.Files {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(file.Path)))
		if err != nil {
			t.Fatalf("read generated %s: %v", file.Path, err)
		}
		if string(content) != string(file.Content) {
			t.Fatalf("generated %s differs from planned bytes", file.Path)
		}
	}

	second, err := render.Apply(ws, pack, false)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || second.Routing == nil || second.Routing.Changed {
		t.Fatalf("byte-stable rerender changed output: %#v", second)
	}

	bundlePath := filepath.Join(repo, filepath.FromSlash(first.Routing.Files[1].Path))
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	drifted := bytes.Replace(bundle, []byte("Commands shown below"), []byte("Direct bundle edit"), 1)
	writeFile(t, bundlePath, string(drifted))
	agentsBefore, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	catalogBefore, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rulepack.RoutingCatalogPath)))
	if err != nil {
		t.Fatal(err)
	}

	_, err = render.Apply(ws, pack, false)
	if !errors.Is(err, render.ErrDrift) {
		t.Fatalf("expected routing drift error, got %v", err)
	}
	agentsAfter, _ := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	catalogAfter, _ := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rulepack.RoutingCatalogPath)))
	bundleAfter, _ := os.ReadFile(bundlePath)
	if !bytes.Equal(agentsAfter, agentsBefore) || !bytes.Equal(catalogAfter, catalogBefore) || !bytes.Equal(bundleAfter, drifted) {
		t.Fatal("failed routing render modified part of the write set")
	}
}

func TestBuildReportsNonblockingRootAndCatalogSizeWarnings(t *testing.T) {
	rootPack := testPack(strings.Repeat("a", 40), "large-rule", strings.Repeat("x", render.RootWarningThresholdBytes))
	rootPlan, err := render.Build(rootPack)
	if err != nil {
		t.Fatal(err)
	}
	if rootPlan.Metrics.Root.Bytes <= render.RootWarningThresholdBytes || !rootPlan.Metrics.Root.Exceeded || len(rootPlan.Warnings) != 1 {
		t.Fatalf("root metrics = %#v warnings = %#v", rootPlan.Metrics, rootPlan.Warnings)
	}

	catalogPack := actionableProjectionPack(strings.Repeat("b", 40))
	catalogPack.Layout = rulepack.LayoutManifest
	catalogPack.Manifest = rulepack.Manifest{Schema: rulepack.ManifestSchema}
	artifactIDs := make([]string, 0, 1_200)
	for index := 0; index < 1_200; index++ {
		id := fmt.Sprintf("route-skill-%04d", index)
		path := fmt.Sprintf(".agents/skills/%s/SKILL.md", id)
		artifactIDs = append(artifactIDs, id)
		catalogPack.Manifest.Artifacts = append(catalogPack.Manifest.Artifacts, rulepack.AcceptedArtifact{
			ID: id, Kind: "skill", Path: path, Lenses: []rulepack.Lens{{Kind: "task", Value: "implementation"}}, Scopes: []string{"**/*"},
		})
		catalogPack.Skills = append(catalogPack.Skills, rulepack.Skill{ID: id, Description: "Apply the routed implementation workflow.", SourcePath: path})
	}
	catalogPack.Routing = &rulepack.RoutingCatalog{
		CatalogPath: rulepack.RoutingCatalogPath,
		Bundles: []rulepack.RoutingBundle{{
			ID: "route-large", Path: rulepack.RoutingBundleDirectory + "/route-large.md",
			Lenses: []rulepack.Lens{{Kind: "task", Value: "implementation"}}, Scopes: []string{"**/*"}, ArtifactIDs: artifactIDs,
		}},
	}
	catalogPlan, err := render.Build(catalogPack)
	if err != nil {
		t.Fatal(err)
	}
	if catalogPlan.Metrics.Catalog == nil || catalogPlan.Metrics.Catalog.Bytes <= render.CatalogWarningThresholdBytes ||
		!catalogPlan.Metrics.Catalog.Exceeded || len(catalogPlan.Warnings) != 1 {
		t.Fatalf("catalog metrics = %#v warnings = %#v", catalogPlan.Metrics, catalogPlan.Warnings)
	}
}

func TestBuildKeepsThreeHundredContextualRulesOutOfFiniteRoot(t *testing.T) {
	const contextualCount = 300
	rootRule := rulepack.Rule{
		ID: "preserve-offline-boundary", Title: "Preserve offline boundary", Directive: "never",
		Category: "architecture", Lenses: []rulepack.Lens{{Kind: "base"}}, Scopes: []string{"**/*"},
		SourcePath: ".software-standards/rules/preserve-offline-boundary.md", Body: "Do not add runtime network behavior.\n",
	}
	pack := rulepack.Pack{
		Layout: rulepack.LayoutManifest, BaselineCommit: strings.Repeat("a", 40),
		ManifestPath: ".software-standards/manifest.yaml",
		Manifest:     rulepack.Manifest{RootCore: []string{rootRule.ID}},
		Rules:        []rulepack.Rule{rootRule},
	}
	pack.Manifest.Artifacts = append(pack.Manifest.Artifacts, rulepack.AcceptedArtifact{
		ID: rootRule.ID, Kind: "rule", Path: rootRule.SourcePath,
		Lenses: rootRule.Lenses, Scopes: rootRule.Scopes,
	})
	contextualIDs := make([]string, 0, contextualCount)
	for index := 0; index < contextualCount; index++ {
		id := fmt.Sprintf("contextual-rule-%03d", index)
		sourcePath := ".software-standards/rules/" + id + ".md"
		contextualIDs = append(contextualIDs, id)
		pack.Rules = append(pack.Rules, rulepack.Rule{
			ID: id, Title: fmt.Sprintf("Contextual rule %03d", index), Directive: "always",
			Category: "correctness", Lenses: []rulepack.Lens{{Kind: "language", Value: "go"}},
			Scopes: []string{"internal/**/*.go"}, SourcePath: sourcePath,
			Body: fmt.Sprintf("Apply contextual contract %03d.\n", index),
		})
		pack.Manifest.Artifacts = append(pack.Manifest.Artifacts, rulepack.AcceptedArtifact{
			ID: id, Kind: "rule", Path: sourcePath,
			Lenses: []rulepack.Lens{{Kind: "language", Value: "go"}}, Scopes: []string{"internal/**/*.go"},
		})
	}
	pack.Routing = &rulepack.RoutingCatalog{
		CatalogPath: rulepack.RoutingCatalogPath,
		Bundles: []rulepack.RoutingBundle{
			{ID: "route-root", Path: rulepack.RoutingBundleDirectory + "/route-root.md", Lenses: rootRule.Lenses, Scopes: rootRule.Scopes, ArtifactIDs: []string{rootRule.ID}},
			{ID: "route-contextual", Path: rulepack.RoutingBundleDirectory + "/route-contextual.md", Lenses: []rulepack.Lens{{Kind: "language", Value: "go"}}, Scopes: []string{"internal/**/*.go"}, ArtifactIDs: contextualIDs},
		},
	}

	first, err := render.Build(pack)
	if err != nil {
		t.Fatal(err)
	}
	second, err := render.Build(pack)
	if err != nil {
		t.Fatal(err)
	}
	if first.Metrics.Root.Bytes >= render.RootWarningThresholdBytes || first.Metrics.Root.Exceeded {
		t.Fatalf("finite root = %#v, want below warning threshold", first.Metrics.Root)
	}
	root := string(first.ManagedSection)
	if !strings.Contains(root, rootRule.Body) || strings.Contains(root, "contextual-rule-000") {
		t.Fatalf("root core/contextual split is wrong:\n%s", root)
	}
	bundle := routingFileContent(t, first.Routing.Files, rulepack.RoutingBundleDirectory+"/route-contextual.md")
	for _, id := range contextualIDs {
		if !strings.Contains(bundle, id) {
			t.Fatalf("contextual rule %s is not discoverable in its route bundle", id)
		}
	}
	if !bytes.Equal(first.ManagedSection, second.ManagedSection) || first.Routing.TreeDigest != second.Routing.TreeDigest {
		t.Fatal("identical inputs did not produce an identical root and routing tree")
	}
}

func routingFileContent(t *testing.T, files []render.FileResult, target string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == target {
			return string(file.Content)
		}
	}
	t.Fatalf("routing output %s is missing", target)
	return ""
}

func TestApplyKeepsOrientationDocumentLinksRepositoryRelative(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{name: "scheme", path: "javascript:alert(1)", want: "](./javascript:alert%281%29)"},
		{name: "protocol relative", path: "//host/path", want: "](.///host/path)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := committedRepository(t)
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			pack := actionableProjectionPack(ws.Baseline())
			pack.Layout = rulepack.LayoutManifest
			pack.Orientation = projectionOrientation()
			pack.Orientation.Documents[0].Path = test.path
			attachManifestRouting(&pack)

			result, err := render.Apply(ws, pack, true)
			if err != nil {
				t.Fatal(err)
			}
			content := string(result.Content)
			if !strings.Contains(content, test.want) {
				t.Fatalf("orientation document link is not explicitly repository-relative:\n%s", content)
			}
		})
	}
}

func TestApplyContainsRawRuleHeadingsInsideTheRuleBody(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		{name: "line feed", body: "Keep public APIs compatible.\n\n## Rationale\n\nDownstream consumers pin versions.\n", want: "\n> ## Rationale\n"},
		{name: "carriage return", body: "Keep public APIs compatible.\r## Rationale\rDownstream consumers pin versions.\r", want: "\r> ## Rationale\r"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := committedRepository(t)
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			pack := actionableProjectionPack(ws.Baseline())
			pack.Rules[0].Body = test.body

			result, err := render.Apply(ws, pack, true)
			if err != nil {
				t.Fatal(err)
			}
			content := string(result.Content)
			if strings.Contains(content, "\n## Rationale\n") || strings.Contains(content, "\r## Rationale\r") ||
				!strings.Contains(content, test.want) {
				t.Fatalf("raw rule heading escaped its rule container:\n%s", content)
			}
			assertOrdered(t, content,
				"> Keep public APIs compatible.",
				"> ## Rationale",
				"- Applies to: `**/*.go`",
				"### Contextual semantic rules",
			)
		})
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
	attachManifestRouting(&manifestLayout)
	manifestResult, err := render.Apply(ws, manifestLayout, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(manifestResult.Content)
	for _, source := range []string{
		"`.software-standards/manifest.yaml`",
		"`.software-standards/inventory.json`",
		"`.software-standards/report.md`",
		"manifest-listed artifacts",
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
		"report-listed artifacts",
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
	attachManifestRouting(&pack)

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
		"#### Universal guidance",
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
	pack.Skills[0].Description = "Review\n### injected skill heading \u202e\u2029tail"

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
		"\u202e",
		"\u2029",
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
		`Review\n\#\#\# injected skill heading \u{202E}\u{2029}tail`,
	} {
		if !strings.Contains(content, visible) {
			t.Errorf("projection did not render control characters visibly as %q:\n%s", visible, content)
		}
	}
}

func TestApplyEscapesTildeFencesAndTrimsProjectedProse(t *testing.T) {
	repo := committedRepository(t)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack := actionableProjectionPack(ws.Baseline())
	pack.Layout = rulepack.LayoutManifest
	pack.Orientation = projectionOrientation()
	pack.Orientation.Summary.Text = "~~~ Repository overview"
	pack.Recipes[0].When = "  Before handoff.\n"
	pack.Skills[0].Description = "  ~~~ Review a change.\n"
	attachManifestRouting(&pack)

	result, err := render.Apply(ws, pack, true)
	if err != nil {
		t.Fatal(err)
	}
	content := string(result.Content)
	for _, file := range result.Routing.Files {
		content += string(file.Content)
	}
	for _, forbidden := range []string{"\n~~~ Repository overview\n", "\n  ~~~ Review a change.\\n\n"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("projected prose opened a tilde fence or retained padding %q:\n%s", forbidden, content)
		}
	}
	for _, required := range []string{
		`\~\~\~ Repository overview`,
		`When: Before handoff.`,
		`\~\~\~ Review a change.`,
	} {
		if !strings.Contains(content, required) {
			t.Errorf("projection missing safe trimmed prose %q:\n%s", required, content)
		}
	}
	if strings.Contains(content, `Before handoff.\n`) || strings.Contains(content, `Review a change.\n`) {
		t.Fatalf("projection rendered YAML scalar padding as visible text:\n%s", content)
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
	attachManifestRouting(&pack)
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

func TestApplyRemovesGeneratedRoutingTreeWithEmptyManifestPack(t *testing.T) {
	repo := committedRepository(t)
	if err := os.MkdirAll(filepath.Join(repo, ".software-standards"), 0o755); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	active := actionableProjectionPack(ws.Baseline())
	active.Layout = rulepack.LayoutManifest
	attachManifestRouting(&active)
	if _, err := render.Apply(ws, active, false); err != nil {
		t.Fatal(err)
	}
	routingPath := filepath.Join(repo, filepath.FromSlash(render.RoutingDirectory))
	if _, err := os.Stat(routingPath); err != nil {
		t.Fatalf("initial render did not publish routing tree: %v", err)
	}

	empty := rulepack.Pack{Layout: rulepack.LayoutManifest, BaselineCommit: ws.Baseline()}
	result, err := render.Apply(ws, empty, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Routing == nil || result.Routing.Exists || !result.Routing.Changed {
		t.Fatalf("empty manifest result = %#v, want removed routing tree", result)
	}
	if _, err := os.Lstat(routingPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty manifest left routing tree behind: %v", err)
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

func attachManifestRouting(pack *rulepack.Pack) {
	if len(pack.Manifest.Artifacts) == 0 {
		pack.Manifest.Artifacts = append([]rulepack.AcceptedArtifact(nil), pack.Report.Artifacts...)
	}
	artifactIDs := make([]string, 0, len(pack.Manifest.Artifacts))
	for _, artifact := range pack.Manifest.Artifacts {
		if artifact.Kind != "automation" {
			artifactIDs = append(artifactIDs, artifact.ID)
		}
	}
	pack.Routing = &rulepack.RoutingCatalog{
		CatalogPath: rulepack.RoutingCatalogPath,
		Bundles: []rulepack.RoutingBundle{{
			ID: "route-test", Path: rulepack.RoutingBundleDirectory + "/route-test.md",
			Lenses: []rulepack.Lens{{Kind: "base"}}, Scopes: []string{"**/*"}, ArtifactIDs: artifactIDs,
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

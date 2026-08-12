package rulepack_test

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/inventory"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
	"go.yaml.in/yaml/v4"
)

func TestValidateAcceptsActionableReportAndSemanticRule(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(
		baseline,
		`  - id: keep-public-api-compatible
    kind: rule
    path: .software-standards/rules/keep-public-api-compatible.md
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 80
      factors:
        marginal_value: 25
        risk_reduction: 20
        actionability: 15
        applicability: 10
        earlier_feedback: 10`,
	))
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "keep-public-api-compatible.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: keep-public-api-compatible
title: Keep public APIs compatible
category: compatibility
lenses:
  - kind: language
    value: go
directive: always
scopes:
  - "**/*.go"
derivation: extracted
evidence:
  - role: declares
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
---
Keep public API changes backward compatible.
`, excerptHash("package main\n")))

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if pack.Layout != rulepack.LayoutEmbedded ||
		pack.ManifestPath != "" || pack.InventoryPath != "" ||
		pack.ReportPath != ".software-standards/report.md" ||
		pack.Report.Schema != rulepack.ReportSchema ||
		pack.Manifest.Schema != rulepack.ReportSchema ||
		pack.Inventory.BaselineCommit != baseline ||
		pack.HumanReport.Body != pack.Report.Body ||
		pack.BaselineCommit != baseline {
		t.Fatalf("unexpected report normalization: %#v", pack)
	}
	if len(pack.Rules) != 1 {
		t.Fatalf("rules = %#v, want one", pack.Rules)
	}
	rule := pack.Rules[0]
	if rule.Schema != rulepack.RuleSchema ||
		rule.Category != "compatibility" ||
		rule.Derivation != "extracted" ||
		rule.Evidence[0].Role != "declares" {
		t.Fatalf("unexpected semantic rule: %#v", rule)
	}
	if got := pack.Report.Artifacts[0].Utility.Total; got != 80 {
		t.Fatalf("utility total = %d, want 80", got)
	}
}

func TestValidateAcceptsZeroArtifactReport(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, "  []"))

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(pack.Rules) != 0 || len(pack.Recipes) != 0 ||
		len(pack.Skills) != 0 || len(pack.Automations) != 0 {
		t.Fatalf("zero-artifact report normalized artifacts: %#v", pack)
	}
}

func TestValidateAcceptsManifestRule(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	fixture := writeValidManifestLayoutPack(t, repo, baseline, true)

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if pack.Layout != rulepack.LayoutManifest ||
		pack.ManifestPath != ".software-standards/manifest.yaml" ||
		pack.InventoryPath != ".software-standards/inventory.json" ||
		pack.ReportPath != ".software-standards/report.md" ||
		pack.BaselineCommit != baseline {
		t.Fatalf("unexpected manifest-layout paths: %#v", pack)
	}
	if pack.Manifest.Schema != rulepack.ManifestSchema ||
		pack.Manifest.Inventory.SHA256 != digestBytes(fixture.inventory) ||
		pack.Inventory.BaselineCommit != baseline ||
		pack.HumanReport.Body != string(fixture.report) {
		t.Fatalf("unexpected normalized machine and report data: %#v", pack)
	}
	if len(pack.Rules) != 1 {
		t.Fatalf("rules = %#v, want one", pack.Rules)
	}
	rule := pack.Rules[0]
	if rule.Schema != rulepack.RuleSchema ||
		rule.ID != "keep-public-api-compatible" ||
		rule.Title != "Keep public APIs compatible" ||
		rule.Category != "compatibility" ||
		rule.Directive != "always" ||
		rule.Body != "Keep public API changes backward compatible.\n" {
		t.Fatalf("unexpected normalized manifest-layout rule: %#v", rule)
	}
}

func TestValidateAcceptsEmptyManifest(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline, false)

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if pack.Layout != rulepack.LayoutManifest || len(pack.Manifest.Artifacts) != 0 ||
		len(pack.Rules) != 0 || len(pack.Recipes) != 0 ||
		len(pack.Skills) != 0 || len(pack.Automations) != 0 {
		t.Fatalf("unexpected zero-artifact manifest-layout pack: %#v", pack)
	}
}

func TestValidateKeepsHumanReportSmallFor2239FileInventory(t *testing.T) {
	repo, _ := evidenceRepository(t)
	for index := 0; index < 2_237; index++ {
		writeFile(
			t,
			filepath.Join(repo, "internal", "examples", fmt.Sprintf("example-%04d.go", index)),
			fmt.Sprintf("package examples\n\nconst Example%d = %d\n", index, index),
		)
	}
	git(t, repo, "add", "internal/examples")
	git(t, repo, "commit", "-m", "large inventory fixture")
	baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	fixture := writeValidManifestLayoutPack(t, repo, baseline, false)

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if pack.Inventory.IndexedFiles != 2_239 || len(pack.Inventory.Files) != 2_239 {
		t.Fatalf("indexed inventory = %d/%d, want 2239", pack.Inventory.IndexedFiles, len(pack.Inventory.Files))
	}
	if len(fixture.report) >= 1_024 || len(fixture.inventory) <= 100*len(fixture.report) {
		t.Fatalf("machine inventory leaked into report sizing: inventory=%d report=%d", len(fixture.inventory), len(fixture.report))
	}
	if !bytes.HasPrefix(fixture.report, []byte("# Software standards report\n")) ||
		pack.HumanReport.Body != string(fixture.report) {
		t.Fatalf("human report did not preserve H1-first presentation: %q", pack.HumanReport.Body)
	}
	if bytes.Contains(fixture.report, []byte("internal/examples/example-")) {
		t.Fatal("human report contains inventory rows")
	}
}

func TestValidateAcceptsManifestSkill(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	fixture := writeValidManifestLayoutPack(t, repo, baseline, false)
	skill := []byte(`---
name: review-change
description: Review a Go change using the repository workflow.
license: MIT
---
# Review change

## Procedure

1. Run the repository verification command.
`)
	manifestEntry := fmt.Sprintf(`  - id: review-change
    kind: skill
    path: .agents/skills/review-change/SKILL.md
    sha256: %s
    category: correctness
    lenses:
      - kind: task
        value: verification
    scopes:
      - "**/*.go"
    derivation: extracted
    evidence:
      - role: enforces
        path: Makefile
        lines: 1-2
        excerpt_sha256: %s
    confidence: medium
    utility:
      method: ssb-utility-v1
      total: 60
      factors:
        marginal_value: 15
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 5`, digestBytes(skill), excerptHash("verify:\n\tgo test ./...\n"))
	writeFile(t, fixture.manifestPath, strings.Replace(fixture.manifest, "  []", manifestEntry, 1))
	writeFile(t, filepath.Join(repo, ".agents", "skills", "review-change", "SKILL.md"), string(skill))

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 || len(pack.Skills) != 1 {
		t.Fatalf("manifest-layout skill failed: pack=%#v diagnostics=%#v", pack, diagnostics)
	}
	if pack.Skills[0].Category != "correctness" || strings.Contains(string(skill), "metadata:") {
		t.Fatalf("manifest-layout skill metadata was not manifest-owned: %#v", pack.Skills[0])
	}
}

func TestValidateManifestPresentation(t *testing.T) {
	tests := []struct {
		name       string
		report     []byte
		rule       []byte
		want       string
		wantBody   string
		wantTitle  string
		wantLayout rulepack.Layout
	}{
		{
			name:       "CRLF",
			report:     []byte("# Software standards report\r\n\r\nInventory complete. See [manifest](manifest.yaml) and [inventory](inventory.json).\r\n"),
			rule:       []byte("# Keep public APIs compatible\r\n\r\nKeep public API changes backward compatible.\r\n"),
			wantBody:   "Keep public API changes backward compatible.\r\n",
			wantTitle:  "Keep public APIs compatible",
			wantLayout: rulepack.LayoutManifest,
		},
		{
			name: "second H1",
			rule: []byte("# Keep public APIs compatible\n\nKeep APIs stable.\n\n# Hidden replacement\n\nBreak them.\n"),
			want: "semantic rule must contain exactly one H1 title",
		},
		{
			name: "heading before actionable text",
			rule: []byte("# Keep public APIs compatible\n\n## Details\n\nKeep APIs stable.\n"),
			want: "actionable text must immediately follow the H1 title",
		},
		{
			name: "Unicode format character in body",
			rule: []byte("# Keep public APIs compatible\n\nKeep APIs \u202estable.\n"),
			want: "format characters",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			fixture := writeValidManifestLayoutPack(t, repo, baseline, true)
			if test.report != nil {
				writeManifestReportAndRefreshManifest(t, fixture, test.report)
				fixture.manifest = strings.Replace(fixture.manifest, digestBytes(fixture.report), digestBytes(test.report), 1)
				fixture.report = test.report
			}
			if test.rule != nil {
				writeManifestRuleAndRefreshManifest(t, fixture, test.rule)
			}
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
			if err != nil {
				t.Fatal(err)
			}
			if test.want != "" {
				if !diagnosticsContain(diagnostics, test.want) {
					t.Fatalf("diagnostics = %#v, want %q", diagnostics, test.want)
				}
				return
			}
			if len(diagnostics) != 0 || pack.Layout != test.wantLayout || len(pack.Rules) != 1 ||
				pack.Rules[0].Title != test.wantTitle || pack.Rules[0].Body != test.wantBody {
				t.Fatalf("CRLF normalization failed: pack=%#v diagnostics=%#v", pack, diagnostics)
			}
		})
	}
}

func TestValidateManifestSizeLimits(t *testing.T) {
	tests := []struct {
		name     string
		path     func(manifestLayoutFixture) string
		maxBytes int64
	}{
		{name: "manifest", path: func(f manifestLayoutFixture) string { return f.manifestPath }, maxBytes: 1 << 20},
		{name: "inventory", path: func(f manifestLayoutFixture) string { return f.inventoryPath }, maxBytes: 128 << 20},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			fixture := writeValidManifestLayoutPack(t, repo, baseline, true)
			file, err := os.OpenFile(test.path(fixture), os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Truncate(test.maxBytes + 1); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			_, diagnostics, err := rulepack.Validate(context.Background(), ws)
			if err != nil {
				t.Fatal(err)
			}
			if !diagnosticsContain(diagnostics, fmt.Sprintf("larger than %d bytes", test.maxBytes)) {
				t.Fatalf("diagnostics = %#v", diagnostics)
			}
		})
	}
}

func TestUpdateManifestArtifacts(t *testing.T) {
	input := []byte(`schema: ssb.dev/manifest/v1
baseline_commit: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
inventory:
  path: .software-standards/inventory.json
  sha256: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
report:
  path: .software-standards/report.md
  sha256: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
artifacts:
  - id: keep-rule
    kind: rule
    path: .software-standards/rules/keep-rule.md
    sha256: sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
    category: maintainability
    lenses: [{kind: base}]
    directive: prefer
    scopes: ["**"]
    derivation: extracted
    evidence:
      - role: declares
        path: README.md
        lines: 1-1
        excerpt_sha256: sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 70
      factors: {marginal_value: 20, risk_reduction: 15, actionability: 15, applicability: 10, earlier_feedback: 10}
    related_artifacts: [remove-skill]
  - id: remove-skill
    kind: skill
    path: .agents/skills/remove-skill/SKILL.md
    sha256: sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee
    category: maintainability
    lenses: [{kind: base}]
    scopes: ["**"]
    derivation: extracted
    evidence:
      - role: declares
        path: README.md
        lines: 1-1
        excerpt_sha256: sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd
    confidence: medium
    utility:
      method: ssb-utility-v1
      total: 50
      factors: {marginal_value: 15, risk_reduction: 10, actionability: 10, applicability: 10, earlier_feedback: 5}
`)
	updatedDigest := "sha256:" + strings.Repeat("f", 64)
	result, err := rulepack.UpdateManifestArtifacts(
		input,
		map[string]struct{}{"remove-skill": {}},
		map[string]string{"keep-rule": updatedDigest},
	)
	if err != nil {
		t.Fatal(err)
	}
	var manifest rulepack.Manifest
	if err := yaml.Load(result, &manifest, yaml.WithKnownFields(), yaml.WithUniqueKeys()); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Artifacts) != 1 || manifest.Artifacts[0].ID != "keep-rule" ||
		manifest.Artifacts[0].SHA256 != updatedDigest ||
		len(manifest.Artifacts[0].RelatedArtifactIDs) != 0 ||
		manifest.Artifacts[0].Category != "maintainability" ||
		manifest.Artifacts[0].Directive != "prefer" ||
		manifest.Inventory.SHA256 != "sha256:"+strings.Repeat("a", 64) ||
		manifest.Report.SHA256 != "sha256:"+strings.Repeat("b", 64) {
		t.Fatalf("unexpected manifest mutation: %#v", manifest)
	}
}

func TestValidateManifestNoFallback(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, "  []"))
	writeFile(t, filepath.Join(repo, ".software-standards", "manifest.yaml"), "schema: ssb.dev/manifest/v0\nunknown: true\n")

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Layout != rulepack.LayoutManifest || len(diagnostics) == 0 ||
		!diagnosticsContain(diagnostics, "field unknown not found") {
		t.Fatalf("invalid manifest fell back or produced unclear diagnostics: pack=%#v diagnostics=%#v", pack, diagnostics)
	}
}

func TestValidateRejectsManifestSources(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, repo string, fixture manifestLayoutFixture)
		want   string
	}{
		{
			name: "unknown manifest field",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				writeFile(t, fixture.manifestPath, strings.Replace(fixture.manifest, "schema:", "unknown: true\nschema:", 1))
			},
			want: "field unknown not found",
		},
		{
			name: "duplicate manifest field",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				writeFile(t, fixture.manifestPath, strings.Replace(fixture.manifest, "schema: ssb.dev/manifest/v1", "schema: ssb.dev/manifest/v1\nschema: ssb.dev/manifest/v1", 1))
			},
			want: "mapping key \"schema\" already defined",
		},
		{
			name: "unknown inventory field",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				changed := bytesReplaceOnce(t, fixture.inventory, []byte("{\n"), []byte("{\n  \"unknown\": true,\n"))
				writeManifestInventoryAndRefreshManifest(t, fixture, changed)
			},
			want: "unknown field \"unknown\"",
		},
		{
			name: "duplicate inventory field",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				changed := bytesReplaceOnce(t, fixture.inventory, []byte("  \"schema_version\": 2,"), []byte("  \"schema_version\": 2,\n  \"schema_version\": 2,"))
				writeManifestInventoryAndRefreshManifest(t, fixture, changed)
			},
			want: "duplicate JSON field \"schema_version\"",
		},
		{
			name: "inventory digest mismatch",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				writeFile(t, fixture.inventoryPath, string(fixture.inventory)+" \n")
			},
			want: "inventory.json SHA-256 does not match manifest",
		},
		{
			name: "report digest mismatch",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				writeFile(t, fixture.reportPath, string(fixture.report)+"Changed.\n")
			},
			want: "report.md SHA-256 does not match manifest",
		},
		{
			name: "rule digest mismatch",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				writeFile(t, fixture.rulePath, string(fixture.rule)+"Changed.\n")
			},
			want: "rule SHA-256 does not match manifest",
		},
		{
			name: "report frontmatter",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				changed := append([]byte("---\nunexpected: true\n---\n"), fixture.report...)
				writeManifestReportAndRefreshManifest(t, fixture, changed)
			},
			want: "report.md must begin at byte zero with # Software standards report",
		},
		{
			name: "rule frontmatter",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				changed := append([]byte("---\nschema: ssb.dev/rule/v2\n---\n"), fixture.rule...)
				writeManifestRuleAndRefreshManifest(t, fixture, changed)
			},
			want: "semantic rule must begin with one H1 title",
		},
		{
			name: "escaping inventory path",
			mutate: func(t *testing.T, repo string, fixture manifestLayoutFixture) {
				writeFile(t, fixture.manifestPath, strings.Replace(fixture.manifest, ".software-standards/inventory.json", "../inventory.json", 1))
			},
			want: "inventory path must be .software-standards/inventory.json",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			fixture := writeValidManifestLayoutPack(t, repo, baseline, true)
			test.mutate(t, repo, fixture)
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			_, diagnostics, err := rulepack.Validate(context.Background(), ws)
			if err != nil {
				t.Fatal(err)
			}
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics = %#v, want %q", diagnostics, test.want)
			}
		})
	}
}

func TestValidateRejectsManifestSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not consistently available on Windows")
	}
	tests := []struct {
		name string
		path func(manifestLayoutFixture) string
	}{
		{name: "manifest", path: func(f manifestLayoutFixture) string { return f.manifestPath }},
		{name: "inventory", path: func(f manifestLayoutFixture) string { return f.inventoryPath }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			fixture := writeValidManifestLayoutPack(t, repo, baseline, true)
			target := test.path(fixture)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			other := target + ".real"
			writeFile(t, other, string(data))
			if err := os.Remove(target); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(other, target); err != nil {
				t.Fatal(err)
			}
			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			_, diagnostics, err := rulepack.Validate(context.Background(), ws)
			if err != nil {
				t.Fatal(err)
			}
			if !diagnosticsContain(diagnostics, "must be a real regular file, not a symlink") {
				t.Fatalf("diagnostics = %#v", diagnostics)
			}
		})
	}
}

func TestValidateRejectsOldRuleContractsAndRejectedCandidateMetadata(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string, string) (string, string)
		want   string
	}{
		{
			name: "rule v1",
			mutate: func(report, rule string) (string, string) {
				return report, strings.Replace(rule, "schema: ssb.dev/rule/v2", "schema: ssb.dev/rule/v1", 1)
			},
			want: "schema must be ssb.dev/rule/v2",
		},
		{
			name: "topic field",
			mutate: func(report, rule string) (string, string) {
				return report, strings.Replace(rule, "category: compatibility", "topic: compatibility", 1)
			},
			want: "field topic not found",
		},
		{
			name: "proof-oriented field",
			mutate: func(report, rule string) (string, string) {
				return report, strings.Replace(rule, "derivation: extracted", "classification: guidance\nderivation: extracted", 1)
			},
			want: "field classification not found",
		},
		{
			name: "stale evidence hash",
			mutate: func(report, rule string) (string, string) {
				return report, strings.Replace(
					rule,
					excerptHash("package main\n"),
					"sha256:"+strings.Repeat("0", 64),
					1,
				)
			},
			want: "excerpt hash does not match",
		},
		{
			name: "invalid evidence role",
			mutate: func(report, rule string) (string, string) {
				return report, strings.Replace(rule, "role: declares", "role: proves", 1)
			},
			want: "evidence role \"proves\" is not supported",
		},
		{
			name: "low confidence",
			mutate: func(report, rule string) (string, string) {
				return strings.Replace(report, "confidence: high", "confidence: low", 1), rule
			},
			want: "remove the candidate",
		},
		{
			name: "utility below threshold",
			mutate: func(report, rule string) (string, string) {
				report = strings.Replace(report, "total: 80", "total: 40", 1)
				report = strings.Replace(report, "marginal_value: 25", "marginal_value: 0", 1)
				report = strings.Replace(report, "risk_reduction: 20", "risk_reduction: 5", 1)
				return report, rule
			},
			want: "utility 40 is below",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			report := actionableReport(baseline, validRuleManifestEntry())
			rule := fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: keep-public-api-compatible
title: Keep public APIs compatible
category: compatibility
lenses:
  - kind: language
    value: go
directive: always
scopes:
  - "**/*.go"
derivation: extracted
evidence:
  - role: declares
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
---
Keep public API changes backward compatible.
`, excerptHash("package main\n"))
			report, rule = test.mutate(report, rule)
			writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), report)
			writeFile(t, filepath.Join(repo, ".software-standards", "rules", "keep-public-api-compatible.md"), rule)

			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			_, diagnostics, err := rulepack.Validate(context.Background(), ws)
			if err != nil {
				t.Fatal(err)
			}
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
		})
	}
}

func TestValidateEnforcesDerivationEvidenceThresholds(t *testing.T) {
	tests := []struct {
		name       string
		derivation string
		evidence   string
		want       string
	}{
		{
			name:       "extracted requires authority",
			derivation: "extracted",
			evidence: fmt.Sprintf(`  - role: demonstrates
    path: main.go
    lines: 1-1
    excerpt_sha256: %s`, excerptHash("package main\n")),
			want: "extracted artifacts require at least one declares or enforces citation",
		},
		{
			name:       "inferred requires repeated occurrences",
			derivation: "inferred",
			evidence: fmt.Sprintf(`  - role: demonstrates
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
  - role: demonstrates
    path: Makefile
    lines: 1-1
    excerpt_sha256: %s`, excerptHash("package main\n"), excerptHash("verify:\n")),
			want: "inferred artifacts require three demonstrates citations across at least two files",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, validRuleManifestEntry()))
			writeFile(t, filepath.Join(repo, ".software-standards", "rules", "keep-public-api-compatible.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: keep-public-api-compatible
title: Keep public APIs compatible
category: compatibility
lenses:
  - kind: language
    value: go
directive: always
scopes:
  - "**/*.go"
derivation: %s
evidence:
%s
---
Keep public API changes backward compatible.
`, test.derivation, test.evidence))

			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			_, diagnostics, err := rulepack.Validate(context.Background(), ws)
			if err != nil {
				t.Fatal(err)
			}
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
		})
	}
}

func TestValidateFailsClosedOnManifestAndArtifactDrift(t *testing.T) {
	t.Run("unlisted artifact", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, "  []"))
		writeFile(t, filepath.Join(repo, ".software-standards", "rules", "unlisted.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: unlisted
title: Unlisted rule
category: correctness
lenses:
  - kind: base
directive: always
scopes: ["**/*"]
derivation: extracted
evidence:
  - role: declares
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
---
Do the thing.
`, excerptHash("package main\n")))

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		if !diagnosticsContain(diagnostics, "not listed in .software-standards/report.md") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})

	t.Run("wrong extension", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, "  []"))
		writeFile(t, filepath.Join(repo, ".software-standards", "rules", "notes.txt"), "not a canonical rule\n")

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		if !diagnosticsContain(diagnostics, "not a supported artifact path") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})

	t.Run("nested artifact", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, "  []"))
		writeFile(t, filepath.Join(repo, ".software-standards", "rules", "nested", "unlisted.md"), "not canonical\n")

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		if !diagnosticsContain(diagnostics, "artifact directories must be flat") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})

	t.Run("missing artifact", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, validRuleManifestEntry()))

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		if !diagnosticsContain(diagnostics, "remove its manifest entry or restore the artifact") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})

	t.Run("unsafe noncanonical path", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		entry := strings.Replace(
			validRuleManifestEntry(),
			".software-standards/rules/keep-public-api-compatible.md",
			"../../outside.md",
			1,
		)
		writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, entry))

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		if !diagnosticsContain(diagnostics, "artifact path must be .software-standards/rules/keep-public-api-compatible.md") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})

	t.Run("duplicate global id and path", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		entries := validRuleManifestEntry() + "\n" + validRuleManifestEntry()
		writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, entries))
		writeFile(t, filepath.Join(repo, ".software-standards", "rules", "keep-public-api-compatible.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: keep-public-api-compatible
title: Keep public APIs compatible
category: compatibility
lenses:
  - kind: base
directive: always
scopes: ["**/*.go"]
derivation: extracted
evidence:
  - role: declares
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
---
Keep public API changes backward compatible.
`, excerptHash("package main\n")))

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		if !diagnosticsContain(diagnostics, "duplicate artifact id") ||
			!diagnosticsContain(diagnostics, "duplicate artifact path") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})

	t.Run("symlinked artifact directory", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, validRuleManifestEntry()))
		outside := t.TempDir()
		writeFile(t, filepath.Join(outside, "keep-public-api-compatible.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: keep-public-api-compatible
title: Keep public APIs compatible
category: compatibility
lenses:
  - kind: base
directive: always
scopes: ["**/*.go"]
derivation: extracted
evidence:
  - role: declares
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
---
Keep public API changes backward compatible.
`, excerptHash("package main\n")))
		if err := os.Symlink(outside, filepath.Join(repo, ".software-standards", "rules")); err != nil {
			t.Fatal(err)
		}

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatalf("symlinked artifact path returned internal error: %v", err)
		}
		if !diagnosticsContain(diagnostics, "contains a symlink component") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})
}

func TestValidateRejectsDeprecatedSkillTopicMetadata(t *testing.T) {
	repo, _ := skillPackRepository(
		t,
		"  category: correctness\n  topic: correctness",
		excerptHash("verify:\n\tgo test ./...\n"),
	)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if !diagnosticsContain(diagnostics, "metadata.topic is not supported") {
		t.Fatalf("deprecated skill topic was accepted: %#v", diagnostics)
	}

	_, candidateDiagnostics := rulepack.ValidateCandidateSkill(
		".agents/skills/review-change/SKILL.md",
		"review-change",
		[]byte(`---
name: review-change
description: Review a Go change using the repository workflow.
metadata:
  category: correctness
  topic: correctness
---
# Review change
`),
	)
	if !diagnosticsContain(candidateDiagnostics, "metadata.topic is not supported") {
		t.Fatalf("deprecated candidate skill topic was accepted: %#v", candidateDiagnostics)
	}
}

func TestValidateReportsSkillProvenanceAtManifestField(t *testing.T) {
	repo, _ := skillPackRepository(
		t,
		"  category: correctness",
		"sha256:"+strings.Repeat("0", 64),
	)
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range diagnostics {
		if !strings.Contains(item.Message, "excerpt hash does not match") {
			continue
		}
		if item.Path != ".software-standards/report.md" ||
			item.Field != "artifacts[0].evidence[0]" {
			t.Fatalf("skill provenance diagnostic points away from its owner: %#v", item)
		}
		return
	}
	t.Fatalf("missing skill provenance diagnostic: %#v", diagnostics)
}

func TestValidateRejectsInventoryThatDoesNotMatchBaseline(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	report := actionableReport(baseline, "  []")
	report = strings.Replace(
		report,
		gitBlobOID("package main\n\nfunc main() {}\n"),
		strings.Repeat("0", 40),
		1,
	)
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), report)

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if !diagnosticsContain(diagnostics, "does not exactly match") {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestValidateRejectsNoncanonicalInventoryFileLimit(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	report := strings.Replace(
		actionableReport(baseline, "  []"),
		"max_file_bytes: 1048576",
		"max_file_bytes: 1",
		1,
	)
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), report)

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if !diagnosticsContain(diagnostics, "max_file_bytes must remain 1048576") {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestValidateNormalizesAllFourActionableArtifactKinds(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(
		baseline,
		fmt.Sprintf(`%s
    related_artifacts:
      - verify-change
      - review-change
      - automate-public-api-compatibility
  - id: verify-change
    kind: verification
    path: .software-standards/verification/verify-change.yaml
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 70
      factors:
        marginal_value: 20
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 10
  - id: review-change
    kind: skill
    path: .agents/skills/review-change/SKILL.md
    confidence: medium
    utility:
      method: ssb-utility-v1
      total: 60
      factors:
        marginal_value: 15
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 5
    category: correctness
    lenses:
      - kind: task
        value: verification
    scopes:
      - "**/*.go"
    derivation: extracted
    evidence:
      - role: enforces
        path: Makefile
        lines: 1-2
        excerpt_sha256: %s
  - id: automate-public-api-compatibility
    kind: automation
    path: .software-standards/automation/automate-public-api-compatibility.yaml
    confidence: medium
    utility:
      method: ssb-utility-v1
      total: 45
      factors:
        marginal_value: 10
        risk_reduction: 10
        actionability: 10
        applicability: 10
        earlier_feedback: 5`,
			validRuleManifestEntry(),
			excerptHash("verify:\n\tgo test ./...\n"),
		),
	))
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "keep-public-api-compatible.md"), fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: keep-public-api-compatible
title: Keep public APIs compatible
category: compatibility
lenses:
  - kind: language
    value: go
directive: always
scopes:
  - "**/*.go"
derivation: extracted
evidence:
  - role: declares
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
---
Keep public API changes backward compatible.
`, excerptHash("package main\n")))
	writeFile(t, filepath.Join(repo, ".software-standards", "verification", "verify-change.yaml"), fmt.Sprintf(`schema: ssb.dev/verification/v1
id: verify-change
title: Verify a Go change
category: correctness
lenses:
  - kind: task
    value: verification
scopes:
  - "**/*.go"
derivation: extracted
evidence:
  - ref: make-verify
    role: enforces
    path: Makefile
    lines: 1-2
    excerpt_sha256: %s
when: Before handing off a Go change.
steps:
  - run: go test ./...
    source_evidence: make-verify
    expected_result: The command exits successfully.
`, excerptHash("verify:\n\tgo test ./...\n")))
	writeFile(t, filepath.Join(repo, ".agents", "skills", "review-change", "SKILL.md"), `---
name: review-change
description: Review a Go change using the repository's established workflow.
metadata:
  source: software-standards-bootstrap
  category: correctness
---

# Review change

Inspect the change, run the related verification recipe, and report findings.
`)
	writeFile(t, filepath.Join(repo, ".software-standards", "automation", "automate-public-api-compatibility.yaml"), fmt.Sprintf(`schema: ssb.dev/automation/v1
id: automate-public-api-compatibility
title: Add a public API compatibility check
category: compatibility
lenses:
  - kind: language
    value: go
scopes:
  - "**/*.go"
derivation: inferred
evidence:
  - role: demonstrates
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
  - role: demonstrates
    path: main.go
    lines: 3-3
    excerpt_sha256: %s
  - role: demonstrates
    path: Makefile
    lines: 1-1
    excerpt_sha256: %s
condition: Public API changes preserve compatibility.
suggested_check: Compare exported declarations against the baseline.
trigger: Run when an in-scope Go declaration changes.
expected_success: No incompatible public API change is found.
expected_failure: Report every incompatible declaration and source location.
`, excerptHash("package main\n"), excerptHash("func main() {}\n"), excerptHash("verify:\n")))

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(pack.Rules) != 1 || len(pack.Recipes) != 1 ||
		len(pack.Skills) != 1 || len(pack.Automations) != 1 {
		t.Fatalf("unexpected normalized artifact counts: %#v", pack)
	}
	if pack.Recipes[0].Steps[0].SourceEvidence != "make-verify" ||
		pack.Skills[0].Category != "correctness" ||
		pack.Automations[0].Derivation != "inferred" {
		t.Fatalf("unexpected normalized artifacts: %#v", pack)
	}
}

func TestValidateRejectsRecipeSkillAndRelationshipDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(report, recipe, skill string) (string, string, string)
		want   string
	}{
		{
			name: "recipe evidence reference",
			mutate: func(report, recipe, skill string) (string, string, string) {
				return report, strings.Replace(recipe, "source_evidence: make-verify", "source_evidence: missing", 1), skill
			},
			want: "references missing enforces evidence",
		},
		{
			name: "skill category mismatch",
			mutate: func(report, recipe, skill string) (string, string, string) {
				return report, recipe, strings.Replace(skill, "category: correctness", "category: reliability", 1)
			},
			want: "must match manifest category",
		},
		{
			name: "dangling relationship",
			mutate: func(report, recipe, skill string) (string, string, string) {
				return strings.Replace(report, "related_artifacts:\n      - verify-change", "related_artifacts:\n      - missing", 1), recipe, skill
			},
			want: "references missing related artifact",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			report := actionableReport(baseline, fmt.Sprintf(`%s
    related_artifacts:
      - verify-change
      - review-change
  - id: verify-change
    kind: verification
    path: .software-standards/verification/verify-change.yaml
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 70
      factors:
        marginal_value: 20
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 10
  - id: review-change
    kind: skill
    path: .agents/skills/review-change/SKILL.md
    confidence: medium
    utility:
      method: ssb-utility-v1
      total: 60
      factors:
        marginal_value: 15
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 5
    category: correctness
    lenses:
      - kind: task
        value: verification
    scopes:
      - "**/*.go"
    derivation: extracted
    evidence:
      - role: enforces
        path: Makefile
        lines: 1-2
        excerpt_sha256: %s`, validRuleManifestEntry(), excerptHash("verify:\n\tgo test ./...\n")))
			rule := fmt.Sprintf(`---
schema: ssb.dev/rule/v2
id: keep-public-api-compatible
title: Keep public APIs compatible
category: compatibility
lenses:
  - kind: language
    value: go
directive: always
scopes: ["**/*.go"]
derivation: extracted
evidence:
  - role: declares
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
---
Keep public API changes backward compatible.
`, excerptHash("package main\n"))
			recipe := fmt.Sprintf(`schema: ssb.dev/verification/v1
id: verify-change
title: Verify a Go change
category: correctness
lenses:
  - kind: task
    value: verification
scopes: ["**/*.go"]
derivation: extracted
evidence:
  - ref: make-verify
    role: enforces
    path: Makefile
    lines: 1-2
    excerpt_sha256: %s
when: Before handoff.
steps:
  - run: go test ./...
    source_evidence: make-verify
    expected_result: The command exits successfully.
`, excerptHash("verify:\n\tgo test ./...\n"))
			skill := `---
name: review-change
description: Review a Go change using the repository's workflow.
metadata:
  category: correctness
---
# Review change
`
			report, recipe, skill = test.mutate(report, recipe, skill)
			writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), report)
			writeFile(t, filepath.Join(repo, ".software-standards", "rules", "keep-public-api-compatible.md"), rule)
			writeFile(t, filepath.Join(repo, ".software-standards", "verification", "verify-change.yaml"), recipe)
			writeFile(t, filepath.Join(repo, ".agents", "skills", "review-change", "SKILL.md"), skill)

			ws, err := workspace.Open(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			_, diagnostics, err := rulepack.Validate(context.Background(), ws)
			if err != nil {
				t.Fatal(err)
			}
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
		})
	}
}

func TestValidateAcceptsAutomationOnlyPack(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(baseline, `  - id: automate-check
    kind: automation
    path: .software-standards/automation/automate-check.yaml
    confidence: medium
    utility:
      method: ssb-utility-v1
      total: 45
      factors:
        marginal_value: 10
        risk_reduction: 10
        actionability: 10
        applicability: 10
        earlier_feedback: 5`))
	writeFile(t, filepath.Join(repo, ".software-standards", "automation", "automate-check.yaml"), fmt.Sprintf(`schema: ssb.dev/automation/v1
id: automate-check
title: Add an automatic check
category: correctness
lenses:
  - kind: task
    value: implementation
scopes: ["**/*.go"]
derivation: inferred
evidence:
  - role: demonstrates
    path: main.go
    lines: 1-1
    excerpt_sha256: %s
  - role: demonstrates
    path: main.go
    lines: 3-3
    excerpt_sha256: %s
  - role: demonstrates
    path: Makefile
    lines: 1-1
    excerpt_sha256: %s
condition: Changes satisfy the inferred condition.
suggested_check: Compare changes against the inferred condition.
trigger: Run when in-scope files change.
expected_success: The condition holds.
expected_failure: Report the violating source locations.
`, excerptHash("package main\n"), excerptHash("func main() {}\n"), excerptHash("verify:\n")))

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 || len(pack.Automations) != 1 {
		t.Fatalf("automation-only pack failed: pack=%#v diagnostics=%#v", pack, diagnostics)
	}
}

func actionableReport(baseline, artifactEntries string) string {
	return fmt.Sprintf(`---
schema: ssb.dev/report/v1
baseline_commit: %s
inventory:
  schema_version: 2
  inventory_version: ssb-inventory-v2
  baseline_commit: %s
  limits:
    max_candidate_files: 40000
    max_candidate_bytes: 134217728
    max_file_bytes: 1048576
  candidate_files: 2
  candidate_bytes: 52
  scanned_files: 2
  scanned_bytes: 52
  indexed_files: 2
  indexed_bytes: 52
  files:
    - path: Makefile
      blob_oid: "%s"
      bytes: 23
      lines: 2
      sha256: %s
    - path: main.go
      blob_oid: "%s"
      bytes: 29
      lines: 3
      language: Go
      sha256: %s
  excluded:
    binary: 0
    generated: 0
    oversized: 0
    secret_like: 0
    symlink: 0
    submodule: 0
    vendor_or_generated_tree: 0
    non_regular: 0
  truncated: false
  remaining_candidate_files: 0
  remaining_candidate_bytes: 0
artifacts:
%s
---
# Software standards report

Inventory coverage was complete. Accepted outputs are listed in the manifest.
`,
		baseline,
		baseline,
		gitBlobOID("verify:\n\tgo test ./...\n"),
		excerptHash("verify:\n\tgo test ./...\n"),
		gitBlobOID("package main\n\nfunc main() {}\n"),
		excerptHash("package main\n\nfunc main() {}\n"),
		artifactEntries,
	)
}

func validRuleManifestEntry() string {
	return `  - id: keep-public-api-compatible
    kind: rule
    path: .software-standards/rules/keep-public-api-compatible.md
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 80
      factors:
        marginal_value: 25
        risk_reduction: 20
        actionability: 15
        applicability: 10
        earlier_feedback: 10`
}

func skillPackRepository(t *testing.T, metadata, evidenceHash string) (string, string) {
	t.Helper()
	repo, baseline := evidenceRepository(t)
	writeFile(t, filepath.Join(repo, ".software-standards", "report.md"), actionableReport(
		baseline,
		fmt.Sprintf(`  - id: review-change
    kind: skill
    path: .agents/skills/review-change/SKILL.md
    confidence: medium
    utility:
      method: ssb-utility-v1
      total: 60
      factors:
        marginal_value: 15
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 5
    category: correctness
    lenses:
      - kind: task
        value: verification
    scopes:
      - "**/*.go"
    derivation: extracted
    evidence:
      - role: enforces
        path: Makefile
        lines: 1-2
        excerpt_sha256: %s`, evidenceHash),
	))
	writeFile(t, filepath.Join(repo, ".agents", "skills", "review-change", "SKILL.md"), fmt.Sprintf(`---
name: review-change
description: Review a Go change using the repository workflow.
metadata:
%s
---
# Review change
`, metadata))
	return repo, baseline
}

type manifestLayoutFixture struct {
	manifestPath  string
	inventoryPath string
	reportPath    string
	rulePath      string
	manifest      string
	inventory     []byte
	report        []byte
	rule          []byte
}

func writeValidManifestLayoutPack(t *testing.T, repo, baseline string, withRule bool) manifestLayoutFixture {
	t.Helper()
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := inventory.ScanAtBaseline(context.Background(), ws, inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	inventoryBytes, err := json.MarshalIndent(recorded, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	inventoryBytes = append(inventoryBytes, '\n')
	reportBytes := []byte(`# Software standards report

Inventory coverage was complete. Review the [manifest](manifest.yaml) and [inventory](inventory.json) for machine metadata.
`)
	ruleBytes := []byte(`# Keep public APIs compatible

Keep public API changes backward compatible.
`)
	artifactEntries := "  []"
	if withRule {
		artifactEntries = fmt.Sprintf(`  - id: keep-public-api-compatible
    kind: rule
    path: .software-standards/rules/keep-public-api-compatible.md
    sha256: %s
    category: compatibility
    lenses:
      - kind: language
        value: go
    directive: always
    scopes:
      - "**/*.go"
    derivation: extracted
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 80
      factors:
        marginal_value: 25
        risk_reduction: 20
        actionability: 15
        applicability: 10
        earlier_feedback: 10`, digestBytes(ruleBytes), excerptHash("package main\n"))
	}
	manifest := fmt.Sprintf(`schema: ssb.dev/manifest/v1
baseline_commit: %s
inventory:
  path: .software-standards/inventory.json
  sha256: %s
report:
  path: .software-standards/report.md
  sha256: %s
artifacts:
%s
`, baseline, digestBytes(inventoryBytes), digestBytes(reportBytes), artifactEntries)
	fixture := manifestLayoutFixture{
		manifestPath:  filepath.Join(repo, ".software-standards", "manifest.yaml"),
		inventoryPath: filepath.Join(repo, ".software-standards", "inventory.json"),
		reportPath:    filepath.Join(repo, ".software-standards", "report.md"),
		rulePath:      filepath.Join(repo, ".software-standards", "rules", "keep-public-api-compatible.md"),
		manifest:      manifest,
		inventory:     inventoryBytes,
		report:        reportBytes,
		rule:          ruleBytes,
	}
	writeFile(t, fixture.manifestPath, fixture.manifest)
	writeFile(t, fixture.inventoryPath, string(fixture.inventory))
	writeFile(t, fixture.reportPath, string(fixture.report))
	if withRule {
		writeFile(t, fixture.rulePath, string(fixture.rule))
	}
	return fixture
}

func writeManifestInventoryAndRefreshManifest(t *testing.T, fixture manifestLayoutFixture, data []byte) {
	t.Helper()
	writeFile(t, fixture.inventoryPath, string(data))
	writeFile(t, fixture.manifestPath, strings.Replace(
		fixture.manifest,
		digestBytes(fixture.inventory),
		digestBytes(data),
		1,
	))
}

func writeManifestReportAndRefreshManifest(t *testing.T, fixture manifestLayoutFixture, data []byte) {
	t.Helper()
	writeFile(t, fixture.reportPath, string(data))
	writeFile(t, fixture.manifestPath, strings.Replace(
		fixture.manifest,
		digestBytes(fixture.report),
		digestBytes(data),
		1,
	))
}

func writeManifestRuleAndRefreshManifest(t *testing.T, fixture manifestLayoutFixture, data []byte) {
	t.Helper()
	writeFile(t, fixture.rulePath, string(data))
	writeFile(t, fixture.manifestPath, strings.Replace(
		fixture.manifest,
		digestBytes(fixture.rule),
		digestBytes(data),
		1,
	))
}

func bytesReplaceOnce(t *testing.T, data, old, replacement []byte) []byte {
	t.Helper()
	if !bytes.Contains(data, old) {
		t.Fatalf("fixture bytes do not contain %q", old)
	}
	return bytes.Replace(data, old, replacement, 1)
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func evidenceRepository(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(repo, "Makefile"), "verify:\n\tgo test ./...\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "baseline")
	return repo, strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
}

func excerptHash(excerpt string) string {
	sum := sha256.Sum256([]byte(excerpt))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func gitBlobOID(content string) string {
	payload := []byte(fmt.Sprintf("blob %d\x00%s", len(content), content))
	sum := sha1.Sum(payload)
	return hex.EncodeToString(sum[:])
}

func diagnosticsContain(diagnostics []rulepack.Diagnostic, want string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, want) || strings.Contains(diagnostic.Recovery, want) {
			return true
		}
	}
	return false
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := append([]string{"-c", "user.name=SSB Test", "-c", "user.email=ssb@example.invalid", "-C", dir}, args...)
	cmd := exec.Command("git", command...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

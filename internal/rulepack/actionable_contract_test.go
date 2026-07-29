package rulepack_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
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
	if pack.ReportPath != ".software-standards/report.md" ||
		pack.Report.Schema != rulepack.ReportSchema ||
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
      blob_oid: "0000000000000000000000000000000000000000"
      bytes: 23
      lines: 2
      language: make
      sha256: %s
    - path: main.go
      blob_oid: "0000000000000000000000000000000000000000"
      bytes: 29
      lines: 3
      language: go
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
`, baseline, baseline, excerptHash("verify:\n\tgo test ./...\n"), excerptHash("package main\n\nfunc main() {}\n"), artifactEntries)
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

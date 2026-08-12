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

func TestValidateNormalizesVerificationVersions(t *testing.T) {
	tests := []struct {
		name             string
		schema           string
		workingDirectory string
		wantDirectories  []string
	}{
		{
			name:            "v1 defaults every step to repository root",
			schema:          "ssb.dev/verification/v1",
			wantDirectories: []string{".", "."},
		},
		{
			name:             "v2 preserves root and nested directories",
			schema:           "ssb.dev/verification/v2",
			workingDirectory: "    working_directory: .\n",
			wantDirectories:  []string{".", "tools"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, fixture := verificationPackRepository(t, test.schema, fmt.Sprintf(`  - run: go test ./...
%s    source_evidence: make-verify
    expected_result: The tests exit successfully.
  - run: go vet ./...
%s    source_evidence: make-verify
    expected_result: Static analysis exits successfully.
`, test.workingDirectory, strings.Replace(test.workingDirectory, ".\n", "tools\n", 1)))

			pack, diagnostics := validatePack(t, repo)
			if len(diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics for %s: %#v\nmanifest:\n%s", test.schema, diagnostics, fixture.manifest)
			}
			if len(pack.Recipes) != 1 || len(pack.Recipes[0].Steps) != 2 {
				t.Fatalf("unexpected normalized recipes: %#v", pack.Recipes)
			}
			for index, want := range test.wantDirectories {
				if got := pack.Recipes[0].Steps[index].WorkingDirectory; got != want {
					t.Fatalf("step %d working_directory = %q, want %q", index, got, want)
				}
			}
			if pack.Recipes[0].Steps[0].Run != "go test ./..." || pack.Recipes[0].Steps[1].Run != "go vet ./..." {
				t.Fatalf("step order changed: %#v", pack.Recipes[0].Steps)
			}
		})
	}
}

func TestValidateRejectsInvalidVerificationVersionsAndWorkingDirectories(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		steps  string
		want   string
	}{
		{
			name:   "v1 rejects the v2-only field",
			schema: "ssb.dev/verification/v1",
			steps:  validVerificationStep("    working_directory: .\n"),
			want:   "working_directory",
		},
		{
			name:   "v2 requires a directory",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep(""),
			want:   "working_directory is required",
		},
		{
			name:   "absolute path",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: /tmp\n"),
			want:   "unsafe working_directory",
		},
		{
			name:   "traversal",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: ../tools\n"),
			want:   "unsafe working_directory",
		},
		{
			name:   "alternate separator",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: 'tools\\sub'\n"),
			want:   "unsafe working_directory",
		},
		{
			name:   "Windows volume",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: C:/tools\n"),
			want:   "unsafe working_directory",
		},
		{
			name:   "empty segment",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: tools//sub\n"),
			want:   "unsafe working_directory",
		},
		{
			name:   "dot segment",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: ./tools\n"),
			want:   "unsafe working_directory",
		},
		{
			name:   "file-valued path",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: main.go\n"),
			want:   "is not a directory",
		},
		{
			name:   "missing path",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: missing\n"),
			want:   "does not exist at the pinned baseline",
		},
		{
			name:   "submodule path",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: vendor/module/subdir\n"),
			want:   "passes through a submodule",
		},
		{
			name:   "duplicate v2 field",
			schema: "ssb.dev/verification/v2",
			steps:  validVerificationStep("    working_directory: .\n    working_directory: tools\n"),
			want:   "working_directory",
		},
		{
			name:   "Unicode format character in exact command",
			schema: "ssb.dev/verification/v2",
			steps: strings.Replace(
				validVerificationStep("    working_directory: .\n"),
				"run: go test ./...",
				"run: \"go test ./... # \u202e\"",
				1,
			),
			want: "format characters",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, _ := verificationPackRepository(t, test.schema, test.steps)
			_, diagnostics := validatePack(t, repo)
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
			for _, item := range diagnostics {
				if strings.Contains(item.Message, test.want) && item.Recovery == "" {
					t.Fatalf("diagnostic lacks recovery guidance: %#v", item)
				}
			}
		})
	}
}

func TestValidateReportsOnlyStrictDecodeFailureForMalformedVerification(t *testing.T) {
	for _, schema := range []string{rulepack.VerificationSchemaV1, rulepack.VerificationSchemaV2} {
		t.Run(schema, func(t *testing.T) {
			workingDirectory := ""
			if schema == rulepack.VerificationSchemaV2 {
				workingDirectory = "    working_directory: .\n"
			}
			repo, _ := verificationPackRepository(t, schema, fmt.Sprintf(`  - run: go test ./...
%s    source_evidence: make-verify
    expected_results: The tests exit successfully.
`, workingDirectory))

			_, diagnostics := validatePack(t, repo)
			if len(diagnostics) != 1 ||
				!strings.Contains(diagnostics[0].Message, "expected_results") ||
				diagnostics[0].Recovery == "" {
				t.Fatalf("malformed %s diagnostics = %#v, want one actionable strict-decode failure", schema, diagnostics)
			}
		})
	}
}

func validVerificationStep(workingDirectory string) string {
	return fmt.Sprintf(`  - run: go test ./...
%s    source_evidence: make-verify
    expected_result: The tests exit successfully.
`, workingDirectory)
}

func verificationPackRepository(t *testing.T, schema, steps string) (string, manifestLayoutFixture) {
	t.Helper()
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	writeFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(repo, "Makefile"), "verify:\n\tgo test ./...\n")
	writeFile(t, filepath.Join(repo, "tools", "check.go"), "package tools\n")
	writeFile(t, filepath.Join(repo, ".env"), "TOKEN=not-a-real-secret\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "initial baseline")
	gitlink := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "update-index", "--add", "--cacheinfo", "160000,"+gitlink+",vendor/module")
	git(t, repo, "commit", "-m", "add pinned submodule entry")
	baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	fixture := writeValidManifestLayoutPack(t, repo, baseline, false)
	recipe := fmt.Sprintf(`schema: %s
id: verify-repository
title: Verify the repository
category: testability
lenses:
  - kind: task
    value: verification
scopes:
  - "**/*"
derivation: extracted
evidence:
  - ref: make-verify
    role: enforces
    path: Makefile
    lines: 1-2
    excerpt_sha256: %s
when: Before handoff.
steps:
%s`, schema, excerptHash("verify:\n\tgo test ./...\n"), steps)
	recipePath := filepath.Join(repo, ".software-standards", "verification", "verify-repository.yaml")
	writeFile(t, recipePath, recipe)
	entry := fmt.Sprintf(`  - id: verify-repository
    kind: verification
    path: .software-standards/verification/verify-repository.yaml
    sha256: %s
    category: testability
    lenses:
      - kind: task
        value: verification
    scopes:
      - "**/*"
    derivation: extracted
    evidence:
      - ref: make-verify
        role: enforces
        path: Makefile
        lines: 1-2
        excerpt_sha256: %s
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 70
      factors:
        marginal_value: 20
        risk_reduction: 15
        actionability: 15
        applicability: 10
        earlier_feedback: 10`, digestBytes([]byte(recipe)), excerptHash("verify:\n\tgo test ./...\n"))
	fixture.manifest = strings.Replace(fixture.manifest, "artifacts:\n  []", "artifacts:\n"+entry, 1)
	writeFile(t, fixture.manifestPath, fixture.manifest)
	return repo, fixture
}

func validatePack(t *testing.T, repo string) (rulepack.Pack, []rulepack.Diagnostic) {
	t.Helper()
	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	pack, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	return pack, diagnostics
}

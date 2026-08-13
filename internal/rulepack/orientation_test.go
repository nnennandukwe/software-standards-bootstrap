package rulepack_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
)

func TestValidateLoadsManifestOrientation(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	fixture := writeValidManifestLayoutPack(t, repo, baseline, true)
	orientation := fmt.Sprintf(`schema: ssb.dev/orientation/v1
summary:
  text: Software Standards Bootstrap is an offline Go CLI.
  evidence:
    - role: declares
      path: main.go
      lines: 1-1
      excerpt_sha256: %s
`, excerptHash("package main\n"))
	bindOrientation(t, repo, &fixture, orientation)

	pack, diagnostics := validatePack(t, repo)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if pack.OrientationPath != ".software-standards/orientation.yaml" ||
		pack.Manifest.Orientation.Path != pack.OrientationPath ||
		pack.Manifest.Orientation.SHA256 != digestBytes([]byte(orientation)) ||
		pack.Orientation == nil || pack.Orientation.Schema != rulepack.OrientationSchema ||
		pack.Orientation.Summary == nil || pack.Orientation.Summary.Text != "Software Standards Bootstrap is an offline Go CLI." {
		t.Fatalf("unexpected normalized orientation: %#v", pack)
	}
	if len(pack.Manifest.Artifacts) != 1 {
		t.Fatalf("orientation changed artifact count: %#v", pack.Manifest.Artifacts)
	}
}

func TestValidateRejectsUnreferencedOrientation(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline, false)
	writeFile(t, filepath.Join(repo, ".software-standards", "orientation.yaml"), "schema: ssb.dev/orientation/v1\n")

	_, diagnostics := validatePack(t, repo)
	if !diagnosticsContain(diagnostics, "bind it in the manifest or remove it") {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestValidateRejectsUnreferencedOrientationSymlinkWithActionableRecovery(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline, false)
	target := filepath.Join(t.TempDir(), "orientation.yaml")
	writeFile(t, target, "schema: ssb.dev/orientation/v1\n")
	orientationPath := filepath.Join(repo, ".software-standards", "orientation.yaml")
	if err := os.Symlink(target, orientationPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	_, diagnostics := validatePack(t, repo)
	for _, item := range diagnostics {
		if strings.Contains(item.Message, "symlink present without a manifest reference") {
			if !strings.Contains(item.Recovery, "replace it with reviewed regular-file bytes and bind those bytes") ||
				!strings.Contains(item.Recovery, "remove the symlink") {
				t.Fatalf("unhelpful symlink recovery: %#v", item)
			}
			return
		}
	}
	t.Fatalf("missing unreferenced orientation symlink diagnostic: %#v", diagnostics)
}

func TestValidateRejectsUnreferencedOrientationDirectoryWithActionableRecovery(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeValidManifestLayoutPack(t, repo, baseline, false)
	orientationPath := filepath.Join(repo, ".software-standards", "orientation.yaml")
	if err := os.Mkdir(orientationPath, 0o755); err != nil {
		t.Fatal(err)
	}

	_, diagnostics := validatePack(t, repo)
	for _, item := range diagnostics {
		if strings.Contains(item.Message, "non-regular entry present without a manifest reference") {
			if !strings.Contains(item.Recovery, "replace it with reviewed regular-file bytes and bind those bytes") ||
				!strings.Contains(item.Recovery, "remove the entry") {
				t.Fatalf("unhelpful non-regular recovery: %#v", item)
			}
			return
		}
	}
	t.Fatalf("missing unreferenced orientation directory diagnostic: %#v", diagnostics)
}

func TestValidateAcceptsCompleteAndSchemaOnlyOrientation(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		repo, fixture := verificationPackRepository(t, rulepack.VerificationSchemaV2, validVerificationStep("    working_directory: .\n"))
		evidence := fmt.Sprintf(` [{role: declares, path: main.go, lines: 1-1, excerpt_sha256: %s}]`, excerptHash("package main\n"))
		orientation := fmt.Sprintf(`schema: ssb.dev/orientation/v1
summary:
  text: Software Standards Bootstrap is an offline Go CLI.
  evidence:%s
areas:
  - path: tools
    purpose: Contains repository verification helpers.
    evidence:%s
prerequisites:
  - requirement: Go is installed.
    evidence:%s
documents:
  - label: Repository entry point
    path: main.go
    evidence:%s
related_artifacts:
  - verify-repository
guidance:
  - kind: planning
    text: Read the repository entry point before planning.
    evidence:%s
  - kind: implementation
    text: Keep changes within the reviewed repository contract.
    evidence:%s
  - kind: verification
    text: Display the retained command before deliberate execution.
    evidence:%s
  - kind: handoff
    text: Report verification and any residual acceptance gaps.
    evidence:%s
`, evidence, evidence, evidence, evidence, evidence, evidence, evidence, evidence)
		bindOrientation(t, repo, &fixture, orientation)

		pack, diagnostics := validatePack(t, repo)
		if len(diagnostics) != 0 {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
		if pack.Orientation == nil || len(pack.Orientation.Areas) != 1 ||
			len(pack.Orientation.Prerequisites) != 1 || len(pack.Orientation.Documents) != 1 ||
			len(pack.Orientation.RelatedArtifactIDs) != 1 || len(pack.Orientation.Guidance) != 4 {
			t.Fatalf("complete orientation was not normalized: %#v", pack.Orientation)
		}
	})

	t.Run("schema only", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		fixture := writeValidManifestLayoutPack(t, repo, baseline, false)
		bindOrientation(t, repo, &fixture, "schema: ssb.dev/orientation/v1\n")

		pack, diagnostics := validatePack(t, repo)
		if len(diagnostics) != 0 || pack.Orientation == nil || pack.Orientation.Summary != nil ||
			len(pack.Orientation.Areas) != 0 || len(pack.Manifest.Artifacts) != 0 {
			t.Fatalf("schema-only orientation failed: pack=%#v diagnostics=%#v", pack, diagnostics)
		}
	})
}

func TestValidateRejectsOrientationSchemaTextEvidenceAndDuplicates(t *testing.T) {
	valid := fmt.Sprintf(`schema: ssb.dev/orientation/v1
summary:
  text: Reviewed repository summary.
  evidence:
    - role: declares
      path: main.go
      lines: 1-1
      excerpt_sha256: %s
`, excerptHash("package main\n"))
	tests := []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{name: "unknown field", mutate: func(value string) string { return value + "unknown: true\n" }, want: "unknown"},
		{name: "duplicate field", mutate: func(value string) string {
			return strings.Replace(value, "schema:", "schema: ssb.dev/orientation/v1\nschema:", 1)
		}, want: "schema"},
		{name: "wrong schema", mutate: func(value string) string {
			return strings.Replace(value, rulepack.OrientationSchema, "ssb.dev/orientation/v0", 1)
		}, want: "schema must be"},
		{name: "untrimmed text", mutate: func(value string) string {
			return strings.Replace(value, "Reviewed repository summary.", "' Reviewed repository summary.'", 1)
		}, want: "trimmed and nonempty"},
		{name: "multiple paragraphs", mutate: func(value string) string {
			return strings.Replace(value, "text: Reviewed repository summary.", "text: |-\n    First paragraph.\n\n    Second paragraph.", 1)
		}, want: "single-paragraph"},
		{name: "Unicode format character", mutate: func(value string) string {
			return strings.Replace(value, "Reviewed repository summary.", "Reviewed repository \u202esummary.", 1)
		}, want: "format characters"},
		{name: "Unicode paragraph separator", mutate: func(value string) string {
			return strings.Replace(value, "text: Reviewed repository summary.", "text: \"Reviewed repository\\u2029summary.\"", 1)
		}, want: "single-paragraph"},
		{name: "text over limit", mutate: func(value string) string {
			return strings.Replace(value, "Reviewed repository summary.", strings.Repeat("x", rulepack.OrientationMaxTextRunes+1), 1)
		}, want: "maximum is 1024"},
		{name: "missing evidence", mutate: func(value string) string { return value[:strings.Index(value, "  evidence:")] }, want: "requires 1-16 citations"},
		{name: "inference-only evidence", mutate: func(value string) string { return strings.Replace(value, "role: declares", "role: demonstrates", 1) }, want: "role \"demonstrates\" is not supported"},
		{name: "evidence ref", mutate: func(value string) string {
			return strings.Replace(value, "    - role:", "    - ref: summary\n      role:", 1)
		}, want: "does not use ref"},
		{name: "stale evidence", mutate: func(value string) string {
			return strings.Replace(value, excerptHash("package main\n"), "sha256:"+strings.Repeat("0", 64), 1)
		}, want: "excerpt hash does not match"},
		{name: "duplicate prerequisite", mutate: func(value string) string {
			return value + fmt.Sprintf(`prerequisites:
  - requirement: Go is installed.
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
  - requirement: Go is installed.
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
`, excerptHash("package main\n"), excerptHash("package main\n"))
		}, want: "duplicate prerequisite"},
		{name: "duplicate guidance", mutate: func(value string) string {
			return value + fmt.Sprintf(`guidance:
  - kind: handoff
    text: Report the result.
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
  - kind: handoff
    text: Report the result.
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
`, excerptHash("package main\n"), excerptHash("package main\n"))
		}, want: "duplicate handoff guidance"},
		{name: "duplicate area", mutate: func(value string) string {
			return value + fmt.Sprintf(`areas:
  - path: main.go
    purpose: Entry point.
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
  - path: main.go
    purpose: Repeated entry point.
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
`, excerptHash("package main\n"), excerptHash("package main\n"))
		}, want: "duplicate area path"},
		{name: "duplicate document", mutate: func(value string) string {
			return value + fmt.Sprintf(`documents:
  - label: Entry point
    path: main.go
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
  - label: Repeated entry point
    path: main.go
    evidence:
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s
`, excerptHash("package main\n"), excerptHash("package main\n"))
		}, want: "duplicate document path"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			fixture := writeValidManifestLayoutPack(t, repo, baseline, true)
			bindOrientation(t, repo, &fixture, test.mutate(valid))
			_, diagnostics := validatePack(t, repo)
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
		})
	}
}

func TestValidateRejectsUnsafeOrientationBaselineEntries(t *testing.T) {
	evidence := fmt.Sprintf(`
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s`, excerptHash("package main\n"))
	tests := []struct {
		name        string
		orientation string
		want        string
	}{
		{
			name: "area through submodule",
			orientation: fmt.Sprintf(`schema: %s
areas:
  - path: vendor/module/subdir
    purpose: Unsafe external tree.
    evidence:%s
`, rulepack.OrientationSchema, evidence),
			want: "passes through a submodule",
		},
		{
			name: "excluded document",
			orientation: fmt.Sprintf(`schema: %s
documents:
  - label: Environment secrets
    path: .env
    evidence:%s
`, rulepack.OrientationSchema, evidence),
			want: "excluded as secret-like",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, fixture := verificationPackRepository(t, rulepack.VerificationSchemaV2, validVerificationStep("    working_directory: .\n"))
			bindOrientation(t, repo, &fixture, test.orientation)
			_, diagnostics := validatePack(t, repo)
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
		})
	}
}

func TestValidateRejectsUnsafeOrientationPathsAndRelationships(t *testing.T) {
	evidence := fmt.Sprintf(`
      - role: declares
        path: main.go
        lines: 1-1
        excerpt_sha256: %s`, excerptHash("package main\n"))
	tests := []struct {
		name        string
		orientation func() string
		want        string
	}{
		{name: "absolute area", orientation: func() string {
			return fmt.Sprintf("schema: %s\nareas:\n  - path: /tmp\n    purpose: Unsafe.\n    evidence:%s\n", rulepack.OrientationSchema, evidence)
		}, want: "unsafe or empty repository-relative path"},
		{name: "missing area", orientation: func() string {
			return fmt.Sprintf("schema: %s\nareas:\n  - path: missing\n    purpose: Missing.\n    evidence:%s\n", rulepack.OrientationSchema, evidence)
		}, want: "does not exist at the pinned baseline"},
		{name: "file relationship", orientation: func() string {
			return fmt.Sprintf("schema: %s\nrelated_artifacts:\n  - keep-public-api-compatible\n", rulepack.OrientationSchema)
		}, want: "must resolve to a verification recipe or Agent Skill"},
		{name: "missing relationship", orientation: func() string {
			return fmt.Sprintf("schema: %s\nrelated_artifacts:\n  - missing-artifact\n", rulepack.OrientationSchema)
		}, want: "must resolve to a verification recipe or Agent Skill"},
		{name: "duplicate relationship", orientation: func() string {
			return fmt.Sprintf("schema: %s\nrelated_artifacts:\n  - missing-artifact\n  - missing-artifact\n", rulepack.OrientationSchema)
		}, want: "duplicate related artifact"},
		{name: "unsupported guidance kind", orientation: func() string {
			return fmt.Sprintf("schema: %s\nguidance:\n  - kind: review\n    text: Review the change.\n    evidence:%s\n", rulepack.OrientationSchema, evidence)
		}, want: "guidance kind \"review\" is not supported"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			fixture := writeValidManifestLayoutPack(t, repo, baseline, true)
			bindOrientation(t, repo, &fixture, test.orientation())
			_, diagnostics := validatePack(t, repo)
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
		})
	}
}

func TestValidateEnforcesOrientationFileLimitAndBinding(t *testing.T) {
	tests := []struct {
		name string
		size int
		want string
	}{
		{name: "at limit", size: int(rulepack.OrientationMaxFileBytes)},
		{name: "over limit", size: int(rulepack.OrientationMaxFileBytes) + 1, want: "larger than 1048576 bytes"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			fixture := writeValidManifestLayoutPack(t, repo, baseline, false)
			prefix := "schema: ssb.dev/orientation/v1\n#"
			orientation := prefix + strings.Repeat("x", test.size-len(prefix))
			bindOrientation(t, repo, &fixture, orientation)
			_, diagnostics := validatePack(t, repo)
			if test.want == "" && len(diagnostics) != 0 {
				t.Fatalf("orientation at limit failed: %#v", diagnostics)
			}
			if test.want != "" && !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
		})
	}

	t.Run("digest substitution", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		fixture := writeValidManifestLayoutPack(t, repo, baseline, false)
		bindOrientation(t, repo, &fixture, "schema: ssb.dev/orientation/v1\n")
		writeFile(t, filepath.Join(repo, ".software-standards", "orientation.yaml"), "schema: ssb.dev/orientation/v1\n# substituted\n")
		_, diagnostics := validatePack(t, repo)
		if !diagnosticsContain(diagnostics, "orientation.yaml SHA-256 does not match manifest") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		fixture := writeValidManifestLayoutPack(t, repo, baseline, false)
		bindOrientation(t, repo, &fixture, "schema: ssb.dev/orientation/v1\n")
		if err := os.Remove(filepath.Join(repo, ".software-standards", "orientation.yaml")); err != nil {
			t.Fatal(err)
		}
		_, diagnostics := validatePack(t, repo)
		if !diagnosticsContain(diagnostics, "does not exist") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
		if !diagnosticsContain(diagnostics, "restore the reviewed orientation bytes or remove the manifest reference") {
			t.Fatalf("missing orientation lacks recovery guidance: %#v", diagnostics)
		}
	})

	t.Run("direct symlink", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		fixture := writeValidManifestLayoutPack(t, repo, baseline, false)
		bindOrientation(t, repo, &fixture, "schema: ssb.dev/orientation/v1\n")
		orientationPath := filepath.Join(repo, ".software-standards", "orientation.yaml")
		if err := os.Remove(orientationPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(repo, "main.go"), orientationPath); err != nil {
			t.Fatal(err)
		}
		_, diagnostics := validatePack(t, repo)
		if !diagnosticsContain(diagnostics, "must be a real regular file, not a symlink") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})
}

func TestValidateRejectsInvalidOrientationManifestReference(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{
			name: "noncanonical path",
			mutate: func(manifest string) string {
				return strings.Replace(manifest, ".software-standards/orientation.yaml", "orientation.yaml", 1)
			},
			want: "orientation path must be .software-standards/orientation.yaml",
		},
		{
			name: "missing digest",
			mutate: func(manifest string) string {
				start := strings.Index(manifest, "orientation:\n")
				return manifest[:start] + strings.Replace(manifest[start:], "sha256: sha256:", "sha256: invalid-", 1)
			},
			want: "orientation sha256 must use",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			fixture := writeValidManifestLayoutPack(t, repo, baseline, false)
			bindOrientation(t, repo, &fixture, "schema: ssb.dev/orientation/v1\n")
			writeFile(t, fixture.manifestPath, test.mutate(fixture.manifest))
			_, diagnostics := validatePack(t, repo)
			if !diagnosticsContain(diagnostics, test.want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, test.want)
			}
		})
	}
}

func TestUpdateManifestArtifactsPreservesOrientationReference(t *testing.T) {
	orientationDigest := "sha256:" + strings.Repeat("c", 64)
	manifest := fmt.Sprintf(`schema: ssb.dev/manifest/v1
baseline_commit: %s
inventory:
  path: .software-standards/inventory.json
  sha256: sha256:%s
report:
  path: .software-standards/report.md
  sha256: sha256:%s
orientation:
  path: .software-standards/orientation.yaml
  sha256: %s
artifacts:
  - id: verify-repository
    kind: verification
    path: .software-standards/verification/verify-repository.yaml
    sha256: sha256:%s
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 1
      factors: {}
`, strings.Repeat("a", 40), strings.Repeat("a", 64), strings.Repeat("b", 64), orientationDigest, strings.Repeat("d", 64))
	updated, err := rulepack.UpdateManifestArtifacts([]byte(manifest), nil, map[string]string{"verify-repository": "sha256:" + strings.Repeat("e", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "path: .software-standards/orientation.yaml") || !strings.Contains(string(updated), "sha256: "+orientationDigest) {
		t.Fatalf("orientation reference changed:\n%s", updated)
	}
}

func bindOrientation(t *testing.T, repo string, fixture *manifestLayoutFixture, contents string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, ".software-standards", "orientation.yaml"), contents)
	reference := fmt.Sprintf(`orientation:
  path: .software-standards/orientation.yaml
  sha256: %s
`, digestBytes([]byte(contents)))
	fixture.manifest = strings.Replace(fixture.manifest, "artifacts:\n", reference+"artifacts:\n", 1)
	writeFile(t, fixture.manifestPath, fixture.manifest)
}

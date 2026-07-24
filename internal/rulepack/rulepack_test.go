package rulepack_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/rulepack"
	"github.com/nnennandukwe/software-standards-bootstrap/internal/workspace"
)

func TestValidateAcceptsGroundedScoredRulesAndPortableSkills(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n\nEvidence review.\n")
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), validRule(
		baseline,
		excerptHash("package main\n"),
		excerptHash("verify:\n\tgo test ./...\n"),
	))
	writeFile(t, filepath.Join(repo, ".agents", "skills", "verify-change", "SKILL.md"), `---
name: verify-change
description: Run the repository's existing verification workflow before handing off a change.
license: Apache-2.0
compatibility: Requires Git 2.39 or newer and the repository's own verification tooling.
metadata:
  source: software-standards-bootstrap
  topic: correctness
---

# Verify change

Run the cited repository check and report its result.
`)

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
	if len(pack.Rules) != 1 || pack.Rules[0].ID != "verify-before-merge" {
		t.Fatalf("unexpected pack: %#v", pack)
	}
	if len(pack.Skills) != 1 || pack.Skills[0].ID != "verify-change" {
		t.Fatalf("unexpected skills: %#v", pack.Skills)
	}
	if pack.Rules[0].Topic != "correctness" || pack.Skills[0].Topic != "correctness" {
		t.Fatalf("unexpected primary topics: rule=%q skill=%q", pack.Rules[0].Topic, pack.Skills[0].Topic)
	}
}

func TestValidateRejectsUngroundedOrInternallyInconsistentRules(t *testing.T) {
	tests := []struct {
		name       string
		mutateRule func(string) string
		want       string
	}{
		{
			name: "score arithmetic",
			mutateRule: func(rule string) string {
				return strings.Replace(rule, "total: 70", "total: 71", 1)
			},
			want: "score total 71 does not equal factor sum 70",
		},
		{
			name: "importance band",
			mutateRule: func(rule string) string {
				return strings.Replace(rule, "importance: high", "importance: very-high", 1)
			},
			want: "importance very-high does not match score 70",
		},
		{
			name: "stale baseline",
			mutateRule: func(rule string) string {
				return strings.Replace(rule, "baseline_commit:", "baseline_commit: deadbeef #", 1)
			},
			want: "baseline_commit must equal current HEAD",
		},
		{
			name: "changed excerpt",
			mutateRule: func(rule string) string {
				return strings.Replace(rule, excerptHash("package main\n"), "sha256:"+strings.Repeat("0", 64), 1)
			},
			want: "excerpt hash does not match",
		},
		{
			name: "deterministic without cited check",
			mutateRule: func(rule string) string {
				start := strings.Index(rule, "verification:\n")
				end := strings.Index(rule[start:], "related_skills:")
				return rule[:start] + "verification:\n  proof_gap: No existing check proves this.\n" + rule[start+end:]
			},
			want: "deterministic rules must cite an existing verification command",
		},
		{
			name: "unknown field",
			mutateRule: func(rule string) string {
				return strings.Replace(rule, "title: Verify before merge", "title: Verify before merge\nmystery: true", 1)
			},
			want: "field mystery not found",
		},
		{
			name: "missing primary topic",
			mutateRule: func(rule string) string {
				return strings.Replace(rule, "topic: correctness\n", "", 1)
			},
			want: "topic is required",
		},
		{
			name: "unsupported primary topic",
			mutateRule: func(rule string) string {
				return strings.Replace(rule, "topic: correctness", "topic: readability", 1)
			},
			want: `topic "readability" is not supported`,
		},
		{
			name: "missing related skill",
			mutateRule: func(rule string) string {
				return strings.Replace(rule, "- verify-change", "- missing-skill", 1)
			},
			want: ".agents/skills/missing-skill/SKILL.md does not exist",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n")
			rule := validRule(baseline, excerptHash("package main\n"), excerptHash("verify:\n\tgo test ./...\n"))
			writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), test.mutateRule(rule))
			writeFile(t, filepath.Join(repo, ".agents", "skills", "verify-change", "SKILL.md"), `---
name: verify-change
description: Verify changes with the repository's existing check.
metadata:
  topic: correctness
---
# Verify
`)

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

func TestValidateRequiresSupportedPrimaryTopicForRelatedSkills(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
		want     string
	}{
		{
			name:     "missing",
			metadata: "  source: software-standards-bootstrap\n",
			want:     "topic is required",
		},
		{
			name:     "unsupported",
			metadata: "  source: software-standards-bootstrap\n  topic: readability\n",
			want:     `topic "readability" is not supported`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, baseline := evidenceRepository(t)
			writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n")
			writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), validRule(
				baseline,
				excerptHash("package main\n"),
				excerptHash("verify:\n\tgo test ./...\n"),
			))
			writeFile(t, filepath.Join(repo, ".agents", "skills", "verify-change", "SKILL.md"), fmt.Sprintf(`---
name: verify-change
description: Verify changes with the repository's existing check.
metadata:
%s---
# Verify
`, test.metadata))

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

func TestValidateRequiresAssessmentAndCandidateEvidenceThreshold(t *testing.T) {
	repo, baseline := evidenceRepository(t)
	rule := validRule(baseline, excerptHash("package main\n"), excerptHash("verify:\n\tgo test ./...\n"))
	rule = strings.Replace(rule, "    authoritative: true\n", "", 1)
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), rule)
	writeFile(t, filepath.Join(repo, ".agents", "skills", "verify-change", "SKILL.md"), `---
name: verify-change
description: Verify changes with the repository's existing check.
---
# Verify
`)

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		".software-standards/assessment.md does not exist",
		"requires one authoritative source or three occurrences across two files",
	} {
		if !diagnosticsContain(diagnostics, want) {
			t.Fatalf("diagnostics %#v do not contain %q", diagnostics, want)
		}
	}
}

func TestValidateRejectsPackAndSkillSymlinkEscapes(t *testing.T) {
	t.Run("pack directory", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		outside := t.TempDir()
		writeFile(t, filepath.Join(outside, "assessment.md"), "# Outside assessment\n")
		writeFile(t, filepath.Join(outside, "rules", "verify-before-merge.md"), validRule(
			baseline,
			excerptHash("package main\n"),
			excerptHash("verify:\n\tgo test ./...\n"),
		))
		if err := os.Symlink(outside, filepath.Join(repo, ".software-standards")); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(repo, ".agents", "skills", "verify-change", "SKILL.md"), `---
name: verify-change
description: Verify changes with the repository's existing check.
---
# Verify
`)

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		if !diagnosticsContain(diagnostics, "must be a real directory, not a symlink") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})

	t.Run("skill directory", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n")
		writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), validRule(
			baseline,
			excerptHash("package main\n"),
			excerptHash("verify:\n\tgo test ./...\n"),
		))
		outside := t.TempDir()
		writeFile(t, filepath.Join(outside, "SKILL.md"), `---
name: verify-change
description: Outside skill.
---
# Outside
`)
		skillParent := filepath.Join(repo, ".agents", "skills")
		if err := os.MkdirAll(skillParent, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(skillParent, "verify-change")); err != nil {
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
		if !diagnosticsContain(diagnostics, "contains a symlink component") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})
}

func TestValidateRejectsEmptyRequiredFilesAndDuplicateEvidenceLocations(t *testing.T) {
	t.Run("empty files", func(t *testing.T) {
		repo, _ := evidenceRepository(t)
		writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "")
		writeFile(t, filepath.Join(repo, ".software-standards", "rules", "empty.md"), "")

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"assessment.md must not be empty", "empty.md must not be empty"} {
			if !diagnosticsContain(diagnostics, want) {
				t.Fatalf("diagnostics %#v do not contain %q", diagnostics, want)
			}
		}
	})

	t.Run("duplicate evidence", func(t *testing.T) {
		repo, baseline := evidenceRepository(t)
		writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n")
		rule := validRule(baseline, excerptHash("package main\n"), excerptHash("verify:\n\tgo test ./...\n"))
		evidence := `  - path: main.go
    lines: 1-1
    excerpt_sha256: %s
    authoritative: true
`
		duplicate := fmt.Sprintf(evidence, excerptHash("package main\n"))
		rule = strings.Replace(rule, "verification:\n", duplicate+"verification:\n", 1)
		writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), rule)
		writeFile(t, filepath.Join(repo, ".agents", "skills", "verify-change", "SKILL.md"), `---
name: verify-change
description: Verify changes with the repository's existing check.
---
# Verify
`)

		ws, err := workspace.Open(context.Background(), repo)
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics, err := rulepack.Validate(context.Background(), ws)
		if err != nil {
			t.Fatal(err)
		}
		if !diagnosticsContain(diagnostics, "duplicate evidence location") {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
	})
}

func TestValidateResolvesLiteralEvidencePathsSupportedByTheHost(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	evidencePath := "docs/a\n$([not-a-command]);*.md"
	if runtime.GOOS == "windows" {
		// Win32 rejects newlines and asterisks in filenames. Spaces, Unicode,
		// and the remaining shell metacharacters still exercise literal paths.
		evidencePath = "docs/a 日本語 $([not-a-command]);.md"
	}
	writeFile(t, filepath.Join(repo, filepath.FromSlash(evidencePath)), "Authoritative contract.\n")
	writeFile(t, filepath.Join(repo, "Makefile"), "verify:\n\tgo test ./...\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "literal evidence")
	baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))

	writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n")
	rule := validRule(baseline, excerptHash("Authoritative contract.\n"), excerptHash("verify:\n\tgo test ./...\n"))
	rule = strings.Replace(rule, "path: main.go", "path: "+strconv.Quote(evidencePath), 1)
	writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), rule)
	writeFile(t, filepath.Join(repo, ".agents", "skills", "verify-change", "SKILL.md"), `---
name: verify-change
description: Verify changes with the repository's existing check.
metadata:
  topic: correctness
---
# Verify
`)

	ws, err := workspace.Open(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics, err := rulepack.Validate(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestValidateRejectsEvidenceExcludedFromTheSafeInventory(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		want    string
	}{
		{"secret-like", ".env", "TOKEN=private\n", "secret-like"},
		{"vendor tree", "vendor/example/rule.md", "vendored\n", "vendor/generated tree"},
		{"generated", "api.pb.go", "// Code generated by fixture. DO NOT EDIT.\n", "generated content"},
		{"binary", "evidence.bin", "text\x00binary", "binary content"},
		{"oversized", "large.txt", strings.Repeat("x", (1<<20)+1), "larger than 1048576 bytes"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := t.TempDir()
			git(t, repo, "init", "-b", "main")
			writeFile(t, filepath.Join(repo, filepath.FromSlash(test.path)), test.content)
			writeFile(t, filepath.Join(repo, "Makefile"), "verify:\n\tgo test ./...\n")
			git(t, repo, "add", ".")
			git(t, repo, "commit", "-m", "excluded evidence")
			baseline := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))

			writeFile(t, filepath.Join(repo, ".software-standards", "assessment.md"), "# Assessment\n")
			rule := validRule(baseline, excerptHash(test.content), excerptHash("verify:\n\tgo test ./...\n"))
			rule = strings.Replace(rule, "path: main.go", fmt.Sprintf("path: %q", test.path), 1)
			writeFile(t, filepath.Join(repo, ".software-standards", "rules", "verify-before-merge.md"), rule)
			writeFile(t, filepath.Join(repo, ".agents", "skills", "verify-change", "SKILL.md"), `---
name: verify-change
description: Verify changes with the repository's existing check.
---
# Verify
`)

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

func validRule(baseline, evidenceHash, verificationHash string) string {
	return fmt.Sprintf(`---
schema: ssb.dev/rule/v1
id: verify-before-merge
title: Verify before merge
topic: correctness
scopes:
  - "**/*.go"
classification: deterministic
importance: high
score:
  method: ssb-score-v1
  total: 70
  factors:
    prevalence: 15
    consistency: 15
    authority: 15
    risk: 15
    applicability: 10
confidence: high
baseline_commit: %s
evidence:
  - path: main.go
    lines: 1-1
    excerpt_sha256: %s
    authoritative: true
verification:
  command: go test ./...
  source:
    path: Makefile
    lines: 1-2
    excerpt_sha256: %s
related_skills:
  - verify-change
---
Run the repository's existing verification command before merging a Go change.
`, baseline, evidenceHash, verificationHash)
}

func excerptHash(excerpt string) string {
	sum := sha256.Sum256([]byte(excerpt))
	return "sha256:" + hex.EncodeToString(sum[:])
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

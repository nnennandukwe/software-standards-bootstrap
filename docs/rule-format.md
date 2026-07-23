# Rule format: `ssb.dev/rule/v1`

Each retained rule is one Markdown file at:

```text
.software-standards/rules/<rule-id>.md
```

The filename must match the stable lower-case kebab-case `id`. YAML frontmatter is strict: unknown and duplicate fields fail validation. The Markdown after the closing marker is the exact rule body projected into `AGENTS.md`.

## Complete example

```markdown
---
schema: ssb.dev/rule/v1
id: verify-before-merge
title: Verify before merge
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
baseline_commit: 0123456789abcdef0123456789abcdef01234567
evidence:
  - path: internal/service/service.go
    lines: 21-29
    excerpt_sha256: sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
    authoritative: true
verification:
  command: go test ./...
  source:
    path: Makefile
    lines: 4-6
    excerpt_sha256: sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789
related_skills:
  - verify-change
---
Run the repository's existing verification command before merging a Go change.
```

## Scoring

`ssb-score-v1` is additive:

| Factor | Range |
|---|---:|
| Prevalence | 0–25 |
| Consistency | 0–20 |
| Existing authority | 0–20 |
| Correctness, security, or architectural risk | 0–20 |
| Applicability | 0–15 |

Bands:

- `very-high`: 80–100
- `high`: 65–79
- `medium`: 45–64
- `low`: 25–44
- below 25: assessment-only

The total must equal the factors. There is no fixed rule-count cap.

## Evidence threshold and hashing

A rule needs either:

- at least one evidence item marked `authoritative: true`; or
- at least three consistent occurrences across two different files.

Evidence paths are repository-relative tracked regular files at `baseline_commit`. Line ranges are one-based and inclusive. The excerpt digest is SHA-256 over the exact bytes in the cited lines, including existing line endings. Validation reads those bytes from the pinned Git blob, not from the worktree.

The same inventory eligibility boundary applies during validation. A rule cannot cite a binary, oversized, secret-like, generated/vendor, symlink, or submodule path that inspection excluded.

## Classification

`deterministic` requires both:

- an existing verification command; and
- an exact `verification.source` evidence citation showing where the repository defines that check.

The command is mapped, not executed.

`guidance` requires exactly one of:

- an existing command plus source citation; or
- a non-empty `proof_gap`.

Procedural work belongs in a portable `.agents/skills/<skill-id>/SKILL.md` and is referenced through `related_skills`.

## Portable Agent Skill fields

Referenced skills use the Agent Skills core frontmatter:

```yaml
---
name: verify-change
description: Run the repository's existing verification workflow before handing off a change.
license: Apache-2.0
compatibility: Requires the repository's own verification tooling.
metadata:
  source: software-standards-bootstrap
---
```

`name` and `description` are required. `license`, `compatibility`, and string-to-string `metadata` are optional. Consumer-specific discovery and optional fields are not portable behavior.

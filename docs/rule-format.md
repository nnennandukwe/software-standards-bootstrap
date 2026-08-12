# Actionable pack format

New Software Standards Bootstrap packs separate machine metadata from human
Markdown while retaining four accepted artifact kinds:

```text
.software-standards/inventory.json
.software-standards/manifest.yaml
.software-standards/orientation.yaml # optional
.software-standards/report.md
.software-standards/rules/<id>.md
.software-standards/verification/<id>.yaml
.agents/skills/<id>/SKILL.md
.software-standards/automation/<id>.yaml
```

The new manifest layout uses `ssb.dev/manifest/v1`. The published embedded
layout embeds `ssb.dev/report/v1` frontmatter in `report.md` and
keeps `ssb.dev/rule/v2` frontmatter in semantic rules. It remains fully
readable for validation, rendering, ADR creation, and governed prune without
conversion. New generation always writes the manifest layout.

All YAML and JSON are strict. Unknown fields and duplicate keys fail
validation. IDs are globally unique lower-case kebab-case and each artifact
kind has one canonical path.

## Manifest: `ssb.dev/manifest/v1`

`.software-standards/manifest.yaml` owns machine metadata:

```yaml
schema: ssb.dev/manifest/v1
baseline_commit: 0123456789abcdef0123456789abcdef01234567
inventory:
  path: .software-standards/inventory.json
  sha256: sha256:<exact-file-digest>
report:
  path: .software-standards/report.md
  sha256: sha256:<exact-file-digest>
orientation:
  path: .software-standards/orientation.yaml
  sha256: sha256:<exact-file-digest>
artifacts:
  - id: keep-public-apis-compatible
    kind: rule
    path: .software-standards/rules/keep-public-apis-compatible.md
    sha256: sha256:<exact-file-digest>
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
        path: CONTRIBUTING.md
        lines: 20-24
        excerpt_sha256: sha256:<64-lowercase-hex>
    confidence: high
    utility:
      method: ssb-utility-v1
      total: 80
      factors:
        marginal_value: 25
        risk_reduction: 20
        actionability: 15
        applicability: 10
        earlier_feedback: 10
    related_artifacts: [verify-api-compatibility]
```

The manifest owns:

- the exact accepted artifact index, kind, and canonical path;
- the exact SHA-256 digest of every primary artifact's raw bytes;
- category, activation, scope, directive, derivation, and evidence;
- confidence and utility;
- cross-artifact relationships;
- the pinned baseline; and
- exact references to the inventory, human report, and optional orientation.

SHA-256 values cover raw file bytes, including line endings. A manifest with
zero artifacts is valid. Relationships name accepted IDs. Self, duplicate,
and dangling relationships fail validation.

`inventory.json` is the complete, unedited `ssb inspect --format json`
response. Validation rejects unknown or duplicate JSON fields, enforces its
128 MiB pre-parse limit, and replays it against the pinned baseline.

`report.md` has no frontmatter and begins at byte zero with:

```markdown
# Software standards report

Inventory coverage was complete. See [manifest.yaml](manifest.yaml) and
[inventory.json](inventory.json). Accepted outputs and limitations are
summarized below.
```

The report narrative is nonempty, links both machine files, and contains no
inventory rows or accepted-artifact metadata. `manifest.yaml` is limited to
1 MiB before parsing.

## Repository orientation: `ssb.dev/orientation/v1`

The manifest layout may bind one canonical
`.software-standards/orientation.yaml` file:

```yaml
schema: ssb.dev/orientation/v1
summary:
  text: This repository provides an offline CLI for evidence-backed standards.
  evidence:
    - role: declares
      path: README.md
      lines: 1-19
      excerpt_sha256: sha256:<64-lowercase-hex>
areas:
  - path: internal/rulepack
    purpose: Validates supported pack layouts and normalizes their contents.
    evidence:
      - role: declares
        path: docs/architecture.md
        lines: 160-210
        excerpt_sha256: sha256:<64-lowercase-hex>
prerequisites:
  - requirement: Go 1.26.5
    evidence:
      - role: declares
        path: go.mod
        lines: 1-3
        excerpt_sha256: sha256:<64-lowercase-hex>
documents:
  - label: Contribution workflow
    path: CONTRIBUTING.md
    evidence:
      - role: declares
        path: CONTRIBUTING.md
        lines: 1-40
        excerpt_sha256: sha256:<64-lowercase-hex>
related_artifacts: [verify-repository]
guidance:
  - kind: handoff
    text: Report the failing test, implementation, verification, and remaining gaps.
    evidence:
      - role: declares
        path: CONTRIBUTING.md
        lines: 31-40
        excerpt_sha256: sha256:<64-lowercase-hex>
```

Orientation is concise reviewed context, not active policy. It does not enter
`artifacts`, artifact counts, benchmark denominators, or ADR eligibility. A
schema-only document is valid and affects source identity without rendering an
empty section. An unreferenced canonical file is rejected with bind-or-remove
recovery. Embedded-layout packs do not load or render orientation.

The raw file limit is 1 MiB. Each optional collection accepts at most 32
entries. Every rendered statement has 1–16 evidence citations. Summary,
purpose, requirement, and guidance text has 1–1024 Unicode code points;
document labels have 1–160. Repository-relative paths have 1–1024 UTF-8 bytes.
Text is trimmed, nonempty, single-paragraph, and control-free.

Evidence uses only `declares` or `enforces`; it has no `ref`. Area paths resolve
to a regular file or tree at the pinned baseline. Document paths resolve to
eligible tracked regular files. Related IDs resolve only to retained
verification recipes or Agent Skills. Duplicate areas, documents,
prerequisites, relationships, or identical guidance entries fail validation.
Guidance kinds are `planning`, `implementation`, `verification`, and `handoff`.

## Confidence and utility

Every accepted artifact has `medium` or `high` confidence. Low-confidence
candidates are removed.

`ssb-utility-v1` is additive:

| Factor | Maximum |
|---|---:|
| Marginal value beyond existing instructions and checks | 30 |
| Risk reduction | 25 |
| Actionability | 20 |
| Applicability to supported coding work | 15 |
| Earlier feedback than the existing failure surface | 10 |

Bands are `very-high` for 80–100, `high` for 65–79, and `medium` for 45–64.
Candidates below 45 are removed.

## Semantic rule: normalized `ssb.dev/rule/v2`

```markdown
# Keep public APIs compatible

Keep in-scope public API changes backward compatible.
```

A manifest-layout rule has no frontmatter. It starts with exactly one H1 whose text
supplies the normalized title, followed immediately by nonempty actionable
text. The manifest owns category, activation, directive, scope, derivation,
and exact evidence. Rules contain no commands or proof metadata.

The rule name should express a falsifiable goal. Include a mechanism in the
name only when that mechanism is the repository contract.

## Verification recipe: `ssb.dev/verification/v2`

```yaml
schema: ssb.dev/verification/v2
id: verify-api-compatibility
title: Verify API compatibility
category: compatibility
lenses:
  - kind: task
    value: verification
scopes:
  - "**/*.go"
derivation: extracted
evidence:
  - ref: compatibility-command
    role: enforces
    path: Makefile
    lines: 10-12
    excerpt_sha256: sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789
when: Before handing off a public API change.
steps:
  - run: make verify-compatibility
    working_directory: .
    source_evidence: compatibility-command
    expected_result: The command reports no incompatible API changes.
```

A recipe records one or more ordered existing commands, when they apply, and
the expected successful result. Every step references exact `enforces`
evidence. Recipes contain no branching, implementation edits, setup decisions,
or semantic judgment. SSB validates but never executes them.

New generation emits only `ssb.dev/verification/v2`. Every step requires
`working_directory`. `.` is repository root; a non-root value uses `/`, is a
canonical relative path with no empty, dot, traversal, volume, or alternate
separator segments, resolves to a tracked tree at the pinned baseline, and
does not pass through a submodule.

Existing `ssb.dev/verification/v1` recipes remain readable in both pack
layouts. Their private v1 representation does not recognize
`working_directory`; valid steps normalize to root `.`. A v1 document carrying
the v2-only field fails strict decoding rather than silently changing command
location.

## Agent Skill

Multi-step procedures with decisions, edits, setup, branching, or recovery use
portable Agent Skills:

```yaml
---
name: review-api-change
description: Review a public API change and its dependent surfaces.
license: Apache-2.0
---
# Review an API change

## Procedure

Inspect the public surface and each dependent package.
```

The entrypoint remains `.agents/skills/<id>/SKILL.md`. Manifest-layout skills
keep portable `name` and `description` frontmatter plus a meaningful standard
`license` or `compatibility` field. They omit SSB-owned `metadata.category`.
The body begins with an H1 and a nonempty procedure. The manifest records SSB
provenance, selection metadata, confidence, utility, and relationships.

## Automation proposal: `ssb.dev/automation/v1`

```yaml
schema: ssb.dev/automation/v1
id: automate-api-compatibility
title: Add an automatic API compatibility check
category: compatibility
lenses:
  - kind: language
    value: go
scopes:
  - "**/*.go"
derivation: inferred
evidence: <exact supporting occurrences>
condition: In-scope public API changes remain backward compatible.
suggested_check: Compare exported declarations against the baseline.
trigger: Run when an in-scope public declaration changes.
expected_success: No incompatible declaration is found.
expected_failure: Report each incompatible declaration and source location.
```

Automation proposals are reviewable designs. They are not generated code,
implemented checks, active standards, or ADR-adopted behavior.

## Category, activation, and evidence

Supported categories are `architecture`, `compatibility`, `compliance`,
`correctness`, `developer-experience`, `documentation`, `maintainability`,
`operability`, `performance`, `quality`, `reliability`, `security`, and
`testability`.

Lenses are:

- one `base` lens without a value; or
- contextual `language`, `framework`, and `task` lenses.

Language and framework values are lower-case kebab-case. Task values are
`planning`, `implementation`, and `verification`. An artifact applies only
when its path scope matches. Contextual activation also requires every
represented lens dimension; values within a dimension are alternatives.

Evidence roles are:

- `declares`: an explicit repository-maintained obligation;
- `demonstrates`: an implementation occurrence supporting an inferred
  invariant; and
- `enforces`: a repository mechanism that actively checks a condition.

`extracted` artifacts require at least one `declares` or `enforces` citation.
`inferred` artifacts require three distinct `demonstrates` citations across at
least two files.

Paths identify eligible tracked regular files at the manifest baseline. Ranges
are one-based and inclusive. Digests hash exact cited bytes including line
endings. Validation also replays the complete inventory at the recorded limits.

## Projection, ADR, and JSON

`AGENTS.md` follows this reading order: derived ownership and lifecycle
boundary, populated repository orientation, routing, action-first standing
orders, contextual semantic rules, verification commands, and Agent Skills.
Contextual rules remain link-only. Recipe steps preserve source order and show
exact inert command bytes, non-root `working_directory`, and expected results.
Relationships show only explicitly declared rule, recipe, and skill IDs in
declared order. Automation remains absent. Empty, orientation-only, and
automation-only packs do not write a managed section; rendering either leaves
an unprojected `AGENTS.md` unchanged or removes a stale managed section.

An ADR includes adopted rules, recipes, and skills with category, derivation,
confidence, utility, and concise evidence sources. It excludes automation
proposals and fails safely when nothing is adoptable.

`ssb validate --format json` uses response schema 3. A valid response names
`pack.layout` as `manifest` or `embedded`, exposes explicit manifest,
inventory, and report paths when separate, and includes normalized manifest,
inventory, human report, optional orientation reference and content, and all
four artifact arrays. Verification steps include normalized
`working_directory`, including `.` for verification/v1 input. Invalid output
includes diagnostics and omits the normalized pack.

Presence of a safe regular `.software-standards/manifest.yaml` selects the
manifest layout. An invalid manifest never falls back to embedded parsing. When
it is absent, `ssb.dev/report/v1` frontmatter selects the embedded layout.
Manifest-layout packs reject embedded report and rule frontmatter. Validation
never rewrites, repairs, or migrates either layout.

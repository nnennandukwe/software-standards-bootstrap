# Actionable pack format

New Software Standards Bootstrap packs separate machine metadata from human
Markdown while retaining four accepted artifact kinds:

```text
.software-standards/inventory.json
.software-standards/manifest.yaml
.software-standards/report.md
.software-standards/rules/<id>.md
.software-standards/verification/<id>.yaml
.agents/skills/<id>/SKILL.md
.software-standards/automation/<id>.yaml
```

The new layout is `split-v1` and uses `ssb.dev/manifest/v1`. The published
`legacy-v1` layout embeds `ssb.dev/report/v1` frontmatter in `report.md` and
keeps `ssb.dev/rule/v2` frontmatter in semantic rules. It remains fully
readable for validation, rendering, ADR creation, and governed prune without
conversion. New generation always writes the split layout.

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
- exact references to the inventory and human report.

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

A split rule has no frontmatter. It starts with exactly one H1 whose text
supplies the normalized title, followed immediately by nonempty actionable
text. The manifest owns category, activation, directive, scope, derivation,
and exact evidence. Rules contain no commands or proof metadata.

The rule name should express a falsifiable goal. Include a mechanism in the
name only when that mechanism is the repository contract.

## Verification recipe: `ssb.dev/verification/v1`

```yaml
schema: ssb.dev/verification/v1
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
    source_evidence: compatibility-command
    expected_result: The command reports no incompatible API changes.
```

A recipe records one or more ordered existing commands, when they apply, and
the expected successful result. Every step references exact `enforces`
evidence. Recipes contain no branching, implementation edits, setup decisions,
or semantic judgment. SSB validates but never executes them.

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

The entrypoint remains `.agents/skills/<id>/SKILL.md`. Split-pack skills keep
portable `name` and `description` frontmatter plus a meaningful standard
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

`AGENTS.md` inlines base semantic rules, links contextual semantic rules and
verification recipes, indexes Agent Skills, and omits automation proposals.
Relationships surface links to related recipes and skills, never inactive
automation proposals. Empty and automation-only packs do not write a managed
section; rendering either leaves an unprojected `AGENTS.md` unchanged or
removes a previously generated managed section.

An ADR includes adopted rules, recipes, and skills with category, derivation,
confidence, utility, and concise evidence sources. It excludes automation
proposals and fails safely when nothing is adoptable.

`ssb validate --format json` uses response schema 3. A valid response names
`pack.format` as `split-v1` or `legacy-v1`, exposes explicit manifest,
inventory, and report paths when separate, and includes normalized manifest,
inventory, human report, and all four artifact arrays. Invalid output includes
diagnostics and omits the normalized pack.

Presence of a safe regular `.software-standards/manifest.yaml` selects the
split layout. An invalid manifest never falls back to legacy parsing. When it
is absent, `ssb.dev/report/v1` frontmatter selects `legacy-v1`. Split packs
reject legacy report and rule frontmatter. Validation never rewrites, repairs,
or migrates either format.

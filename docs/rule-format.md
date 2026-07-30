# Actionable pack format

Software Standards Bootstrap uses one required report manifest and four
accepted artifact kinds:

```text
.software-standards/report.md
.software-standards/rules/<id>.md
.software-standards/verification/<id>.yaml
.agents/skills/<id>/SKILL.md
.software-standards/automation/<id>.yaml
```

This is an intentional pre-release cutover. Semantic rules use the rewritten
`ssb.dev/rule/v2` contract. Earlier rule contracts and proof-oriented fields
are unsupported.

All YAML is strict. Unknown fields and duplicate keys fail validation. IDs are
globally unique lower-case kebab-case and each artifact kind has one canonical
path.

## Report: `ssb.dev/report/v1`

`.software-standards/report.md` is the required pack manifest and run
narrative:

```yaml
---
schema: ssb.dev/report/v1
baseline_commit: 0123456789abcdef0123456789abcdef01234567
inventory: <complete unedited schema 2 ssb-inventory-v2 response>
artifacts:
  - id: keep-public-apis-compatible
    kind: rule
    path: .software-standards/rules/keep-public-apis-compatible.md
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
    related_artifacts:
      - verify-api-compatibility
---
# Software standards report

Inventory coverage was complete. Accepted outputs and run-wide limitations
are summarized here.
```

The report owns:

- the exact accepted artifact index, kind, and canonical path;
- confidence and utility;
- cross-artifact relationships;
- complete inventory and run-wide limitations; and
- accepted-output summaries.

It does not duplicate native artifact provenance and contains no rejected
candidates, reasons, or counts. A report with zero artifacts is valid.

Agent Skills are the exception to provenance locality. Portable frontmatter
cannot carry the complete SSB contract, so each skill manifest entry also
records `category`, `lenses`, `scopes`, `derivation`, and `evidence`. Its
category must match `metadata.category` in the skill.

Relationships name accepted IDs. Self, duplicate, and dangling relationships
fail validation.

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

## Semantic rule: `ssb.dev/rule/v2`

```markdown
---
schema: ssb.dev/rule/v2
id: keep-public-apis-compatible
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
    path: CONTRIBUTING.md
    lines: 20-24
    excerpt_sha256: sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
---
Keep in-scope public API changes backward compatible.
```

A rule owns category, activation, directive, scope, derivation, exact evidence,
and an actionable body. It does not contain classification, confidence,
utility, baseline, commands, command sources, coverage, proved properties,
gaps, or replacement verification-approach metadata.

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
metadata:
  category: compatibility
---
```

The entrypoint remains `.agents/skills/<id>/SKILL.md`. The report records its
SSB provenance, selection metadata, confidence, utility, and relationships.

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

Paths identify eligible tracked regular files at the report baseline. Ranges
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

`ssb validate --format json` uses response schema 2. A valid response includes
the normalized report and all four artifact arrays. Invalid output includes
diagnostics and omits the normalized pack.

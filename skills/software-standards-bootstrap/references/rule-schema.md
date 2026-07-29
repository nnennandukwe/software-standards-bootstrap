# Actionable artifact schema quick reference

The accepted pack is indexed by `.software-standards/report.md`. All YAML is
strict: unknown fields and duplicate keys fail validation.

## Report manifest

```yaml
schema: ssb.dev/report/v1
baseline_commit: <40-character commit>
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
```

The report owns the accepted index, confidence, utility, relationships,
complete inventory, limitations, and accepted-output summaries. It contains
no rejected candidates, reasons, or counts. Native artifacts own their
category, lenses, scopes, derivation, and evidence.

A skill manifest entry additionally records those native fields because
portable Agent Skill frontmatter cannot carry the complete SSB provenance
contract. Its `category` must match the skill's `metadata.category`.

Accepted confidence is `medium` or `high`.

`ssb-utility-v1` factor maxima are 30 marginal value, 25 risk reduction, 20
actionability, 15 applicability, and 10 earlier feedback. The total must equal
the factors. Totals below 45 are rejected and removed. Bands are 80–100
very-high, 65–79 high, and 45–64 medium.

Artifact kinds and canonical paths are:

| Kind | Path |
|---|---|
| `rule` | `.software-standards/rules/<id>.md` |
| `verification` | `.software-standards/verification/<id>.yaml` |
| `skill` | `.agents/skills/<id>/SKILL.md` |
| `automation` | `.software-standards/automation/<id>.yaml` |

IDs are globally unique lower-case kebab-case. Relationships name accepted
artifact IDs and cannot dangle, repeat, or refer to the source artifact.

## Semantic rule

```yaml
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
    excerpt_sha256: sha256:<64 lowercase hex>
---
Keep changes to in-scope public APIs backward compatible.
```

The Markdown body is canonical. A rule contains no classification, score,
confidence, baseline, command, source, coverage, proves, proof-gap, or other
verification-approach field.

`directive` is `always`, `ask-first`, `never`, or `prefer`.

## Verification recipe

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
    excerpt_sha256: sha256:<64 lowercase hex>
when: Before handing off a public API change.
steps:
  - run: make verify-compatibility
    source_evidence: compatibility-command
    expected_result: The command reports no incompatible API changes.
```

Recipes contain one or more ordered existing commands. Every step references
exact `enforces` evidence and states an expected successful result. Recipes
contain no branching, edits, setup decisions, or semantic judgment.

## Agent Skill

```yaml
---
name: review-api-change
description: Review a public API change and its dependent surfaces.
metadata:
  category: compatibility
---
```

The report entry owns the skill's lenses, scopes, derivation, evidence,
confidence, utility, and relationships. Multi-step procedures with decisions,
edits, setup, branching, or recovery belong here.

## Automation proposal

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
suggested_check: Compare exported API declarations against the baseline.
trigger: Run when an in-scope public declaration changes.
expected_success: No incompatible declaration is found.
expected_failure: Report each incompatible declaration and source location.
```

Automation proposals are reviewable designs. They are not executable checks,
active standards, or ADR-adopted behavior.

## Shared fields

Supported categories are `architecture`, `compatibility`, `compliance`,
`correctness`, `developer-experience`, `documentation`, `maintainability`,
`operability`, `performance`, `quality`, `reliability`, `security`, and
`testability`.

Lenses are `base`, `language`, `framework`, or `task`. `base` has no value and
must be the sole lens. Language and framework values are lower-case kebab-case.
Task values are `planning`, `implementation`, and `verification`.

Every artifact has at least one repository-relative scope. Contextual
activation requires a matching scope and every represented lens dimension;
values within one dimension are alternatives.

Derivation is `extracted` or `inferred`. Evidence roles are `declares`,
`demonstrates`, and `enforces`. Extracted artifacts need at least one
`declares` or `enforces` citation. Inferred artifacts need three distinct
`demonstrates` citations across at least two files.

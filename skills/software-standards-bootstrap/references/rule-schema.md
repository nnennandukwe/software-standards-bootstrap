# Rule schema quick reference

New proposals use `ssb.dev/rule/v2`. Existing `ssb.dev/rule/v1` files remain
valid, but v1 files must not contain v2-only fields.

## Complete v2 shape

```yaml
schema: ssb.dev/rule/v2
id: lower-kebab-case
title: Developer-facing title
topic: correctness
lenses:
  - kind: language
    value: go
  - kind: framework
    value: cobra
  - kind: task
    value: review
directive: always
scopes:
  - "repository/relative/**"
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
baseline_commit: <exact inspect baseline>
evidence:
  - path: tracked/file
    lines: 10-14
    excerpt_sha256: sha256:<64 lowercase hex>
    authoritative: true
verification:
  command: existing command
  source:
    path: tracked/check-definition
    lines: 1-4
    excerpt_sha256: sha256:<64 lowercase hex>
  coverage: full
  proves: State the bounded property established when the command passes.
related_skills:
  - procedural-skill
```

Use one `base` lens without a value for a repository-wide rule:

```yaml
lenses:
  - kind: base
```

`base` must be the sole lens. Other supported kinds are `language`,
`framework`, and `task`; they require lower-case kebab-case values. Task values
are `implementation`, `review`, `testing`, `security`, `documentation`, and
`release`.

A rule's path scope must match. For a contextual rule, every represented lens
dimension must also match. Multiple values within one dimension are
alternatives. If context is unknown, consumers load the potentially relevant
rule instead of excluding it.

`directive` is exactly one of `always`, `ask-first`, `never`, or `prefer`.

## Proof classification

A deterministic v2 rule requires:

- an existing command and exact source citation;
- `coverage: full`; and
- a non-empty `proves` statement limited to the property established when the
  command passes.

Guidance with an existing check uses the same command and source shape with
`coverage: partial` and a bounded `proves` statement. Guidance without an
existing check uses only:

```yaml
verification:
  proof_gap: Explain the precise unautomated boundary.
```

Proof gaps do not declare `coverage` or `proves`. Every mapped command is
mapped, not executed by `ssb`; its presence is never a passing result.

## Shared v1/v2 fields

`topic` is required and must be exactly one value from: `architecture`,
`compatibility`, `compliance`, `correctness`, `developer-experience`,
`documentation`, `maintainability`, `operability`, `performance`, `quality`,
`reliability`, `security`, or `testability`. Choose the concern that best
explains the rule's engineering risk or change obligation. Use `quality` only
when no narrower topic fits.

Factor ranges are 0–25 prevalence, 0–20 consistency, 0–20 authority, 0–20 risk,
and 0–15 applicability. Bands are 80–100 very-high, 65–79 high, 45–64 medium,
and 25–44 low.

Referenced Agent Skills must set `metadata.topic` to one value from the same
taxonomy, based on the workflow's primary engineering outcome.

The Markdown after the closing `---` is the canonical rule body. Unknown
fields, duplicate keys, score mismatch, stale baseline, missing evidence, hash
mismatch, unsafe scopes, invalid lens combinations, unsupported directive,
inconsistent proof coverage, and missing related skills fail validation.

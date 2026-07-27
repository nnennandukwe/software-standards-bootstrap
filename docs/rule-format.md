# Rule format: `ssb.dev/rule/v2`

New proposals use one Markdown file per retained rule at:

```text
.software-standards/rules/<rule-id>.md
```

The filename must match the stable lower-case kebab-case `id`. YAML frontmatter
is strict: unknown and duplicate fields fail validation. The Markdown after
the closing marker is the canonical rule body. `AGENTS.md` inlines base rule
bodies and links contextual rule bodies for on-demand loading.

Existing `ssb.dev/rule/v1` files remain valid and renderable with their original
validation semantics. They must not declare v2-only `lenses`, `directive`,
`verification.coverage`, or `verification.proves` fields. The bundled Agent
Skill emits v2. A v1 rule records no directive or explicit proof coverage, so
consumers apply its canonical body as written rather than inferring v2
semantics.

## Complete v2 example

```markdown
---
schema: ssb.dev/rule/v2
id: verify-before-merge
title: Verify before merge
topic: correctness
lenses:
  - kind: language
    value: go
  - kind: task
    value: review
directive: always
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
  coverage: full
  proves: The retained Go assertions when the command passes.
related_skills:
  - verify-change
---
Run the repository's existing verification command before merging a Go change.
```

## Activation lenses

Every v2 rule declares either:

```yaml
lenses:
  - kind: base
```

or one or more contextual lenses:

```yaml
lenses:
  - kind: language
    value: go
  - kind: framework
    value: cobra
  - kind: task
    value: review
```

Supported kinds are `base`, `language`, `framework`, and `task`.

- `base` has no value and must be the sole lens.
- Language and framework values are lower-case kebab-case identifiers.
- Task values are `implementation`, `review`, `testing`, `security`,
  `documentation`, and `release`.
- Duplicate kind/value pairs fail validation.

A rule applies only when its path scope matches. For a contextual rule, every
represented lens dimension must also match. Multiple values within one
dimension are alternatives. For example, two language values mean Go or
Python, while a language plus a task requires both dimensions. When a consumer
cannot determine a dimension or affected path confidently, it loads the
potentially relevant rule instead of excluding it.

The canonical directory remains flat. Lenses provide deterministic virtual
grouping without duplicating a rule that spans language, framework, and task
contexts.

## Directive

`directive` records how the agent should interpret the rule:

- `never`: prohibited within the rule's evidence-backed scope;
- `ask-first`: requires developer authorization before the action;
- `always`: required whenever the rule applies; and
- `prefer`: the default approach, subject to documented exceptions or explicit
  user direction.

The directive does not replace importance, classification, or scope.

## Scoring

`ssb-score-v1` remains additive:

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

## Primary topic

Every rule declares exactly one primary software-engineering topic. It
identifies the concern that best explains the rule's risk or change obligation;
it does not replace lenses, scope, classification, importance, directive, or
confidence.

The controlled vocabulary is `architecture`, `compatibility`, `compliance`,
`correctness`, `developer-experience`, `documentation`, `maintainability`,
`operability`, `performance`, `quality`, `reliability`, `security`, and
`testability`. Prefer the narrowest accurate topic and use `quality` only when
no narrower topic fits. See [the topic taxonomy](../skills/software-standards-bootstrap/references/topics.md).

## Evidence threshold and hashing

A rule needs either:

- at least one evidence item marked `authoritative: true`; or
- at least three consistent occurrences across two different files.

Evidence paths are repository-relative tracked regular files at
`baseline_commit`. Line ranges are one-based and inclusive. The excerpt digest
is SHA-256 over the exact bytes in the cited lines, including existing line
endings. Validation reads those bytes from the pinned Git blob, not from the
worktree.

The same inventory eligibility boundary applies during validation. A rule
cannot cite a binary, oversized, secret-like, generated/vendor, symlink, or
submodule path that inspection excluded.

Rule bodies may point to preferred examples when authority or repeated evidence
establishes them as patterns to follow. They may identify counterexamples only
when repository authority marks the pattern deprecated, unsafe, or intentionally
avoided. Do not invent examples or infer that differing legacy code is wrong.

## Classification and proof coverage

`ssb` maps existing proof but never executes a cited command or claims it
passed.

A v2 `deterministic` rule requires:

- an existing verification command;
- an exact `verification.source` citation;
- `coverage: full`; and
- a non-empty `proves` statement naming the bounded property established when
  the command passes.

A v2 `guidance` rule uses exactly one of:

- an existing command plus source citation, `coverage: partial`, and a bounded
  `proves` statement; or
- a non-empty `proof_gap` with no coverage or proved-property fields.

This separation prevents a downstream loader from treating a mapped command as
a result or assuming a partial check proves the complete guidance.

## Portable Agent Skill fields

Procedural work belongs in a portable
`.agents/skills/<skill-id>/SKILL.md` and is referenced through
`related_skills`. Referenced skills use the Agent Skills core frontmatter:

```yaml
---
name: verify-change
description: Run the repository's existing verification workflow before handing off a change.
license: Apache-2.0
compatibility: Requires the repository's own verification tooling.
metadata:
  source: software-standards-bootstrap
  topic: correctness
---
```

`name` and `description` are required by the Agent Skills core format. Software
Standards Bootstrap additionally requires `metadata.topic` for every referenced
skill, using the same controlled vocabulary as rules. Consumer-specific
discovery and optional fields are not portable behavior.

## Validated JSON interchange

`ssb validate --format json` returns response schema 2. A valid response
includes the normalized `pack` containing the baseline, assessment, rules,
canonical bodies, evidence, lenses, directives, proof metadata, and referenced
skills. Invalid output contains diagnostics and omits `pack`.

Stable rule IDs are repository-local. A future centralized catalog owns its
external namespace, version, import review, and lifecycle while preserving the
source baseline and evidence. `ssb` does not fetch, import, synchronize, or
activate catalog rules.

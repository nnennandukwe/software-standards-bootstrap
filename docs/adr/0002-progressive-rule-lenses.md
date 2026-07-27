# 0002: Add progressive rule lenses without becoming a catalog

- Status: Proposed
- Date: 2026-07-27

## Context

Rule v1 records evidence, path scopes, one engineering topic, importance,
confidence, and proof classification. The root `AGENTS.md` projection renders
every retained rule body in stable ID order.

That representation is reviewable, but it cannot distinguish repository-wide
standing orders from language, framework, or request-specific guidance.
`topic` explains engineering risk rather than activation context, and path
scope alone cannot express that a rule is relevant to review or security work.
As packs grow, copying every body into root context also consumes tokens for
rules unrelated to the current task.

The project must preserve its differentiation from centralized catalogs:
repository evidence remains the input, Git review remains the selection and
adoption surface, and `ssb` must not fetch, synchronize, or activate external
policy.

## Decision

Introduce `ssb.dev/rule/v2` while retaining v1 validation and rendering.

- Every v2 rule has one base lens or one or more language, framework, and task
  lenses.
- Path scope and all represented lens dimensions select contextual rules;
  alternatives within one dimension are OR matches.
- Uncertain context fails open for loading: consumers include a potentially
  relevant rule instead of silently omitting it.
- Every v2 rule declares `always`, `ask-first`, `never`, or `prefer`.
- A deterministic rule maps a command with full coverage and states the exact
  property proved when it passes.
- Guidance either maps partial proof or records a proof gap.
- `ssb` never executes the mapped command and never converts its presence into
  a passing result.
- Root `AGENTS.md` inlines base standing orders, deduplicates mapped commands,
  and links contextual rule sources by lens instead of copying their bodies.
- The canonical rule directory remains flat. One rule may span several
  dimensions without being duplicated into arbitrary directory trees.
- Valid JSON validation output includes the normalized pack as a local
  interchange envelope. A future catalog owns any external namespace,
  versioning, import review, and lifecycle.

The bundled Agent Skill emits v2 and can consume an existing pack by loading
scope-matching base rules plus scope- and lens-matching contextual rules. It
does not overwrite or refresh an existing pack.

## Consequences

- New packs support progressive disclosure without hooks or repository code
  execution.
- Root agent context stays concise while complete rule bodies remain
  Git-reviewable canonical sources.
- Downstream consumers can distinguish rule modality, activation context, full
  proof, partial proof, and proof gaps.
- Existing v1 packs remain valid but do not gain explicit v2 activation or
  proof-coverage metadata.
- Validation JSON advances to response schema 2 for valid-pack interchange.
- Organizational catalog ingestion remains possible later without adding
  runtime network behavior or synchronization to `ssb`.

## Alternatives considered

- Split canonical rules into base/language/framework/task directories:
  rejected because cross-dimensional rules would require duplication or an
  arbitrary primary directory.
- Render every rule body and add headings: rejected because grouping alone
  does not reduce irrelevant context.
- Add a catalog import or synchronization command: rejected because it changes
  the repository-specific, offline trust boundary.
- Infer directives and lenses during rendering: rejected because semantic
  intent belongs in developer-reviewed canonical sources, not deterministic
  renderer heuristics.

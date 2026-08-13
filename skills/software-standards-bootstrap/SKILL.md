---
name: software-standards-bootstrap
description: Generate an evidence-backed actionable standards pack from a clean Git snapshot, or load the relevant rules, verification recipes, and Agent Skills from an existing pack. Use when a developer wants to extract or apply repository-specific engineering standards.
license: Apache-2.0
compatibility: Requires the ssb CLI, Git 2.39 or newer, a commit-backed branch, and host access to read and write the target repository.
metadata:
  project: software-standards-bootstrap
  schema: ssb.dev/manifest/v1
  category: developer-experience
  version: 0.6.0
---

# Software Standards Bootstrap

Generate or consume repository-specific actionable artifacts. The host agent
makes semantic judgments. The `ssb` CLI validates schemas, exact evidence,
inventory, confidence, utility, relationships, projection, and ADR output.

Do not use a generic rules catalog. Do not execute repository code or recipe
commands merely because they are recorded. Do not implement an automation
proposal. Do not stage, commit, branch, push, open a pull request, or activate
artifacts in another system.

Before generating artifacts, read [the actionable schemas](references/rule-schema.md),
[the category taxonomy](references/categories.md),
[the evidence workflow](references/evidence-workflow.md), and
[the structural-pattern workflow](references/structural-patterns.md). Read
[the governed lifecycle review](references/prune-review.md) before writing a
prune proposal.

## Choose the mode

If neither `.software-standards/manifest.yaml` nor
`.software-standards/report.md` exists, use generation mode. A safe regular
`manifest.yaml` selects the manifest layout; otherwise a frontmatter-based
`ssb.dev/report/v1` report selects the embedded layout. When either pack exists:

- use reviewed-pack maintenance mode only for an explicit request to validate
  or rerender developer-edited sources;
- use requested-ADR mode only after an explicit request for the adoption record;
- use governed lifecycle review mode only for an explicit adopted-pack review;
- stop on regeneration and explain that the existing pack must be reviewed and
  removed deliberately; never overwrite it; or
- otherwise use existing-pack consumption mode.

### Existing-pack consumption mode

Do not run `ssb inspect`, `ssb validate`, or `ssb render`, and do not rewrite
the pack.

1. Read `.software-standards/manifest.yaml` and the linked human report and
   optional orientation for a manifest-layout pack. Inspect the inventory's
   top-level baseline, completeness, limits, and counts plus only the file rows
   needed to confirm active evidence. Do not load the complete raw inventory
   into context. For an embedded-layout pack, read
   `.software-standards/report.md` and its accepted artifact index.
2. If the pack contains no rule, verification recipe, or Agent Skill, report
   that it has no active guidance. Automation proposals are not active policy.
3. Otherwise read the managed Software Standards Bootstrap section in root
   `AGENTS.md`. Stop as stale if it is missing, malformed, or disagrees with
   the detected manifest, human report, or canonical sources.
4. Identify affected repository-relative paths and classify the request as
   `planning`, `implementation`, or `verification`.
5. Identify languages and frameworks only from the request and repository
   evidence already available.
6. Select base rules whose scopes match. Select contextual rules, recipes, and
   skills only when scopes match and every represented lens dimension matches;
   values within one dimension are alternatives.
7. When a dimension or affected path is uncertain, load the potentially
   relevant artifact instead of excluding it.
8. Read the complete canonical source for every active semantic rule and Agent
   Skill. Recipes remain links to existing commands with explicit use
   conditions and expected results.

Treat projected repository orientation as reviewed context, not active policy.
Use it to understand the repository, important areas, prerequisites, canonical
documents, and evidence-backed task guidance. It does not change artifact
selection or prove that a related command ran.

Report active artifact IDs before applying them. Follow semantic rules and
procedural skills as applicable. Run recipe commands only when the developer's
request and host permissions independently authorize execution; never infer a
passing result from the recipe. Ignore automation proposals unless the
developer explicitly asks to review proposed check designs.

Consumption mode never changes the standards pack or projection.

### Reviewed-pack maintenance mode

Do not run `ssb inspect` and do not edit canonical sources. Run:

```bash
ssb validate --repo . --format text
ssb render --repo . --dry-run
ssb render --repo .
```

Stop on validation diagnostics without rendering. Otherwise report every
changed and untracked path. State that recipe commands were not executed and
automation proposals were not implemented. Do not create an ADR unless
separately requested.

### Requested-ADR mode

Only after developer review and an explicit request, run:

```bash
ssb adr --repo . --dry-run
ssb adr --repo .
```

If ADR conventions are ambiguous, stop and request the intended `--adr-dir`.
An empty or automation-only pack has nothing adoptable and must not create an
ADR. The ADR remains `Proposed`.

### Governed lifecycle review mode

Use this mode only for an explicit lifecycle-review request. `prune` is not
permission to delete. Follow
[the governed lifecycle review contract](references/prune-review.md).

Require a local point-in-time capability profile and optional provenance
declaration. Never select an implicit latest model, query an online registry,
or treat release notes as conformance proof. Run:

```bash
ssb prune inspect --repo . --review <id> --capabilities <profile> [--provenance <manifest>]
```

Never use partial inventory. Read `context.json`; evaluate canonical artifacts
separately and together; then write only `proposal.yaml` and complete candidate
files inside the review bundle. Do not edit canonical sources, `AGENTS.md`, or
ADRs.

Validate without approving:

```bash
ssb prune validate --repo . --review <id> --format text
ssb prune status --repo . --review <id>
```

Report every disposition, rationale, evidence reference or structured evidence
gap, confidence band, and unresolved question. Stop for human review. Do not
run `approve`, `apply`, `render`, `adr`, or `verify` unless the developer
separately requests that exact transition.
Application remains a dry run unless `--write` is explicitly authorized.

## Generation mode

### 1. Establish the immutable input

Run:

```bash
ssb inspect --repo . --format json
```

Never pass `--allow-partial`. Stop on any failure, including exit `4`, and
report recovery guidance verbatim. Record the exact `baseline_commit`, the
complete schema 2 inventory response, confirmation that `truncated` is false,
and the safe tracked paths available for semantic reads.

Preserve the command's complete, unedited `ssb inspect --format json` response
for `.software-standards/inventory.json`. Do not parse and re-encode it. Its
digest binds the exact raw file bytes, including line endings.

### 2. Perform targeted semantic reads

Read only inventory-listed paths. Never execute repository files, hooks, build
scripts, tests, linters, formatters, package managers, or existing commands.

Review:

- explicit repository obligations and engineering risks;
- dependency boundaries, parallel implementations and families, platform seams,
  compatibility surfaces, and source/test/documentation symmetry;
- existing commands, where invocation is defined, their triggers, and the
  exact condition they enforce; and
- existing automatic enforcement that already handles a condition completely.

For every candidate, collect exact one-based line ranges and hashes from the
pinned Git blobs. Narrow scope when the evidence supports only one package,
family, seam, or surface.

Separately collect concise repository orientation only when authoritative
repository evidence supports useful context that does not belong in a rule,
verification recipe, Agent Skill, or automation proposal. Orientation may
describe the repository summary, important areas, prerequisites, canonical
documents, related verification recipes or skills, and `planning`,
`implementation`, `verification`, or `handoff` guidance. Every rendered
statement requires exact `declares` or `enforces` evidence. Do not infer
orientation from `demonstrates` evidence. If no reviewed orientation is useful,
write no orientation file or manifest reference.

### 3. Route and evaluate candidates

Before presenting final candidates:

1. Reject anything outside planning, implementation, or verification work.
2. Classify derivation as `extracted` or `inferred` and collect exact evidence.
3. Review existing commands, invocation, triggers, and automatic enforcement.
4. Emit nothing when the condition is already handled completely and
   automatically.
5. Route remaining value to exactly one primary destination:
   - implementation condition → semantic rule;
   - deliberately invoked existing command → verification recipe;
   - multi-step workflow with decisions, edits, setup, branching, or recovery
     → Agent Skill;
   - valuable proposed automatic check → automation proposal;
   - otherwise → reject and discard.
6. Reject and discard candidates below `medium` confidence.
7. Score every remaining candidate with `ssb-utility-v1`; reject and discard a
   total below 45.
8. Review each semantic name. A rule name must express a falsifiable goal and
   include a mechanism only when that mechanism is the repository contract.
9. Write accepted artifacts and then the final report manifest. For the
   manifest layout, write `manifest.yaml` last, after every digest-bound
   primary file is final.

Do not persist rejected candidates, rejection reasons, or rejection counts.
Do not split a compound observation unless semantic clarity requires it.

### 4. Write the actionable pack

Create only accepted outputs:

```text
.software-standards/inventory.json
.software-standards/manifest.yaml
.software-standards/orientation.yaml
.software-standards/report.md
.software-standards/rules/<rule-id>.md
.software-standards/verification/<recipe-id>.yaml
.agents/skills/<skill-id>/SKILL.md
.software-standards/automation/<proposal-id>.yaml
```

`manifest.yaml` uses `ssb.dev/manifest/v1`. It owns the baseline, exact
inventory and report references, accepted index, category, activation lenses,
repository-relative scopes, directive where applicable, derivation, exact
evidence, `medium` or `high` confidence, utility of at least 45,
relationships, and each primary file's SHA-256 digest. Hash exact raw file
bytes, including line endings. IDs are globally unique stable kebab-case.

When orientation was retained, write `.software-standards/orientation.yaml`
with `ssb.dev/orientation/v1`, then bind its exact raw bytes through the
manifest `orientation` reference. Orientation is optional reviewed context.
It is not a manifest artifact, does not contribute to actionable artifact
counts, and is not eligible for the ADR. Use only eligible pinned-baseline
paths and exact `declares` or `enforces` evidence. Related IDs may name only a
retained verification recipe or Agent Skill.

Evidence roles are:

- `declares`: an explicit repository-maintained obligation;
- `demonstrates`: an implementation occurrence supporting an inferred
  invariant; and
- `enforces`: a repository mechanism that actively checks a condition.

An `extracted` artifact needs at least one `declares` or `enforces` citation.
An `inferred` artifact needs at least three distinct `demonstrates` citations
across at least two files.

`ssb-utility-v1` is additive: marginal value 0–30, risk reduction 0–25,
actionability 0–20, applicability 0–15, and earlier feedback 0–10. Bands are
80–100 very-high, 65–79 high, and 45–64 medium.

Semantic-rule Markdown has no frontmatter. It begins with one H1 title, then
immediately presents nonempty actionable text. The H1 supplies the normalized
title; machine metadata lives only in `manifest.yaml`.

New verification recipes use only `ssb.dev/verification/v2`. They contain
ordered existing commands, a required `working_directory` of `.` or a
canonical tracked repository directory for each step, exact `enforces`
evidence references, when they apply, and expected successful results. They
contain no branching, edits, or semantic judgment; existing v1 recipes remain
readable and normalize each step to repository root; never add
`working_directory` to a v1 document.

Agent Skills use portable `name` and `description` frontmatter plus a
meaningful standard `license` or `compatibility` field. Generation omits
`metadata.category`; the manifest stores all SSB selection and provenance
metadata. The body begins with an H1 and contains a nonempty `## Procedure`
section.

Automation proposals describe a condition, suggested check, trigger, and
expected success and failure. They are designs, not implemented checks or
adopted standards.

`inventory.json` is the complete, unedited inspection response. `report.md`
has no frontmatter, begins at byte zero with `# Software standards report`,
contains nonempty narrative, and links to `manifest.yaml` and
`inventory.json`. It summarizes limitations and accepted outputs without
inventory rows or machine metadata. A zero-artifact manifest is valid.

Verification recipes and automation proposals use their native YAML schemas.
Their exact primary bytes are digest-bound by the manifest.

If any target exists, stop instead of overwriting developer work.

### 5. Validate and project

Run:

```bash
ssb validate --repo . --format text
ssb render --repo . --dry-run
ssb render --repo .
```

Stop on diagnostics before projection. Do not edit the managed `AGENTS.md`
section. When a digest-bound source changes, update its exact manifest digest,
then rerun.

The projection identifies its derived lifecycle boundary, presents populated
orientation first, explains routing, inlines action-first base semantic rules,
links contextual rules, displays recipe commands and expected results without
executing them, indexes primary Agent Skills, and omits automation proposals.
An empty, orientation-only, or automation-only pack does not create or rewrite
an unprojected `AGENTS.md`, but removes a stale generated managed section when
one exists.

### 6. Disclose the complete uncommitted result

Run:

```bash
git --no-optional-locks status --short --untracked-files=all
```

List every changed and untracked path. State explicitly:

- no repository code or recipe command was executed;
- no automation proposal was implemented;
- no Git mutation was performed;
- `AGENTS.md` is derived; and
- the manifest, inventory, optional orientation, report, and canonical
  artifact files are the editable sources.

Stop before the ADR. The developer-created pull request and its merge are the
adoption decision.

<!-- Source: https://github.com/nnennandukwe/software-standards-bootstrap · Author: Nnenna Ndukwe · Apache-2.0 -->

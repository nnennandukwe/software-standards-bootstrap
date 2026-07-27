---
name: software-standards-bootstrap
description: Generate an evidence-backed repository rules pack from a clean Git snapshot, or load the relevant subset of an existing pack for the current language, framework, task, and path. Use when a developer wants to extract or apply repository-specific coding standards, conventions, checks, or agent guidance.
license: Apache-2.0
compatibility: Requires the ssb CLI, Git 2.39 or newer, a commit-backed branch, and host access to read and write the target repository.
metadata:
  project: software-standards-bootstrap
  schema: ssb.dev/rule/v2
  topic: developer-experience
  version: 0.3.0
---

# Software Standards Bootstrap

Generate or consume a repository-specific proposal. Do not use a generic rules catalog. Do not execute repository code or verification commands merely because a rule cites them. Do not stage, commit, branch, push, open a pull request, or activate rules in another system.

Read [the rule schema](references/rule-schema.md), [the topic taxonomy](references/topics.md), [the evidence workflow](references/evidence-workflow.md), and [the structural-pattern workflow](references/structural-patterns.md) before writing generation proposal files. Read [the governed lifecycle review](references/prune-review.md) before writing a prune proposal.

## Choose the mode

If `.software-standards/rules/` does not exist, use generation mode. When it
does exist, route the developer's request before acting:

- use reviewed-pack maintenance mode only for an explicit request to validate
  or rerender developer-edited sources;
- use requested-ADR mode only after the developer explicitly asks for the
  adoption record;
- use governed lifecycle review mode only when the developer explicitly asks
  to assess an adopted pack for stale, redundant, contradictory, or
  unsupported rules or skills;
- stop on a regeneration request and explain that the existing pack must be
  reviewed and removed deliberately; never overwrite it; or
- otherwise use existing-pack consumption mode.

### Existing-pack consumption mode

In consumption mode, do not run `ssb inspect`, `ssb validate`, or `ssb render`,
and do not rewrite the pack. Read the Software Standards Bootstrap managed
section in root `AGENTS.md`. If the file or managed section is missing,
malformed, or points to a rule source that does not exist, stop and report the
pack as undiscoverable instead of guessing at selection metadata. Otherwise:

1. enumerate `.software-standards/rules/*.md` and read only each file's YAML
   frontmatter;
2. reconcile each canonical source ID and its selection metadata against every
   managed-index occurrence, stopping as stale if either side has a missing
   entry or the canonical frontmatter disagrees with the index; expect one
   contextual rule to occur under each of its lens values;
3. identify the affected repository-relative paths;
4. identify the request task as `implementation`, `review`, `testing`,
   `security`, `documentation`, or `release`;
5. identify languages and frameworks only from the request and repository
   evidence already available;
6. mark a base rule active when its path scope matches;
7. mark a contextual rule active when its path scope matches and every represented lens dimension matches, treating multiple values within one dimension as alternatives; and
8. when context is uncertain, load the potentially relevant rule instead of
   excluding it.

Only after selection, read the complete canonical source for every active rule,
including base rules; do not rely on an inline managed copy. Before applying
them, report the active rule IDs. Legacy v1 rules have no directive, so apply
their canonical bodies as written without inventing one. Treat every cited
command as mapped, not executed: do not claim it passed, and run it only when
the developer's request and the host's permissions independently authorize
execution.

Stop after fulfilling the current request. Consumption mode never changes the
standards pack or its projection.

### Reviewed-pack maintenance mode

Use this mode only when the developer explicitly asks to validate or rerender
sources they have reviewed, edited, or deleted. Do not run `ssb inspect` and do
not edit rule or skill sources. Run:

```bash
ssb validate --repo . --format text
ssb render --repo . --dry-run
ssb render --repo .
```

Stop on validation diagnostics without rendering. Otherwise, report every
changed and untracked path and restate that mapped commands were not executed.
Do not create an ADR unless it was separately and explicitly requested.

### Requested-ADR mode

Use this mode only after developer review and an explicit ADR request. Do not
run `ssb inspect`, rewrite source files, or infer adoption. Preview and create:

```bash
ssb adr --repo . --dry-run
ssb adr --repo .
```

If ADR conventions are ambiguous, stop and request the intended `--adr-dir`.
The ADR remains `Proposed`.

### Governed lifecycle review mode

Use this mode only for an explicit lifecycle-review request. `prune` is a
working concept, not permission to delete. Follow
[the governed lifecycle review contract](references/prune-review.md).

Require the developer to identify the local point-in-time capability profile
and optional provenance declaration. Never select an implicit latest model,
query an online registry, or treat release notes as conformance proof. Run:

```bash
ssb prune inspect --repo . --review <id> --capabilities <profile> [--provenance <manifest>]
```

Never use partial inventory for a prune proposal. Read `context.json`; evaluate
rules and skills separately and together; then write only `proposal.yaml` and
complete candidate files inside that review bundle. Do not edit canonical
rules, skills, `AGENTS.md`, or ADRs.

Validate without approving:

```bash
ssb prune validate --repo . --review <id> --format text
ssb prune status --repo . --review <id>
```

Report every disposition, rationale, evidence reference, confidence band, and
unresolved question. Stop for human review. Do not run `approve`, `apply`,
`render`, `adr`, or `verify` on the developer's behalf unless the developer
separately and explicitly requests that exact transition. Application remains a dry run unless `--write` is explicitly authorized.

## Generation mode

### 1. Establish the immutable input

From the target repository, run:

```bash
ssb inspect --repo . --format json
```

Never pass `--allow-partial` during this workflow. Stop if the command fails,
including exit `4` for incomplete coverage, and report its recovery guidance
verbatim. Do not create proposal files from incomplete inventory coverage. Do
not bypass a dirty, detached, unborn, non-Git, existing-pack, or missing-baseline
precondition.

Record:

- `baseline_commit`;
- candidate, scanned, indexed, and remaining counts and bytes;
- confirmation that `truncated` is false;
- excluded-category counts; and
- the safe tracked files available for targeted reads.

### 2. Perform targeted semantic reads

Read only inventory-listed paths, selecting exact sections relevant to repository conventions, architecture, risk, and existing checks. Never execute repository files, hooks, build scripts, tests, linters, formatters, package managers, or verification commands.

Perform both an authority-and-risk pass and the structural-pattern workflow. Complete the structural-pattern pass before scoring or writing candidates. Do not reject a structural candidate only because no repository policy states it explicitly: three consistent occurrences across at least two files are an alternative evidence threshold. Narrow an otherwise useful candidate to the evidence-backed scope instead of manufacturing a repository-wide rule.

Distinguish:

- repository context that belongs only in the assessment;
- durable declarative guidance that can become a rule;
- genuinely procedural work that belongs in an Agent Skill; and
- existing deterministic checks that can be cited but were not executed.

If an explicitly diagnostic partial inventory is supplied from outside this
workflow, disclose it and stop. Do not score candidates or write proposal
sources from it.

### 3. Write editable proposal sources

Create:

```text
.software-standards/assessment.md
.software-standards/rules/<rule-id>.md
.agents/skills/<skill-id>/SKILL.md
```

The assessment must name the baseline, complete inventory limits, repository context, evidence reviewed, the completed structural pattern review, candidates retained, candidates kept assessment-only, primary-topic rationale, and classification rationale.

Use the dynamic number of candidates supported by evidence. Do not impose a five-rule or other fixed cap. Keep candidates below 25 in the assessment.

Every emitted rule must:

- conform to `ssb.dev/rule/v2`;
- assign exactly one primary topic from the controlled taxonomy;
- declare one base lens or one or more language, framework, and task lenses;
- declare an `always`, `ask-first`, `never`, or `prefer` directive;
- use `ssb-score-v1` with visible factor arithmetic;
- have one authoritative source or three consistent occurrences across two files;
- cite exact one-based line ranges and SHA-256 hashes of the exact baseline bytes;
- use repository-relative scopes;
- declare honest confidence;
- classify `deterministic` only when an existing command and its defining source are cited;
- record full verification coverage and the bounded property proved for a deterministic rule;
- record partial verification coverage and the bounded property proved when guidance cites a check; and
- record a proof gap when guidance has no existing deterministic check.

Choose the topic that best explains the rule's engineering risk or change obligation. Use `quality` only when no narrower topic fits. Topic is independent of classification, importance, confidence, and scope.

Use evidence-backed preferred examples and counterexamples when the repository
contains them. Cite their exact locations in the rule body and evidence; never
invent an example or label ordinary legacy code as a counterexample without
repository evidence.

Create a portable Agent Skill only for a procedural workflow. Use core Agent Skills frontmatter and set `metadata.topic` to the workflow's one primary engineering outcome. Do not add consumer-specific fields to the portable source.

If a target already exists, stop instead of overwriting developer work.

### 4. Validate before projection

Run:

```bash
ssb validate --repo . --format text
```

If validation fails, keep the uncommitted source files, report each file-specific diagnostic, correct only proposal sources, and rerun validation. Do not render or create an ADR while diagnostics remain.

Then preview:

```bash
ssb render --repo . --dry-run
```

Review the bounded section and run:

```bash
ssb render --repo .
```

Do not edit inside the managed `AGENTS.md` section. Edit or delete rule source files and rerun instead.

The managed section is a progressive router: it inlines base standing orders
and links contextual rules by language, framework, and task. Confirm that a
contextual rule body appears only in its canonical source file.

### 5. Disclose the complete uncommitted result

Run a read-only status query that includes all untracked paths:

```bash
git --no-optional-locks status --short --untracked-files=all
```

List every changed and untracked path. State explicitly:

- no repository code or cited verification command was executed;
- no Git mutation was performed;
- `AGENTS.md` is derived;
- rule and skill files are the editable sources; and
- the developer should edit or delete sources and rerun validation/rendering.

### 6. Stop before the ADR

Do not run `ssb adr` as part of the initial generation workflow.

Wait for the developer to review retained files and explicitly request the ADR. After that request, preview and then create it:

```bash
ssb adr --repo . --dry-run
ssb adr --repo .
```

If multiple ADR conventions exist, report the ambiguity and use `--adr-dir PATH` only after the developer identifies the intended directory.

The ADR must remain `Proposed`. The developer-created pull request and its merge constitute adoption.

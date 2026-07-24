---
name: software-standards-bootstrap
description: Analyze a clean Git repository with targeted semantic and structural-pattern review, then generate an evidence-backed, scored rule pack plus portable procedural skills as uncommitted files. Use when a developer wants to extract repository-specific coding standards, best practices, conventions, linting rules, or agent guidance from existing code, documentation, architecture, and checks.
license: Apache-2.0
compatibility: Requires the ssb CLI, Git 2.39 or newer, a commit-backed branch, and host access to read and write the target repository.
metadata:
  project: software-standards-bootstrap
  schema: ssb.dev/rule/v1
  topic: developer-experience
  version: 0.1.0
---

# Software Standards Bootstrap

Create a repository-specific proposal. Do not use a generic rules catalog. Do not execute repository code or verification commands. Do not stage, commit, branch, push, open a pull request, or activate rules in another system.

Read [the rule schema](references/rule-schema.md), [the topic taxonomy](references/topics.md), [the evidence workflow](references/evidence-workflow.md), and [the structural-pattern workflow](references/structural-patterns.md) before writing proposal files.

## 1. Establish the immutable input

From the target repository, run:

```bash
ssb inspect --repo . --format json
```

Stop if the command fails. Report its recovery guidance verbatim. Do not bypass a dirty, detached, unborn, non-Git, existing-pack, or missing-baseline precondition.

Record:

- `baseline_commit`;
- whether coverage is truncated and the exact reason;
- excluded-category counts; and
- the safe tracked files available for targeted reads.

## 2. Perform targeted semantic reads

Read only inventory-listed paths, selecting exact sections relevant to repository conventions, architecture, risk, and existing checks. Never execute repository files, hooks, build scripts, tests, linters, formatters, package managers, or verification commands.

Perform both an authority-and-risk pass and the structural-pattern workflow. Complete the structural-pattern pass before scoring or writing candidates. Do not reject a structural candidate only because no repository policy states it explicitly: three consistent occurrences across at least two files are an alternative evidence threshold. Narrow an otherwise useful candidate to the evidence-backed scope instead of manufacturing a repository-wide rule.

Distinguish:

- repository context that belongs only in the assessment;
- durable declarative guidance that can become a rule;
- genuinely procedural work that belongs in an Agent Skill; and
- existing deterministic checks that can be cited but were not executed.

Disclose inventory truncation. Do not describe a truncated scan as complete.

## 3. Write editable proposal sources

Create:

```text
.software-standards/assessment.md
.software-standards/rules/<rule-id>.md
.agents/skills/<skill-id>/SKILL.md
```

The assessment must name the baseline, inventory limits or truncation, repository context, evidence reviewed, the completed structural pattern review, candidates retained, candidates kept assessment-only, primary-topic rationale, and classification rationale.

Use the dynamic number of candidates supported by evidence. Do not impose a five-rule or other fixed cap. Keep candidates below 25 in the assessment.

Every emitted rule must:

- conform to `ssb.dev/rule/v1`;
- assign exactly one primary topic from the controlled taxonomy;
- use `ssb-score-v1` with visible factor arithmetic;
- have one authoritative source or three consistent occurrences across two files;
- cite exact one-based line ranges and SHA-256 hashes of the exact baseline bytes;
- use repository-relative scopes;
- declare honest confidence;
- classify `deterministic` only when an existing command and its defining source are cited; and
- record a proof gap when guidance has no existing deterministic check.

Choose the topic that best explains the rule's engineering risk or change obligation. Use `quality` only when no narrower topic fits. Topic is independent of classification, importance, confidence, and scope.

Create a portable Agent Skill only for a procedural workflow. Use core Agent Skills frontmatter and set `metadata.topic` to the workflow's one primary engineering outcome. Do not add consumer-specific fields to the portable source.

If a target already exists, stop instead of overwriting developer work.

## 4. Validate before projection

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

## 5. Disclose the complete uncommitted result

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

## 6. Stop before the ADR

Do not run `ssb adr` as part of the initial generation workflow.

Wait for the developer to review retained files and explicitly request the ADR. After that request, preview and then create it:

```bash
ssb adr --repo . --dry-run
ssb adr --repo .
```

If multiple ADR conventions exist, report the ambiguity and use `--adr-dir PATH` only after the developer identifies the intended directory.

The ADR must remain `Proposed`. The developer-created pull request and its merge constitute adoption.

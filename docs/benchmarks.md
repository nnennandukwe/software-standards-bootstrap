# Pinned public benchmark evaluation

The v0.1 benchmark set is recorded in [`testdata/benchmarks.yaml`](../testdata/benchmarks.yaml). The pins were resolved from each repository's public `HEAD` on 2026-07-23.

| Repository | Commit | Evaluation role |
|---|---|---|
| `spf13/cobra` | `adbc8813901bba65827259daa8e22ff94ec1f30e` | Small Go repository |
| `pallets/flask` | `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81` | Medium Python repository |
| `django/django` | `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f` | Large repository |
| `vercel/next.js` | `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd` | TypeScript monorepo |

## Reproduction

Prepare each benchmark in its own fresh clone. The evaluator—not `ssb`—creates an attached local branch at the pin because `ssb inspect` intentionally rejects detached `HEAD`:

```bash
git clone <repository-url> <local-path>
git -C <local-path> switch --create ssb-evaluation <pinned-commit>
git -C <local-path> status --short
ssb inspect --repo <local-path> --format json
```

Do not install dependencies or execute repository code.

Then run the Agent Skill workflow described in [agent-smoke-tests.md](agent-smoke-tests.md). Keep proposal files uncommitted. Record:

Keep proposal generation, developer retention, rendering, ADR creation, and
release evidence as separate recorded states. Do not infer a later state from
an earlier artifact.

- consumer and version;
- operating system;
- exact pin;
- inventory schema, candidate/scanned/indexed counts and bytes, exclusions, and
  confirmation of complete coverage;
- counts for every emitted rule, verification recipe, Agent Skill, and
  automation proposal;
- confidence, utility, and category for each emitted artifact;
- structural-pattern and existing-check review coverage;
- evidence resolution result;
- keep/edit-and-keep/defer/reject judgment;
- complete changed-path list; and
- whether any forbidden Git or repository-code action occurred.

Orientation is repository context, not an actionable artifact. It may be
recorded separately for a fixture but does not enter the artifact denominator
or ADR eligibility.

## Acceptance

- 100% of emitted artifacts have resolvable evidence.
- The structural-pattern review covers all five required dimensions and the
  existing-command and automatic-enforcement review.
- Every accepted candidate has exactly one actionable artifact kind.
- At least 70% of final artifacts are judged “keep” or “edit and keep” in each
  pinned repository independently. A pooled cross-repository average is not
  acceptance, because it lets a strong fixture mask a weak one. The denominator
  is every final artifact emitted for that fixture before developer review;
  deferred and rejected artifacts remain in the denominator, and the exact
  fraction must meet the threshold without rounding.
- Inventory v2 completes all four pinned repositories under its defaults.
- Any truncated run blocks proposal generation and is not benchmark acceptance.
- Procedural candidates become Agent Skills.
- Existing deliberately invoked commands become recipes without execution.
- Proposed checks remain automation proposals and never enter `AGENTS.md` or
  the ADR.
- Every artifact uses one supported category, and projection/ADR output
  preserves the applicable category.
- Artifact edits and deletions, together with report updates, propagate to
  `AGENTS.md` and the ADR.
- At least one conforming agent host finishes the workflow through its
  documented skill location, recorded with its exact version.

The benchmark pins are evaluation inputs, not runtime dependencies. `ssb` never clones or contacts these repositories.

## Recorded results

The public-safe result ledger for the 2026-07-23 release evaluation is in
[`docs/benchmarks/results/2026-07-23/`](benchmarks/results/2026-07-23/README.md).
Each record separates a generated proposal from developer acceptance and
discloses truncation, evidence resolution, structural dispositions, changed
paths, and forbidden-action checks.

Those files remain immutable historical evidence for their recorded contract.
They are not actionable-artifact acceptance. Record a fresh blind pass over all
four pins separately.

The [2026-07-29 actionable gate record](benchmarks/results/2026-07-29-actionable/README.md)
captures a fresh complete inventory pass over all four pins. Proposal
generation and developer retention remain explicitly open in that ledger.

The [2026-07-31 actionable host acceptance](benchmarks/results/2026-07-31-actionable/README.md)
records a fresh isolated Codex CLI 0.145.0 pass through proposal generation,
per-artifact developer retention, rendering, edit/delete propagation, and
requested `Proposed` ADR creation on all four pins. Release tagging and
publication remain separate gates.

The [2026-08-12 Hoop AGENTS contract](benchmarks/results/2026-08-12-hoop-agents-contract/run.yaml)
records a Codex CLI 0.145.0 and `gpt-5.6-sol` behavioral conformance run for
orientation, planning/implementation/verification routing, action-first rules,
and exact inert verification commands. Its source commit, target pin, host,
proposal inputs, routing outputs, and source/content/output digests are bound
byte-for-byte. The retained projection is `proposal/AGENTS.proposed.md`, never
a nested active `AGENTS.md`.

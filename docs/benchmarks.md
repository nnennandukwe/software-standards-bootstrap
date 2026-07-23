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

- consumer and version;
- operating system;
- exact pin;
- inventory file count, excluded counts, indexed bytes, and truncation;
- candidate scores and classifications;
- evidence resolution result;
- keep/edit-and-keep/defer/reject judgment;
- complete changed-path list; and
- whether any forbidden Git or repository-code action occurred.

## Acceptance

- 100% of emitted rules have resolvable evidence.
- At least 70% of high and very-high candidates are judged “keep” or “edit and keep.”
- Truncation is visible wherever a bound is reached.
- Procedural candidates become Agent Skills.
- Existing commands are mapped without execution.
- Rule edits and deletions propagate to `AGENTS.md` and the ADR.
- Codex and Claude Code both finish the workflow through their documented skill locations.

The benchmark pins are evaluation inputs, not runtime dependencies. `ssb` never clones or contacts these repositories.

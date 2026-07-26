# Next.js / Claude Code inventory-v2 proposal record

Generated on 2026-07-26. This record is at the mandatory developer-review gate;
it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.220
- Model: reported by the session environment as Opus 5 (1M context), model id
  `claude-opus-5[1m]`. **Not independently observable** from `claude --version`;
  recorded as self-reported rather than verified.
- Observable configuration: `learning` output style active. No reasoning-effort
  setting is exposed by the CLI, so none is claimed.
- Host: macOS 15.7.3 build 24G419 (`arm64`)
- Git: 2.39.5 (Apple Git-154)
- Repository: `github.com/vercel/next.js`
- Baseline commit: `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`
- Evaluation branch: `ssb-claude-v2-evaluation`
- SSB source commit: `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Evaluator binary SHA-256:
  `c9d93a2aeef27249a1fc828fef025e77f6f0dd742d847bc4927cd8c538fd6fba`
- Raw inventory output SHA-256:
  `791e4b443dad3b80eed3cd85e194f0c09fee2a2e6f222ec24456a8a18addc180`

The Codex inventory-v2 Next.js pack was not read before this evaluation. This run
is independent.

## Inventory

- Contract: `ssb-inventory-v2`, schema 2
- Candidate: 29,073 files, 111,110,455 bytes
- Scanned: 29,073 files, 111,110,455 bytes
- Indexed: 28,403 files, 88,643,646 bytes
- Remaining: 0 files, 0 bytes
- Truncated: no
- Limits: 40,000 candidate files; 134,217,728 candidate bytes; 1,048,576 bytes
  per file
- Excluded: 1,060 vendor-or-generated tree, 652 binary, 113 secret-like, 29
  symlink, 21 oversized, 18 generated

This is the repository where the inventory contract change mattered most. Under
the previous contract the scan truncated inside `packages/next/src/compiled/`,
leaving core source, `test/`, and `turbopack/` unindexed. All 28,403 files are
indexed here.

## Pending proposal

`ssb validate` passed with 8 rules, 0 related skills, and 100% evidence
resolution. Validation passed on the first attempt with no corrections.

| Rule | Primary topic | Classification | Score | Decision |
|---|---|---|---:|---|
| `guard-trusted-types-exemptions` | security | guidance | 80 | Pending |
| `keep-toolchain-pins-synchronized` | compatibility | guidance | 78 | Pending |
| `run-repository-lint-gate` | maintainability | deterministic | 78 | Pending |
| `sign-all-commits` | security | guidance | 74 | Pending |
| `protect-secrets-in-agent-workflows` | security | guidance | 74 | Pending |
| `rebuild-before-integration-tests` | correctness | guidance | 74 | Pending |
| `attach-error-links-to-new-errors` | documentation | guidance | 70 | Pending |
| `leave-pull-requests-in-draft` | developer-experience | guidance | 62 | Pending |

**No skill was proposed.** The repository already carries 17 Agent Skills plus an
authoring guide under `.agents/skills/`, tracked in `skills-lock.json`, and a
roughly 450-line root `AGENTS.md`. Its own authoring guide states that a skill
should not be created for one-liner rules or guardrails, nor for content every
session needs. Every procedural workflow this evaluation identified is already
covered by an existing skill — `flags`, `dce-edge`, `react-vendoring`,
`runtime-debug`, `create-pr`, `pr-status-triage`, `gh-stack`, `backport-pr`,
`update-docs`, `write-guide`, `write-api-reference`, `insight-error-page`,
`next-rspack`, or `v8-jit`. Generating a duplicate would introduce drift.

The one workflow without a dedicated existing skill is adding a `tsec`
exemption. It was deliberately expressed as a rule rather than a skill because
the obligation is a declarative constraint on a boundary file rather than a
multi-step procedure.

## Structural review and candidate dispositions

All five dimensions were reviewed and recorded in
`.software-standards/assessment.md`.

| Dimension | Disposition |
|---|---|
| Package and dependency boundaries | Assessment-only. The 996-file `packages/next/src/compiled/` vendoring boundary is already owned by the `react-vendoring` skill and the `AGENTS.md` anti-patterns section. |
| Parallel implementation families | Emitted `attach-error-links-to-new-errors` for the 254-document `errors/` family. The Webpack/Turbopack bundler family was left to existing guidance. |
| Platform and configuration seams | Emitted `keep-toolchain-pins-synchronized`, the strongest structural finding of the pass. |
| Public compatibility surfaces | Assessment-only. Flag plumbing belongs to the `flags` skill and release-channel policy is documented under `contributing/repository/`. |
| Source, test, and documentation symmetry | Emitted `rebuild-before-integration-tests`. `test/` is organized by mode rather than mirroring source paths, so no one-to-one pairing exists to enforce. |

The `keep-toolchain-pins-synchronized` finding rests on the repository stating
the coupling itself: `rust-toolchain.toml` records in its own comments that
changing the channel also requires updating the devcontainer Rust feature
definition, and that moving the file requires updating `turbo.json` inputs that
reference it. Nothing checks either coupling.

## Rendered output

Next.js has a **pre-existing tracked `AGENTS.md`**. `ssb render` appended only
the bounded generated block:

- Before: 512 lines, 25,280 bytes, SHA-256
  `513c9c6e439505e9d6f06ecafd748b7f52e8da2192a89683086720ec3d696466`
- After: 618 lines, 34,460 bytes
- `git diff --stat` reports 106 insertions and 0 deletions
- The original 512 lines were verified byte-identical to `HEAD:AGENTS.md` after
  rendering

Digests:

- source digest:
  `sha256:5c5c968b43652b8e8c45814876b70270e338e79954ca61aaecd06420c5d2a229`
- content digest:
  `sha256:65a05668bd641890f021f84fe1cd74e6088aaff0e3ab6b0be9532dcc9d41bb0c`

`ssb validate` was run again after rendering and passed.

## Proposed paths

```text
.software-standards/assessment.md
.software-standards/rules/attach-error-links-to-new-errors.md
.software-standards/rules/guard-trusted-types-exemptions.md
.software-standards/rules/keep-toolchain-pins-synchronized.md
.software-standards/rules/leave-pull-requests-in-draft.md
.software-standards/rules/protect-secrets-in-agent-workflows.md
.software-standards/rules/rebuild-before-integration-tests.md
.software-standards/rules/run-repository-lint-gate.md
.software-standards/rules/sign-all-commits.md
AGENTS.md  (pre-existing tracked file; append-only modification)
```

Because this repository's `.claude/skills` is a tracked symlink to
`../.agents/skills`, the evaluator's own skill attachment appears in status as
`.agents/skills/software-standards-bootstrap`. It is untracked and is not part of
the proposal.

## Safety and review boundary

- The clone was fresh and `HEAD` remained at
  `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd` throughout.
- No Next.js code, test, hook, build script, linter, formatter, or package
  manager was executed. `pnpm lint`, `pnpm types`, `pnpm build-all`,
  `pnpm new-error`, and the prettier and eslint pre-commit invocations are cited
  only.
- No Next.js dependency was installed; `pnpm install` was never run.
- The Git index is clean: 0 staged paths. One tracked file is modified —
  `AGENTS.md`, the intended render target — with append-only changes as recorded
  above. No other pre-existing tracked file was edited, and no pre-existing skill
  was overwritten.
- No Git mutation beyond creating the initial attached evaluation branch.
- No ADR was previewed or created, and `ssb adr` was never run.

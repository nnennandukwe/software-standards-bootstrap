# 2026-07-29 Claude Code `rule/v2` evidence

A single-consumer Claude Code pass against the four repositories pinned in
[`testdata/benchmarks.yaml`](../../../../testdata/benchmarks.yaml), run at
`origin/main` and emitting `ssb.dev/rule/v2`. This is release evidence, not adopted
policy for the evaluated repositories.

The developer review instrument for this pass is
[`retention-ledger.md`](retention-ledger.md), which carries all 37 rules with their
bodies, evidence, and proof coverage, and an empty decision cell for each.

## Why this pass exists

The [2026-07-26 records](../2026-07-26/README.md) are valid `ssb-inventory-v2`
evidence, but they emitted `ssb.dev/rule/v1` because v1 was the only rule schema in
existence at their SSB source commit `820c3a8` — `rule/v2` appears nowhere in that
tree. The progressive standards pack (`f55cb40`, merged via PR #9) subsequently
introduced `rule/v2`, made it the documented emission format, and rewrote the
renderer. `main` is now `04bc830`.

Two consequences followed, neither a defect in the earlier evidence:

- step 8 of [the conformance suite](../../../agent-smoke-tests.md) now requires newly
  emitted rules to use v2 with valid lenses and directive, so a conformance claim
  against the shipped contract needs a pass at `main`; and
- `main`'s renderer projects v1 rules into a *Legacy v1 rules (directive not
  recorded)* group, so the `AGENTS.md` content digests recorded in the 2026-07-26
  per-run files no longer reproduce.

This pass supplies v2 evidence against the current contract. It does not supersede
the 2026-07-26 records, which remain the inventory-v2 evidence for two consumers.

## Immutable evaluator input

- SSB source commit: `04bc830`
- Evaluator binary SHA-256:
  `88adf70d853e1ef34ab7392d02649dfc897ceb556cf6633ae4ef500c7543145d`
- Build: `go build -trimpath -o ssb ./cmd/ssb` with Go 1.26.5 (`darwin/arm64`)
- Inventory contract: `ssb-inventory-v2`, schema 2
- Rule schema: `ssb.dev/rule/v2`
- Inventory limits: 40,000 candidate files, 134,217,728 candidate bytes, and
  1,048,576 bytes per file
- Host: macOS 15.7.3 build 24G419 (`arm64`)
- `go test ./...`, `go vet ./...`, and `go build` all pass at `04bc830`

### The evaluator digest is reproducible; the 2026-07-26 one is not

The digest above was produced with `-trimpath` and confirmed byte-identical across
two different build directories.

The `c9d93a2a…` digest recorded for the 2026-07-26 pass cannot be verified by anyone.
It came from a plain `go build`, which embeds absolute source paths into the binary:
building `820c3a8` at two different paths yields `7e7ed6f4…` and `1d6001c2…`. The
original binary has since been overwritten and exists nowhere on disk. Future runs
should record a `-trimpath` digest so the claim is checkable.

## Consumer

- Consumer: Claude Code 2.1.220 — the same host version as the 2026-07-26 Claude pass
- Model: reported by the session environment as Opus 5 (1M context), model id
  `claude-opus-5[1m]`. Not independently observable from `claude --version`, which
  prints only the CLI version; recorded as self-reported rather than verified.
- Observable configuration: `learning` output style active. No reasoning-effort
  setting is exposed by the CLI, so none is claimed.
- Evaluation branch: `ssb-claude-v3-evaluation`

| Repository | Inventory | Rules | New skills | Evidence | Developer decisions |
|---|---|---:|---:|---|---|
| Cobra | Complete | 8 | 1 | 19 items, 0 diagnostics | 8 pending |
| Flask | Complete | 8 | 1 | 18 items, 0 diagnostics | 8 pending |
| Django | Complete | 9 | 1 | 22 items, 0 diagnostics | 9 pending |
| Next.js | Complete | 12 | 0 | 22 items, 0 diagnostics | 12 pending |
| **Total** | **4/4** | **37** | **3** | **81 items, 0 diagnostics** | **37 pending** |

## Independence limitation

**This pass is not blind for any of the four repositories.** Before it ran, the
operating session had read a v1 rule body in full and had generated a review ledger
containing all 66 rule ids, bodies, and evidence from the 2026-07-26 pass, covering
every fixture.

The 2026-07-26 record carried this qualification for Cobra alone. Here it applies to
all four, and it is the most significant limitation on this evidence. Every rule was
derived from reads of the pinned files and every digest was computed from the pinned
Git blob, but the rule *set* is not an independent second opinion, and its overlap
with the v1 sets is not corroboration.

A genuinely blind v2 pass would require a session with no prior exposure to the v1
rule sets.

## Schema conformance

Verified mechanically across all 37 rules: every rule declares `ssb.dev/rule/v2`, a
directive, and at least one lens; every `importance` matches its score band; every
score's factors sum to its stated total; all 7 `deterministic` rules carry an
existing command, a cited defining source, `coverage: full`, and a bounded `proves`
statement.

Distribution: `always` 28, `never` 7, `ask-first` 2. Classification: `guidance` 30,
`deterministic` 7. Of the guidance rules, 2 cite a command with `coverage: partial`
and 28 record a `proof_gap`.

Of 81 evidence items, 53 are marked authoritative. Four rules carry no authoritative
source and rest on the alternative threshold of three consistent occurrences across
at least two files.

Lens usage: `base` 12, `language:python` 6, `language:go` 5, `task:testing` 5,
`task:implementation` 4, `task:security` 4, `language:typescript` 3,
`task:documentation` 1, `task:release` 1, `task:review` 1. The 12 base rules render
inline as standing orders; the remaining 25 are contextual and load only when every
represented lens dimension matches.

## Review boundary

For every run: no evaluated repository code, hook, test, linter, formatter, build
script, package manager, or cited verification command was executed, and no target
dependency was installed. No database was started. `ssb validate` and `ssb render`
succeeded for every proposal, and `ssb validate` was rerun after rendering. No ADR was
previewed or created, and `ssb adr` was never run.

Every target repository's `HEAD` is at its pin with a clean index. Proposal sources
are uncommitted on attached `ssb-claude-v3-evaluation` branches. `--allow-partial` was
never passed; `ssb inspect` exited `0` for all four.

One tracked file is modified across the four runs: Next.js's pre-existing
`AGENTS.md`, which is the intended render target. It received 170 insertions and 0
deletions, and the original 512 lines were verified byte-identical afterwards
(`513c9c6e439505e9d6f06ecafd748b7f52e8da2192a89683086720ec3d696466` both before and
after the render).

The clones were created from local mirrors of the upstream repositories rather than
fetched fresh from GitHub. Each `HEAD` was verified equal to its pinned commit, which
is what fixes the content.

## Remaining gates

This evidence does not satisfy the acceptance threshold. Still outstanding:

- a developer must record keep, edit-and-keep, defer, or reject for all 37 proposed
  rules and the 3 proposed skills;
- at least 70% of the 36 high and very-high candidates must be kept or
  edit-and-kept — 26 of 36;
- the edit, delete, rerender, and explicitly requested ADR behavior
  (steps 12–18 of the conformance suite) remains unverified for this pass;
- a genuinely blind pass is needed if independence is to be claimed;
- Codex-side conformance was dropped for this pass by developer instruction, so no
  second-consumer evidence exists at `rule/v2`; and
- a signed `v0.1.0` tag and published-artifact verification remain pending.

Proposal completion is not developer retention, and neither is end-to-end acceptance.

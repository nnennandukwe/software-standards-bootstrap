# 2026-07-26 inventory-v2 evidence

These records capture two independent consumer passes — Codex and Claude Code —
against the public repositories pinned in
[`testdata/benchmarks.yaml`](../../../../testdata/benchmarks.yaml), plus the
native Linux amd64 resource-envelope run. They are release evidence, not adopted
policy for the evaluated repositories.

## Immutable evaluator input

Shared by both consumer passes:

- SSB source commit:
  `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Evaluator binary SHA-256:
  `c9d93a2aeef27249a1fc828fef025e77f6f0dd742d847bc4927cd8c538fd6fba`
- Inventory contract: `ssb-inventory-v2`, schema 2
- Inventory limits: 40,000 candidate files, 134,217,728 candidate bytes, and
  1,048,576 bytes per file
- Evaluation host: macOS 15.7.3 build 24G419 (`arm64`)
- Git: 2.39.5 (Apple Git-154)
- Go: 1.26.5 (`darwin/arm64`)

## Codex pass

- Consumer: Codex desktop 26.721.31836 (build 5828), Codex CLI
  0.146.0-alpha.3.1
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Evaluation branch: `ssb-codex-v2-evaluation`

| Repository | Inventory | Rules | New skills | Evidence | Developer decisions |
|---|---:|---:|---:|---:|---|
| Cobra | Complete | 7 | 1 | 100% resolved | 7 pending |
| Flask | Complete | 7 | 1 | 100% resolved | 7 pending |
| Django | Complete | 9 | 1 | 100% resolved | 9 pending |
| Next.js | Complete | 10 | 0 | 100% resolved | 10 pending |
| **Total** | **4/4** | **33** | **3** | **100% resolved** | **33 pending** |

Records: [Cobra](codex-cobra.md) · [Flask](codex-flask.md) ·
[Django](codex-django.md) · [Next.js](codex-nextjs.md)

## Claude Code pass

- Consumer: Claude Code 2.1.220
- Model: reported by the session environment as Opus 5 (1M context), model id
  `claude-opus-5[1m]`. Not independently observable from `claude --version`,
  which prints only the CLI version; recorded as self-reported rather than
  verified.
- Observable configuration: `learning` output style active. No reasoning-effort
  setting is exposed by the CLI, so none is claimed.
- Evaluation branch: `ssb-claude-v2-evaluation`

| Repository | Inventory | Rules | New skills | Evidence | Developer decisions |
|---|---:|---:|---:|---:|---|
| Cobra | Complete | 8 | 1 | 100% resolved | 8 pending |
| Flask | Complete | 8 | 1 | 100% resolved | 8 pending |
| Django | Complete | 9 | 1 | 100% resolved | 9 pending |
| Next.js | Complete | 8 | 0 | 100% resolved | 8 pending |
| **Total** | **4/4** | **33** | **3** | **100% resolved** | **33 pending** |

Records: [Cobra](claude-cobra.md) · [Flask](claude-flask.md) ·
[Django](claude-django.md) · [Next.js](claude-nextjs.md)

## Consumer comparison

Both passes emitted 33 rules and 3 skills in total, but the per-repository
distribution and the rule sets themselves differ. Neither pass is the reference
answer for the other.

| Property | Codex | Claude Code |
|---|---|---|
| Total rules | 33 | 33 |
| Total proposed skills | 3 | 3 |
| Repository with no proposed skill | Next.js | Next.js |
| Validation corrections needed | 0 | 1 (Flask YAML quoting) |
| Inventory results | Identical to Claude | Identical to Codex |

Both consumers independently declined to propose a skill for Next.js, each
citing the repository's pre-existing skill set.

Raw inventory output digests are byte-identical between the two passes for the
repositories where both recorded them, which is the expected determinism result
for the same evaluator binary against the same pin and is not evidence of shared
work.

### Independence limitation

The Claude Code **Cobra** record carries a documented qualification: the
operating session had read the Codex Cobra rule ids before that evaluation ran,
so it is not a blind run. The Flask, Django, and Next.js Claude evaluations did
not read their Codex counterparts and carry no such qualification. See
[`claude-cobra.md`](claude-cobra.md) for the full statement.

## Review boundary

For both passes: no evaluated repository code, hook, test, linter, build script,
package manager, or cited verification command was executed, and no target
dependency was installed. Proposal sources remain uncommitted in fresh clones on
attached branches at the exact benchmark pins. `ssb validate` and `ssb render`
succeeded for every proposal, and `ssb validate` was rerun after rendering. No
ADR was previewed or created in any run.

Every target repository's index is clean with `HEAD` at its pin. One tracked file
is modified across all eight runs: Next.js's pre-existing `AGENTS.md`, in both
passes, which is the intended render target and received append-only changes. The
Claude record verifies the pre-existing 512 lines remained byte-identical.

## Remaining gates

This evidence does not satisfy the acceptance threshold. Still outstanding:

- a developer must record keep, edit-and-keep, defer, or reject for all 66
  proposed rules across both passes;
- the six proposed skills must be explicitly reviewed;
- at least 70% of high and very-high candidates must be kept or edit-and-kept;
- the edit, delete, rerender, and explicitly requested ADR behavior must be
  verified for both consumers; and
- a signed `v0.1.0` tag and published-artifact verification remain pending.

Proposal completion is not developer retention, and neither is end-to-end
acceptance.

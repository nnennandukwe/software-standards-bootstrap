# 2026-07-26 inventory-v2 evidence

These records capture four fresh Codex proposal runs against the public
repositories pinned in
[`testdata/benchmarks.yaml`](../../../../testdata/benchmarks.yaml), plus the
native Linux amd64 resource-envelope run. They are release evidence, not
adopted policy for the evaluated repositories.

## Immutable evaluator input

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
- Consumer: Codex desktop 26.721.31836 (build 5828), Codex CLI
  0.146.0-alpha.3.1
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`

## Review status

| Repository | Inventory | Rules | New skills | Evidence | Developer decisions |
|---|---:|---:|---:|---:|---|
| Cobra | Complete | 7 | 1 | 100% resolved | 7 pending |
| Flask | Complete | 7 | 1 | 100% resolved | 7 pending |
| Django | Complete | 9 | 1 | 100% resolved | 9 pending |
| Next.js | Complete | 10 | 0 | 100% resolved | 10 pending |
| **Total** | **4/4** | **33** | **3** | **100% resolved** | **33 pending** |

All four inventories completed without truncation and with zero remaining
candidate files or bytes. Every proposal stops at the mandatory
developer-review gate. No retention decision from an earlier inventory-v1 run
was carried into these fresh proposals.

The reviewable decisions are in:

- [Cobra / Codex](codex-cobra.md)
- [Flask / Codex](codex-flask.md)
- [Django / Codex](codex-django.md)
- [Next.js / Codex](codex-nextjs.md)
- [Linux amd64 resource envelope](linux-amd64-resource-envelope.md)

## Review boundary

No evaluated repository code, hook, test, linter, build script, package
manager, or cited verification command was executed. Proposal sources remain
uncommitted in fresh clones on attached branches at the exact benchmark pins.
`ssb validate` and `ssb render` succeeded for every proposal. No ADR was
previewed or created.

This evidence does not satisfy the acceptance threshold until a developer
records keep, edit-and-keep, defer, or reject for every rule and explicitly
reviews the three proposed skills. Claude Code inventory-v2 proposal records
also remain a separate release gate.

# Next.js / Codex proposal record

Generated on 2026-07-23. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.715.71837 (build 5702)
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/vercel/next.js`
- Baseline commit: `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`
- Evaluation branch: `ssb-evaluation`

## Inventory

- Contract: `ssb-inventory-v1`
- Safe tracked files: 8,287
- Indexed bytes: 25,974,724
- Limits: 20,000 files; 25 MiB total; 1 MiB per file
- Truncated: yes, at the total-byte limit
- Excluded: 321 binary; 8 generated; 21 oversized; 113 secret-like;
  29 symlinks; 1,060 vendor/generated-tree; all other categories 0
- Indexed coverage ends in `packages/next/src/compiled/`; later source, `test/`,
  and `turbopack/` paths were not used for negative conclusions.

## Proposal

Validation passed with 10 rules, no new procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `verify-across-nextjs-modes` | correctness | guidance | 92 | very-high | Pending |
| `wire-feature-flags-end-to-end` | architecture | guidance | 90 | very-high | Pending |
| `preserve-edge-dce-boundaries` | architecture | guidance | 88 | very-high | Pending |
| `preserve-react-server-vendoring-boundary` | architecture | guidance | 87 | very-high | Pending |
| `filter-internal-request-headers` | security | guidance | 85 | very-high | Pending |
| `run-nextjs-lint-gate` | quality | deterministic | 85 | very-high | Pending |
| `generate-isolated-regression-tests` | testability | guidance | 84 | very-high | Pending |
| `use-pinned-node-pnpm-toolchain` | compatibility | guidance | 80 | very-high | Pending |
| `preserve-source-generated-output-boundary` | maintainability | guidance | 78 | high | Pending |
| `attach-helpful-error-links` | developer-experience | guidance | 72 | high | Pending |

No new skill was created because the repository already has targeted skills for
the procedural candidates; duplicating them would introduce drift.

The structural review covered all five required dimensions. It emitted package
and dependency boundaries, parallel runtime/build families, configuration and
feature-flag seams, public request/error compatibility surfaces, and
repository-specific source/test/generated-output symmetry. It disclosed the
truncation boundary rather than making whole-repository negative claims.

The existing root `AGENTS.md` was preserved and received a bounded section with
source digest
`sha256:0a18fbee3854217fc8d4509a048b82c5c6c0f086f6403d2a298fbdfdabe81bae`
and content digest
`sha256:14204269242529887a7b3c92f7195aa8d0ec55ac7f3ca15751c40829967f98dd`.

## Changed and untracked paths

```text
AGENTS.md (tracked file modified only by the bounded rendered section)
.software-standards/assessment.md
.software-standards/rules/attach-helpful-error-links.md
.software-standards/rules/filter-internal-request-headers.md
.software-standards/rules/generate-isolated-regression-tests.md
.software-standards/rules/preserve-edge-dce-boundaries.md
.software-standards/rules/preserve-react-server-vendoring-boundary.md
.software-standards/rules/preserve-source-generated-output-boundary.md
.software-standards/rules/run-nextjs-lint-gate.md
.software-standards/rules/use-pinned-node-pnpm-toolchain.md
.software-standards/rules/verify-across-nextjs-modes.md
.software-standards/rules/wire-feature-flags-end-to-end.md
```

## Safety and review boundary

- No Next.js code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index remained unchanged.
- SSB performed no network or Git mutation.
- Rule sources remained untracked. The only tracked worktree change was the
  derived bounded section in `AGENTS.md`.
- No ADR was previewed or created.

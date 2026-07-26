# Next.js / Codex inventory-v2 proposal record

Generated on 2026-07-26. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.721.31836 (build 5828), Codex CLI
  0.146.0-alpha.3.1
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/vercel/next.js`
- Baseline commit: `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`
- Evaluation branch: `ssb-codex-v2-evaluation`
- SSB source commit: `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Raw inventory output SHA-256:
  `791e4b443dad3b80eed3cd85e194f0c09fee2a2e6f222ec24456a8a18addc180`

## Inventory

- Contract: `ssb-inventory-v2`, schema 2
- Candidate and scanned coverage: 29,073 files, 111,110,455 bytes
- Indexed coverage: 28,403 files, 88,643,646 bytes
- Limits: 40,000 files; 128 MiB total; 1 MiB per file
- Remaining: 0 files, 0 bytes
- Truncated: no
- Excluded: 652 binary, 18 generated, 21 oversized, 113 secret-like,
  29 symlinks, and 1,060 files in vendor or generated trees; all other
  categories 0

## Pending proposal

Validation passed with 10 rules, no new procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Decision |
|---|---|---|---:|---|
| `verify-across-nextjs-modes` | correctness | guidance | 92 | Pending |
| `wire-feature-flags-end-to-end` | architecture | guidance | 90 | Pending |
| `preserve-edge-dce-boundaries` | architecture | guidance | 88 | Pending |
| `preserve-react-server-vendoring-boundary` | architecture | guidance | 87 | Pending |
| `filter-internal-request-headers` | security | guidance | 85 | Pending |
| `run-nextjs-lint-gate` | quality | deterministic | 85 | Pending |
| `generate-isolated-regression-tests` | testability | guidance | 84 | Pending |
| `use-pinned-node-pnpm-toolchain` | compatibility | guidance | 80 | Pending |
| `preserve-source-generated-output-boundary` | maintainability | guidance | 78 | Pending |
| `attach-helpful-error-links` | developer-experience | guidance | 72 | Pending |

No new skill was created because the target already has focused skills for
flags, edge DCE, React vendoring, documentation, runtime debugging, Rspack,
test workflows, and PR-status triage.

The assessment recorded all five structural dimensions. It emitted the
source/generated and React vendoring boundaries, runtime and bundler families,
feature-flag and edge seams, request/toolchain/error compatibility surfaces,
and isolated test symmetry. General monorepo dependencies and broad
documentation synchronization remained assessment-only.

The pre-existing root `AGENTS.md` was preserved outside the bounded managed
section. The rendered section has source digest
`sha256:64bb3d582dc322d364e1e464a235e571758d127c6d58dfa0881317a82d5aef10`
and content digest
`sha256:ba0bb73258013163cb5ed43838bfccd4b3e1a0d5b882b5ce0fa954af9a0908de`.

## Proposed paths

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

The evaluator-only `software-standards-bootstrap` skill attachment is untracked
but is not part of the proposal.

## Safety and review boundary

- The clone was fresh and `HEAD` stayed at the benchmark pin.
- No Next.js code, test, hook, build script, linter, package manager, or cited
  verification command was executed.
- Proposal sources and derived output remain uncommitted; the index is clean.
- No ADR was previewed or created.

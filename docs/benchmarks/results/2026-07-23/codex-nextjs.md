# Next.js / Codex proposal record

Proposal generated on 2026-07-23. Developer retention decisions were recorded
on 2026-07-26; see [Developer review](#developer-review). All proposed rules
were approved as Keep. Edit/delete/rerender propagation and the explicitly
requested ADR remain unverified.

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

## Developer review

Decisions below are the developer's judgment, not generator output. On
2026-07-26, the developer approved every pending rule as **Keep**.

**High-band retention: 10 of 10 (100%).** No related skill was generated.

Evidence paths are relative to Next.js baseline `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`.

### High-band rules (count toward the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `verify-across-nextjs-modes` | 92 | `AGENTS.md:71-105`, `AGENTS.md:114-166`, `package.json:21-81` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `wire-feature-flags-end-to-end` | 90 | `AGENTS.md:464-471`, `.agents/skills/flags/SKILL.md:17-39` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `preserve-edge-dce-boundaries` | 88 | `AGENTS.md:464-471`, `.agents/skills/dce-edge/SKILL.md:18-62` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `preserve-react-server-vendoring-boundary` | 87 | `AGENTS.md:464-471`, `.agents/skills/react-vendoring/SKILL.md:19-33`, `.agents/skills/react-vendoring/SKILL.md:64-66` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `filter-internal-request-headers` | 85 | `AGENTS.md:508-512` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `run-nextjs-lint-gate` | 85 | `AGENTS.md:227-235` | Yes | `pnpm lint` | Keep | Approved as written by the developer on 2026-07-26. |
| `generate-isolated-regression-tests` | 84 | `AGENTS.md:150-225`, `contributing/core/testing.md:44-78` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `use-pinned-node-pnpm-toolchain` | 80 | `package.json:306-310`, `.node-version:1-1`, `contributing/core/developing.md:7-29` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `preserve-source-generated-output-boundary` | 78 | `AGENTS.md:22-46`, `AGENTS.md:396-412`, `contributing/core/building.md:14-30` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `attach-helpful-error-links` | 72 | `contributing/core/adding-error-links.md:1-12`, `package.json:8-15` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |

### Related skill

No skill was generated for this pack; no skill decision is required.

### What each rule asks

- **`verify-across-nextjs-modes`** (92, very-high) — Rebuild the affected packages before integration tests, then run focused tests in each runtime mode and bundler the change can affect. Match CI environment flags when reproducing a failure; do not treat a passing default Turbopack development test as proof for Webpack, Rspack, production, deploy, or experimental variants.
- **`wire-feature-flags-end-to-end`** (90, very-high) — When adding or changing a framework flag, follow `.agents/skills/flags/SKILL.md` and wire every applicable surface: configuration type and schema, user-bundle definition, runtime configuration, server and export startup, bundle definitions, and runtime selection. Distinguish user-bundled code from precompiled runtime bundles before choosing the wiring path.
- **`preserve-edge-dce-boundaries`** (88, very-high) — Keep platform-only `require()` calls inside compile-time `if/else` branches that the bundler can eliminate. Force flags guarding Node-only imports to false for edge builds, do not use `NEXT_RUNTIME` as a feature flag, and verify affected paths with the isolated Webpack edge test prescribed by `.agents/skills/dce-edge/SKILL.md`.
- **`preserve-react-server-vendoring-boundary`** (87, very-high) — Route all `react-server-dom-webpack/*` server and static APIs through `entry-base.ts`; access them elsewhere through the exposed component module. When adding vendored React APIs, update the internal declarations and affected stable, experimental, Webpack, and Turbopack surfaces described by `.agents/skills/react-vendoring/SKILL.md`.
- **`filter-internal-request-headers`** (85, very-high) — Treat any newly consumed nonstandard request header as attacker-controlled until reviewed. If it is framework-internal, add it to the `INTERNAL_HEADERS` filtering boundary before downstream server code can read it, and add a focused regression for direct external requests.
- **`run-nextjs-lint-gate`** (85, very-high) — Run `pnpm lint` before handing off repository changes so TypeScript, formatting, ESLint, AST-grep, language, and unused-task checks execute through the maintained aggregate gate.
- **`generate-isolated-regression-tests`** (84, very-high) — Create new suites with `pnpm new-test` so they use the repository's typed fixture structure and `nextTestSetup` isolation. Demonstrate that a fix's regression test fails without the fix, add checks to a closely related existing suite when appropriate, and use condition-based polling rather than fixed sleeps.
- **`use-pinned-node-pnpm-toolchain`** (80, very-high) — Use pnpm through the root `packageManager` declaration and use the Node version selected by `.node-version`, while preserving the package's declared minimum Node version. Update toolchain metadata, lockfile behavior, contributor guidance, and CI together for an intentional version change.
- **`preserve-source-generated-output-boundary`** (78, high) — Make core framework changes in `packages/next/src/` or the owning build configuration, not in derived `packages/next/dist/` output. Regenerate compiled JavaScript, runtime bundles, source maps, and declarations through the repository build tasks, and review generated diffs only as outputs of those source changes.
- **`attach-helpful-error-links`** (72, high) — For each new user-facing warning or error, run `pnpm new-error`, write actionable explanatory documentation, and attach the generated URL to the runtime message. Keep the logged message concise without removing the context users need to resolve it.

### Open judgment questions

Auto-flagged by evidence profile. These are prompts for review, not verdicts.

1. **Single-citation rules (2).** Each rests on one authoritative source. Does that one source support the obligation as written, or is the rule broader than its evidence?
   - `filter-internal-request-headers` (85) — `AGENTS.md`
   - `run-nextjs-lint-gate` (85) — `AGENTS.md`
2. **Lowest-scored rule.** `attach-helpful-error-links` (72) sits at the bottom of this pack.

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

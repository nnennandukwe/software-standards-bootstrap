# Next.js / Claude Code proposal record

Proposal generated on 2026-07-23. Developer retention decisions were recorded
on 2026-07-26; see [Developer review](#developer-review). All proposed rules
and the related skill were approved as Keep. Edit/delete/rerender propagation
and the explicitly requested ADR remain unverified.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.191
- Model: `claude-sonnet-4-6`
- Reasoning profile: `medium`
- Repository: `github.com/vercel/next.js`
- Baseline commit: `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`
- Evaluation branch: `ssb-claude-evaluation`
- Project skill location:
  `.claude/skills/software-standards-bootstrap/SKILL.md`, resolved through the
  repository's existing `.claude/skills -> ../.agents/skills` link

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

Validation passed with 9 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `browser-variant-splitting` | correctness | deterministic | 92 | very-high | Pending |
| `rust-turbopack-cell-map-ordering` | correctness | deterministic | 83 | very-high | Pending |
| `ci-script-injection-env-var` | security | guidance | 77 | high | Pending |
| `ci-third-party-action-pin-sha` | security | guidance | 75 | high | Pending |
| `ci-workflow-permissions-least-privilege` | security | guidance | 75 | high | Pending |
| `rust-lazy-error-context` | performance | deterministic | 75 | high | Pending |
| `rust-context-naming` | maintainability | deterministic | 75 | high | Pending |
| `ci-pull-request-trigger-safety` | security | guidance | 73 | high | Pending |
| `rust-use-bail-macro` | maintainability | deterministic | 73 | high | Pending |

Related skill: `browser-module-variant` (`correctness`).

The structural review covered all five required dimensions. It recorded
package/build boundaries, emitted the browser-module parallel family,
evaluated bundler and configuration seams, kept a narrow public-import surface
assessment-only, and disclosed that test-tree truncation prevented broader
symmetry claims.

The existing root `AGENTS.md` was preserved and received a bounded section with
source digest
`sha256:ce16769500465afcbc652e7aaf034d84d2db9f59a4b7ea22c92d3cbc913d8b69`
and content digest
`sha256:7c6e3d00dea2ffbbec9cde4d5f83c94619b9ac3235fa398931e8096842f2a84e`.

## Validation and assessment repair

The initial assessment omitted the exact exclusion counts and misstated the
indexed-byte boundary. It was corrected to the authoritative inventory above.

The initial proposal rule also referenced the repository's pre-existing
`dce-edge` skill. That target-owned skill does not satisfy SSB's portable skill
contract, so validation correctly blocked rendering. Claude did not edit the
pre-existing skill. It removed only the unsupported `dce-edge` relationship
from `browser-variant-splitting`, retained the generated
`browser-module-variant` relationship, revalidated successfully, previewed the
render, and then rendered the bounded section.

## Developer review

Decisions below are the developer's judgment, not generator output. On
2026-07-26, the developer approved every pending rule and the related skill as
**Keep**.

**High-band retention: 9 of 9 (100%).** The related skill was also kept.

Evidence paths are relative to Next.js baseline `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`.

### High-band rules (count toward the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `browser-variant-splitting` | 92 | `.config/ast-grep/rules/no-typeof-window-require.yml:1-33`, `.config/ast-grep/rules/no-typeof-window-require-tsx.yml:1-33`, `packages/next/src/client/components/server-async-storage.browser.ts:1-7`, `packages/next/src/client/request/io.browser.ts:1-13` | Yes (2 of 4) | `pnpm lint-ast-grep` | Keep | Approved as written by the developer on 2026-07-26. |
| `rust-turbopack-cell-map-ordering` | 83 | `.config/ast-grep/rules/no-map-async-cell.yml:1-28` | Yes | `pnpm lint-ast-grep` | Keep | Approved as written by the developer on 2026-07-26. |
| `ci-script-injection-env-var` | 77 | `.github/AGENTS.md:61-78` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `ci-third-party-action-pin-sha` | 75 | `.github/AGENTS.md:28-43`, `.github/workflows/build_and_test.yml:34-37` | Yes (1 of 2) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `ci-workflow-permissions-least-privilege` | 75 | `.github/AGENTS.md:8-24` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `rust-context-naming` | 75 | `.config/ast-grep/rules/no-context.yml:1-22` | Yes | `pnpm lint-ast-grep` | Keep | Approved as written by the developer on 2026-07-26. |
| `rust-lazy-error-context` | 75 | `.config/ast-grep/rules/no-context-format.yml:1-20` | Yes | `pnpm lint-ast-grep` | Keep | Approved as written by the developer on 2026-07-26. |
| `ci-pull-request-trigger-safety` | 73 | `.github/AGENTS.md:3-5` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `rust-use-bail-macro` | 73 | `.config/ast-grep/rules/no-err-anyhow.yml:1-38` | Yes | `pnpm lint-ast-grep` | Keep | Approved as written by the developer on 2026-07-26. |

### Related skill

| Artifact | Decision | Rationale |
|---|---|---|
| `browser-module-variant` | Keep | Approved as written by the developer on 2026-07-26. |

Generated at `.agents/skills/browser-module-variant/SKILL.md`.

### What each rule asks

- **`browser-variant-splitting`** (92, very-high) — Do not guard a `require()` call on `typeof window` in `packages/next/src/`. Doing so causes both the server branch and the browser branch to be included in the browser bundle. Instead, split the module into a default file (`<name>.ts`) and a browser sibling (`<name>.browser.ts`). The bundler aliases `.browser` files automatically via `scripts/generate-browser-variant-aliases.mjs`. **Prohibited:** ```ts // packages/next/src/client/some-module.ts const impl = typeof window === 'undefined' ? require('./server-impl') : require('./browser-impl') ``` **Required:** ```ts // packages/next/src/client/some-module.ts — server / unbundled runtime export { impl } from './server-impl' // packages/next/src/client/some-module.browser.ts — browser bundle export { impl } from './browser-impl' ``` Optional sub-variants for dev/prod splits follow the same pattern: `<name>.browser.dev.ts` and `<name>.browser.prod.ts`, with the `.browser.ts` entry delegating between them based on `process.env.NODE_ENV`. The ast-grep rules `no-typeof-window-require` (TypeScript) and `no-typeof-window-require-tsx` (TSX) enforce this at severity `error` and are run by `pnpm lint-ast-grep` in CI.
- **`rust-turbopack-cell-map-ordering`** (83, very-high) — Calling `.cell()`, `.resolved_cell()`, `Vc::cell()`, `ResolvedVc::cell()`, `ReadRef::cell()`, or `TraitRef::cell()` inside an `async` closure that is passed to `.map()` causes non-deterministic cell ordering in Turbopack's task graph. Non-deterministic ordering corrupts build-cache reproducibility and can cause incorrect incremental recompilation. **Prohibited:** ```rust let results = items.map(async |item| { let computed = expensive(item).await?; computed.cell() // ← cell created inside async .map() closure }); ``` **Required:** `try_join_all()` (or `try_join!`) the async computations first, then call `.cell()` on the joined results outside the closure: ```rust let computed: Vec<_> = try_join_all(items.iter().map(|item| async { expensive(item).await })).await?; let cells: Vec<_> = computed.into_iter().map(|c| c.cell()).collect(); ``` The ast-grep rule `no-map-async-cell` enforces this at severity `error` and is run by `pnpm lint-ast-grep` in CI.
- **`ci-script-injection-env-var`** (77, high) — `${{ ... }}` expressions are interpolated into the shell script before bash parses it. Values derived from untrusted sources — pull request title, branch name, issue body, commit message — can escape the script and execute arbitrary commands. Route every untrusted value through an environment variable and quote it on use: ```yaml - name: Check PR title env: TITLE: ${{ github.event.pull_request.title }} run: | set -euo pipefail case "$TITLE" in octocat*) echo "starts with octocat" ;; *) exit 1 ;; esac ``` Additional requirements for multi-line `run:` blocks: - Start with `set -euo pipefail`. - Double-quote every `$VAR`. - Never `echo` or `printf` a secret directly. Use `::add-mask::` for dynamic secrets. - The same injection risk applies to `bash -c`, `sh -c`, and any `child_process` invocation that builds a command string from context data.
- **`ci-third-party-action-pin-sha`** (75, high) — Prefer GitHub-provided (`actions/*`) and Vercel-owned actions. For any third-party action, always pin to the full 40-character commit SHA and include the tag as a trailing comment: ```yaml uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2 ``` Never pin to a tag or branch name — these are mutable and can be silently redirected after a supply-chain compromise. Before pinning a new action, run `git grep -n 'owner/repo@'` and reuse any SHA already present in the repository for the same action. If no existing SHA is found, look up the latest tag using: ```sh gh api repos/{owner}/{repo}/tags --jq '.[0:10] | .[] | {name, sha: .commit.sha}' ``` When using `actions/checkout`, always pass `persist-credentials: false` unless the job genuinely needs to push commits or call the GitHub API using the checkout credential.
- **`ci-workflow-permissions-least-privilege`** (75, high) — Set `permissions: {}` at the top-level workflow scope to deny all permissions by default, then grant the minimum required permissions to each individual job. ```yaml permissions: {} jobs: lint: permissions: contents: read comment-on-pr: permissions: contents: read pull-requests: write ``` This prevents a compromised step in one job from inheriting write permissions that were only needed by another job. The GITHUB_TOKEN's default permissions are broader than most jobs require; always explicitly scope them down.
- **`rust-context-naming`** (75, high) — In the Turbopack Rust codebase, the word `context` is ambiguous because many distinct context types exist (`ChunkingContext`, `AssetContext`, `CompileTimeInfo`, etc.). Naming a variable, parameter, or struct field `context` makes it impossible to determine which kind is meant without reading the type annotation. Use a specific name instead: | Ambiguous | Preferred | |---|---| | `context` | `chunking_context` | | `context` | `asset_context` | | `context` | `compile_info` | The ast-grep rule `no-context` enforces this for identifiers in closure parameters, function parameters, function bodies, and let declarations, as well as field names in struct declarations. It runs at severity `error` via `pnpm lint-ast-grep` in CI.
- **`rust-lazy-error-context`** (75, high) — `.context(format!(...))` eagerly allocates and formats the error string on every call, even when no error actually occurred. Use the lazy variant `.with_context(|| format!(...))` so the allocation only happens when the error path is taken. The same rule applies to `turbofmt!` macros: use `.with_context(|| turbofmt!(...).await?)` instead of `.context(turbofmt!(...).await?)`. **Prohibited:** ```rust file.read().context(format!("failed to read {path}"))?; ``` **Required:** ```rust file.read().with_context(|| format!("failed to read {path}"))?; ``` The ast-grep rule `no-context-format` (which includes `no-context-turbofmt`) enforces this at severity `error`, with an auto-fix, and is run by `pnpm lint-ast-grep` in CI.
- **`ci-pull-request-trigger-safety`** (73, high) — Use the `pull_request` trigger for CI workflows, not `pull_request_target`. `pull_request_target` runs with write permissions and access to repository secrets, using the base branch context. If a workflow using `pull_request_target` checks out the PR's HEAD ref and then runs the checked-out code, an attacker can submit a malicious PR that executes arbitrary code with full repository access. `workflow_run` carries the same risk profile as `pull_request_target` — apply equal caution. If write access or secrets are genuinely required for a fork PR workflow, use a separate job gated on a trusted event (for example, a maintainer manually triggering a `workflow_dispatch`) and never check out the PR's HEAD ref in that privileged job.
- **`rust-use-bail-macro`** (73, high) — `Err(anyhow!(...))` and `Err(anyhow::anyhow!(...))` are more verbose than necessary. The `bail!` macro from the anyhow crate is the idiomatic equivalent and reduces visual noise. **Prohibited:** ```rust return Err(anyhow!("something went wrong: {}", detail)); return Err(anyhow::anyhow!("something went wrong: {}", detail)); ``` **Required:** ```rust bail!("something went wrong: {}", detail); ``` The ast-grep rule `no-err-anyhow` enforces this at severity `error` and is run by `pnpm lint-ast-grep` in CI. The rule excludes cases where `Err(anyhow!(...))` appears inside function arguments, assignment expressions, let bindings, or field expressions — those patterns have no direct `bail!` equivalent.

### Open judgment questions

Auto-flagged by evidence profile. These are prompts for review, not verdicts.

1. **Single-citation rules (7).** Each rests on one authoritative source. Does that one source support the obligation as written, or is the rule broader than its evidence?
   - `rust-turbopack-cell-map-ordering` (83) — `.config/ast-grep/rules/no-map-async-cell.yml`
   - `ci-script-injection-env-var` (77) — `.github/AGENTS.md`
   - `ci-workflow-permissions-least-privilege` (75) — `.github/AGENTS.md`
   - `rust-context-naming` (75) — `.config/ast-grep/rules/no-context.yml`
   - `rust-lazy-error-context` (75) — `.config/ast-grep/rules/no-context-format.yml`
   - `ci-pull-request-trigger-safety` (73) — `.github/AGENTS.md`
   - `rust-use-bail-macro` (73) — `.config/ast-grep/rules/no-err-anyhow.yml`

## Changed and untracked paths

```text
AGENTS.md (tracked file modified only by the bounded rendered section)
.agents/skills/browser-module-variant/SKILL.md
.agents/skills/software-standards-bootstrap
.software-standards/assessment.md
.software-standards/rules/browser-variant-splitting.md
.software-standards/rules/ci-pull-request-trigger-safety.md
.software-standards/rules/ci-script-injection-env-var.md
.software-standards/rules/ci-third-party-action-pin-sha.md
.software-standards/rules/ci-workflow-permissions-least-privilege.md
.software-standards/rules/rust-context-naming.md
.software-standards/rules/rust-lazy-error-context.md
.software-standards/rules/rust-turbopack-cell-map-ordering.md
.software-standards/rules/rust-use-bail-macro.md
```

The `.agents/skills/software-standards-bootstrap` path is the evaluator's
uncommitted project-skill harness, not generated repository policy.

## Safety and review boundary

- No Next.js code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index remained unchanged.
- No Git mutation occurred after evaluator setup.
- Rule and generated-skill sources remained untracked. The only tracked
  worktree change was the derived bounded section in `AGENTS.md`.
- No ADR was previewed or created.

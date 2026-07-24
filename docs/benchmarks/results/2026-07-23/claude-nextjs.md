# Next.js / Claude Code proposal record

Generated on 2026-07-23. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

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

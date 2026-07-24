# Flask / Claude Code proposal record

Generated on 2026-07-23. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.191
- Model: `claude-sonnet-4-6`
- Reasoning profile: `medium`
- Repository: `github.com/pallets/flask`
- Baseline commit: `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`
- Evaluation branch: `ssb-claude-evaluation`
- Project skill location:
  `.claude/skills/software-standards-bootstrap/SKILL.md`

## Inventory

- Contract: `ssb-inventory-v1`
- Safe tracked files: 230
- Indexed bytes: 1,474,850
- Limits: 20,000 files; 25 MiB total; 1 MiB per file
- Truncated: no
- Excluded: 5 binary; 1 secret-like; all other exclusion categories 0

## Proposal

Validation passed with 12 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `gha-sha-pinned-actions` | security | guidance | 84 | very-high | Pending |
| `public-api-as-alias-reexport` | compatibility | guidance | 82 | very-high | Pending |
| `ruff-enforced-style` | maintainability | deterministic | 82 | very-high | Pending |
| `type-checking-guard-for-runtime-imports` | correctness | guidance | 81 | very-high | Pending |
| `dual-type-checker-coverage` | correctness | deterministic | 80 | very-high | Pending |
| `gha-empty-default-permissions` | security | guidance | 77 | high | Pending |
| `pytest-warnings-as-errors` | testability | deterministic | 74 | high | Pending |
| `gha-no-persist-credentials` | security | guidance | 73 | high | Pending |
| `sansio-protocol-agnostic-boundary` | architecture | guidance | 72 | high | Pending |
| `future-annotations-import` | maintainability | guidance | 72 | high | Pending |
| `uv-lock-consistency` | reliability | deterministic | 72 | high | Pending |
| `no-app-context-leaks-in-tests` | testability | deterministic | 68 | high | Pending |

Related skill: `propose-new-public-api-export` (`compatibility`).

The structural review covered all five required dimensions. It emitted the
sans-I/O package boundary and public re-export surface, evaluated shared
Scaffold implementations, recorded async/free-threaded/minimum-version
configuration seams, and rejected generic source/test colocation.

`AGENTS.md` rendered with source digest
`sha256:c433528327ef1c3d024c19832912235057945c8713b819a7e510715035e89c7b`
and content digest
`sha256:0399b8e06587a11791c41d3dc84775e6f6bbf72d6f61ddf45522fff710b92044`.

## Assessment correction

The initial assessment copied an incorrect safe-file total of 152 even though
the immutable `ssb inspect` result contained 230. An evaluator audit caught the
mismatch. A targeted Claude Code recovery turn corrected the assessment to the
authoritative inventory, revalidated the unchanged rule pack, previewed the
render, and rerendered it. The incorrect assessment is not counted as final
evidence.

## Changed and untracked paths

```text
.agents/skills/propose-new-public-api-export/SKILL.md
.claude/skills/software-standards-bootstrap
.software-standards/assessment.md
.software-standards/rules/dual-type-checker-coverage.md
.software-standards/rules/future-annotations-import.md
.software-standards/rules/gha-empty-default-permissions.md
.software-standards/rules/gha-no-persist-credentials.md
.software-standards/rules/gha-sha-pinned-actions.md
.software-standards/rules/no-app-context-leaks-in-tests.md
.software-standards/rules/public-api-as-alias-reexport.md
.software-standards/rules/pytest-warnings-as-errors.md
.software-standards/rules/ruff-enforced-style.md
.software-standards/rules/sansio-protocol-agnostic-boundary.md
.software-standards/rules/type-checking-guard-for-runtime-imports.md
.software-standards/rules/uv-lock-consistency.md
AGENTS.md
```

The `.claude/skills/software-standards-bootstrap` path is the evaluator's
uncommitted project-skill harness, not generated repository policy.

## Safety and review boundary

- No Flask code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index and tracked source tree remained
  unchanged.
- No Git mutation occurred after evaluator setup.
- The proposal remained uncommitted. Rules and skills are editable sources;
  `AGENTS.md` is derived.
- No ADR was previewed or created.

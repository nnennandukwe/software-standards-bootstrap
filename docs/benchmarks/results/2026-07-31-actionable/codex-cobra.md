# Cobra / Codex actionable-artifact acceptance

Generated and reviewed on 2026-07-31. The developer explicitly kept every
final artifact. The pack remains uncommitted in the disposable fixture; its
ADR remains `Proposed`.

## Runtime and immutable input

- Consumer: `codex-cli 0.145.0`
- Session: `019fb874-3a3f-7400-b6c7-9baa5ad514ba`
- Model and reasoning: `gpt-5.6-sol`, `xhigh`
- Repository: `github.com/spf13/cobra`
- Baseline: `adbc8813901bba65827259daa8e22ff94ec1f30e`
- Attached branch: `ssb-evaluation`
- Inventory: schema 2; 66 candidate and scanned files; 65 indexed files;
  705,271 candidate/scanned bytes; 631,792 indexed bytes
- Exclusions: 1 binary; every other exclusion category 0
- Truncated: no; remaining candidate files and bytes: 0

## Final artifacts and decisions

All seven artifacts resolved their exact evidence and passed validation.

| Artifact | Kind | Category | Confidence | Utility | Decision |
|---|---|---|---|---:|---|
| `preserve-go-115-compatibility` | Rule | compatibility | high | 79 | Keep |
| `cover-go-changes-with-tests` | Rule | testability | high | 75 | Keep |
| `preserve-declared-v1-compatibility` | Rule | compatibility | high | 86 | Keep |
| `update-affected-document-generators-together` | Rule | correctness | high | 66 | Keep |
| `verify-go-changes` | Recipe | quality | high | 78 | Keep |
| `change-shell-completions` | Agent Skill | compatibility | high | 75 | Keep |
| `check-minimum-go-version` | Automation proposal | compatibility | high | 71 | Keep |

Retention is 7/7 (100%), exceeding the required exact per-fixture threshold of
70%.

The structural review covered the root and `doc` package boundary, completion
and documentation generator families, Windows build-constraint seams, public
v1 compatibility, source/test/documentation symmetry, the `make all` command,
and the gap between the declared Go 1.15 minimum and the automated matrix that
begins at Go 1.17.

## Projection and ADR

- `ssb validate`: pass, 4 rules, 1 recipe, 1 Agent Skill, 1 automation proposal
- `ssb render --dry-run`: current
- ADR: `docs/adr/0001-actionable-standards.md`
- ADR status: `Proposed`
- ADR SHA-256:
  `d3fb9927dbc9cd9da576ef5a8a65e3ba7f2e6bff0f5a34fa717bb9596ea51a8b`
- ADR contents: 4 rules, 1 recipe, and 1 Agent Skill; automation omitted

## Complete changed and untracked paths

```text
.agents/skills/change-shell-completions/SKILL.md
.agents/skills/software-standards-bootstrap/SKILL.md
.agents/skills/software-standards-bootstrap/references/categories.md
.agents/skills/software-standards-bootstrap/references/evidence-workflow.md
.agents/skills/software-standards-bootstrap/references/prune-review.md
.agents/skills/software-standards-bootstrap/references/rule-schema.md
.agents/skills/software-standards-bootstrap/references/structural-patterns.md
.software-standards/automation/check-minimum-go-version.yaml
.software-standards/report.md
.software-standards/rules/cover-go-changes-with-tests.md
.software-standards/rules/preserve-declared-v1-compatibility.md
.software-standards/rules/preserve-go-115-compatibility.md
.software-standards/rules/update-affected-document-generators-together.md
.software-standards/verification/verify-go-changes.yaml
AGENTS.md
docs/adr/0001-actionable-standards.md
```

## Safety boundary

No Cobra code or recorded recipe command was executed. No automation was
implemented. `HEAD` stayed at the pin, the index remained empty, and no Git or
network mutation occurred.

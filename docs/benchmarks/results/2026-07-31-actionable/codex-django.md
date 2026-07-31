# Django / Codex actionable-artifact acceptance

Generated and reviewed on 2026-07-31. The developer explicitly kept every
final artifact. The pack remains uncommitted in the disposable fixture; its
ADR remains `Proposed`.

## Runtime and immutable input

- Consumer: `codex-cli 0.145.0`
- Session: `019fb88c-17b1-7f93-88b3-f1b60aa76085`
- Model and reasoning: `gpt-5.6-sol`, `xhigh`
- Repository: `github.com/django/django`
- Baseline: `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`
- Attached branch: `ssb-evaluation`
- Inventory: schema 2; 7,001 candidate and scanned files; 5,619 indexed files;
  45,506,636 candidate/scanned bytes; 36,820,618 indexed bytes
- Exclusions: 1,382 binary, 4 symlink, and 72 vendor/generated-tree; every
  other exclusion category 0
- Truncated: no; remaining candidate files and bytes: 0

## Final artifacts and decisions

All seven artifacts resolved their exact evidence and passed validation.

| Artifact | Kind | Category | Confidence | Utility | Decision |
|---|---|---|---|---:|---|
| `keep-release-changes-compatible` | Rule | compatibility | high | 88 | Keep |
| `cover-behavior-changes-with-tests` | Rule | testability | high | 85 | Keep |
| `document-versioned-behavior-changes` | Rule | documentation | high | 78 | Keep |
| `defer-settings-access-until-runtime` | Rule | correctness | high | 72 | Keep |
| `keep-database-backends-contract-aligned` | Rule | architecture | high | 77 | Keep |
| `verify-django-change-with-tox` | Recipe | quality | high | 82 | Keep |
| `deprecate-django-feature` | Agent Skill | compatibility | high | 91 | Keep |

Retention is 7/7 (100%), exceeding the required exact per-fixture threshold of
70%.

The structural review covered framework/package boundaries, the built-in
database-backend family, compatibility and deprecation seams, settings import
behavior, source/test/documentation symmetry, the existing tox command, and
existing automatic enforcement. No automation proposal was emitted.

## Projection and ADR

- `ssb validate`: pass, 5 rules, 1 recipe, 1 Agent Skill, 0 automation proposals
- `ssb render --dry-run`: current
- ADR: `docs/adr/0001-actionable-standards.md`
- ADR status: `Proposed`
- ADR SHA-256:
  `c0fbb8580a62ca4aa9750e36c97ed896ca07ca4f5d63b98f91667d59914c330d`
- ADR contents: 5 rules, 1 recipe, and 1 Agent Skill

## Complete changed and untracked paths

```text
.agents/skills/deprecate-django-feature/SKILL.md
.agents/skills/software-standards-bootstrap/SKILL.md
.agents/skills/software-standards-bootstrap/references/categories.md
.agents/skills/software-standards-bootstrap/references/evidence-workflow.md
.agents/skills/software-standards-bootstrap/references/prune-review.md
.agents/skills/software-standards-bootstrap/references/rule-schema.md
.agents/skills/software-standards-bootstrap/references/structural-patterns.md
.software-standards/report.md
.software-standards/rules/cover-behavior-changes-with-tests.md
.software-standards/rules/defer-settings-access-until-runtime.md
.software-standards/rules/document-versioned-behavior-changes.md
.software-standards/rules/keep-database-backends-contract-aligned.md
.software-standards/rules/keep-release-changes-compatible.md
.software-standards/verification/verify-django-change-with-tox.yaml
AGENTS.md
docs/adr/0001-actionable-standards.md
```

## Safety boundary

No Django code or recorded recipe command was executed. `HEAD` stayed at the
pin, the index remained empty, and no Git or network mutation occurred. The
accepted run was repository-isolated; an earlier shared-parent diagnostic was
terminated and excluded after it observed sibling fixture output.

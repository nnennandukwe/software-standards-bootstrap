# Django / Codex proposal record

Generated on 2026-07-23. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.715.71837 (build 5702)
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/django/django`
- Baseline commit: `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`
- Evaluation branch: `ssb-evaluation`

## Inventory

- Contract: `ssb-inventory-v1`
- Safe tracked files: 3,730
- Indexed bytes: 26,184,396
- Limits: 20,000 files; 25 MiB total; 1 MiB per file
- Truncated: yes, at the total-byte limit
- Excluded: 1,267 binary; 4 symlinks; 72 vendor/generated-tree; all other
  exclusion categories 0
- Indexed coverage ends at `tests/delete/models.py`; later paths were not used
  for positive or negative conclusions.

## Proposal

Validation passed with 9 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `verify-django-changes` | correctness | deterministic | 95 | very-high | Pending |
| `preserve-deprecation-schedule` | compatibility | guidance | 88 | very-high | Pending |
| `preserve-python-support-boundary` | compatibility | guidance | 87 | very-high | Pending |
| `disclose-ai-assistance` | compliance | deterministic | 83 | very-high | Pending |
| `keep-database-backends-aligned` | compatibility | guidance | 82 | very-high | Pending |
| `keep-sync-async-apis-aligned` | compatibility | guidance | 80 | very-high | Pending |
| `validate-documentation-changes` | documentation | deterministic | 78 | high | Pending |
| `check-test-migrations` | correctness | deterministic | 75 | high | Pending |
| `do-not-request-automated-ai-review` | compliance | guidance | 70 | high | Pending |

Related skill: `maintain-sync-async-api-parity` (`compatibility`).

The structural review covered all five required dimensions. It evaluated
package boundaries, emitted the database-backend and sync/async implementation
families, recorded platform/configuration seams within the truncated inventory,
emitted narrow public compatibility obligations, and retained
source/test/documentation symmetry only where evidence met the threshold.

`AGENTS.md` rendered with source digest
`sha256:9f4eecee5895ab6fd3b416c6699dedc7372b6d6cdde09fcd46958c510e55003d`
and content digest
`sha256:26e1da021604ab7551ee23a3170fbe06c03958307ba791cbf983b0d9034603e1`.

## Changed and untracked paths

```text
.agents/skills/maintain-sync-async-api-parity/SKILL.md
.software-standards/assessment.md
.software-standards/rules/check-test-migrations.md
.software-standards/rules/disclose-ai-assistance.md
.software-standards/rules/do-not-request-automated-ai-review.md
.software-standards/rules/keep-database-backends-aligned.md
.software-standards/rules/keep-sync-async-apis-aligned.md
.software-standards/rules/preserve-deprecation-schedule.md
.software-standards/rules/preserve-python-support-boundary.md
.software-standards/rules/validate-documentation-changes.md
.software-standards/rules/verify-django-changes.md
AGENTS.md
```

## Safety and review boundary

- No Django code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index and tracked source tree remained
  unchanged.
- SSB performed no network or Git mutation.
- The proposal remained uncommitted. Rules and skills are editable sources;
  `AGENTS.md` is derived.
- No ADR was previewed or created.

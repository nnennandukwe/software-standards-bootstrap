# Django / Codex proposal record

Proposal generated on 2026-07-23. Developer retention decisions were recorded
on 2026-07-26; see [Developer review](#developer-review). All proposed rules
and the related skill were approved as Keep. Edit/delete/rerender propagation
and the explicitly requested ADR remain unverified.

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

## Developer review

Decisions below are the developer's judgment, not generator output. On
2026-07-26, the developer approved every pending rule and the related skill as
**Keep**.

**High-band retention: 9 of 9 (100%).** The related skill was also kept.

Evidence paths are relative to Django baseline `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`.

### High-band rules (count toward the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `verify-django-changes` | 95 | `docs/internals/contributing/writing-code/unit-tests.txt:1-35`, `.github/workflows/python_matrix.yml:29-60` | Yes (all) | `python -Wall tests/runtests.py -v2` | Keep | Approved as written by the developer on 2026-07-26. |
| `preserve-deprecation-schedule` | 88 | `docs/internals/release-process.txt:73-120`, `docs/internals/deprecation.txt:5-16` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `preserve-python-support-boundary` | 87 | `pyproject.toml:5-39`, `.github/workflows/python_matrix.yml:29-60` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `disclose-ai-assistance` | 83 | `.github/pull_request_template.md:10-21`, `scripts/pr_quality/check_pr.py:351-388` | Yes (all) | `python scripts/pr_quality/check_pr.py` | Keep | Approved as written by the developer on 2026-07-26. |
| `keep-database-backends-aligned` | 82 | `django/db/backends/base/features.py:5-36`, `django/db/backends/mysql/features.py:3-33`, `django/db/backends/postgresql/features.py:3-40`, `django/db/backends/sqlite3/features.py:5-44` +2 more | Yes (2 of 6) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `keep-sync-async-apis-aligned` | 80 | `docs/topics/cache.txt:1304-1310`, `docs/ref/models/querysets.txt:2148-2154`, `django/core/cache/backends/base.py:127-177`, `tests/cache/tests_async.py:16-42` | Yes (2 of 4) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `validate-documentation-changes` | 78 | `.github/workflows/docs.yml:3-13` | Yes | `cd docs && make lint && make black &&…` | Keep | Approved as written by the developer on 2026-07-26. |
| `check-test-migrations` | 75 | `.github/workflows/check-migrations.yml:3-9` | Yes | `python scripts/check_migrations.py` | Keep | Approved as written by the developer on 2026-07-26. |
| `do-not-request-automated-ai-review` | 70 | `.github/pull_request_template.md:10-21`, `.github/copilot-instructions.md:1-10` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |

### Related skill

| Artifact | Decision | Rationale |
|---|---|---|
| `maintain-sync-async-api-parity` | Keep | Approved as written by the developer on 2026-07-26. |

Generated at `.agents/skills/maintain-sync-async-api-parity/SKILL.md`.

### What each rule asks

- **`verify-django-changes`** (95, very-high) — Add or update focused tests for behavior changes and use Django's `runtests.py` gate before handoff. Preserve warning visibility so deprecations and other runtime warnings are not hidden.
- **`preserve-deprecation-schedule`** (88, very-high) — Keep deprecated public behavior working with its scheduled warning for at least the two feature releases required by Django's release policy. Add the deprecation and eventual removal to the appropriate release notes and `docs/internals/deprecation.txt`; remove a shim only in its scheduled release.
- **`preserve-python-support-boundary`** (87, very-high) — Treat `requires-python` and the Python classifiers in `pyproject.toml` as the supported runtime boundary. When changing that boundary, update formatter targets, dependencies, requirements, documentation, and the standard, Windows, and free-threaded CI surfaces together; otherwise avoid syntax and APIs outside it.
- **`disclose-ai-assistance`** (83, very-high) — Select exactly one AI-assistance disclosure option in every pull request. When AI tools were used, name them, describe their use with the required detail, and fully review and verify their output before submission.
- **`keep-database-backends-aligned`** (82, very-high) — When changing shared ORM or database behavior, review MySQL, Oracle, PostgreSQL, and SQLite as one implementation family. Update each affected backend's capability flags and implementation together with backend-focused tests; preserve intentional differences through explicit feature flags, skips, or expected failures.
- **`keep-sync-async-apis-aligned`** (80, very-high) — Treat an `a`-prefixed asynchronous method as the public counterpart of its synchronous method unless documentation states an intentional difference. Keep names, arguments, results, exceptions, documentation, and focused tests aligned, while preserving necessary async scheduling and thread-sensitivity behavior.
- **`validate-documentation-changes`** (78, high) — Keep Django documentation lint-clean, consistently formatted, and free of spelling failures. Run the same lint, Black, and warning-as-error spelling checks as the documentation workflow before handoff.
- **`check-test-migrations`** (75, high) — When changing test models or test migration files, keep their migration state complete and run the repository migration checker with the PostgreSQL test settings used by CI.
- **`do-not-request-automated-ai-review`** (70, high) — Do not request automated AI review for a pull request in the main Django repository. If such a review is useful during development, run it only in your own fork and keep the upstream review process human-led.

### Open judgment questions

Auto-flagged by evidence profile. These are prompts for review, not verdicts.

1. **Single-citation rules (2).** Each rests on one authoritative source. Does that one source support the obligation as written, or is the rule broader than its evidence?
   - `validate-documentation-changes` (78) — `.github/workflows/docs.yml`
   - `check-test-migrations` (75) — `.github/workflows/check-migrations.yml`
2. **Lowest-scored rule.** `do-not-request-automated-ai-review` (70) sits at the bottom of this pack.

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

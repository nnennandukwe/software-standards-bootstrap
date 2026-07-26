# Django / Codex inventory-v2 proposal record

Generated on 2026-07-26. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.721.31836 (build 5828), Codex CLI
  0.146.0-alpha.3.1
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/django/django`
- Baseline commit: `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`
- Evaluation branch: `ssb-codex-v2-evaluation`
- SSB source commit: `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Raw inventory output SHA-256:
  `7dfbdc561cf24d8d2bef2807e4e1bf4b81266740deff444f36da9dee230d585c`

## Inventory

- Contract: `ssb-inventory-v2`, schema 2
- Candidate and scanned coverage: 7,001 files, 45,506,636 bytes
- Indexed coverage: 5,619 files, 36,820,618 bytes
- Limits: 40,000 files; 128 MiB total; 1 MiB per file
- Remaining: 0 files, 0 bytes
- Truncated: no
- Excluded: 1,382 binary files, 4 symlinks, and 72 files in vendor or
  generated trees; all other categories 0

## Pending proposal

Validation passed with 9 rules, 1 new procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Decision |
|---|---|---|---:|---|
| `verify-django-changes` | correctness | deterministic | 94 | Pending |
| `preserve-deprecation-schedule` | compatibility | guidance | 91 | Pending |
| `preserve-python-support-boundary` | compatibility | guidance | 88 | Pending |
| `disclose-ai-assistance` | compliance | guidance | 87 | Pending |
| `keep-database-backends-aligned` | compatibility | guidance | 83 | Pending |
| `keep-sync-async-apis-aligned` | compatibility | guidance | 81 | Pending |
| `validate-documentation-changes` | documentation | deterministic | 79 | Pending |
| `check-test-migrations` | correctness | deterministic | 76 | Pending |
| `do-not-request-automated-ai-review` | compliance | guidance | 74 | Pending |

Proposed skill: `maintain-sync-async-api-parity` — **Pending review**.

The assessment recorded all five structural dimensions. It emitted the
database extension boundary, sync/async and database implementation families,
Python and free-threading support seam, formal deprecation surface, and
test/documentation/migration symmetry. A repository-wide package-layer rule was
not supported; a universal full-coverage rule was rejected because Django's own
guidance requires contextual coverage review.

The derived `AGENTS.md` has source digest
`sha256:a0463c0724c0cd9a689851b7fa8ca49035e9ef5bf41d3c5094f299d5dc8aa89c`
and content digest
`sha256:59694ce9691841131b75d0225365ac98778207f40a2dbaa8e843f2cdfa68aa56`.

## Proposed paths

```text
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
.agents/skills/maintain-sync-async-api-parity/SKILL.md
AGENTS.md
```

The evaluator-only `software-standards-bootstrap` skill attachment is untracked
but is not part of the proposal.

## Safety and review boundary

- The clone was fresh and `HEAD` stayed at the benchmark pin.
- No Django code, test, hook, build script, linter, package manager, or cited
  verification command was executed.
- Proposal sources and derived output remain uncommitted; the index is clean.
- No ADR was previewed or created.

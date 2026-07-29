# Flask / Codex proposal record

Proposal generated on 2026-07-23. Developer retention decisions were recorded
on 2026-07-26; see [Developer review](#developer-review). All proposed rules
and the related skill were approved as Keep. Edit/delete/rerender propagation
and the explicitly requested ADR remain unverified.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.715.71837 (build 5702)
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/pallets/flask`
- Baseline commit: `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`
- Evaluation branch: `ssb-evaluation`

## Inventory

- Contract: `ssb-inventory-v1`
- Safe tracked files: 230
- Indexed bytes: 1,474,850
- Limits: 20,000 files; 25 MiB total; 1 MiB per file
- Truncated: no
- Excluded: 5 binary; 1 secret-like; all other exclusion categories 0

## Proposal

Validation passed with 7 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `verify-flask-runtime-matrix` | correctness | deterministic | 95 | very-high | Pending |
| `keep-sansio-boundary-pure` | architecture | guidance | 85 | very-high | Pending |
| `preserve-python-support-boundary` | compatibility | guidance | 83 | very-high | Pending |
| `preserve-async-wsgi-bridge` | compatibility | guidance | 78 | high | Pending |
| `keep-dual-type-checkers-green` | correctness | deterministic | 75 | high | Pending |
| `run-repository-precommit-checks` | quality | deterministic | 75 | high | Pending |
| `document-public-behavior-changes` | documentation | guidance | 70 | high | Pending |

Related skill: `maintain-async-wsgi-bridge` (`compatibility`).

The structural review covered all five required dimensions. It emitted the
sans-I/O architecture boundary and async-to-WSGI compatibility seam, recorded
the Python/platform configuration boundary, evaluated the public behavior
surface, and retained only repository-specific source/test/documentation
symmetry.

`AGENTS.md` rendered with source digest
`sha256:ac9e8cd1a7dd45b58f5944292cd42b87ba0352e085b512793a1c66d81b1bbd12`
and content digest
`sha256:8960eb2d7831dff4668fc1bb8fa8c604736ff968cebc118c1c14fa2eb6722e51`.

## Developer review

Decisions below are the developer's judgment, not generator output. On
2026-07-26, the developer approved every pending rule and the related skill as
**Keep**.

**High-band retention: 7 of 7 (100%).** The related skill was also kept.

Evidence paths are relative to Flask baseline `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`.

### High-band rules (count toward the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `verify-flask-runtime-matrix` | 95 | `pyproject.toml:170-212`, `.github/workflows/tests.yaml:13-44` | Yes (all) | `uv run --locked --no-default-groups --…` | Keep | Approved as written by the developer on 2026-07-26. |
| `keep-sansio-boundary-pure` | 85 | `src/flask/sansio/README.md:1-6`, `src/flask/app.py:31-54`, `src/flask/blueprints.py:7-18` | Yes (1 of 3) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `preserve-python-support-boundary` | 83 | `pyproject.toml:9-30`, `pyproject.toml:126-145`, `.github/workflows/tests.yaml:18-30` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `preserve-async-wsgi-bridge` | 78 | `docs/design.rst:188-204`, `src/flask/app.py:1065-1100`, `tests/test_async.py:81-145` | Yes (2 of 3) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `keep-dual-type-checkers-green` | 75 | `pyproject.toml:126-145`, `pyproject.toml:239-245`, `.github/workflows/tests.yaml:45-63` | Yes (all) | `uv run --locked --no-default-groups --…` | Keep | Approved as written by the developer on 2026-07-26. |
| `run-repository-precommit-checks` | 75 | `.github/workflows/pre-commit.yaml:11-29`, `pyproject.toml:233-237` | Yes (all) | `uv run --locked --no-default-groups --…` | Keep | Approved as written by the developer on 2026-07-26. |
| `document-public-behavior-changes` | 70 | `.github/pull_request_template.md:18-25` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |

### Related skill

| Artifact | Decision | Rationale |
|---|---|---|
| `maintain-async-wsgi-bridge` | Keep | Approved as written by the developer on 2026-07-26. |

Generated at `.agents/skills/maintain-async-wsgi-bridge/SKILL.md`.

### What each rule asks

- **`verify-flask-runtime-matrix`** (95, very-high) — Add focused tests for behavior changes and use the repository's tox environments to verify every affected supported runtime, platform, and dependency boundary before handoff.
- **`keep-sansio-boundary-pure`** (85, very-high) — Keep `src/flask/sansio` reusable by alternative implementations: do not introduce I/O, likely I/O paths, or Flask globals there. Put runtime integration in the `Flask` and `Blueprint` subclasses outside the sans-I/O package.
- **`preserve-python-support-boundary`** (83, very-high) — Treat Python 3.10 as the supported language floor. Avoid syntax, typing constructs, and dependencies that require a newer version unless the change intentionally updates package metadata, mypy and Pyright targets, tox and CI matrices, and relevant documentation together.
- **`preserve-async-wsgi-bridge`** (78, high) — Route async views, handlers, and request hooks through the overridable `ensure_sync` and `async_to_sync` bridge. Preserve the WSGI and extension compatibility described in the design documentation, and update focused async tests and limitations documentation with behavioral changes.
- **`keep-dual-type-checkers-green`** (75, high) — When changing public annotations or callback protocols, update the dedicated type-check fixtures and keep both mypy and Pyright green; do not treat success in one checker as sufficient.
- **`run-repository-precommit-checks`** (75, high) — Before handing off a change, run the repository's pinned all-files pre-commit command and address the reported formatting, lint, or repository-policy failures.
- **`document-public-behavior-changes`** (70, high) — For public behavior changes, add a regression test that fails without the change, update relevant user and code documentation, add the appropriate `.. versionchanged::` entry, and summarize the change in `CHANGES.rst` with its issue link.

### Open judgment questions

Auto-flagged by evidence profile. These are prompts for review, not verdicts.

1. **Single-citation rules (1).** Each rests on one authoritative source. Does that one source support the obligation as written, or is the rule broader than its evidence?
   - `document-public-behavior-changes` (70) — `.github/pull_request_template.md`

## Changed and untracked paths

```text
.agents/skills/maintain-async-wsgi-bridge/SKILL.md
.software-standards/assessment.md
.software-standards/rules/document-public-behavior-changes.md
.software-standards/rules/keep-dual-type-checkers-green.md
.software-standards/rules/keep-sansio-boundary-pure.md
.software-standards/rules/preserve-async-wsgi-bridge.md
.software-standards/rules/preserve-python-support-boundary.md
.software-standards/rules/run-repository-precommit-checks.md
.software-standards/rules/verify-flask-runtime-matrix.md
AGENTS.md
```

## Safety and review boundary

- No Flask code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index and tracked source tree remained
  unchanged.
- SSB performed no network or Git mutation.
- The proposal remained uncommitted. Rules and skills are editable sources;
  `AGENTS.md` is derived.
- No ADR was previewed or created.

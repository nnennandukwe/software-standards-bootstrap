# ADR 0001: Adopt actionable repository standards

- Status: Proposed
- Baseline commit: `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`
- Manifest: `.software-standards/manifest.yaml`
- Inventory: `.software-standards/inventory.json`
- Report: `.software-standards/report.md`

## Context

The repository was inspected at the pinned baseline above. The developer retained the following evidence-backed actionable artifacts after review. Verification recipes are recorded here but were not executed by SSB.

## Semantic rules

### Cover behavior changes with regression tests (`cover-behavior-changes-with-tests`)

- Source: `.software-standards/rules/cover-behavior-changes-with-tests.md`
- Scope: `django/**/*.py`, `tests/**/*.py`
- Lenses: `base`
- Directive: `always`
- Category: `correctness`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (88/100, `ssb-utility-v1`)
- Evidence: `docs/internals/contributing/writing-code/submitting-patches.txt:110-120` (`declares`), `docs/internals/contributing/writing-code/submitting-patches.txt:508-509` (`declares`), `docs/internals/contributing/writing-code/submitting-patches.txt:515-522` (`declares`)

Add a regression test for every bug fix and make sure it fails before the fix is applied. For every new feature, add tests that exercise all new code.

### Document user-visible behavior changes (`document-user-visible-behavior-changes`)

- Source: `.software-standards/rules/document-user-visible-behavior-changes.md`
- Scope: `django/**/*.py`, `docs/**/*.txt`
- Lenses: `base`
- Directive: `always`
- Category: `documentation`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `high` (78/100, `ssb-utility-v1`)
- Evidence: `docs/internals/contributing/writing-code/submitting-patches.txt:122-123` (`declares`), `docs/internals/contributing/writing-code/submitting-patches.txt:479-503` (`declares`), `docs/internals/contributing/writing-code/submitting-patches.txt:515-522` (`declares`)

Include documentation whenever code adds a feature or changes existing behavior. For a new feature, also add a release note and mark its documentation with the appropriate ``versionadded`` or ``versionchanged`` directive. Record any backward-incompatible change in the release notes.

### Gate database tests by feature (`gate-database-tests-by-feature`)

- Source: `.software-standards/rules/gate-database-tests-by-feature.md`
- Scope: `django/db/backends/**/*.py`, `tests/**/*.py`
- Lenses: `language:python`, `framework:django`
- Directive: `prefer`
- Category: `compatibility`
- Derivation: `inferred`
- Confidence: `high`
- Utility: `high` (70/100, `ssb-utility-v1`)
- Evidence: `tests/queries/test_db_returning.py:3-11` (`demonstrates`), `tests/queries/test_qs_combinators.py:699-706` (`demonstrates`), `tests/schema/tests.py:365-367` (`demonstrates`)

When a shared database test depends on a backend capability, guard it with ``skipUnlessDBFeature`` or ``skipIfDBFeature`` and the relevant database feature name instead of assuming every supported backend behaves alike.

## Verification recipes

### Check Django documentation (`check-documentation`)

- Source: `.software-standards/verification/check-documentation.yaml`
- Scope: `docs/**`
- Lenses: `task:verification`
- When: Before handing off a documentation change.
- Category: `documentation`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `high` (72/100, `ssb-utility-v1`)
- Evidence: `docs/internals/contributing/writing-documentation.txt:207-220` (`enforces`)

### Run the Django test suite (`run-django-tests`)

- Source: `.software-standards/verification/run-django-tests.yaml`
- Scope: `django/**/*.py`, `tests/**/*.py`
- Lenses: `language:python`, `framework:django`, `task:verification`
- When: Before handing off a change to Django Python behavior or its tests.
- Category: `testability`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (92/100, `ssb-utility-v1`)
- Evidence: `docs/internals/contributing/writing-code/unit-tests.txt:5-12` (`declares`), `docs/internals/contributing/writing-code/unit-tests.txt:27-35` (`enforces`)

## Agent Skills

### Deprecate django feature (`deprecate-django-feature`)

- Source: `.agents/skills/deprecate-django-feature/SKILL.md`
- Description: Apply Django's compatibility, warning, test, cleanup, and documentation workflow when deprecating a public feature or behavior.
- Scope: `django/**/*.py`, `tests/**/*.py`, `docs/**/*.txt`
- Lenses: `framework:django`, `task:planning`, `task:implementation`
- Category: `compatibility`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (86/100, `ssb-utility-v1`)
- Evidence: `docs/internals/release-process.txt:87-92` (`declares`), `docs/internals/contributing/writing-code/submitting-patches.txt:304-316` (`declares`), `docs/internals/contributing/writing-code/submitting-patches.txt:336-350` (`declares`), `docs/internals/contributing/writing-code/submitting-patches.txt:374-388` (`declares`)

## Consequences

- `AGENTS.md` is a derived projection; the manifest, inventory, human report, and canonical artifact source files remain editable.
- Verification recipes remain deliberately invoked repository procedures; this record does not claim their commands passed.
- The developer-created pull request and its merge constitute adoption; this ADR remains Proposed until then.

# Django / Claude Code proposal record

Proposal generated on 2026-07-23. Developer retention decisions were recorded
on 2026-07-26; see [Developer review](#developer-review). All proposed rules
and the related skill were approved as Keep. Edit/delete/rerender propagation
and the explicitly requested ADR remain unverified.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.191
- Model: `claude-sonnet-4-6`
- Reasoning profile: `medium`
- Repository: `github.com/django/django`
- Baseline commit: `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`
- Evaluation branch: `ssb-claude-evaluation`
- Project skill location:
  `.claude/skills/software-standards-bootstrap/SKILL.md`

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

Validation passed with 10 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `python-black-formatting` | maintainability | deterministic | 83 | very-high | Pending |
| `import-order-isort` | maintainability | deterministic | 81 | very-high | Pending |
| `flake8-line-length` | maintainability | deterministic | 78 | high | Pending |
| `test-use-assertraisesmessage` | testability | guidance | 75 | high | Pending |
| `python-requires-3-12-plus` | compatibility | guidance | 74 | high | Pending |
| `no-settings-at-module-level` | correctness | guidance | 73 | high | Pending |
| `deprecation-warning-class` | compatibility | guidance | 71 | high | Pending |
| `f-string-no-i18n` | correctness | guidance | 71 | high | Pending |
| `model-field-lowercase` | maintainability | guidance | 71 | high | Pending |
| `pr-trac-ticket-required` | developer-experience | guidance | 64 | medium | Pending |

Related skill: `add-deprecation` (`compatibility`).

The structural review covered all five required dimensions. It recorded
database-backend package boundaries, evaluated database/cache implementation
families and test symmetry, mapped Python/platform CI seams, emitted typed
deprecation compatibility guidance, and kept broad release-note symmetry
assessment-only.

`AGENTS.md` rendered with source digest
`sha256:ee46c9914007a2bc73b9a8d9766fcfcbbaa673742880cb4450cd281a02124246`
and content digest
`sha256:c0683806f3121202881e08b461936591dbecf541bf465b3356b7dcea97d9e66a`.

## Assessment correction

The initial assessment disclosed truncation and exclusions but omitted the
3,730 safe-file and 26,184,396 indexed-byte totals. An evaluator audit caught
the omission. A targeted Claude Code recovery turn added the authoritative
inventory values, revalidated the unchanged rule pack, previewed the render,
and rerendered it. The incomplete assessment is not counted as final evidence.

## Developer review

Decisions below are the developer's judgment, not generator output. On
2026-07-26, the developer approved every pending rule and the related skill as
**Keep**.

**High-band retention: 9 of 9 (100%).** The medium-band rule and related skill
were also kept.

Evidence paths are relative to Django baseline `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`.

### High-band rules (count toward the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `python-black-formatting` | 83 | `docs/internals/contributing/writing-code/coding-style.txt:38-39` | Yes | `black --check --diff .` | Keep | Approved as written by the developer on 2026-07-26. |
| `import-order-isort` | 81 | `docs/internals/contributing/writing-code/coding-style.txt:134-155` | Yes | `isort --check-only --diff django tests…` | Keep | Approved as written by the developer on 2026-07-26. |
| `flake8-line-length` | 78 | `docs/internals/contributing/writing-code/coding-style.txt:46-55`, `.flake8:1-10` | Yes (all) | `flake8 .` | Keep | Approved as written by the developer on 2026-07-26. |
| `test-use-assertraisesmessage` | 75 | `docs/internals/contributing/writing-code/coding-style.txt:99-105` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `python-requires-3-12-plus` | 74 | `pyproject.toml:8-8`, `pyproject.toml:30-33` | Yes (all) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `no-settings-at-module-level` | 73 | `docs/internals/contributing/writing-code/coding-style.txt:445-480` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `deprecation-warning-class` | 71 | `django/utils/deprecation.py:13-22` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `f-string-no-i18n` | 71 | `docs/internals/contributing/writing-code/coding-style.txt:82-85` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `model-field-lowercase` | 71 | `docs/internals/contributing/writing-code/coding-style.txt:379-393` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |

### Medium-band rules (outside the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `pr-trac-ticket-required` | 64 | `.github/pull_request_template.md:1-5`, `CONTRIBUTING.rst:18-19` | Yes (1 of 2) | none | Keep | Approved as written by the developer on 2026-07-26. |

### Related skill

| Artifact | Decision | Rationale |
|---|---|---|
| `add-deprecation` | Keep | Approved as written by the developer on 2026-07-26. |

Generated at `.agents/skills/add-deprecation/SKILL.md`.

### What each rule asks

- **`python-black-formatting`** (83, very-high) — All Python files in `django/`, `tests/`, and `scripts/` must be formatted with [Black](https://black.readthedocs.io/). Black is non-negotiable: it runs in CI on every pull request and push to `main` via `.github/workflows/linters.yml`. The configured line length is **88 characters** for code. Documentation, comments, and docstrings use **79 characters** (enforced separately by flake8; see `flake8-line-length`). Black is configured in `pyproject.toml`: ```toml [tool.black] target-version = ["py312"] force-exclude = "tests/test_runner_apps/tagged/tests_syntax_error.py" ``` **To fix locally** before committing: ```bash black . ``` If pre-commit is installed (`pre-commit install`), Black runs automatically on `git commit` and applies fixes in place. Review the changes and re-stage before pushing. One file is intentionally excluded from formatting: `tests/test_runner_apps/tagged/tests_syntax_error.py` (it contains deliberate syntax errors used to test the test runner's error handling).
- **`import-order-isort`** (81, very-high) — All Python imports in `django/`, `tests/`, and `scripts/` must be sorted with [isort](https://pycqa.github.io/isort/) using the project's `black` profile. isort runs in CI on every pull request and push to `main`. **Group order** (each group sorted alphabetically, `import X` before `from X import Y`): 1. `__future__` 2. Standard library 3. Third-party libraries 4. Other Django components (absolute imports) 5. Local Django component (`from .foo import Bar` — one-dot relative only) 6. `try/except` imports **isort is configured in `pyproject.toml`**: ```toml [tool.isort] profile = "black" default_section = "THIRDPARTY" known_first_party = "django" ``` **To fix locally**: ```bash isort django tests scripts ``` **Exceptions**: to keep an import out of sorted order (e.g., to break a circular import), add a comment on that line: ```python import module # isort:skip ``` Use convenience re-export paths when available — import from `django.views` rather than `django.views.generic.base`. Avoid multi-dot relative imports (`from ..foo import Bar`).
- **`flake8-line-length`** (78, high) — Python source lines must not exceed **88 characters**. Documentation strings, inline comments, and documentation files (`.txt`, `.rst` under `docs/`) must not exceed **79 characters**. Both limits are enforced by flake8 in CI on every pull request. The limits are defined in `.flake8`: ```ini [flake8] max-line-length = 88 max-doc-length = 79 extend-ignore = E203 exclude = build,.git,.tox,./tests/.env ``` `E203` (whitespace before `:`) is suppressed because Black formats slice notation differently from what PEP 8 prescribes — Black's output is authoritative for code style. **Per-file overrides**: `W601` is silenced for `django/core/cache/backends/filebased.py`, `django/core/cache/backends/base.py`, `django/core/cache/backends/redis.py`, and `tests/cache/tests.py` (these use the dict `.has_key()` check pattern specific to the cache API). **To check locally**: ```bash flake8 . ``` When Black is formatting a line that flake8 would otherwise flag (e.g., a continuation), Black's result takes precedence and the `extend-ignore = E203` setting handles the common overlap.
- **`test-use-assertraisesmessage`** (75, high) — When a test must verify that code raises an exception or emits a warning, always use Django's message-checking variants: | Instead of | Use | |---|---| | `assertRaises(ExcType)` | `assertRaisesMessage(ExcType, "expected message")` | | `assertWarns(WarnType)` | `assertWarnsMessage(WarnType, "expected message")` | | `assertRaisesRegex(...)` | Only when the check genuinely requires a regex pattern | | `assertWarnsRegex(...)` | Only when the check genuinely requires a regex pattern | | `assertTrue(...)` / `assertFalse(...)` for booleans | `assertIs(..., True)` / `assertIs(..., False)` | **Why**: `assertRaises` catches any exception of the given type, including accidental ones with the wrong message. `assertRaisesMessage` pins both the type and the human-readable message, preventing tests from silently passing when the wrong code path raises the same exception class. This convention is observed across 1,595 occurrences in 250+ test files in the inventory — it is the established test idiom for this codebase. `assertRaisesRegex` and `assertWarnsRegex` are acceptable only when the error message contains variable content (e.g., object IDs or paths) that cannot be matched exactly. For boolean assertions, `assertIs(result, True)` rather than `assertTrue(result)` — the former verifies the value is the actual boolean `True`, not merely truthy.
- **`python-requires-3-12-plus`** (74, high) — All contributed Python code must be compatible with **Python 3.12, 3.13, and 3.14** (including the free-threaded `3.14t` variant). Support for Python 3.11 and earlier was dropped; code that uses APIs or syntax removed in 3.12 will fail CI. From `pyproject.toml`: ```toml [project] requires-python = ">= 3.12" ``` Supported classifiers (at baseline): ```toml "Programming Language :: Python :: 3.12", "Programming Language :: Python :: 3.13", "Programming Language :: Python :: 3.14", "Programming Language :: Python :: Free Threading :: 2 - Beta", ``` **Practical implications**: - Do not use `asyncio.coroutine` decorator (removed in 3.11). - Do not use `distutils` (removed in 3.12). - `tomllib` is available in the standard library from 3.11 — no need to add `tomli` as a dependency. - Type parameter syntax (`type X = ...`, PEP 695) is available from 3.12. - The free-threaded build (`3.14t`) removes the GIL — code that relies on the GIL for thread safety is a latent risk. New concurrency-affecting code should be reviewed against free-threading correctness. The CI matrix in `.github/workflows/tests.yml` explicitly tests Windows/SQLite, Ubuntu/SQLite (free-threading), and the standard Python matrix including `3.14t`.
- **`no-settings-at-module-level`** (73, high) — Django modules must not access `django.conf.settings` at the top level (evaluated when the module is imported). Any access to settings before `settings.configure()` is called will trigger the `LazyObject` auto-configuration, breaking callers who configure Django manually. **Wrong**: ```python from django.conf import settings from django.urls import get_callable # This evaluates settings at import time — breaks manual configuration. default_foo_view = get_callable(settings.FOO_VIEW) ``` **Correct** — use laziness or indirection: ```python from django.conf import settings from django.utils.functional import LazyObject, lazy # Option 1: wrap in a function called at use time def get_default_foo_view(): return get_callable(settings.FOO_VIEW) # Option 2: use django.utils.functional.lazy() get_callable_lazy = lazy(get_callable, str) default_foo_view = get_callable_lazy(settings.FOO_VIEW) ``` `settings` is a `LazyObject` that auto-configures on first attribute access. If any setting is accessed before the application calls `settings.configure(...)`, the window for manual configuration is permanently closed. This is a correctness issue for third-party packages and Django itself. Acceptable patterns for deferred access: - `django.utils.functional.LazyObject` subclass - `django.utils.functional.lazy()` - `lambda` that reads settings inside a function call - Accessing settings inside a function body, not at module scope
- **`deprecation-warning-class`** (71, high) — Every deprecation warning issued from Django source must use one of the typed versioned classes defined in `django/utils/deprecation.py`, not the built-in `DeprecationWarning` or `PendingDeprecationWarning` directly. **Current warning hierarchy** (at baseline `50c2b7c`): ```python class RemovedInDjango70Warning(DeprecationWarning): pass class RemovedInDjango71Warning(PendingDeprecationWarning): pass RemovedInNextVersionWarning = RemovedInDjango70Warning RemovedAfterNextVersionWarning = RemovedInDjango71Warning ``` Use `RemovedInNextVersionWarning` (alias for the current-cycle class) when the feature will be removed in the next major release, and `RemovedAfterNextVersionWarning` when it will be removed in the release after that. **Why typed classes matter**: downstream code — including test utilities like `assertWarnsMessage` — filters warnings by type. Using raw `DeprecationWarning` conflates Django's own deprecations with third-party library warnings, preventing consumers from selectively silencing or surfacing only Django deprecations. Additionally, the `docs/internals/deprecation.txt` timeline is keyed to these typed classes. When adding a new deprecation, also: 1. Update `docs/internals/deprecation.txt` to list the item under the appropriate removal version. 2. Write a test that asserts the warning is raised with `assertWarnsMessage(RemovedInDjangoXXWarning, ...)`. 3. Add a release notes entry under the "Deprecated features" section. See the `add-deprecation` Agent Skill for the complete procedural workflow.
- **`f-string-no-i18n`** (71, high) — f-strings must not be used for any string that may require translation, including: - User-facing error messages - Validation messages passed to `ValidationError` - Log messages (logging calls) - Any string wrapped in `_()`, `gettext()`, `ngettext()`, `ugettext_lazy()`, or similar i18n helpers **Why**: gettext message extraction (`makemessages`) relies on static string literals. An f-string is not a string literal — it is evaluated at runtime, so `makemessages` cannot extract it. The result is silently untranslated messages. **Wrong**: ```python raise ValidationError(f"Value {value!r} is not valid.") logger.error(f"Failed to load {module_name}") ``` **Correct**: ```python raise ValidationError("Value %r is not valid." % (value,)) raise ValidationError("Value %(value)r is not valid." % {"value": value}) logger.error("Failed to load %s", module_name) ``` For non-translatable strings (internal assertions, developer-facing `ImproperlyConfigured`, debugging output), f-strings are acceptable. When in doubt, use `%`-formatting or `str.format()` so the message remains extractable. `str.format()` is more verbose than f-strings; `%`-formatting is generally preferred for log messages because the logging framework defers interpolation until the message is actually emitted.
- **`model-field-lowercase`** (71, high) — Django model field names must use **lowercase with underscores** (snake_case). camelCase and mixed-case field names are not permitted. **Wrong**: ```python class Person(models.Model): FirstName = models.CharField(max_length=20) Last_Name = models.CharField(max_length=40) ``` **Correct**: ```python class Person(models.Model): first_name = models.CharField(max_length=20) last_name = models.CharField(max_length=40) ``` **Why**: camelCase field names produce counterintuitive ORM query expressions (`Person.objects.filter(FirstName="Alice")`) and generate database column names that violate SQL naming conventions. Django's ORM lowercases column names by default, but a mismatch between field attribute name and column name causes confusion during migration review and introspection. **Choices**: when `choices` is defined for a field, declare each choice as an uppercase class attribute, or use an `IntegerChoices`/`TextChoices` enum: ```python class MyModel(models.Model): class Direction(models.TextChoices): UP = "U", "Up" DOWN = "D", "Down" direction = models.CharField(choices=Direction) ``` The field name (`direction`) is lowercase; the enum members (`UP`, `DOWN`) are uppercase — this is the prescribed pattern.
- **`pr-trac-ticket-required`** (64, medium) — Every non-trivial pull request must include a Trac ticket number in the PR description. The PR template provides the field: ```markdown #### Trac ticket number ticket-XXXXX ``` Replace `XXXXX` with the corresponding ticket number from [code.djangoproject.com](https://code.djangoproject.com/). For typo fixes only, delete the line and write `N/A - typo`. **Why**: Django uses Trac — not GitHub Issues — as its authoritative bug and feature tracker. Pull requests without a corresponding ticket are closed without merging, because the Django project requires community consensus on the ticket before code review begins. This is an explicit policy stated in `CONTRIBUTING.rst`: > non-trivial pull requests (anything more than fixing a typo) without Trac tickets will be closed! **Scope of "trivial"**: fixing a single typo in documentation or a comment qualifies. Any code change, behavioral change, new feature, or multi-line documentation rewrite requires a ticket. Before opening a PR, check the ticket's "Owned by" field. If it is assigned to another contributor, coordinate with them rather than duplicating work.

### Open judgment questions

Auto-flagged by evidence profile. These are prompts for review, not verdicts.

1. **Single-citation rules (7).** Each rests on one authoritative source. Does that one source support the obligation as written, or is the rule broader than its evidence?
   - `python-black-formatting` (83) — `docs/internals/contributing/writing-code/coding-style.txt`
   - `import-order-isort` (81) — `docs/internals/contributing/writing-code/coding-style.txt`
   - `test-use-assertraisesmessage` (75) — `docs/internals/contributing/writing-code/coding-style.txt`
   - `no-settings-at-module-level` (73) — `docs/internals/contributing/writing-code/coding-style.txt`
   - `deprecation-warning-class` (71) — `django/utils/deprecation.py`
   - `f-string-no-i18n` (71) — `docs/internals/contributing/writing-code/coding-style.txt`
   - `model-field-lowercase` (71) — `docs/internals/contributing/writing-code/coding-style.txt`
2. **Lowest-scored rule.** `pr-trac-ticket-required` (64) sits at the bottom of this pack.

## Changed and untracked paths

```text
.agents/skills/add-deprecation/SKILL.md
.claude/skills/software-standards-bootstrap
.software-standards/assessment.md
.software-standards/rules/deprecation-warning-class.md
.software-standards/rules/f-string-no-i18n.md
.software-standards/rules/flake8-line-length.md
.software-standards/rules/import-order-isort.md
.software-standards/rules/model-field-lowercase.md
.software-standards/rules/no-settings-at-module-level.md
.software-standards/rules/pr-trac-ticket-required.md
.software-standards/rules/python-black-formatting.md
.software-standards/rules/python-requires-3-12-plus.md
.software-standards/rules/test-use-assertraisesmessage.md
AGENTS.md
```

The `.claude/skills/software-standards-bootstrap` path is the evaluator's
uncommitted project-skill harness, not generated repository policy.

## Safety and review boundary

- No Django code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index and tracked source tree remained
  unchanged.
- No Git mutation occurred after evaluator setup.
- The proposal remained uncommitted. Rules and skills are editable sources;
  `AGENTS.md` is derived.
- No ADR was previewed or created.

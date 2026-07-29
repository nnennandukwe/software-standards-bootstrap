# Flask / Claude Code proposal record

Proposal generated on 2026-07-23. Developer retention decisions were recorded
on 2026-07-26; see [Developer review](#developer-review). All proposed rules
and the related skill were approved as Keep. Edit/delete/rerender propagation
and the explicitly requested ADR remain unverified.

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

## Developer review

Decisions below are the developer's judgment, not generator output. On
2026-07-26, the developer approved every pending rule and the related skill as
**Keep**.

**High-band retention: 12 of 12 (100%).** The related skill was also kept.

Evidence paths are relative to Flask baseline `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`.

### High-band rules (count toward the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `gha-sha-pinned-actions` | 84 | `.github/workflows/tests.yaml:32-40`, `.github/workflows/pre-commit.yaml:14-17`, `.github/workflows/publish.yaml:15-16`, `.github/workflows/zizmor.yaml:15-17` +1 more | **No — source-only** | none | Keep | Approved as written by the developer on 2026-07-26. |
| `public-api-as-alias-reexport` | 82 | `src/flask/__init__.py:1-39` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `ruff-enforced-style` | 82 | `pyproject.toml:147-165`, `.pre-commit-config.yaml:1-23` | Yes (all) | `pre-commit run ruff-check ruff-format…` | Keep | Approved as written by the developer on 2026-07-26. |
| `type-checking-guard-for-runtime-imports` | 81 | `src/flask/typing.py:6-9`, `src/flask/sansio/app.py:36-42`, `src/flask/sansio/scaffold.py:21-22`, `src/flask/sansio/blueprints.py:14-16` | **No — source-only** | none | Keep | Approved as written by the developer on 2026-07-26. |
| `dual-type-checker-coverage` | 80 | `pyproject.toml:126-145`, `pyproject.toml:239-245` | Yes (all) | `tox run -e typing` | Keep | Approved as written by the developer on 2026-07-26. |
| `gha-empty-default-permissions` | 77 | `.github/workflows/tests.yaml:7-8`, `.github/workflows/pre-commit.yaml:6-6`, `.github/workflows/publish.yaml:5-5`, `.github/workflows/zizmor.yaml:8-8` | **No — source-only** | none | Keep | Approved as written by the developer on 2026-07-26. |
| `pytest-warnings-as-errors` | 74 | `pyproject.toml:107-110` | Yes | `pytest` | Keep | Approved as written by the developer on 2026-07-26. |
| `gha-no-persist-credentials` | 73 | `.github/workflows/tests.yaml:32-40`, `.github/workflows/pre-commit.yaml:14-17`, `.github/workflows/publish.yaml:15-16`, `.github/workflows/zizmor.yaml:15-17` | **No — source-only** | none | Keep | Approved as written by the developer on 2026-07-26. |
| `future-annotations-import` | 72 | `src/flask/app.py:1-1`, `src/flask/sansio/app.py:1-1`, `src/flask/views.py:1-1`, `src/flask/sansio/scaffold.py:1-1` +1 more | **No — source-only** | none | Keep | Approved as written by the developer on 2026-07-26. |
| `sansio-protocol-agnostic-boundary` | 72 | `src/flask/sansio/README.md:1-6` | Yes | none | Keep | Approved as written by the developer on 2026-07-26. |
| `uv-lock-consistency` | 72 | `.pre-commit-config.yaml:7-10` | Yes | `pre-commit run uv-lock` | Keep | Approved as written by the developer on 2026-07-26. |
| `no-app-context-leaks-in-tests` | 68 | `tests/conftest.py:84-96` | Yes | `pytest` | Keep | Approved as written by the developer on 2026-07-26. |

### Related skill

| Artifact | Decision | Rationale |
|---|---|---|
| `propose-new-public-api-export` | Keep | Approved as written by the developer on 2026-07-26. |

Generated at `.agents/skills/propose-new-public-api-export/SKILL.md`.

### What each rule asks

- **`gha-sha-pinned-actions`** (84, very-high) — Every `uses:` line in all four workflow files pins to a full 40-character commit SHA, with the human-readable version tag in a trailing comment: ```yaml - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2 - uses: astral-sh/setup-uv@cec208311dfd045dd5311c1add060b2062131d57 # v8.0.0 ``` `.pre-commit-config.yaml` uses the same convention for hook repositories, with `# frozen:` version comments: ```yaml rev: 5e2fb545eba1ea9dc051f6f962d52fe8f76a9794 # frozen: v0.15.13 ``` **Why this matters:** Mutable tags (e.g. `@v4`) can be silently redirected to attacker-controlled commits. SHA pins ensure the exact bytecode of the action is evaluated, not whatever the tag points to at run time. **When this applies:** Adding or upgrading any `uses:` reference in a workflow file or any `rev:` entry in `.pre-commit-config.yaml`. To update a pin: use `tox run -e update-actions` (workflows) or `tox run -e update-pre_commit` (pre-commit), as defined in `pyproject.toml`.
- **`public-api-as-alias-reexport`** (82, very-high) — All 39 existing exports in `src/flask/__init__.py` use the explicit re-export form: ```python from .module import Symbol as Symbol ``` This pattern marks each symbol as intentionally part of the public API under PEP 484 / `py.typed` semantics. Type checkers (mypy in strict mode, pyright) treat a bare `from .module import Symbol` as a private implementation detail and will not re-export it to downstream consumers of the `flask` package. **When this applies:** Any time a new symbol is added to `src/flask/__init__.py`. **Required form:** ```python from .new_module import NewSymbol as NewSymbol ``` **Incorrect form:** ```python from .new_module import NewSymbol # missing "as NewSymbol" ``` The incorrect form causes type checkers used by Flask application authors to fail when those authors write `from flask import NewSymbol`, because the symbol is not recognised as a public re-export.
- **`ruff-enforced-style`** (82, very-high) — `pyproject.toml` configures ruff with rules `B` (flake8-bugbear), `E` (pycodestyle error), `F` (pyflakes), `I` (isort), `UP` (pyupgrade), and `W` (pycodestyle warning). Both `ruff-check` and `ruff-format` hooks run via pre-commit and are enforced by the `pre-commit` CI job (`.github/workflows/pre-commit.yaml`). Key isort settings in effect: - `force-single-line = true` — each import on its own line - `order-by-type = false` — imports ordered by module name, not type All Python changes to `src/`, `tests/`, and `examples/` must pass both hooks before merge. Running `pre-commit run --all-files` locally reproduces the CI check. The `tox` `style` environment (`pyproject.toml` lines 233–237) also provides a convenient entry point.
- **`type-checking-guard-for-runtime-imports`** (81, very-high) — Every `src/flask/` module that imports symbols only needed for type annotations wraps those imports in a `if t.TYPE_CHECKING:` guard. The pattern appears consistently across at least four source files (`typing.py`, `sansio/app.py`, `sansio/scaffold.py`, `sansio/blueprints.py`). ```python import typing as t if t.TYPE_CHECKING: # pragma: no cover from werkzeug.wrappers import Response as BaseResponse from .testing import FlaskClient ``` **Why this matters:** Flask's module graph contains potential circular imports. An import that is only needed to satisfy a type annotation at runtime would execute those cycles. The guard ensures such imports run only under type checkers (where no live objects are created) and not when the module is imported by user code. **When this applies:** Any new import in `src/flask/` that is referenced exclusively inside type annotations (function signatures, variable annotations, `TypeVar` bounds, or other `t.*` constructs). **Required pattern:** ```python if t.TYPE_CHECKING: from some.module import SomeType ``` **Incorrect pattern:** ```python from some.module import SomeType # top-level, used only in annotations ```
- **`dual-type-checker-coverage`** (80, very-high) — `pyproject.toml` configures **mypy** in `strict` mode (`python_version = "3.10"`) and **pyright** in `basic` mode (`pythonVersion = "3.10"`), both targeting `src/` and `tests/type_check/`. The `typing` CI job runs both sequentially (`.github/workflows/tests.yaml` lines 45–63). `tests/type_check/` contains dedicated type-checking smoke tests (`typing_app_decorators.py`, `typing_error_handler.py`, `typing_route.py`) that verify the public API signatures work correctly under both checkers. **When this applies:** Any change to public API signatures, new type aliases in `src/flask/typing.py`, new class attributes with type annotations, or additions to `src/flask/__init__.py`. Run `tox run -e typing` locally to reproduce both checks before pushing.
- **`gha-empty-default-permissions`** (77, high) — All four workflows declare `permissions: {}` immediately after the `on:` trigger block, restricting the GitHub token to zero permissions at the workflow level. Jobs that need write access (`publish.yaml` create-release and publish-pypi) grant the minimum required permission at the job level only: ```yaml jobs: create-release: permissions: contents: write # narrowly scoped to this job ``` **Why this matters:** Without an explicit `permissions:` block, GitHub Actions grants the default token permissions (read/write for contents and packages in some configurations), which can be exploited if a step runs compromised code. **When this applies:** Every new workflow file added to `.github/workflows/`. Existing workflows must retain `permissions: {}` at the top level; additional per-job grants must be scoped to the minimum required scope.
- **`pytest-warnings-as-errors`** (74, high) — `pyproject.toml` sets `filterwarnings = ["error"]` in `[tool.pytest.ini_options]`. Every `DeprecationWarning`, `PendingDeprecationWarning`, `ResourceWarning`, or any other warning category raised during any test is promoted to a hard test failure. **When this applies:** Any code path exercised by the test suite. Common triggers: - Calling deprecated Werkzeug/Jinja2/Click APIs - Leaving open file handles or unclosed resources - Using deprecated stdlib functions - Issuing `DeprecationWarning` for Flask's own deprecated code paths If a new deprecation warning is intentional (Flask is deprecating a feature), add a matching `pytest.warns` assertion in the relevant test rather than suppressing the error globally.
- **`gha-no-persist-credentials`** (73, high) — All four workflow files that use `actions/checkout` include `persist-credentials: false`: ```yaml - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2 with: persist-credentials: false ``` **Why this matters:** By default, `actions/checkout` persists the GITHUB_TOKEN credential in the Git config of the checked-out repository. If a subsequent step runs untrusted code (e.g. from a pull-request branch), it can read the token. Setting `persist-credentials: false` removes it from the Git config immediately after checkout. **When this applies:** Every new or modified `actions/checkout` step in any workflow file.
- **`future-annotations-import`** (72, high) — Every Python source file in `src/flask/` starts with `from __future__ import annotations` as its first line. This is consistent across all five source files examined (`app.py`, `sansio/app.py`, `sansio/scaffold.py`, `sansio/blueprints.py`, `views.py`) and in `src/flask/typing.py`. ```python from __future__ import annotations ``` **Why this matters:** This import activates PEP 563 deferred evaluation of annotations for the module. It: 1. Allows forward references in type annotations without quoting them (`def foo(x: SomeClass)` works even when `SomeClass` is defined later in the file or in a `TYPE_CHECKING` block). 2. Keeps annotation evaluation cost zero at runtime (annotations are stored as strings, not evaluated). 3. Is required to be consistent with the project's minimum Python version (3.10), which does not have PEP 563 active by default. **When this applies:** Any new `.py` file added to `src/flask/` or its subdirectories. The import must be the first line of the file (after a possible module docstring, but before other imports).
- **`sansio-protocol-agnostic-boundary`** (72, high) — `src/flask/sansio/README.md` is the authoritative statement of this boundary: > This folder contains code that can be used by alternative Flask implementations, for example Quart. The code therefore cannot do any IO, nor be part of a likely IO path. Finally this code cannot use the Flask globals. The three constraints in practice: 1. **No IO** — sansio modules must not open files, make network calls, or invoke WSGI callables. 2. **No Flask globals** — sansio modules must not import or reference `flask.globals` (`request`, `g`, `session`, `current_app`). 3. **No parent-layer imports** — sansio modules must not import from `flask.app` (the WSGI `Flask` class), `flask.ctx`, `flask.sessions`, or other WSGI-only modules. The `src/flask/sansio/` ↔ `src/flask/` dependency direction is strictly one-way: the WSGI layer imports from sansio, not the reverse. **When this applies:** Any change to `src/flask/sansio/scaffold.py`, `src/flask/sansio/app.py`, or `src/flask/sansio/blueprints.py`. Routing logic, configuration, error-handler registration, and blueprint registration all live in the sansio layer and must remain IO-free. **Why this matters:** Quart and other ASGI-based frameworks inherit from these base classes. A WSGI import or IO call in the sansio layer would silently break those frameworks.
- **`uv-lock-consistency`** (72, high) — `.pre-commit-config.yaml` includes the `uv-lock` hook from `astral-sh/uv-pre-commit`. The hook verifies that `uv.lock` is consistent with `pyproject.toml` and fails if they have drifted. It runs on every commit and is also enforced by the `pre-commit` CI job. **Why this matters:** Flask's CI uses `uv run --locked` (e.g. in `.github/workflows/tests.yaml` line 42), which fails if the lockfile is inconsistent. A drifted lockfile also breaks the `tests-min` tox environment, which pins exact minimum dependency versions. Drifted lockfiles have caused reproducibility failures across the Pallets dependency matrix. **When this applies:** Any change to `pyproject.toml` that adds, removes, or modifies a dependency — including dependency-group changes and version constraint changes. Run `uv lock` to regenerate, then commit both `pyproject.toml` and `uv.lock` together. To update the lock: `tox run -e update-requirements` (as defined in `pyproject.toml`).
- **`no-app-context-leaks-in-tests`** (68, high) — `tests/conftest.py` registers an `autouse=True` fixture `leak_detector` that runs after every test: ```python @pytest.fixture(autouse=True) def leak_detector(): yield leaks = [] while _app_ctx: leaks.append(_app_ctx._get_current_object()) _app_ctx.pop() assert not leaks ``` If a test leaves one or more app contexts pushed on the stack, the assertion fires and the test fails. The fixture also pops the leaked contexts so that subsequent tests are not affected. **When this applies:** Any test that manually pushes an app context (e.g. `app.app_context().push()` without a `with` statement or a `yield`). The correct pattern is to use a `with` block or the `app_ctx` fixture from `conftest.py`: ```python def test_something(app_ctx): # app context is active here, automatically cleaned up ... ``` Never call `ctx.push()` without a paired `ctx.pop()`. Tests that rely on the leaked context being present in a subsequent test will fail non-deterministically.

### Open judgment questions

Auto-flagged by evidence profile. These are prompts for review, not verdicts.

1. **Source-only evidence (5 rules).** `gha-sha-pinned-actions` (84), `type-checking-guard-for-runtime-imports` (81), `gha-empty-default-permissions` (77), `gha-no-persist-credentials` (73), `future-annotations-import` (72) cite only code or config that SSB did not mark authoritative — no maintainer-written policy document. For each: is this an invariant the maintainers would defend, or a pattern the generator inferred from repetition?
   - `gha-sha-pinned-actions` — `.github/workflows/tests.yaml`, `.github/workflows/pre-commit.yaml`, `.github/workflows/publish.yaml`, `.github/workflows/zizmor.yaml`, `.pre-commit-config.yaml`
   - `type-checking-guard-for-runtime-imports` — `src/flask/typing.py`, `src/flask/sansio/app.py`, `src/flask/sansio/scaffold.py`, `src/flask/sansio/blueprints.py`
   - `gha-empty-default-permissions` — `.github/workflows/tests.yaml`, `.github/workflows/pre-commit.yaml`, `.github/workflows/publish.yaml`, `.github/workflows/zizmor.yaml`
   - `gha-no-persist-credentials` — `.github/workflows/tests.yaml`, `.github/workflows/pre-commit.yaml`, `.github/workflows/publish.yaml`, `.github/workflows/zizmor.yaml`
   - `future-annotations-import` — `src/flask/app.py`, `src/flask/sansio/app.py`, `src/flask/views.py`, `src/flask/sansio/scaffold.py`, `src/flask/sansio/blueprints.py`
2. **Single-citation rules (5).** Each rests on one authoritative source. Does that one source support the obligation as written, or is the rule broader than its evidence?
   - `public-api-as-alias-reexport` (82) — `src/flask/__init__.py`
   - `pytest-warnings-as-errors` (74) — `pyproject.toml`
   - `sansio-protocol-agnostic-boundary` (72) — `src/flask/sansio/README.md`
   - `uv-lock-consistency` (72) — `.pre-commit-config.yaml`
   - `no-app-context-leaks-in-tests` (68) — `tests/conftest.py`

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

<!-- software-standards-bootstrap:start -->
<!-- source-digest: sha256:4b26e1cd9a90669832c351290e9cb6b943689ce91a3e8990064bf29ce45c6c7b -->
<!-- content-digest: sha256:f4a4f2bc62a59ff77948935716b68ad1e083950117a7325a22db4fc36487b09e -->
## Software Standards Bootstrap

This managed section is derived from retained canonical sources. An unmerged generated change is a proposal; repository review and merge are the adoption decision. File presence alone does not prove adoption.

Generated from `.software-standards/manifest.yaml`, `.software-standards/inventory.json`, `.software-standards/report.md`, `.software-standards/orientation.yaml` and the manifest-listed artifacts by `ssb render`. Edit canonical sources and the manifest together, then rerun the command.

Baseline: `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`

SSB did not stage, commit, push, open a pull request, execute any displayed recipe command, or activate another system. Recipe presence and expected results are not execution evidence.

### Repository orientation

Django is a high-level Python web framework focused on rapid development and clean, pragmatic design.

- Evidence: `README.rst:5-6 (declares)`

#### Important areas

- `docs` - Contains the project's tutorials, topical guides, how-to guides, reference material, and documentation build instructions.
  - Evidence: `README.rst:8-24 (declares)`

- `tests` - Contains Django's own test suite, which uses Django's application testing infrastructure.
  - Evidence: `docs/internals/contributing/writing-code/unit-tests.txt:5-12 (declares)`

#### Related standards

- Related recipe: [Run the Django test suite](.software-standards/verification/run-django-tests.yaml)
- Related recipe: [Check Django documentation](.software-standards/verification/check-documentation.yaml)
- Related skill: [Deprecate django feature](.agents/skills/deprecate-django-feature/SKILL.md)

#### Task guidance

- **Planning:** Use docs/internals/contributing as the canonical contribution guidance.
  - Evidence: `CONTRIBUTING.rst:13-16 (declares)`

- **Handoff:** Review applicable style checks, backward-compatibility release notes, the Django test suite, and documentation build and quality checks before handoff.
  - Evidence: `docs/internals/contributing/writing-code/submitting-patches.txt:479-503 (declares)`

### How routing works

- **Host-specific:** `AGENTS.md` discovery, directory placement, and nested-file precedence depend on the active host; consult that host's documented behavior.
- **SSB generator-defined:** Scopes and lenses are SSB routing metadata, not native `AGENTS.md` glob activation. A semantic rule applies when its affected path scope matches; contextual artifacts also require every represented lens dimension to match, with values inside one dimension treated as alternatives.
- If the language, framework, task, or affected path is uncertain, load the potentially relevant rule, recipe, or skill instead of excluding it.
- **SSB generator-defined:** Directives mean: `never` is prohibited, `ask-first` requires developer authorization, `always` is required, and `prefer` is the default when no documented exception or explicit user direction applies.
- Linked artifact files are canonical. This projection is a concise router, not a replacement for their complete content.

### Standing orders

#### Always

##### Cover behavior changes with regression tests (`cover-behavior-changes-with-tests`)

> Add a regression test for every bug fix and make sure it fails before the fix is applied. For every new feature, add tests that exercise all new code.

- Applies to: `django/**/*.py`, `tests/**/*.py`
- Related recipe: [Run the Django test suite](.software-standards/verification/run-django-tests.yaml)
- Category: `correctness`
- Canonical rule: [.software-standards/rules/cover-behavior-changes-with-tests.md](.software-standards/rules/cover-behavior-changes-with-tests.md)
- Evidence: `docs/internals/contributing/writing-code/submitting-patches.txt:110-120`, `docs/internals/contributing/writing-code/submitting-patches.txt:508-509`, `docs/internals/contributing/writing-code/submitting-patches.txt:515-522`

##### Document user-visible behavior changes (`document-user-visible-behavior-changes`)

> Include documentation whenever code adds a feature or changes existing behavior. For a new feature, also add a release note and mark its documentation with the appropriate ``versionadded`` or ``versionchanged`` directive. Record any backward-incompatible change in the release notes.

- Applies to: `django/**/*.py`, `docs/**/*.txt`
- Related recipe: [Check Django documentation](.software-standards/verification/check-documentation.yaml)
- Related skill: [Deprecate django feature](.agents/skills/deprecate-django-feature/SKILL.md)
- Category: `documentation`
- Canonical rule: [.software-standards/rules/document-user-visible-behavior-changes.md](.software-standards/rules/document-user-visible-behavior-changes.md)
- Evidence: `docs/internals/contributing/writing-code/submitting-patches.txt:122-123`, `docs/internals/contributing/writing-code/submitting-patches.txt:479-503`, `docs/internals/contributing/writing-code/submitting-patches.txt:515-522`

### Contextual semantic rules

#### [Gate database tests by feature](.software-standards/rules/gate-database-tests-by-feature.md) (`gate-database-tests-by-feature`) - `prefer`

- Load when: `language:python`, `framework:django`
- Applies to: `django/db/backends/**/*.py`, `tests/**/*.py`
- Related recipe: [Run the Django test suite](.software-standards/verification/run-django-tests.yaml)
- Category: `compatibility`
- Canonical rule: [.software-standards/rules/gate-database-tests-by-feature.md](.software-standards/rules/gate-database-tests-by-feature.md)
- Evidence: `tests/queries/test_db_returning.py:3-11`, `tests/queries/test_qs_combinators.py:699-706`, `tests/schema/tests.py:365-367`

### Verification commands

#### [Check Django documentation](.software-standards/verification/check-documentation.yaml) (`check-documentation`)

- When: Before handing off a documentation change.
- Route when: `task:verification`
- Applies to: `docs/**`
- Related rule: [Document user-visible behavior changes](.software-standards/rules/document-user-visible-behavior-changes.md)
- Related skill: [Deprecate django feature](.agents/skills/deprecate-django-feature/SKILL.md)
- Category: `documentation`
- Canonical recipe: [.software-standards/verification/check-documentation.yaml](.software-standards/verification/check-documentation.yaml)
- Evidence: `docs/internals/contributing/writing-documentation.txt:207-220`

##### Step 1

Working directory: `docs`

```
make check
```

Expected result: All current documentation quality checks pass.

#### [Run the Django test suite](.software-standards/verification/run-django-tests.yaml) (`run-django-tests`)

- When: Before handing off a change to Django Python behavior or its tests.
- Route when: `language:python`, `framework:django`, `task:verification`
- Applies to: `django/**/*.py`, `tests/**/*.py`
- Related rule: [Cover behavior changes with regression tests](.software-standards/rules/cover-behavior-changes-with-tests.md)
- Related rule: [Gate database tests by feature](.software-standards/rules/gate-database-tests-by-feature.md)
- Category: `testability`
- Canonical recipe: [.software-standards/verification/run-django-tests.yaml](.software-standards/verification/run-django-tests.yaml)
- Evidence: `docs/internals/contributing/writing-code/unit-tests.txt:5-12`, `docs/internals/contributing/writing-code/unit-tests.txt:27-35`

##### Step 1

Working directory: `tests`

```
./runtests.py
```

Expected result: The Django test suite completes without failures.

### Agent Skills

#### [Deprecate django feature](.agents/skills/deprecate-django-feature/SKILL.md) (`deprecate-django-feature`)

Apply Django's compatibility, warning, test, cleanup, and documentation workflow when deprecating a public feature or behavior.

- Use when: `framework:django`, `task:planning`, `task:implementation`
- Applies to: `django/**/*.py`, `tests/**/*.py`, `docs/**/*.txt`
- Related rule: [Document user-visible behavior changes](.software-standards/rules/document-user-visible-behavior-changes.md)
- Related recipe: [Run the Django test suite](.software-standards/verification/run-django-tests.yaml)
- Related recipe: [Check Django documentation](.software-standards/verification/check-documentation.yaml)
- Category: `compatibility`
- Evidence: `docs/internals/release-process.txt:87-92`, `docs/internals/contributing/writing-code/submitting-patches.txt:304-316`, `docs/internals/contributing/writing-code/submitting-patches.txt:336-350`, `docs/internals/contributing/writing-code/submitting-patches.txt:374-388`
<!-- software-standards-bootstrap:end -->

# Developer retention review — 2026-07-26 proposals

Review instrument for the 66 rule proposals and 6 proposed skills recorded in
[`README.md`](README.md). Every decision below is **the developer's to make**;
this file supplies the metadata, body text, and evidence needed to make them and
nothing more. It records no decision on the developer's behalf.

Record one of `keep`, `edit-and-keep`, `defer`, or `reject` per row.

## Scope qualification

These proposals satisfy the contract in force when they were generated. The
evaluation used the `ssb-inventory-v2` inventory contract at schema 2, which is
what these records claim, and it emitted `ssb.dev/rule/v1` rules because v1 was
the only rule schema that existed at SSB source commit `820c3a8` — `rule/v2`
appears nowhere in that tree. All 66 rules declare `ssb.dev/rule/v1` and none
carries a `directive` or `lenses` field, as expected for that schema.

Two things changed on `main` afterwards, both introduced by the progressive
standards pack, and neither is a defect in this evidence:

- `main` now documents [`ssb.dev/rule/v2`](../../../rule-format.md) as the
  emission format for new proposals, and step 8 of
  [the conformance suite](../../../agent-smoke-tests.md) requires *newly* emitted
  rules to use v2 with valid lenses and directive. A conformance claim against
  the shipped contract therefore needs a fresh pass; these records do not make
  that claim.
- `main`'s renderer projects v1 rules into the *Legacy v1 rules (directive not
  recorded)* group and links rather than copies their bodies. This is the
  documented backward-compatibility path — v1 files remain valid and renderable,
  and `ssb validate` exits `0` on all eight packs under both the `820c3a8` and
  `main` binaries. The consequence is narrow but real: the `AGENTS.md` content
  digests recorded in the per-run files no longer reproduce under `main`.

Retention decisions recorded here satisfy the 70% threshold gate for this
evidence set.

## Threshold arithmetic

Bands follow [`docs/rule-format.md`](../../../rule-format.md): `very-high` 80–100,
`high` 65–79, `medium` 45–64, `low` 25–44. The gate is *at least 70% of high and
very-high candidates kept or edit-and-kept*, so only the high-band rows count
toward the denominator.

| Pass | Rules | very-high | high | medium | High-band denominator | Keeps needed for 70% |
|---|---:|---:|---:|---:|---:|---:|
| Codex | 33 | 22 | 11 | 0 | 33 | 24 |
| Claude | 33 | 10 | 21 | 2 | 31 | 22 |
| **Combined** | **66** | **32** | **32** | **2** | **64** | **45** |

Two Claude rules score in the `medium` band and are excluded from the denominator;
they still need a recorded decision. Every rule's declared `importance` matches its
score band, and every score's factors sum to its stated total.

## Cross-pass convergence

11 rule ids were proposed independently by both consumers. Agreement is
not evidence of correctness, but a rule two consumers reached separately is a
cheaper place to start reviewing than one only a single pass proposed.

- `check-test-migrations` — Django (Claude 74, Codex 76)
- `coordinate-security-fixes-privately` — Cobra (Claude 76, Codex 75)
- `disclose-ai-assistance` — Django (Claude 70, Codex 87)
- `do-not-request-automated-ai-review` — Django (Claude 68, Codex 74)
- `keep-dual-type-checkers-green` — Flask (Claude 78, Codex 84)
- `keep-shell-completion-family-aligned` — Cobra (Claude 81, Codex 70)
- `preserve-explicit-compatibility-shims` — Cobra (Claude 73, Codex 72)
- `preserve-go-support-boundary` — Cobra (Claude 84, Codex 86)
- `preserve-license-headers` — Cobra (Claude 84, Codex 82)
- `verify-flask-runtime-matrix` — Flask (Claude 88, Codex 92)
- `verify-go-changes` — Cobra (Claude 89, Codex 90)

## Proposed skills

| Pass | Repository | Skill | Decision |
|---|---|---|---|
| Codex | Cobra | `maintain-shell-completions` | |
| Codex | Flask | `maintain-async-wsgi-bridge` | |
| Codex | Django | `maintain-sync-async-api-parity` | |
| Claude | Cobra | `add-shell-completion-generator` | |
| Claude | Flask | `add-supported-python-version` | |
| Claude | Django | `add-async-api-variant` | |

Neither pass proposed a skill for Next.js; both cited that repository's
pre-existing skill set.

## Decision tables

### Codex

#### Cobra (7 rules)

| Rule | Topic | Class | Score | Band | Conf. | Evidence | Proof gap | Decision |
|---|---|---|---:|---|---|---:|---|---|
| `verify-go-changes` | correctness | deterministic | 90 | very-high | high | 1 | — | |
| `preserve-go-support-boundary` | compatibility | guidance | 86 | very-high | high | 3 | yes | |
| `preserve-license-headers` | compliance | deterministic | 82 | very-high | high | 1 | — | |
| `keep-shell-completions-portable` | compatibility | guidance | 79 | high | high | 3 | yes | |
| `coordinate-security-fixes-privately` | security | guidance | 75 | high | high | 1 | yes | |
| `preserve-explicit-compatibility-shims` | compatibility | guidance | 72 | high | high | 1 | yes | |
| `keep-shell-completion-family-aligned` | maintainability | guidance | 70 | high | medium | 3 | yes | |

#### Flask (7 rules)

| Rule | Topic | Class | Score | Band | Conf. | Evidence | Proof gap | Decision |
|---|---|---|---:|---|---|---:|---|---|
| `verify-flask-runtime-matrix` | correctness | deterministic | 92 | very-high | high | 2 | — | |
| `preserve-python-support-boundary` | compatibility | guidance | 87 | very-high | high | 3 | yes | |
| `keep-dual-type-checkers-green` | correctness | deterministic | 84 | very-high | high | 2 | — | |
| `run-repository-precommit-checks` | correctness | deterministic | 82 | very-high | high | 2 | — | |
| `keep-sansio-boundary-pure` | architecture | guidance | 80 | very-high | high | 1 | yes | |
| `preserve-async-wsgi-bridge` | compatibility | guidance | 78 | high | high | 3 | yes | |
| `document-public-behavior-changes` | documentation | guidance | 72 | high | high | 1 | yes | |

#### Django (9 rules)

| Rule | Topic | Class | Score | Band | Conf. | Evidence | Proof gap | Decision |
|---|---|---|---:|---|---|---:|---|---|
| `verify-django-changes` | correctness | deterministic | 94 | very-high | high | 1 | — | |
| `preserve-deprecation-schedule` | compatibility | guidance | 91 | very-high | high | 2 | yes | |
| `preserve-python-support-boundary` | compatibility | guidance | 88 | very-high | high | 2 | yes | |
| `disclose-ai-assistance` | compliance | guidance | 87 | very-high | high | 2 | yes | |
| `keep-database-backends-aligned` | compatibility | guidance | 83 | very-high | high | 3 | yes | |
| `keep-sync-async-apis-aligned` | compatibility | guidance | 81 | very-high | high | 3 | yes | |
| `validate-documentation-changes` | documentation | deterministic | 79 | high | high | 2 | — | |
| `check-test-migrations` | correctness | deterministic | 76 | high | high | 1 | — | |
| `do-not-request-automated-ai-review` | compliance | guidance | 74 | high | high | 2 | yes | |

#### Next.js (10 rules)

| Rule | Topic | Class | Score | Band | Conf. | Evidence | Proof gap | Decision |
|---|---|---|---:|---|---|---:|---|---|
| `verify-across-nextjs-modes` | correctness | guidance | 92 | very-high | high | 3 | yes | |
| `wire-feature-flags-end-to-end` | architecture | guidance | 90 | very-high | high | 2 | yes | |
| `preserve-edge-dce-boundaries` | architecture | guidance | 88 | very-high | high | 2 | yes | |
| `preserve-react-server-vendoring-boundary` | architecture | guidance | 87 | very-high | high | 3 | yes | |
| `filter-internal-request-headers` | security | guidance | 85 | very-high | high | 1 | yes | |
| `run-nextjs-lint-gate` | quality | deterministic | 85 | very-high | high | 1 | — | |
| `generate-isolated-regression-tests` | testability | guidance | 84 | very-high | high | 2 | yes | |
| `use-pinned-node-pnpm-toolchain` | compatibility | guidance | 80 | very-high | high | 3 | yes | |
| `preserve-source-generated-output-boundary` | maintainability | guidance | 78 | high | high | 3 | yes | |
| `attach-helpful-error-links` | developer-experience | guidance | 72 | high | high | 2 | yes | |

### Claude

#### Cobra (8 rules)

| Rule | Topic | Class | Score | Band | Conf. | Evidence | Proof gap | Decision |
|---|---|---|---:|---|---|---:|---|---|
| `verify-go-changes` | correctness | deterministic | 89 | very-high | high | 2 | — | |
| `preserve-go-support-boundary` | compatibility | guidance | 84 | very-high | high | 3 | yes | |
| `preserve-license-headers` | compliance | deterministic | 84 | very-high | high | 1 | — | |
| `keep-shell-completion-family-aligned` | compatibility | guidance | 81 | very-high | high | 4 | yes | |
| `coordinate-security-fixes-privately` | security | guidance | 76 | high | high | 1 | yes | |
| `preserve-dual-build-tag-syntax` | compatibility | guidance | 74 | high | high | 3 | yes | |
| `preserve-explicit-compatibility-shims` | compatibility | guidance | 73 | high | medium | 4 | yes | |
| `keep-doc-generator-autogen-tag-aligned` | maintainability | guidance | 62 | medium | medium | 4 | yes | |

#### Flask (8 rules)

| Rule | Topic | Class | Score | Band | Conf. | Evidence | Proof gap | Decision |
|---|---|---|---:|---|---|---:|---|---|
| `verify-flask-runtime-matrix` | correctness | deterministic | 88 | very-high | high | 2 | — | |
| `harden-github-actions-workflows` | security | guidance | 86 | very-high | high | 3 | yes | |
| `preserve-python-support-floor` | compatibility | guidance | 86 | very-high | high | 3 | yes | |
| `keep-sansio-layer-protocol-agnostic` | architecture | guidance | 84 | very-high | high | 3 | yes | |
| `keep-dual-type-checkers-green` | correctness | deterministic | 78 | high | high | 3 | — | |
| `pin-and-freeze-pre-commit-hooks` | security | guidance | 76 | high | high | 2 | yes | |
| `document-public-changes-in-changes-rst` | documentation | guidance | 68 | high | medium | 1 | yes | |
| `keep-test-warnings-fatal` | maintainability | deterministic | 66 | high | high | 1 | — | |

#### Django (9 rules)

| Rule | Topic | Class | Score | Band | Conf. | Evidence | Proof gap | Decision |
|---|---|---|---:|---|---|---:|---|---|
| `preserve-deprecation-cycle` | compatibility | guidance | 86 | very-high | high | 2 | yes | |
| `keep-database-backend-modules-aligned` | architecture | guidance | 78 | high | high | 3 | yes | |
| `run-repository-linters` | maintainability | deterministic | 78 | high | high | 2 | — | |
| `check-test-migrations` | correctness | deterministic | 74 | high | high | 1 | — | |
| `file-trac-ticket-before-patch` | developer-experience | guidance | 74 | high | high | 2 | yes | |
| `keep-sync-async-api-parity` | compatibility | guidance | 74 | high | high | 3 | yes | |
| `disclose-ai-assistance` | compliance | guidance | 70 | high | high | 1 | yes | |
| `end-commit-messages-with-a-period` | maintainability | guidance | 70 | high | high | 2 | yes | |
| `do-not-request-automated-ai-review` | compliance | guidance | 68 | high | high | 2 | yes | |

#### Next.js (8 rules)

| Rule | Topic | Class | Score | Band | Conf. | Evidence | Proof gap | Decision |
|---|---|---|---:|---|---|---:|---|---|
| `guard-trusted-types-exemptions` | security | guidance | 80 | very-high | medium | 2 | yes | |
| `keep-toolchain-pins-synchronized` | compatibility | guidance | 78 | high | high | 3 | yes | |
| `run-repository-lint-gate` | maintainability | deterministic | 78 | high | high | 1 | — | |
| `protect-secrets-in-agent-workflows` | security | guidance | 74 | high | high | 1 | yes | |
| `rebuild-before-integration-tests` | correctness | guidance | 74 | high | high | 1 | yes | |
| `sign-all-commits` | security | guidance | 74 | high | high | 1 | yes | |
| `attach-error-links-to-new-errors` | documentation | guidance | 70 | high | high | 1 | yes | |
| `leave-pull-requests-in-draft` | developer-experience | guidance | 62 | medium | high | 1 | yes | |

## Rule bodies

The text a `keep` decision adopts as a standard. Grouped by pass and repository,
ordered by score.

### Codex rule bodies

#### Cobra

##### `verify-go-changes` — Verify Go changes with the repository gate

correctness · deterministic · 90/100 (very-high) · confidence high · **also proposed by the other pass**

Add focused tests for behavior changes and keep Go source formatted. Before
handoff, use the repository-owned `make all` gate, which combines formatting
and tests. This standard maps the existing gate; the assessment did not execute
it.

Evidence:

- `CONTRIBUTING.md` lines 29-36 — authoritative

Decision:
##### `preserve-go-support-boundary` — Preserve the declared Go support boundary

compatibility · guidance · 86/100 (very-high) · confidence high · **also proposed by the other pass**

Treat the Go version in `go.mod` as the language floor. Preserve legacy build
constraints and avoid newer-only syntax or APIs while that floor remains. An
intentional support change must update `go.mod`, build-tag lint policy, the CI
matrix, and affected compatibility documentation together.

Evidence:

- `go.mod` lines 1-3 — authoritative
- `.golangci.yml` lines 75-81 — authoritative
- `.github/workflows/test.yml` lines 56-86 — authoritative

Proof gap: No single existing command proves source, build tags, lint configuration, and the complete supported Go matrix remain aligned.

Decision:

##### `preserve-license-headers` — Preserve required Apache license headers

compliance · deterministic · 82/100 (very-high) · confidence high · **also proposed by the other pass**

Keep the configured Apache 2.0 header on Go source files with the owner and
year range defined by CI. Preserve the workflow's `.github/**` exclusion. This
standard maps the existing license check; the assessment did not execute it.

Evidence:

- `.github/workflows/test.yml` lines 17-32 — authoritative

Decision:

##### `keep-shell-completions-portable` — Keep shell completions portable

compatibility · guidance · 79/100 (high) · confidence high

Treat Bash, Zsh, Fish, and PowerShell as the supported completion surface.
Review every affected shell when changing directives, annotations, descriptions,
or generated output. Preserve documented shell-specific differences rather than
assuming identical APIs or behavior.

Evidence:

- `README.md` lines 40-51 — authoritative
- `site/content/completions/_index.md` lines 513-520 — authoritative
- `site/content/completions/_index.md` lines 539-546 — authoritative

Proof gap: Existing tests cover individual generators but no single check proves equivalent supported behavior across Bash, Zsh, Fish, and PowerShell.

Decision:

##### `coordinate-security-fixes-privately` — Coordinate security fixes privately

security · guidance · 75/100 (high) · confidence high · **also proposed by the other pass**

Do not open a public issue or pull request for a suspected vulnerability.
Report it through the private security address with reproduction, impact, and
mitigation details. Coordinate any patch, release, and disclosure directly with
the maintainers.

Evidence:

- `SECURITY.md` lines 9-23 — authoritative

Proof gap: Private disclosure and maintainer coordination are human-governed and are intentionally not proven by a repository command.

Decision:

##### `preserve-explicit-compatibility-shims` — Preserve explicit public compatibility shims

compatibility · guidance · 72/100 (high) · confidence high · **also proposed by the other pass**

Do not remove or bypass public methods explicitly marked as retained for
backward compatibility. When changing template inheritance, preserve the
behavior of `UsageTemplate`, `HelpTemplate`, and `VersionTemplate`, or document
and test an intentional compatibility break.

Evidence:

- `command.go` lines 590-627 — authoritative

Proof gap: The repository has no single compatibility check that proves callers still receive the legacy template behavior.

Decision:

##### `keep-shell-completion-family-aligned` — Keep the shell completion family aligned

maintainability · guidance · 70/100 (high) · confidence medium · **also proposed by the other pass**

Treat the completion generators as a sibling implementation family. For a
cross-shell behavior change, review every affected generator and its focused
tests; preserve intentional signature and description differences. A new
supported shell requires a generator, corresponding tests, and user
documentation.

Evidence:

- `zsh_completions.go` lines 25-42
- `fish_completions.go` lines 276-284
- `powershell_completions.go` lines 331-348

Proof gap: Per-shell tests exist, but no aggregate command encodes which sibling generators and tests every cross-shell change must review.

Decision:

#### Flask

##### `verify-flask-runtime-matrix` — Verify Flask changes across the runtime matrix

correctness · deterministic · 92/100 (very-high) · confidence high · **also proposed by the other pass**

Add focused pytest coverage for behavior changes. Use the repository-owned tox
environment that matches the supported runtime or dependency case, then rely on
the CI matrix for the complete CPython, free-threaded, PyPy, minimum-version,
and development-version sweep. This standard maps the existing gate; the
assessment did not execute it.

Evidence:

- `pyproject.toml` lines 170-193 — authoritative
- `.github/workflows/tests.yaml` lines 13-44 — authoritative

Decision:

##### `preserve-python-support-boundary` — Preserve the declared Python support boundary

compatibility · guidance · 87/100 (very-high) · confidence high

Treat Python 3.10 as the implementation and typing floor while
`requires-python`, mypy, and Pyright declare it. Avoid newer-only syntax and
APIs unless the floor changes intentionally. Update project metadata, both type
checker configurations, tox environments, and the CI matrix together for a
support-policy change.

Evidence:

- `pyproject.toml` lines 1-30 — authoritative
- `pyproject.toml` lines 126-145 — authoritative
- `.github/workflows/tests.yaml` lines 13-44 — authoritative

Proof gap: No single existing command proves syntax, runtime metadata, both type-checker targets, and the complete CI matrix remain aligned.

Decision:

##### `keep-dual-type-checkers-green` — Keep mypy and Pyright green

correctness · deterministic · 84/100 (very-high) · confidence high · **also proposed by the other pass**

Keep public and internal annotations acceptable to both mypy and Pyright. Add
or update focused cases under `tests/type_check` for typing behavior changes,
and use the repository-owned typing tox environment. This standard maps the
existing gate; the assessment did not execute it.

Evidence:

- `pyproject.toml` lines 126-145 — authoritative
- `pyproject.toml` lines 233-245 — authoritative

Decision:

##### `run-repository-precommit-checks` — Run the repository pre-commit gate

correctness · deterministic · 82/100 (very-high) · confidence high

Use the repository-owned aggregate pre-commit command before handoff so style,
generated-file, and configured static checks see the whole tree. This standard
maps the existing gate; the assessment did not execute it.

Evidence:

- `pyproject.toml` lines 233-245 — authoritative
- `.github/workflows/pre-commit.yaml` lines 11-29 — authoritative

Decision:

##### `keep-sansio-boundary-pure` — Keep the sans-I/O boundary pure

architecture · guidance · 80/100 (very-high) · confidence high

Code under `src/flask/sansio` must remain usable by alternative
implementations. Do not perform I/O, enter a likely I/O path, or depend on Flask
globals there. Keep runtime-specific integration in the surrounding Flask
layers.

Evidence:

- `src/flask/sansio/README.md` lines 1-6 — authoritative

Proof gap: The repository states the boundary explicitly but does not provide a single check that proves a change avoids I/O paths and Flask globals.

Decision:

##### `preserve-async-wsgi-bridge` — Preserve the async-to-WSGI compatibility bridge

compatibility · guidance · 78/100 (high) · confidence high

Preserve `ensure_sync` as the boundary that adapts coroutine views and hooks to
WSGI execution. Keep plain callables unchanged, retain the overridable
conversion seam and missing-extra error, and update focused async tests and
documentation for intentional behavior changes.

Evidence:

- `docs/design.rst` lines 188-204 — authoritative
- `src/flask/app.py` lines 1065-1100 — authoritative
- `tests/test_async.py` lines 81-145

Proof gap: Focused async tests exist, but no single check proves extension compatibility, WSGI semantics, and every async hook remain aligned.

Decision:

##### `document-public-behavior-changes` — Document public behavior changes

documentation · guidance · 72/100 (high) · confidence high

For a public behavior change, add tests that fail without it, update relevant
user and code documentation, add a linked `CHANGES.rst` entry, and add
`versionchanged` annotations to affected API documentation.

Evidence:

- `.github/pull_request_template.md` lines 17-24 — authoritative

Proof gap: Documentation quality, changelog scope, and version annotations require review and are not fully proven by a repository command.

Decision:

#### Django

##### `verify-django-changes` — Verify Django changes with the repository gate

correctness · deterministic · 94/100 (very-high) · confidence high

Add focused regression coverage for behavior changes and use the
repository-owned tox gate before handoff. The default gate runs the SQLite test
suite plus configured formatting, import, documentation, workflow, and spelling
checks. This standard maps the existing gate; the assessment did not execute
it.

Evidence:

- `docs/internals/contributing/writing-code/unit-tests.txt` lines 60-91 — authoritative

Decision:

##### `preserve-deprecation-schedule` — Preserve Django's deprecation schedule

compatibility · guidance · 91/100 (very-high) · confidence high

Keep a deprecated feature working with its matching
`RemovedInDjangoXXWarning` for the required schedule. Add a warning assertion,
eliminate accidental internal use, annotate compatibility-only code, document
the upgrade path in the current release notes, and add the removal to the
deprecation timeline. Remove the shim only in its scheduled release.

Evidence:

- `docs/internals/release-process.txt` lines 68-103 — authoritative
- `docs/internals/contributing/writing-code/submitting-patches.txt` lines 304-390 — authoritative

Proof gap: The full two-feature-release lifecycle spans multiple releases and cannot be proven by one repository command.

Decision:

##### `preserve-python-support-boundary` — Preserve the declared Python support boundary

compatibility · guidance · 88/100 (very-high) · confidence high

Treat Python 3.12 as the source floor while project metadata declares it, and
preserve the 3.12 through 3.14 plus free-threading support surface. Avoid
newer-only syntax or APIs unless the support policy changes intentionally.
Update `requires-python`, classifiers, formatter targets, and runtime workflows
together for a support change.

Evidence:

- `pyproject.toml` lines 5-39 — authoritative
- `.github/workflows/python_matrix.yml` lines 17-60 — authoritative

Proof gap: The label-gated matrix checks declared runtimes, but no single command proves syntax, classifiers, free-threading behavior, and all platform seams remain aligned.

Decision:

##### `disclose-ai-assistance` — Disclose AI assistance accurately

compliance · guidance · 87/100 (very-high) · confidence high · **also proposed by the other pass**

For an AI-assisted contribution, identify the tool and version and describe how
it was used. Fully review and verify its output, do not invent APIs, behavior,
or citations, complete the pull-request template, and call out any uncertain
compliance to reviewers.

Evidence:

- `docs/internals/contributing/writing-code/submitting-patches.txt` lines 192-218 — authoritative
- `.github/pull_request_template.md` lines 10-21 — authoritative

Proof gap: Accurate disclosure, complete human review, and absence of fabricated material require contributor and reviewer judgment.

Decision:

##### `keep-database-backends-aligned` — Keep supported database backends aligned

compatibility · guidance · 83/100 (very-high) · confidence high

Review ORM and schema changes against PostgreSQL, MariaDB, MySQL, Oracle, and
SQLite. Keep shared behavior in the base backend, encode intentional capability
differences in backend features and skips, and add backend-specific tests where
semantics differ. Do not treat a passing SQLite run as proof for every backend.

Evidence:

- `docs/ref/databases.txt` lines 5-19 — authoritative
- `docs/internals/contributing/writing-code/unit-tests.txt` lines 92-109 — authoritative
- `docs/internals/contributing/writing-code/unit-tests.txt` lines 160-170 — authoritative

Proof gap: SQLite is the bundled default, while full PostgreSQL, MariaDB, MySQL, and Oracle coverage requires separate settings and CI builders.

Decision:

##### `keep-sync-async-apis-aligned` — Keep synchronous and asynchronous APIs aligned

compatibility · guidance · 81/100 (very-high) · confidence high

Treat a synchronous method and its `a`-prefixed asynchronous variant as one
public API family. Keep arguments, return behavior, validation, backend routing,
and documented limitations aligned unless a difference is intentional. Update
focused tests and documentation for both call paths.

Evidence:

- `docs/topics/async.txt` lines 140-154 — authoritative
- `docs/topics/tasks.txt` lines 136-143 — authoritative
- `docs/topics/cache.txt` lines 1301-1310 — authoritative

Proof gap: Paired tests exist across subsystems, but no single command proves every synchronous and asynchronous API pair has matching arguments and semantics.

Decision:

##### `validate-documentation-changes` — Validate Django documentation changes

documentation · deterministic · 79/100 (high) · confidence high

Run the repository's documentation lint, formatting, and warning-as-error
spelling checks for documentation changes. Mark new and changed public behavior
with self-contained `versionadded` or `versionchanged` blocks. This standard
maps the existing workflow; the assessment did not execute it.

Evidence:

- `.github/workflows/docs.yml` lines 22-48 — authoritative
- `docs/internals/contributing/writing-documentation.txt` lines 619-640 — authoritative

Decision:

##### `check-test-migrations` — Check test migration integrity

correctness · deterministic · 76/100 (high) · confidence high · **also proposed by the other pass**

Keep test app models and checked-in test migrations consistent. Use the
repository migration checker with the workflow's PostgreSQL settings after the
service and settings file are prepared. This standard maps the existing
workflow; the assessment did not execute it.

Evidence:

- `.github/workflows/check-migrations.yml` lines 22-68 — authoritative

Decision:

##### `do-not-request-automated-ai-review` — Do not request automated AI review

compliance · guidance · 74/100 (high) · confidence high · **also proposed by the other pass**

Do not request an automated AI review on a pull request submitted to the Django
repository. Use such tools only in a fork before submission, and leave review
of the submitted contribution to human reviewers.

Evidence:

- `docs/internals/contributing/writing-code/submitting-patches.txt` lines 182-190 — authoritative
- `.github/pull_request_template.md` lines 10-21 — authoritative

Proof gap: Review requests occur on the hosting platform and require contributor compliance rather than repository-local proof.

Decision:

#### Next.js

##### `verify-across-nextjs-modes` — Verify changes in the affected modes and bundlers

correctness · guidance · 92/100 (very-high) · confidence high

Rebuild the affected packages before integration tests, then run focused tests
in each runtime mode and bundler the change can affect. Match CI environment
flags when reproducing a failure; do not treat a passing default Turbopack
development test as proof for Webpack, Rspack, production, deploy, or
experimental variants.

Evidence:

- `AGENTS.md` lines 71-105 — authoritative
- `AGENTS.md` lines 114-166 — authoritative
- `package.json` lines 21-81 — authoritative

Proof gap: The repository provides deterministic mode-specific commands, but selecting every affected dev, start, deploy, stable, experimental, Turbopack, Webpack, and Rspack path requires change-specific judgment.

Decision:

##### `wire-feature-flags-end-to-end` — Wire feature flags end to end

architecture · guidance · 90/100 (very-high) · confidence high

When adding or changing a framework flag, follow
`.agents/skills/flags/SKILL.md` and wire every applicable surface:
configuration type and schema, user-bundle definition, runtime configuration,
server and export startup, bundle definitions, and runtime selection.
Distinguish user-bundled code from precompiled runtime bundles before choosing
the wiring path.

Evidence:

- `AGENTS.md` lines 464-471 — authoritative
- `.agents/skills/flags/SKILL.md` lines 17-39 — authoritative

Proof gap: Type, schema, build-time, runtime, export, bundle-definition, and bundle-selection checks are distributed; no single gate proves that a flag reaches every consumer and variant without leaking into unrelated bundles.

Decision:

##### `preserve-edge-dce-boundaries` — Preserve edge dead-code-elimination boundaries

architecture · guidance · 88/100 (very-high) · confidence high

Keep platform-only `require()` calls inside compile-time `if/else` branches
that the bundler can eliminate. Force flags guarding Node-only imports to false
for edge builds, do not use `NEXT_RUNTIME` as a feature flag, and verify
affected paths with the isolated Webpack edge test prescribed by
`.agents/skills/dce-edge/SKILL.md`.

Evidence:

- `AGENTS.md` lines 464-471 — authoritative
- `.agents/skills/dce-edge/SKILL.md` lines 18-62 — authoritative

Proof gap: The focused edge build catches known regressions, but no checker proves that every conditional import uses a compile-time-eliminable branch or that all Node-only feature flags are false in edge builds.

Decision:

##### `preserve-react-server-vendoring-boundary` — Preserve the React server vendoring boundary

architecture · guidance · 87/100 (very-high) · confidence high

Route all `react-server-dom-webpack/*` server and static APIs through
`entry-base.ts`; access them elsewhere through the exposed component module.
When adding vendored React APIs, update the internal declarations and affected
stable, experimental, Webpack, and Turbopack surfaces described by
`.agents/skills/react-vendoring/SKILL.md`.

Evidence:

- `AGENTS.md` lines 464-471 — authoritative
- `.agents/skills/react-vendoring/SKILL.md` lines 19-33 — authoritative
- `.agents/skills/react-vendoring/SKILL.md` lines 64-66 — authoritative

Proof gap: Development can mask a direct React server import that fails in production, and no single check proves stable and experimental vendored channels, declarations, gateway exports, and Turbopack remaps remain aligned.

Decision:

##### `filter-internal-request-headers` — Filter new internal request headers

security · guidance · 85/100 (very-high) · confidence high

Treat any newly consumed nonstandard request header as attacker-controlled
until reviewed. If it is framework-internal, add it to the `INTERNAL_HEADERS`
filtering boundary before downstream server code can read it, and add a focused
regression for direct external requests.

Evidence:

- `AGENTS.md` lines 508-512 — authoritative

Proof gap: The router filters a maintained header list, but no checker determines whether every newly consumed nonstandard request header is internal, forgeable, and represented at the entry-point filter.

Decision:

##### `run-nextjs-lint-gate` — Run the repository lint gate

quality · deterministic · 85/100 (very-high) · confidence high

Run `pnpm lint` before handing off repository changes so TypeScript,
formatting, ESLint, AST-grep, language, and unused-task checks execute through
the maintained aggregate gate. This standard maps the existing gate; the
assessment did not execute it.

Evidence:

- `AGENTS.md` lines 227-235 — authoritative

Decision:

##### `generate-isolated-regression-tests` — Generate isolated regression tests

testability · guidance · 84/100 (very-high) · confidence high

Create new suites with `pnpm new-test` so they use the repository's typed
fixture structure and `nextTestSetup` isolation. Demonstrate that a fix's
regression test fails without the fix, add checks to a closely related existing
suite when appropriate, and use condition-based polling rather than fixed
sleeps.

Evidence:

- `AGENTS.md` lines 150-225 — authoritative
- `contributing/core/testing.md` lines 44-78 — authoritative

Proof gap: The generator standardizes new suites, but no check proves that a regression test fails without the fix, uses the right suite, avoids timing sleeps, and remains isolated from monorepo state.

Decision:

##### `use-pinned-node-pnpm-toolchain` — Use the pinned Node and pnpm toolchain

compatibility · guidance · 80/100 (very-high) · confidence high

Use pnpm through the root `packageManager` declaration and use the Node version
selected by `.node-version`, while preserving the package's declared minimum
Node version. Update toolchain metadata, lockfile behavior, contributor
guidance, and CI together for an intentional version change.

Evidence:

- `package.json` lines 306-310 — authoritative
- `.node-version` lines 1-1 — authoritative
- `contributing/core/developing.md` lines 7-29 — authoritative

Proof gap: Package-manager metadata aligns local installs with CI, but no check proves that all syntax, scripts, lockfile behavior, and native tooling remain valid across the declared Node floor and selected Node major.

Decision:

##### `preserve-source-generated-output-boundary` — Preserve the source and generated-output boundary

maintainability · guidance · 78/100 (high) · confidence high

Make core framework changes in `packages/next/src` or the owning build
configuration, not in derived `packages/next/dist` output. Regenerate compiled
JavaScript, runtime bundles, source maps, and declarations through repository
build tasks, and review generated diffs only as outputs of those source
changes.

Evidence:

- `AGENTS.md` lines 22-46 — authoritative
- `AGENTS.md` lines 396-412 — authoritative
- `contributing/core/building.md` lines 14-30 — authoritative

Proof gap: Build tasks regenerate compiled JavaScript, bundles, maps, and declarations, but no check proves that a submitted change modified the canonical source rather than patching derived output.

Decision:

##### `attach-helpful-error-links` — Attach helpful links to warnings and errors

developer-experience · guidance · 72/100 (high) · confidence high

For each new user-facing warning or error, run `pnpm new-error`, write
actionable explanatory documentation, and attach the generated URL to the
runtime message. Keep the logged message concise without removing the context
users need to resolve it.

Evidence:

- `contributing/core/adding-error-links.md` lines 1-12 — authoritative
- `package.json` lines 8-15 — authoritative

Proof gap: The generator creates documentation and updates the manifest, but no check proves that every new user-facing warning or error includes its generated URL and actionable resolution guidance.

Decision:

### Claude rule bodies

#### Cobra

##### `verify-go-changes` — Verify Go changes with the repository gate

correctness · deterministic · 89/100 (very-high) · confidence high · **also proposed by the other pass**

Add adequate tests for any Go change, then run the repository's `make all` gate, which checks `gofmt` formatting and runs the full Go test suite. The contribution guide names both obligations together: tests for the feature, and formatted code verified through `make all`.

Evidence:

- `CONTRIBUTING.md` lines 33-36 — authoritative
- `Makefile` lines 12-16 — authoritative

Decision:

##### `preserve-go-support-boundary` — Keep the Go support boundary coherent

compatibility · guidance · 84/100 (very-high) · confidence high · **also proposed by the other pass**

Treat the `go 1.15` directive in `go.mod` as the declared language floor and avoid syntax, standard-library APIs, or dependencies that require a newer version. Raising the floor is a coordinated change: update `go.mod`, the lint configuration's `buildtag` exemption and its explanatory comment, the CI version matrix, and the build-tag syntax assumptions in the same reviewed change. Note the existing gap between the declared floor and the tested matrix when reasoning about what is actually verified.

Evidence:

- `go.mod` lines 1-3 — authoritative
- `.golangci.yml` lines 75-81 — authoritative
- `.github/workflows/test.yml` lines 56-71 — authoritative

Proof gap: The CI matrix's lowest tested version is Go 1.17, while `go.mod` declares `go 1.15`. No automated job builds or tests the declared floor, so the declared boundary is asserted rather than proven.

Decision:

##### `preserve-license-headers` — Preserve the Apache license header

compliance · deterministic · 84/100 (very-high) · confidence high · **also proposed by the other pass**

Keep the `The Cobra Authors` Apache-2.0 header on every new or modified file that the `lic-headers` CI job covers. All 36 Go source files carry it at this baseline. The job excludes `.github/**`; do not widen or narrow that exclusion without maintainer intent, because the exclusion is part of the checked contract.

Evidence:

- `.github/workflows/test.yml` lines 17-32 — authoritative

Decision:

##### `keep-shell-completion-family-aligned` — Keep the shell completion family aligned

compatibility · guidance · 81/100 (very-high) · confidence high · **also proposed by the other pass**

Bash, Zsh, fish, and PowerShell are the documented supported shells, and each one has a generator, a focused test file, and a page under `site/content/completions`. When you change shared completion behavior — directives, descriptions, active help, or flag completion — review every affected generator and update its test and its documentation together. Respect the documented per-shell limitations instead of assuming behavior is uniform; the documentation records real differences per shell. Not every change touches every shell, but every shell a change affects must move together with its test and docs.

Evidence:

- `site/content/completions/_index.md` lines 1-8 — authoritative
- `zsh_completions.go` lines 25-42
- `fish_completions.go` lines 276-284
- `powershell_completions.go` lines 331-348

Proof gap: Each generator has its own focused test file, but no check asserts that a shared completion behavior was applied to every supported shell or that the documentation set stayed complete.

Related skill: `add-shell-completion-generator`

Decision:

##### `coordinate-security-fixes-privately` — Coordinate security fixes privately

security · guidance · 76/100 (high) · confidence high · **also proposed by the other pass**

Never open a public issue or public pull request for a suspected Cobra vulnerability or for its fix. Report it to `cobra-security@googlegroups.com` with description, reproduction steps, impact, and any known mitigations, then coordinate the patch, release, and disclosure privately with the maintainers. This applies to fixes you already have in hand, not only to unfixed reports.

Evidence:

- `SECURITY.md` lines 9-23 — authoritative

Proof gap: Disclosure behavior is a human process. No repository check can detect a vulnerability discussed in a public issue or pull request before it is published.

Decision:

##### `preserve-dual-build-tag-syntax` — Keep both build-tag syntaxes on platform files

compatibility · guidance · 74/100 (high) · confidence high

Platform-specific files must carry both the modern `//go:build` constraint and the legacy `// +build` constraint. The lint configuration disables the govet `buildtag` check specifically to allow this, and records that it is required for Go 1.15 compatibility because `//go:build` was only introduced in Go 1.17. Removing the legacy line would silently drop support for the declared minimum Go version, and the disabled check means nothing will catch it.

Evidence:

- `.golangci.yml` lines 75-81 — authoritative
- `command_win.go` lines 15-16
- `command_notwin.go` lines 15-16

Proof gap: The repository deliberately disables the govet `buildtag` check to permit the dual syntax, so no automated check verifies that the legacy `// +build` line is still present alongside `//go:build`.

Decision:

##### `preserve-explicit-compatibility-shims` — Preserve explicitly marked compatibility shims

compatibility · guidance · 73/100 (high) · confidence medium · **also proposed by the other pass**

Where a declaration is annotated as existing only for compatibility with downstream users, treat it as a public contract rather than dead code. At this baseline that covers the `Gt`, `Eq`, and `appendIfNotPresent` helpers marked `FIXME ... exists only for compatibility`, and the `UsageTemplate`, `HelpTemplate`, and `VersionTemplate` methods marked `kept for backwards-compatibility reasons`. Do not delete or change the signature of a shim annotated this way outside an intentional, maintainer-approved major-version break, even if static analysis reports it as unused.

Evidence:

- `cobra.go` lines 109-114 — authoritative
- `command.go` lines 590-592 — authoritative
- `command.go` lines 603-605 — authoritative
- `command.go` lines 616-618 — authoritative

Proof gap: No check distinguishes an intentional compatibility export from dead code, so a coverage or dead-code tool may report these symbols as removable when they are not.

Decision:

##### `keep-doc-generator-autogen-tag-aligned` — Keep the auto-generated tag aligned across doc generators

maintainability · guidance · 62/100 (medium) · confidence medium

The Man, Markdown, and reStructuredText generators each emit an "Auto generated by spf13/cobra" line gated on `DisableAutoGenTag`, and the documentation states the option removes that string from documentation output. The YAML generator is intentionally outside this obligation because it emits no such tag. When changing the auto-generated tag text, its date format, or the meaning of `DisableAutoGenTag`, update every generator that emits the tag and its focused test, and keep the documented behavior accurate.

Evidence:

- `site/content/docgen/_index.md` lines 8-13 — authoritative
- `doc/man_docs.go` lines 242-243
- `doc/md_docs.go` lines 112-113
- `doc/rest_docs.go` lines 125-126

Proof gap: Each generator has focused tests, but no check asserts that the documented `DisableAutoGenTag` behavior is honored by every generator that emits the tag.

Decision:

#### Flask

##### `verify-flask-runtime-matrix` — Verify changes across the supported runtime matrix

correctness · deterministic · 88/100 (very-high) · confidence high · **also proposed by the other pass**

Add focused tests for a behavior change, then verify it through the repository's tox environments rather than a single interpreter. The declared matrix is broader than "latest Python": CPython 3.10 through 3.14, the 3.14 free-threaded build, PyPy 3.11, a minimum-dependency-version environment, and a development-dependency environment built from the Pallets main branches. Windows and macOS are covered too. A change that passes only on the newest CPython with current dependencies has not been verified against the supported surface.

Evidence:

- `pyproject.toml` lines 170-179 — authoritative
- `.github/workflows/tests.yaml` lines 16-30 — authoritative

Decision:

##### `harden-github-actions-workflows` — Keep GitHub Actions workflows hardened

security · guidance · 86/100 (very-high) · confidence high

Workflow files follow a deliberate hardening posture, and the repository runs `zizmor` — a GitHub Actions security scanner — against every workflow change to enforce it. When adding or editing a workflow, keep all three properties: pin every third-party action to a full commit SHA with the human-readable version in a trailing comment, declare `permissions: {}` at workflow level and grant only the narrower permissions a job actually needs, and set `persist-credentials: false` on every `actions/checkout` step. Action pins are refreshed through the `update-actions` tox environment rather than by hand.

Evidence:

- `.github/workflows/zizmor.yaml` lines 12-22 — authoritative
- `.github/workflows/tests.yaml` lines 8-8 — authoritative
- `.github/workflows/tests.yaml` lines 32-34 — authoritative

Proof gap: The `zizmor` job enforces this posture in CI on any workflow change, but the repository defines no locally runnable zizmor invocation, so there is no repository-local command to cite or run before pushing.

Decision:

##### `preserve-python-support-floor` — Keep the Python support floor coherent

compatibility · guidance · 86/100 (very-high) · confidence high

Python 3.10 is the support floor, declared in three places that must agree: `requires-python`, the mypy `python_version`, and the pyright `pythonVersion`. Avoid syntax, typing constructs, and standard-library APIs introduced after 3.10. Raising the floor is a coordinated change — update all three declarations, the tox environment list, and the CI matrix in the same reviewed change.

Evidence:

- `pyproject.toml` lines 22-22 — authoritative
- `pyproject.toml` lines 126-131 — authoritative
- `pyproject.toml` lines 142-145 — authoritative

Proof gap: The type checkers are pinned to 3.10 and will reject typing syntax newer than the floor, but no check catches a runtime-only standard-library API that exists solely in a later version, because the 3.10 test environment must actually execute the affected path to fail.

Related skill: `add-supported-python-version`

Decision:

##### `keep-sansio-layer-protocol-agnostic` — Keep the sans-I/O layer protocol agnostic

architecture · guidance · 84/100 (very-high) · confidence high

`src/flask/sansio` exists so that alternative implementations such as Quart can reuse Flask's routing and application scaffolding. Its stated constraints are that the code cannot perform I/O, cannot sit on a likely I/O path, and cannot use the Flask globals. Honor all three: put WSGI-specific behavior in the `Flask` and `Blueprint` subclasses that extend the sans-I/O classes, and reach for `current_app`, `request`, `session`, or `g` only outside this package. Importing a module such as `os` for pure path computation does not by itself breach the boundary; performing or enabling I/O does.

Evidence:

- `src/flask/sansio/README.md` lines 1-6 — authoritative
- `src/flask/app.py` lines 44-44
- `src/flask/blueprints.py` lines 10-12

Proof gap: No linter rule, import-graph check, or test asserts the sans-I/O constraints. The boundary is maintained by review only, so a violating import or a Flask-global reference inside `src/flask/sansio` would pass every existing check.

Decision:

##### `keep-dual-type-checkers-green` — Keep both type checkers green

correctness · deterministic · 78/100 (high) · confidence high · **also proposed by the other pass**

The repository runs two independent type checkers over the same targets: mypy in `strict` mode and pyright in `basic` mode, both across `src` and `tests/type_check`. The `typing` tox environment runs both in sequence. Treat passing one checker as insufficient — they disagree in practice, which is why both are configured. When you change public annotations or callback protocols, update the fixtures under `tests/type_check` that exercise the affected signature and keep both checkers clean.

Evidence:

- `pyproject.toml` lines 126-131 — authoritative
- `pyproject.toml` lines 142-145 — authoritative
- `pyproject.toml` lines 239-245 — authoritative

Decision:

##### `pin-and-freeze-pre-commit-hooks` — Keep pre-commit hooks pinned and frozen

security · guidance · 76/100 (high) · confidence high

Every pre-commit repository is pinned to a full commit SHA with a trailing `# frozen: <version>` comment, matching the same supply-chain posture applied to GitHub Actions. Do not replace a pinned SHA with a mutable tag or branch. Refresh pins through the `update-pre_commit` tox environment, which runs `pre-commit autoupdate --freeze` and preserves the frozen-SHA form, rather than editing revisions by hand.

Evidence:

- `.pre-commit-config.yaml` lines 1-23 — authoritative
- `pyproject.toml` lines 264-269 — authoritative

Proof gap: Nothing rejects a hand-written mutable version tag in place of a pinned SHA. The frozen-SHA convention is maintained by using the update environment rather than by a check.

Decision:

##### `document-public-changes-in-changes-rst` — Document public changes in CHANGES and version directives

documentation · guidance · 68/100 (high) · confidence medium

The pull-request template is the in-repository contribution contract; `docs/contributing.rst` is a short pointer to the external Pallets guide rather than a checklist. For a public behavior change the template requires four things together: a test that demonstrates the correct behavior and would fail without the change, updated documentation in the `docs` folder and in code, an entry in `CHANGES.rst` summarizing the change and linking the issue, and a `.. versionchanged::` directive in the affected code documentation. Treat all four as one obligation rather than a menu.

Evidence:

- `.github/pull_request_template.md` lines 17-25 — authoritative

Proof gap: No check verifies that a `CHANGES.rst` entry or a `.. versionchanged::` directive accompanies a behavior change, and none can confirm that a new test actually fails without the change.

Decision:

##### `keep-test-warnings-fatal` — Keep warnings fatal in the test suite

maintainability · deterministic · 66/100 (high) · confidence high

The pytest configuration turns every warning into an error. A deprecation warning raised by Flask itself or by an upstream Pallets dependency therefore fails the suite rather than scrolling past. Do not silence a new warning with a broad `filterwarnings` entry or a blanket suppression; fix the deprecated usage. When a warning is genuinely expected, assert it in the specific test with `pytest.warns` so the expectation stays local and visible.

Evidence:

- `pyproject.toml` lines 106-110 — authoritative

Decision:

#### Django

##### `preserve-deprecation-cycle` — Follow the two-release deprecation cycle

compatibility · guidance · 86/100 (very-high) · confidence high

Deprecations run over at least two feature releases. A feature deprecated in A.x keeps working through every A.x release while raising a warning, and is removed in B.0 — or B.1 when it was deprecated in the last A.x release. Never remove a public feature in the same release that deprecates it.

Use the warning classes rather than a bare `DeprecationWarning`: `RemovedInDjango70Warning` for the deprecation ending in the next major release, and `RemovedInDjango71Warning` for the one after. The `RemovedInNextVersionWarning` and `RemovedAfterNextVersionWarning` aliases exist so that references do not need editing each cycle; prefer them when writing code that should survive the version bump. When you add a deprecation, add the release-note entry and the timeline in the same change.

Evidence:

- `docs/internals/release-process.txt` lines 84-92 — authoritative
- `django/utils/deprecation.py` lines 13-22 — authoritative

Proof gap: No check confirms that a removal is due in the current release or that a new deprecation used the correct warning class. Removing a feature one release early passes every test that does not assert on the warning.

Decision:

##### `keep-database-backend-modules-aligned` — Keep database backend modules aligned with the base interface

architecture · guidance · 78/100 (high) · confidence high

`django/db/backends/base` defines the backend contract across `base`, `client`, `creation`, `features`, `introspection`, `operations`, `schema`, and `validation`. The `postgresql`, `mysql`, `sqlite3`, and `oracle` backends each implement that contract by subclassing, and each declares its own `DatabaseFeatures`.

When you add or change a capability flag, a SQL-generating operation, or a schema behavior on a base class, review every affected backend and set the flag explicitly where the default is wrong for that database. Not every change touches every backend, but a change that silently relies on an inherited default for a backend that behaves differently is a defect. Membership of the module set differs by backend on purpose — `sqlite3` and `postgresql` have no `validation.py`, and `oracle` and `sqlite3` have no `compiler.py` — so match the modules the affected backend actually defines rather than assuming a uniform set.

Evidence:

- `django/db/backends/base/features.py` lines 5-10
- `django/db/backends/postgresql/features.py` lines 9-12
- `django/db/backends/mysql/features.py` lines 7-10

Proof gap: The full backend matrix is exercised only by the scheduled and per-database CI runs, not by the default test job. A capability flag added to the base class without a per-backend override silently inherits the base default.

Decision:

##### `run-repository-linters` — Run the repository linters before handoff

maintainability · deterministic · 78/100 (high) · confidence high

Python changes must satisfy the repository's lint gates: flake8 with an 88-character code limit, a 79-character documentation limit, and `E203` extended-ignore; isort across `django`, `tests`, and `scripts`; and black formatting. JavaScript changes must satisfy biome, and workflow changes must satisfy zizmor. The flake8 configuration carries a small set of deliberate per-file ignores; do not add to that list to silence a new finding without maintainer intent.

Evidence:

- `.flake8` lines 1-10 — authoritative
- `.github/workflows/linters.yml` lines 55-60 — authoritative

Decision:

##### `check-test-migrations` — Check for missing test migrations

correctness · deterministic · 74/100 (high) · confidence high · **also proposed by the other pass**

When you change a model under `tests/`, generate the corresponding migration in the same change. The repository has a dedicated check that fails when a test app's models and migrations have drifted apart, and it runs on any change to `tests/**/models.py` or a test migrations directory. The check requires a PostgreSQL settings module, which CI supplies from a template.

Evidence:

- `.github/workflows/check-migrations.yml` lines 63-68 — authoritative

Decision:

##### `file-trac-ticket-before-patch` — File a Trac ticket before a non-trivial patch

developer-experience · guidance · 74/100 (high) · confidence high

Django tracks work in Trac, not in GitHub issues. Any change beyond a typo fix needs a Trac ticket before the pull request, and the pull-request description must carry the ticket number in the `ticket-XXXXX` line — or `N/A - typo` for a typo fix. The contribution guide states plainly that non-trivial pull requests without tickets will be closed. Target the `main` branch; mergers evaluate and perform backports.

Evidence:

- `CONTRIBUTING.rst` lines 18-26 — authoritative
- `.github/pull_request_template.md` lines 1-5 — authoritative

Proof gap: No automated check resolves the `ticket-XXXXX` line against Trac or rejects a placeholder, so an unticketed pull request fails review rather than CI.

Decision:

##### `keep-sync-async-api-parity` — Keep sync and async public API variants in parity

compatibility · guidance · 74/100 (high) · confidence high

Django exposes async variants of public sync APIs under an `a` prefix — `aget` beside `get`, `aset` beside `set`, `acount` beside `count`, `ahas_perm` beside `has_perm`, and many more across the cache backends, paginator, auth models, ORM, dispatcher, tasks, and test client.

When you change the signature, default arguments, return type, or raised exceptions of a public method that has an `a`-prefixed counterpart, change both together and cover both in tests. When you add a public method to a class that already exposes async variants, add the async variant too rather than leaving the pair incomplete. Keep the semantics identical apart from awaiting: a difference in behavior between a pair is a bug, not a design choice.

Evidence:

- `django/core/cache/backends/base.py` lines 144-166
- `django/core/paginator.py` lines 242-260
- `django/contrib/auth/models.py` lines 383-407

Proof gap: Nothing asserts that a sync public method has its `a`-prefixed counterpart, or that the two agree on signature and semantics. A signature change applied to only one half of a pair passes the suite unless a test happens to cover the async variant.

Related skill: `add-async-api-variant`

Decision:

##### `disclose-ai-assistance` — Disclose AI assistance on every pull request

compliance · guidance · 70/100 (high) · confidence high · **also proposed by the other pass**

The pull-request template makes AI-assistance disclosure required, not optional. Select exactly one of the two statements: that no AI tools were used, or that AI tools were used, are named, and their output was fully reviewed and verified by you. When tools were used, list which ones. Leaving both boxes unchecked, or checking the second without naming the tools, does not satisfy the requirement.

Evidence:

- `.github/pull_request_template.md` lines 10-14 — authoritative

Proof gap: The disclosure is a template checkbox. No check verifies that one option was selected, that the named tools are complete, or that the stated review actually happened.

Decision:

##### `end-commit-messages-with-a-period` — End commit messages with a period

maintainability · guidance · 70/100 (high) · confidence high

Every commit message subject must end with a period. The `check-commit-suffix` CI job inspects each commit in the pull request and fails when one does not. Write the subject in past tense and mention the Trac ticket number where applicable. Pull requests against a `stable/` branch additionally require the release prefix — for example `[5.2]` — on both the pull-request title and every commit subject, enforced by the companion `check-commit-prefix` job.

Evidence:

- `.github/workflows/check_commit_messages.yml` lines 93-114 — authoritative
- `.github/pull_request_template.md` lines 20-20 — authoritative

Proof gap: The `check-commit-suffix` job enforces the trailing period, but it runs only on the `django/django` repository and only in CI. The repository defines no locally runnable command or commit hook, so the failure appears after pushing rather than before.

Decision:

##### `do-not-request-automated-ai-review` — Do not request an automated AI review

compliance · guidance · 68/100 (high) · confidence high · **also proposed by the other pass**

Do not request an automated AI review on a pull request to this repository, and do not commit to doing so later. The repository states this twice: the pull-request checklist asks you to confirm you have not requested and will not request one, and the repository's assistant instruction file directs any configured reviewer to decline and reply that the reviewer should be used in a fork instead. Automated review in your own fork is explicitly welcome; on the upstream pull request it is not.

Evidence:

- `.github/pull_request_template.md` lines 21-21 — authoritative
- `.github/copilot-instructions.md` lines 8-10 — authoritative

Proof gap: The repository can suppress its own configured assistant's output but cannot prevent a contributor from invoking a third-party review bot, so compliance is social rather than enforced.

Decision:

#### Next.js

##### `guard-trusted-types-exemptions` — Guard the Trusted Types exemption allowlist

security · guidance · 80/100 (very-high) · confidence medium

The repository runs `tsec` to ban unsafe DOM sinks, with a per-rule allowlist in `tsec-exemptions.json` naming the exact files permitted to use each banned pattern — for example `ban-element-innerhtml-assignments` and `ban-element-setattribute`. Treat that file as a security boundary, not boilerplate. Prefer a safe API over an exemption. When an exemption is genuinely unavoidable, add the narrowest possible file entry under the specific rule and explain why in the pull request; do not add a directory, a glob, or a whole package. Removing a file from the allowlist when its unsafe usage goes away is part of the change.

Evidence:

- `tsconfig-tsec.json` lines 1-10 — authoritative
- `tsec-exemptions.json` lines 1-8 — authoritative

Proof gap: tsec is configured through a separate `tsconfig-tsec.json` rather than the default `tsconfig.json`, and the inventory does not show it wired into the `pnpm lint` gate. Whether a new unexempted violation fails a routine local run could not be confirmed from repository files alone.

Decision:

##### `keep-toolchain-pins-synchronized` — Keep toolchain pins synchronized

compatibility · guidance · 78/100 (high) · confidence high

The repository pins its toolchains in several coupled places: the Rust channel and components in `rust-toolchain.toml`, the Node version in `.node-version`, and the Node engine range plus the exact pnpm version in `package.json`. `rust-toolchain.toml` carries an explicit instruction in its own comments — updating the channel also requires updating the devcontainer Rust feature definition, and moving the file requires updating any `turbo.json` inputs that reference it. Honor that coupling: change every dependent declaration in the same reviewed change rather than leaving the container or the task graph pinned to a different toolchain than the repository.

Evidence:

- `rust-toolchain.toml` lines 1-6 — authoritative
- `.node-version` lines 1-1 — authoritative
- `package.json` lines 306-309 — authoritative

Proof gap: No check compares the pinned Rust channel against the devcontainer feature definition or the `turbo.json` inputs that reference the file. A drifted pin surfaces as an environment-specific build difference rather than a failing job.

Decision:

##### `run-repository-lint-gate` — Run the repository lint gate before handoff

maintainability · deterministic · 78/100 (high) · confidence high

`pnpm lint` is the single gate covering TypeScript types, prettier formatting, eslint, and ast-grep rules; `pnpm types` runs type checking alone. Run the gate before handing off a change rather than relying on the pre-commit hook, which is slow enough that a failure costs a full cycle. For a targeted pre-check, run prettier and eslint directly against the changed files using the repository's own configuration paths, which is what the hook does. Do not silence an ast-grep or eslint finding by broadening an ignore file when the underlying pattern is the problem.

Evidence:

- `AGENTS.md` lines 227-234 — authoritative

Decision:

##### `protect-secrets-in-agent-workflows` — Never expose secret values

security · guidance · 74/100 (high) · confidence high

Treat every environment variable value as sensitive unless it is a known test-mode flag. Never print or paste tokens, API keys, or cookies into responses, commits, or shared logs. Mirror CI environment variable *names and modes* exactly, but never inline a literal secret value into a command. When a required secret is missing locally, stop and ask rather than inventing placeholder credentials. Never commit a local secret file, and use placeholder-only examples when documenting environment setup. When sharing command output, summarize and redact anything that looks sensitive.

Evidence:

- `AGENTS.md` lines 350-358 — authoritative

Proof gap: No scanner in the repository detects a secret pasted into a chat response, a shared log, or a command transcript. The obligation covers channels outside the repository entirely.

Decision:

##### `rebuild-before-integration-tests` — Rebuild before running integration tests

correctness · guidance · 74/100 (high) · confidence high

Next.js integration tests run against built output, not source. A test result obtained without rebuilding after a source change is meaningless in both directions — it can pass despite a broken change or fail despite a correct one. Rebuild according to what changed: `pnpm build-all` on the first run after a branch switch or bootstrap, or whenever you are unsure; `pnpm --filter=next build` when only files under `packages/next` changed after bootstrap; and `pnpm build-all` again whenever Rust or Turbopack code changed. Treat "did I rebuild?" as the first question when a test result looks surprising.

Evidence:

- `AGENTS.md` lines 441-447 — authoritative

Proof gap: Nothing detects a stale build. Integration tests run happily against previously built output, so the failure mode is a passing or failing result that does not reflect the working tree.

Decision:

##### `sign-all-commits` — Sign every commit with a verified key

security · guidance · 74/100 (high) · confidence high

Protected branches require verified commit signatures. Configure Git to sign with a GPG, SSH, or S/MIME key that is registered on your GitHub account, and confirm commits show as `Verified` before opening a pull request. A `Signed-off-by` trailer does not satisfy this requirement — it is a text line, not a signature. If a branch already contains unsigned commits, re-sign them and force-push; they cannot be merged as-is.

Evidence:

- `contributing.md` lines 8-16 — authoritative

Proof gap: Signature verification is enforced by repository branch-protection rules on the hosting platform, not by any check defined in the repository. Nothing in the working tree will reject an unsigned commit locally.

Decision:

##### `attach-error-links-to-new-errors` — Attach a documentation link to every new error

documentation · guidance · 70/100 (high) · confidence high

New user-facing warnings and errors should carry a documentation link so the logged message can stay short while the full explanation lives in the docs. The repository maintains 254 error documents under `errors/` at this baseline. Use `pnpm new-error` to create the document and update the manifest rather than hand-writing either — the command prints the URL to embed in the error message. Do not reference an error slug that does not exist, and do not add an error document without wiring its link into the message it explains.

Evidence:

- `contributing/core/adding-error-links.md` lines 1-12 — authoritative

Proof gap: No check requires a new user-facing warning or error to carry a documentation link, and none verifies that a referenced slug exists under `errors/`.

Decision:

##### `leave-pull-requests-in-draft` — Leave pull requests in draft and keep commit trailers clean

developer-experience · guidance · 62/100 (medium) · confidence high

Do not add "Generated with Claude Code" or co-author footers to commits or pull requests. Keep commit subjects concise and descriptive, and write pull-request descriptions around what changed and why. Do not mark a pull request as ready for review — leave it in draft and let the author decide when it is ready. These are explicit repository conventions rather than defaults, so tooling that adds trailers or promotes drafts automatically must be configured off for this repository.

Evidence:

- `AGENTS.md` lines 419-424 — authoritative

Proof gap: Nothing rejects a generated-by trailer or a pull request marked ready for review. Both are review-time observations rather than checks.

Decision:

# Developer retention review — 2026-07-29 Claude Code `rule/v2` pass

Review instrument for the 37 `ssb.dev/rule/v2` proposals and 3 proposed skills
generated against `main`. Every decision below is **the developer's to make**;
this file supplies the metadata, body text, evidence, and proof coverage needed to
make them. It records no decision on the developer's behalf — all 37 decision
cells are deliberately empty.

Record one of `keep`, `edit-and-keep`, `defer`, or `reject` per row.

## Run provenance

| Property | Value |
|---|---|
| Consumer | Claude Code 2.1.220 |
| Model | Reported by the session environment as Opus 5 (1M context), `claude-opus-5[1m]`. **Self-reported, not independently observable** — `claude --version` prints only the CLI version. |
| Observable configuration | `learning` output style active. No reasoning-effort setting is exposed by the CLI, so none is claimed. |
| SSB source commit | `04bc830` (`origin/main`) |
| Evaluator digest | `88adf70d853e1ef34ab7392d02649dfc897ceb556cf6633ae4ef500c7543145d` |
| Evaluator build | `go build -trimpath -o ssb ./cmd/ssb`, Go 1.26.5 `darwin/arm64` |
| Inventory contract | `ssb-inventory-v2`, schema 2 |
| Rule schema emitted | `ssb.dev/rule/v2` |
| Host | macOS 15.7.3 build 24G419 (`arm64`) |
| Evaluation branch | `ssb-claude-v3-evaluation` in each clone |
| Skill exposure | `.claude/skills/software-standards-bootstrap` → the `main` worktree's portable skill |

The evaluator digest is **reproducible**, unlike the `c9d93a2a…` digest recorded
for the 2026-07-26 pass. That build used a plain `go build`, which embeds absolute
source paths: building the same commit at two different paths yields two different
digests, so no third party could ever verify it, and the original binary has since
been overwritten. `-trimpath` removes the path dependence — the digest above was
confirmed identical across two build directories.

## Independence limitation

**This pass is not blind for any of the four repositories.** Before it ran, the
operating session had read a v1 rule body in full and had generated a ledger
containing all 66 rule ids, bodies, and evidence from the 2026-07-26 pass, covering
every fixture. The 2026-07-26 record carried this qualification for Cobra alone; here
it applies to all four.

Every rule below was derived from reads of the pinned files, and every digest was
computed from the pinned Git blob rather than copied. But the rule *set* cannot be
treated as an independent second opinion, and overlap with the v1 sets is not
corroboration.

## Inventory coverage

| Repository | Pin | Candidate | Indexed | Remaining | Truncated |
|---|---|---:|---:|---:|---|
| Cobra | `adbc881390` | 66 files / 705,271 B | 65 files / 631,792 B | 0 / 0 | no |
| Flask | `36e4a824f3` | 235 files / 1,814,782 B | 230 files / 1,474,850 B | 0 / 0 | no |
| Django | `50c2b7c836` | 7,001 files / 45,506,636 B | 5,619 files / 36,820,618 B | 0 / 0 | no |
| Next.js | `6c6d1632e1` | 29,073 files / 111,110,455 B | 28,403 files / 88,643,646 B | 0 / 0 | no |

Coverage is complete for all four. `ssb inspect` exited `0` in every case,
`--allow-partial` was never passed, and no rule cites an excluded path.

## Threshold arithmetic

Bands follow [`docs/rule-format.md`](../../../rule-format.md): `very-high` 80–100,
`high` 65–79, `medium` 45–64. The gate is *at least 70% of high and very-high
candidates kept or edit-and-kept*, so only those rows count toward the denominator.

| Repository | Rules | very-high | high | medium | High-band denominator | Keeps needed |
|---|---:|---:|---:|---:|---:|---:|
| Cobra | 8 | 4 | 3 | 1 | 7 | 5 |
| Flask | 8 | 4 | 4 | 0 | 8 | 6 |
| Django | 9 | 2 | 7 | 0 | 9 | 7 |
| Next.js | 12 | 4 | 8 | 0 | 12 | 9 |
| **Total** | **37** | **14** | **22** | **1** | **36** | **26** |

One rule (`keep-doc-generator-autogen-tag-aligned`, 62) falls in the `medium` band
and is outside the denominator. It still needs a recorded decision.

## Schema conformance

Checked mechanically across all 37 rules:

| Property | Result |
|---|---|
| Declare `ssb.dev/rule/v2` | 37/37 |
| Declare a `directive` | 37/37 |
| Declare at least one lens | 37/37 |
| `importance` matches score band | 37/37 |
| Score factors sum to stated total | 37/37 |
| `deterministic` rules with `coverage: full` | 7/7 |
| Validation diagnostics | 0 across all four packs |
| Evidence items | 81 total, 53 marked authoritative |

Directive distribution: `always` 28, `never` 7, `ask-first` 2.

Classification: `guidance` 30, `deterministic` 7. Of the 30 guidance rules, 2 cite a command with `coverage: partial` and 28 record a `proof_gap`.

Four rules carry no authoritative source and rest on the alternative threshold of
three consistent occurrences across at least two files: `keep-doc-generator-autogen-tag-aligned`, `preserve-explicit-compatibility-shims`, `keep-sansio-boundary-pure`, `keep-sync-async-apis-aligned`.

## Lens coverage

Lenses are the v2 addition that makes progressive selection possible. Usage here:

| Lens | Rules |
|---|---:|
| `base` | 12 |
| `language:python` | 6 |
| `language:go` | 5 |
| `task:testing` | 5 |
| `task:implementation` | 4 |
| `task:security` | 4 |
| `language:typescript` | 3 |
| `task:documentation` | 1 |
| `task:release` | 1 |
| `task:review` | 1 |

The 12 `base` rules are standing orders: they render inline in `AGENTS.md` and load
whenever their path scope matches. The remaining 25 are contextual and load only
when every represented lens dimension also matches.

## Proposed skills

| Repository | Skill | Topic | Referenced by | Decision |
|---|---|---|---|---|
| Cobra | `add-shell-completion-generator` | compatibility | `keep-shell-completion-family-aligned` | |
| Flask | `add-supported-python-version` | compatibility | `preserve-python-support-boundary` | |
| Django | `add-async-api-variant` | compatibility | `keep-sync-async-apis-aligned` | |

**No skill is proposed for Next.js.** It already ships 17 skills under
`.agents/skills/`, and every procedural workflow this pass identified is already
covered by one. No Next.js rule declares `related_skills` either: the target-owned
skills do not carry the `metadata.topic` field this project requires of referenced
skills, so referencing one would fail validation. Those rules point to the existing
skill files in prose instead.

## Decision tables

### Cobra — 8 rules

| Rule | Topic | Directive | Lenses | Class | Score | Band | Proof | Decision |
|---|---|---|---|---|---:|---|---|---|
| `preserve-go-support-boundary` | compatibility | `always` | `language:go` | guidance | 84 | very-high | gap | |
| `verify-go-changes` | correctness | `always` | `base` | deterministic | 83 | very-high | `full` | |
| `preserve-license-headers` | compliance | `always` | `base` | deterministic | 82 | very-high | `full` | |
| `keep-shell-completion-family-aligned` | compatibility | `always` | `language:go` | guidance | 81 | very-high | gap | |
| `coordinate-security-fixes-privately` | security | `never` | `task:security` | guidance | 76 | high | gap | |
| `preserve-dual-build-tag-syntax` | compatibility | `always` | `language:go` | guidance | 68 | high | gap | |
| `preserve-explicit-compatibility-shims` | compatibility | `ask-first` | `language:go` | guidance | 68 | high | gap | |
| `keep-doc-generator-autogen-tag-aligned` | maintainability | `always` | `language:go` | guidance | 62 | medium | gap | |

### Flask — 8 rules

| Rule | Topic | Directive | Lenses | Class | Score | Band | Proof | Decision |
|---|---|---|---|---|---:|---|---|---|
| `verify-flask-test-matrix` | correctness | `always` | `base` | deterministic | 85 | very-high | `full` | |
| `keep-dual-type-checkers-green` | correctness | `always` | `base` | deterministic | 82 | very-high | `full` | |
| `preserve-python-support-boundary` | compatibility | `always` | `language:python` | guidance | 80 | very-high | gap | |
| `run-repository-precommit-checks` | maintainability | `always` | `base` | deterministic | 80 | very-high | `full` | |
| `pin-actions-to-commit-sha` | security | `always` | `task:release`, `task:security` | guidance | 78 | high | gap | |
| `route-user-callables-through-ensure-sync` | compatibility | `always` | `language:python` | guidance | 74 | high | gap | |
| `document-public-behavior-changes` | documentation | `always` | `task:documentation`, `task:implementation` | guidance | 70 | high | gap | |
| `keep-sansio-boundary-pure` | architecture | `never` | `language:python` | guidance | 70 | high | gap | |

### Django — 9 rules

| Rule | Topic | Directive | Lenses | Class | Score | Band | Proof | Decision |
|---|---|---|---|---|---:|---|---|---|
| `keep-database-backends-aligned` | compatibility | `always` | `language:python` | guidance | 82 | very-high | gap | |
| `keep-sync-async-apis-aligned` | compatibility | `always` | `language:python` | guidance | 80 | very-high | gap | |
| `preserve-python-support-boundary` | compatibility | `always` | `language:python` | guidance | 78 | high | gap | |
| `run-repository-linters` | maintainability | `always` | `base` | guidance | 78 | high | `partial` | |
| `check-test-migrations` | correctness | `always` | `task:testing`, `task:implementation` | deterministic | 74 | high | `full` | |
| `report-security-issues-privately` | security | `never` | `task:security` | guidance | 74 | high | gap | |
| `do-not-request-automated-ai-review` | compliance | `never` | `base` | guidance | 72 | high | gap | |
| `disclose-ai-assistance` | compliance | `always` | `base` | guidance | 70 | high | gap | |
| `end-commit-messages-with-a-period` | maintainability | `always` | `base` | guidance | 66 | high | gap | |

### Next.js — 12 rules

| Rule | Topic | Directive | Lenses | Class | Score | Band | Proof | Decision |
|---|---|---|---|---|---:|---|---|---|
| `verify-across-nextjs-modes` | correctness | `always` | `task:implementation`, `task:testing` | guidance | 84 | very-high | gap | |
| `rebuild-before-integration-tests` | correctness | `always` | `task:testing` | guidance | 82 | very-high | gap | |
| `run-nextjs-lint-gate` | quality | `always` | `base` | deterministic | 82 | very-high | `full` | |
| `wire-feature-flags-end-to-end` | architecture | `always` | `language:typescript` | guidance | 80 | very-high | gap | |
| `never-expose-secret-values` | security | `never` | `base` | guidance | 78 | high | gap | |
| `keep-require-dce-safe` | compatibility | `always` | `language:typescript` | guidance | 76 | high | `partial` | |
| `filter-internal-request-headers` | security | `ask-first` | `task:security`, `task:review` | guidance | 74 | high | gap | |
| `follow-agent-pr-conduct-limits` | compliance | `never` | `base` | guidance | 74 | high | gap | |
| `keep-react-server-imports-in-entry-base` | architecture | `always` | `language:typescript` | guidance | 72 | high | gap | |
| `use-retry-not-check-in-tests` | testability | `never` | `task:testing` | guidance | 72 | high | gap | |
| `generate-tests-with-new-test` | testability | `always` | `task:testing` | guidance | 70 | high | gap | |
| `keep-ast-grep-rule-pairs-in-sync` | maintainability | `always` | `task:implementation` | guidance | 66 | high | gap | |

## Rule bodies

The text a `keep` decision adopts as a standard, with its evidence and proof
coverage. Grouped by repository, ordered by score.

### Cobra

Baseline `adbc8813901bba65827259daa8e22ff94ec1f30e`

#### `preserve-go-support-boundary` — Preserve the declared Go support boundary

compatibility · directive `always` · guidance · 84/100 (very-high) · confidence high

Lenses: `language:go` · Scope: `**/*.go`, `go.mod`, `.golangci.yml`, `.github/workflows/test.yml`

Treat the Go version in `go.mod` as the language floor. Avoid newer-only syntax
and standard-library APIs while that floor stands, and preserve the legacy build
constraints that exist to satisfy it. The `.golangci.yml` `govet` settings record
this obligation explicitly: the `buildtag` check is disabled for Go 1.15
compatibility, with a note that it can be re-enabled once Cobra requires Go 1.17
or higher.

An intentional support change is not a single-file edit. Move `go.mod`, the
build-tag lint policy, the CI matrix, and any affected compatibility
documentation together in the same change.

*Evidence:*

- `go.mod` lines 1-3 — **authoritative**
- `.golangci.yml` lines 75-81 — **authoritative**
- `.github/workflows/test.yml` lines 56-71 — **authoritative**

*Proof gap:* The declared floor is asserted but never built or tested: `go.mod` declares `go 1.15` while the lowest version in the CI test matrix is 1.17. No existing command proves that source, build tags, lint configuration, and the supported matrix remain mutually consistent.

**Decision:**
#### `verify-go-changes` — Verify Go changes with the repository gate

correctness · directive `always` · deterministic · 83/100 (very-high) · confidence high

Lenses: `base` · Scope: `**/*.go`

Add focused tests for behavior changes and keep Go source formatted. Before
handoff, run the repository-owned `make all` gate, which combines the formatting
check and the test run. `CONTRIBUTING.md` states both obligations and names the
gate. This standard maps the existing gate; `ssb` did not execute it, and its
presence here is not a passing result.

*Evidence:*

- `CONTRIBUTING.md` lines 33-36 — **authoritative**

*Mapped command:* `make all` — coverage `full`, not executed by `ssb`.

*Proves:* Go sources are gofmt-clean and the repository's Go test suite passes.

**Decision:**

#### `preserve-license-headers` — Preserve required Apache license headers

compliance · directive `always` · deterministic · 82/100 (very-high) · confidence high

Lenses: `base` · Scope: `**/*`

Keep the configured Apache 2.0 header on new and modified files, using the owner
and year range the CI job defines. Preserve the job's `.github/**` exclusion
rather than adding headers there or widening the ignore list. The header text and
its parameters are owned by the license check, not by individual files. This
standard maps the existing check; `ssb` did not execute it.

*Evidence:*

- `.github/workflows/test.yml` lines 23-32 — **authoritative**

*Mapped command:* `docker run -v $(pwd):/wrk -w /wrk ghcr.io/google/addlicense -c 'The Cobra Authors' -y '2013-2023' -l apache -ignore '.github/**' -check .` — coverage `full`, not executed by `ssb`.

*Proves:* Every checked file outside `.github/**` carries the Apache 2.0 header with owner `The Cobra Authors` and year range `2013-2023`.

**Decision:**

#### `keep-shell-completion-family-aligned` — Keep the shell completion family aligned

compatibility · directive `always` · guidance · 81/100 (very-high) · confidence high

Lenses: `language:go` · Scope: `bash_completions.go`, `bash_completionsV2.go`, `zsh_completions.go`, `fish_completions.go`, `powershell_completions.go`, `completions.go`, `shell_completions.go`, `site/content/completions/**`

Bash, Zsh, fish, and PowerShell are the documented supported shells, and each has
a generator, a focused test file, and a page under `site/content/completions`.
When changing shared completion behavior — directives, descriptions, active help,
or flag completion — move every affected generator together with its test and its
documentation page.

Not every change touches every shell. Every shell a change does affect must move
as a unit. Respect the documented per-shell limitations rather than assuming
behavior is uniform; the documentation records real differences between shells,
so a behavior that is correct for one may be inapplicable to another.

*Evidence:*

- `site/content/completions/_index.md` lines 1-8 — **authoritative**
- `zsh_completions.go` lines 25-27
- `fish_completions.go` lines 276-281
- `powershell_completions.go` lines 331-333

*Proof gap:* Each generator has its own focused test file, but no existing command asserts that a shared completion behavior reached every supported shell or that the documentation set stayed complete.

*Related skill:* `add-shell-completion-generator`

**Decision:**

#### `coordinate-security-fixes-privately` — Coordinate security fixes privately

security · directive `never` · guidance · 76/100 (high) · confidence high

Lenses: `task:security` · Scope: `**/*`

Never open a public issue or a public pull request for a suspected Cobra
vulnerability or for its fix. Report it to `cobra-security@googlegroups.com` with
a description, reproduction steps, potential impact, and any mitigations already
identified, then coordinate the patch, release, and disclosure privately with the
maintainers. The policy applies to a fix you already hold, not only to an unfixed
report: it asks contributors with a patch to work through the security address so
the push, release, and disclosure stay coordinated.

*Evidence:*

- `SECURITY.md` lines 9-22 — **authoritative**

*Proof gap:* Disclosure behavior is a human process. No repository check can detect a vulnerability discussed in a public issue or pull request before it is published.

**Decision:**

#### `preserve-dual-build-tag-syntax` — Keep both build-tag syntaxes on platform files

compatibility · directive `always` · guidance · 68/100 (high) · confidence high

Lenses: `language:go` · Scope: `command_win.go`, `command_notwin.go`

Platform-specific files carry both build-constraint forms: the modern
`//go:build` line and the legacy `// +build` line. Keep both, and keep them
expressing the same condition, when adding or editing a platform-gated file.

This is not redundancy to clean up. `//go:build` was introduced in Go 1.17, and
the module still declares a Go 1.15 floor, so the legacy form is what makes the
constraint effective on the oldest supported toolchains. `.golangci.yml` disables
the `govet` `buildtag` check specifically to permit the pair, and records that the
exemption can be dropped once Cobra requires Go 1.17 or higher.

*Evidence:*

- `command_win.go` lines 15-16
- `command_notwin.go` lines 15-16
- `.golangci.yml` lines 77-81 — **authoritative**

*Proof gap:* The `govet` `buildtag` check is deliberately disabled in `.golangci.yml`, so no existing command proves that both constraint forms remain present and agree. Removing one form fails only on a Go 1.15 or 1.16 build, which the CI matrix does not cover.

**Decision:**

#### `preserve-explicit-compatibility-shims` — Preserve explicitly marked compatibility shims

compatibility · directive `ask-first` · guidance · 68/100 (high) · confidence high

Lenses: `language:go` · Scope: `cobra.go`, `command.go`

Some symbols in this repository exist only for downstream compatibility and say so
in a comment. `cobra.go` marks `Gt`, `Eq`, and `appendIfNotPresent` as unused by
Cobra itself and retained "only for compatibility with users of cobra". The
`UsageTemplate`, `HelpTemplate`, and `VersionTemplate` methods in `command.go` are
each marked "kept for backwards-compatibility reasons".

Ask before removing or narrowing one of these. Cobra is a widely depended-upon
library, so an apparently dead exported symbol may be load-bearing for callers
this repository cannot see, and its own tooling will not object to the deletion.
The `cobra.go` comments schedule removal for a future major version; treat that as
the intended path rather than as license to remove them now.

This rule covers only symbols the repository itself marks as compatibility
retentions. A plain `Deprecated:` marker is an ordinary Go convention and is out
of scope.

*Evidence:*

- `cobra.go` lines 109-109
- `cobra.go` lines 163-163
- `command.go` lines 590-591

*Proof gap:* No existing command distinguishes a symbol retained for downstream compatibility from genuinely dead code. The `unused` linter is enabled but does not flag exported identifiers, so removal breaks only downstream builds, which this repository's CI cannot observe.

**Decision:**

#### `keep-doc-generator-autogen-tag-aligned` — Keep the auto-generated tag aligned across doc generators

maintainability · directive `always` · guidance · 62/100 (medium) · confidence medium

Lenses: `language:go` · Scope: `doc/**`, `site/content/docgen/**`

The Man, Markdown, and reStructuredText generators each emit an "Auto generated by
spf13/cobra" line gated on `DisableAutoGenTag`. When changing the tag text, its
date format, or the meaning of `DisableAutoGenTag`, update every generator that
emits the tag along with its focused test, and keep the documented behavior
accurate.

The YAML generator is deliberately outside this obligation. `doc/yaml_docs.go`
emits no auto-generated tag at all, so `DisableAutoGenTag` is vacuous there rather
than drifted. Do not "fix" it into emitting one on the strength of this rule.

*Evidence:*

- `doc/man_docs.go` lines 242-243
- `doc/md_docs.go` lines 112-113
- `doc/rest_docs.go` lines 125-126

*Proof gap:* Each generator has focused tests, but no existing command asserts that every generator emitting the tag honors `DisableAutoGenTag` consistently or that the documented behavior stays accurate.

**Decision:**

### Flask

Baseline `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`

#### `verify-flask-test-matrix` — Verify changes through the repository test runner

correctness · directive `always` · deterministic · 85/100 (very-high) · confidence high

Lenses: `base` · Scope: `**/*.py`

Run the repository's own test runner rather than invoking `pytest` directly. The
`tox` environments are what CI executes, and they pin the interpreter and resolve
dependencies from the lock file, so a bare `pytest` run in an ad-hoc virtualenv is
not the same check.

Note that `pyproject.toml` sets `filterwarnings = ["error"]`, so a warning raised
during a test is a failure. A change that introduces a deprecation warning from a
dependency will fail here even when the assertion logic is correct.

This standard maps the existing runner; `ssb` did not execute it, and its presence
here is not a passing result.

*Evidence:*

- `.github/workflows/tests.yaml` lines 16-31 — **authoritative**

*Mapped command:* `uv run --locked --no-default-groups --group dev tox run` — coverage `full`, not executed by `ssb`.

*Proves:* The pytest suite passes on the selected interpreter using the locked dependency versions.

**Decision:**

#### `keep-dual-type-checkers-green` — Keep both type checkers green

correctness · directive `always` · deterministic · 82/100 (very-high) · confidence high

Lenses: `base` · Scope: `src/**/*.py`, `tests/type_check/**`

This repository runs two type checkers, not one. `mypy` is configured with
`strict = true`, and `pyright` runs in `basic` mode over the same sources. The
`typing` environment runs both in sequence, so a change must satisfy both.

They disagree in practice. Satisfying `mypy` alone is not sufficient, and a
targeted `# type: ignore` that silences one may leave the other reporting an
error — or become an unused-ignore error under strict mypy. When adding a
suppression, check both checkers rather than the one that reported first.

Flask is a typed package (`Typing :: Typed`), so annotations here are part of the
public surface that downstream users type-check against.

*Evidence:*

- `pyproject.toml` lines 126-131 — **authoritative**
- `pyproject.toml` lines 142-145 — **authoritative**

*Mapped command:* `uv run --locked --no-default-groups --group dev tox run -e typing` — coverage `full`, not executed by `ssb`.

*Proves:* Both mypy in strict mode and pyright report no errors over `src` and `tests/type_check`.

**Decision:**

#### `preserve-python-support-boundary` — Preserve the declared Python support boundary

compatibility · directive `always` · guidance · 80/100 (very-high) · confidence high

Lenses: `language:python` · Scope: `pyproject.toml`, `.github/workflows/tests.yaml`, `src/**/*.py`

Python 3.10 is the declared floor, and it is declared in four separate places:
`requires-python = ">=3.10"`, the `mypy` `python_version`, the `pyright`
`pythonVersion`, and the lowest entry in both the `tox` environment list and the
CI matrix.

Avoid syntax and standard-library APIs newer than the floor in `src`, and remember
that the type checkers are pinned to the floor as well, so a newer-only construct
surfaces as a typing failure rather than a syntax error.

A support change is not a one-line edit. Move `requires-python`, both type-checker
target versions, the `tox` environment list, and the CI matrix together, and add
the corresponding `CHANGES.rst` entry. The matrix also carries a free-threaded
build (`3.14t`) and a PyPy environment; a change that assumes CPython
reference-counting semantics can pass on the default entries and fail on those.

*Evidence:*

- `pyproject.toml` lines 22-22 — **authoritative**
- `pyproject.toml` lines 170-179 — **authoritative**
- `.github/workflows/tests.yaml` lines 19-30 — **authoritative**

*Proof gap:* No existing command asserts that the four declarations of the floor agree. `requires-python`, the `mypy` and `pyright` target versions, the `tox` environment list, and the CI matrix are maintained independently, so a bump to one can pass every check while leaving the others stale.

*Related skill:* `add-supported-python-version`

**Decision:**

#### `run-repository-precommit-checks` — Run the repository pre-commit checks

maintainability · directive `always` · deterministic · 80/100 (very-high) · confidence high

Lenses: `base` · Scope: `**/*`

Formatting, import order, lint, spelling, and lock-file freshness are all enforced
through pre-commit rather than by hand. The configured hooks are `ruff-check` and
`ruff-format`, `uv-lock`, `codespell`, and the merge-conflict, debug-statement,
byte-order-marker, trailing-whitespace, and end-of-file fixers.

Two consequences worth knowing before you hand off:

- `uv-lock` means a dependency edit in `pyproject.toml` must be accompanied by a
  refreshed `uv.lock`. The lock file is checked, not advisory.
- `ruff` is configured with `force-single-line = true` for imports, so import
  blocks look unusual by wider Python convention and should not be "tidied" into
  grouped imports.

Each hook is pinned to a frozen upstream commit, so results are stable until
someone runs the update environment deliberately.

*Evidence:*

- `.pre-commit-config.yaml` lines 1-23 — **authoritative**

*Mapped command:* `uv run --locked --no-default-groups --group dev tox run -e style` — coverage `full`, not executed by `ssb`.

*Proves:* Every configured pre-commit hook passes across all files.

**Decision:**

#### `pin-actions-to-commit-sha` — Pin GitHub Actions to full commit SHAs

security · directive `always` · guidance · 78/100 (high) · confidence high

Lenses: `task:release`, `task:security` · Scope: `.github/workflows/**`

Reference every GitHub Action by a full 40-character commit SHA with the human
version in a trailing comment, as in
`uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2`. Never
reference a moving tag or branch.

The convention is followed without exception at this baseline: all 21 `uses:`
references across the five workflow files are SHA-pinned. Pins are maintained
deliberately rather than by hand — the `update-actions` environment runs
`gha-update` — so add a new action at its current SHA and let that environment move
it later.

Two related habits appear alongside the pins and are worth preserving in a new
workflow: jobs declare `permissions: {}` or a minimal read scope, and `checkout`
steps set `persist-credentials: false`. The repository also runs `zizmor`, a
GitHub Actions security analyzer, on every workflow change.

*Evidence:*

- `.github/workflows/tests.yaml` lines 32-35
- `.github/workflows/zizmor.yaml` lines 16-21
- `pyproject.toml` lines 257-262 — **authoritative**

*Proof gap:* The repository runs `zizmor` on workflow changes, which is the closest existing control, but the repository does not configure or assert an unpinned-action rule. From repository evidence alone the convention is maintained by the `gha-update` environment and by review, not by a check this repository owns.

**Decision:**

#### `route-user-callables-through-ensure-sync` — Invoke user-supplied callables through ensure_sync

compatibility · directive `always` · guidance · 74/100 (high) · confidence high

Lenses: `language:python` · Scope: `src/flask/app.py`

When adding code that calls a developer-registered callable — a view function, an
error handler, a `before_request` or `teardown` function, a signal receiver, or a
template context processor — invoke it through `self.ensure_sync(func)(...)` rather
than calling it directly.

`ensure_sync` is the single documented seam that makes async handlers work under a
WSGI worker: it returns plain functions unchanged and wraps coroutine functions via
`async_to_sync`. Its docstring marks it as an override point, so subclasses depend
on every user callable passing through it. The existing dispatch paths are
consistent about this — view dispatch, error handlers, teardown, and signal
delivery all route through it.

Calling a registered callable directly silently drops async support on that path
only. Prefer extending an existing dispatch path over adding a new invocation
site; when a new site is genuinely needed, add a test that registers an
`async def` handler for it.

*Evidence:*

- `src/flask/app.py` lines 1065-1077 — **authoritative**
- `src/flask/app.py` lines 990-990
- `src/flask/app.py` lines 946-946

*Proof gap:* No existing check asserts that a newly added invocation of a user-supplied callable routes through `ensure_sync`. A direct call works for every synchronous view, so the omission is invisible to the suite unless a test specifically exercises an async handler on that path.

**Decision:**

#### `document-public-behavior-changes` — Document public behavior changes in CHANGES and code docs

documentation · directive `always` · guidance · 70/100 (high) · confidence high

Lenses: `task:documentation`, `task:implementation` · Scope: `src/**/*.py`, `CHANGES.rst`, `docs/**`

A change to public behavior carries documentation obligations, and the pull request
template states them: add tests that fail without the change, add or update the
relevant docs both in the `docs` folder and in code, add a `CHANGES.rst` entry
summarizing the change and linking to the issue, and add `.. versionchanged::`
entries in the affected code documentation.

The `versionchanged` and `versionadded` directives are load-bearing rather than
decorative. Flask's published documentation is how downstream users determine when
a behavior became available, and the existing docstrings use these markers
consistently for that purpose.

Note that the template points contributors at `CONTRIBUTING.rst`, which does not
exist at the repository root; the tracked file is `docs/contributing.rst`, and it
is a short stub that refers to the Pallets-wide contributing guide. Treat the
template's own list as the operative in-repository statement of these steps.

*Evidence:*

- `.github/pull_request_template.md` lines 17-25 — **authoritative**

*Proof gap:* No existing check asserts that a behavior change carries a `CHANGES.rst` entry or a `.. versionchanged::` directive. The documentation build runs with `-W`, so it fails on malformed markup, but it cannot detect a missing entry.

**Decision:**

#### `keep-sansio-boundary-pure` — Keep the sans-IO layer free of runtime IO imports

architecture · directive `never` · guidance · 70/100 (high) · confidence high

Lenses: `language:python` · Scope: `src/flask/sansio/**`

Never add a runtime import of the request/response or WSGI layer to a module under
`src/flask/sansio/`. That package holds the framework logic that does not depend on
a concrete IO model, which is what lets an ASGI framework reuse it.

The existing code shows the intended shape. `sansio/app.py` needs
`werkzeug.wrappers.Response` only as an annotation and imports it inside
`if t.TYPE_CHECKING:`; `sansio/blueprints.py` defers its own imports the same way.
The concrete layer is where IO enters: `src/flask/app.py` imports
`from .sansio.app import App` and adds `Request` and `Response`, with
`class Flask(App)` composing the two.

The dependency direction is one-way — `app.py` depends on `sansio`, never the
reverse. When a sansio module needs a type from the IO layer, put the import under
`TYPE_CHECKING` and quote the annotation. When it needs behavior from the IO layer,
that is a signal the logic belongs in `app.py` instead.

*Evidence:*

- `src/flask/sansio/app.py` lines 36-37
- `src/flask/sansio/blueprints.py` lines 14-14
- `src/flask/app.py` lines 44-44
- `src/flask/app.py` lines 53-54

*Proof gap:* The type checkers read the deferred annotations but neither asserts the boundary. No configured check fails a new runtime import of the WSGI layer into `src/flask/sansio/`, so the regression is invisible until a downstream sans-IO consumer breaks.

**Decision:**

### Django

Baseline `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`

#### `keep-database-backends-aligned` — Declare backend capabilities through feature flags

compatibility · directive `always` · guidance · 82/100 (very-high) · confidence high

Lenses: `language:python` · Scope: `django/db/backends/**`

Backend capability differences are expressed as feature flags, never as
backend-name checks in shared code. `BaseDatabaseFeatures` declares each capability
with a conservative default and a comment explaining what it means; each backend's
`DatabaseFeatures` subclass overrides only the flags that differ for it.

When adding behavior that not every database supports:

- add a flag to `BaseDatabaseFeatures` with the safe default and a comment
  describing the capability;
- override it in each backend where the real answer differs; and
- gate shared code and tests on the flag, using `skipUnlessDBFeature` and
  `skipIfDBFeature` rather than inspecting the vendor name.

Do not branch on `connection.vendor` in shared code to work around a capability
gap. The flag set is how five backends stay comparable, and a vendor check hides
the difference from every other backend and from third-party backends that are not
in this repository at all. `minimum_database_version` on each backend follows the
same principle for version-dependent behavior.

*Evidence:*

- `django/db/backends/base/features.py` lines 25-36 — **authoritative**
- `django/db/backends/postgresql/features.py` lines 9-20
- `django/db/backends/mysql/features.py` lines 9-16
- `django/db/backends/oracle/features.py` lines 15-16

*Proof gap:* Proving this requires running the suite against every supported backend, including Oracle and MySQL, which the default local test run does not do. A capability difference left undeclared surfaces only in a backend-specific CI job or in a downstream user's database.

**Decision:**

#### `keep-sync-async-apis-aligned` — Pair every public sync API with its async counterpart

compatibility · directive `always` · guidance · 80/100 (very-high) · confidence high

Lenses: `language:python` · Scope: `django/db/models/**`, `django/core/cache/**`, `django/contrib/auth/**`

Public APIs in these packages come in pairs: a synchronous method and an
`a`-prefixed asynchronous counterpart that delegates to it through
`sync_to_async`. `QuerySet.get` has `aget`, `create` has `acreate`, the cache
backends pair `get`/`aget` and `set`/`aset`, and `django.contrib.auth` pairs
`authenticate`/`aauthenticate` and `login`/`alogin`.

When adding a public synchronous method to one of these surfaces, add its async
counterpart in the same change. Follow the established shape rather than
reimplementing the logic: the async method awaits `sync_to_async` over the sync
one, so behavior stays defined in exactly one place. The cache backends pass
`thread_sensitive=True`; match the convention of the surrounding module rather
than choosing per method.

Name the counterpart by prefixing `a`, even when the result reads awkwardly. The
convention is mechanical on purpose so callers can predict the name.

*Evidence:*

- `django/db/models/query.py` lines 717-718
- `django/db/models/query.py` lines 742-743
- `django/core/cache/backends/base.py` lines 151-152
- `django/contrib/auth/__init__.py` lines 148-148

*Proof gap:* No existing check asserts that a newly added public sync method has an async counterpart. The omission is not a failure — it is a silently missing API that downstream async code cannot call, and it surfaces only as a user-reported gap.

*Related skill:* `add-async-api-variant`

**Decision:**

#### `preserve-python-support-boundary` — Preserve the declared Python support boundary

compatibility · directive `always` · guidance · 78/100 (high) · confidence high

Lenses: `language:python` · Scope: `django/**/*.py`, `pyproject.toml`, `tox.ini`

Python 3.12 is the declared floor (`requires-python = ">= 3.12"`). Avoid syntax and
standard-library APIs newer than the floor in `django/`.

Be aware of what the local tooling does and does not cover. The `py3` tox
environment uses `basepython = python3`, so a local run tests whichever interpreter
happens to be current — passing locally on a newer Python says nothing about the
floor. A separate `py314t` environment targets the free-threaded build, where code
assuming the GIL can fail even though every default environment passes.

A support change moves `requires-python`, the tox environment list, and the CI
matrix definition together, and needs a release-note entry.

*Evidence:*

- `pyproject.toml` lines 8-8 — **authoritative**
- `tox.ini` lines 9-17 — **authoritative**
- `tox.ini` lines 23-25 — **authoritative**

*Proof gap:* The default `tox` environment runs whatever `python3` resolves to on the developer's machine rather than pinning the floor, and the CI Python matrix is computed dynamically at workflow run time. Nothing in the repository asserts that `requires-python` and the tested set agree.

**Decision:**

#### `run-repository-linters` — Satisfy the repository linter and formatter set

maintainability · directive `always` · guidance · 78/100 (high) · confidence high

Lenses: `base` · Scope: `**/*`

Six separate tools gate this repository, each with its own CI job and pre-commit
hook: `black` for Python formatting, `blacken-docs` for code blocks inside
`docs/*.txt`, `isort` for import order, `flake8` for lint, `biome` for JavaScript
and JSON, and `zizmor` for workflow security.

Two of these are easy to miss because they cover non-Python files. `blacken-docs`
formats Python embedded in reStructuredText, so a docs example is held to the same
formatter as source. `biome` covers the JavaScript and JSON in the repository,
including `django/contrib/admin` static assets.

Note the two different line limits: 88 characters for code, 79 for documentation
and comments. `E203` is intentionally ignored because it conflicts with `black`;
do not "fix" that by reformatting against `black`.

Every mapped command here is existing repository evidence only. `ssb` executed
none of them, and `flake8` alone proves only the lint portion of this rule.

*Evidence:*

- `.pre-commit-config.yaml` lines 1-31 — **authoritative**
- `.github/workflows/linters.yml` lines 21-40 — **authoritative**

*Mapped command:* `flake8` — coverage `partial`, not executed by `ssb`.

*Proves:* Python sources satisfy the configured flake8 rules, including the 88-character code line limit and the 79-character documentation line limit.

**Decision:**

#### `check-test-migrations` — Keep test app migrations complete

correctness · directive `always` · deterministic · 74/100 (high) · confidence high

Lenses: `task:testing`, `task:implementation` · Scope: `tests/**/models.py`, `tests/**/migrations/**`

When changing a model under `tests/`, generate the matching migration in the same
change. A test app whose models have drifted from its migrations breaks the suite
for unrelated work, which is why this has its own dedicated CI job rather than
riding along with the main test run.

The check runs against PostgreSQL specifically, using a generated settings module,
so a migration that happens to be a no-op on SQLite is still required.

This standard maps the existing check; `ssb` did not execute it, and its presence
here is not a passing result.

*Evidence:*

- `.github/workflows/check-migrations.yml` lines 5-9 — **authoritative**

*Mapped command:* `python scripts/check_migrations.py` — coverage `full`, not executed by `ssb`.

*Proves:* No test-app model change is left without its corresponding migration.

**Decision:**

#### `report-security-issues-privately` — Never disclose a security vulnerability in a pull request

security · directive `never` · guidance · 74/100 (high) · confidence high

Lenses: `task:security` · Scope: `**/*`

Never open a pull request that discloses a security vulnerability. The pull request
checklist requires contributors to affirm that the change does not, and the
repository's security policy directs reports through the Django project's security
process rather than through the public repository.

This applies to the fix as much as to the report. A patch that quietly closes an
exploitable hole still discloses it — the diff, and often the accompanying test,
tells a reader exactly what was wrong and how to trigger it. Route it through the
security process instead.

When you are uncertain whether something you found qualifies, treat it as a
security issue and use the private channel. The cost of asking privately is low;
the cost of a public disclosure cannot be undone.

*Evidence:*

- `.github/pull_request_template.md` lines 18-18 — **authoritative**
- `.github/SECURITY.md` lines 1-3 — **authoritative**

*Proof gap:* Whether a change discloses a vulnerability is a human judgment about the change's meaning, not a property of the diff. No repository check can make it.

**Decision:**

#### `do-not-request-automated-ai-review` — Never request an automated AI review on this repository

compliance · directive `never` · guidance · 72/100 (high) · confidence high

Lenses: `base` · Scope: `**/*`

Never request an automated AI review for a pull request against this repository.
The pull request checklist requires contributors to affirm that they have not and
will not, and `.github/copilot-instructions.md` instructs the reviewing tool to
decline: its only permitted output is a message telling the requester to do it in
their own fork.

This is a deliberate maintainer position, stated twice and in two mechanisms. Doing
it in a personal fork is explicitly acceptable; doing it on `django/django` is not.

The prohibition covers the review request itself. It does not restrict the
disclosed use of AI assistance while preparing a change, which
`disclose-ai-assistance` governs separately.

*Evidence:*

- `.github/copilot-instructions.md` lines 8-10 — **authoritative**
- `.github/pull_request_template.md` lines 21-21 — **authoritative**

*Proof gap:* Requesting a review is an action taken in the pull request interface, not in the repository. No repository check can observe or prevent it.

**Decision:**

#### `disclose-ai-assistance` — Disclose AI assistance on every pull request

compliance · directive `always` · guidance · 70/100 (high) · confidence high

Lenses: `base` · Scope: `**/*`

Every pull request must carry the AI Assistance Disclosure, and the template marks
it `(REQUIRED)`. Select exactly one of the two options: either no AI tools were
used, or AI tools were used — in which case name which ones and confirm you fully
reviewed and verified their output.

Two things this rule asks of an agent operating in this repository. First, if you
produced or modified any part of a change, that fact belongs in the disclosure;
saying nothing is not the same as selecting "No AI tools were used". Second, the
second option asserts human review and verification, so it should not be checked
on the strength of the agent's own confidence.

The obligation is about the pull request body rather than the code, so it applies
to every change regardless of which files it touches.

*Evidence:*

- `.github/pull_request_template.md` lines 10-14 — **authoritative**

*Proof gap:* The disclosure is a checkbox in the pull request body. No repository check verifies that a box was selected, that the named tools are accurate, or that the claimed review actually happened.

**Decision:**

#### `end-commit-messages-with-a-period` — Write commit messages in the repository house style

maintainability · directive `always` · guidance · 66/100 (high) · confidence high

Lenses: `base` · Scope: `**/*`

Write each commit subject in past tense, mention the ticket number when there is
one, and end it with a period. The pull request checklist states all three.

Only the trailing period is mechanically enforced, and only partly. A CI job walks
every commit in the pull request and fails any subject that does not end with `.`.
For a pull request targeting a `stable/` branch, a second job additionally requires
every commit subject and the pull request title to start with the release prefix,
such as `[5.2]`.

Both jobs are gated on the repository being `django/django`, so neither runs in a
fork. Getting the format right locally is therefore cheaper than discovering it
upstream, and rewriting history after the fact is the more expensive path.

*Evidence:*

- `.github/pull_request_template.md` lines 20-20 — **authoritative**
- `.github/workflows/check_commit_messages.yml` lines 93-107 — **authoritative**
- `.github/workflows/check_commit_messages.yml` lines 73-75 — **authoritative**

*Proof gap:* The trailing-period job is gated on `if: github.repository == 'django/django'`, so it does not run on the forks where contributors actually develop. Past tense and the ticket reference are not checked at all. A contributor sees this failure only after opening the pull request upstream.

**Decision:**

### Next.js

Baseline `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`

#### `verify-across-nextjs-modes` — Verify behavior across dev, start, Turbopack, and webpack

correctness · directive `always` · guidance · 84/100 (very-high) · confidence high

Lenses: `task:implementation`, `task:testing` · Scope: `packages/next/**`, `test/**`

Behavior is not uniform across the four supported combinations, and the test
scripts are split accordingly: `test-dev-turbo`, `test-dev-webpack`,
`test-start-turbo`, and `test-start-webpack`. Each pins one mode and one bundler.

Pick the script matching what your change affects, and run more than one when the
change touches shared code. Module resolution, bundling, and snapshot output are
the areas that diverge most: Turbopack resolves `react-dom/server.edge` while
webpack resolves the `.node` build, so an inline snapshot can legitimately differ
between them.

Turbopack is the default for both `next dev` and `next build`; force webpack with
the `--webpack` flag. There is no `--no-turbopack` flag.

When reproducing a CI failure, mirror the job's environment variables — notably
`IS_WEBPACK_TEST=1`, which changes bundler selection and therefore snapshot
output. Capture output to a file once and analyze it from there rather than
re-running the suite with different filters.

*Evidence:*

- `AGENTS.md` lines 138-143 — **authoritative**
- `AGENTS.md` lines 269-277 — **authoritative**
- `package.json` lines 29-32 — **authoritative**

*Proof gap:* No single command covers all four mode and bundler combinations. Each of the four scripts exercises exactly one, so a change verified in one combination can regress in another without any check failing locally.

**Decision:**

#### `rebuild-before-integration-tests` — Rebuild before running integration tests

correctness · directive `always` · guidance · 82/100 (very-high) · confidence high

Lenses: `task:testing` · Scope: `packages/next/**`, `test/**`

Integration tests run against built output, not source. If source changed and you
did not rebuild, the result describes the previous build — which is worse than a
failure, because it looks like a real answer.

Which rebuild depends on what moved:

- first run after a branch switch or bootstrap, or when unsure — `pnpm build-all`;
- edits confined to `packages/next/**` after bootstrap — `pnpm --filter=next build`;
- edits touching Turbopack or other Rust — `pnpm build-all`.

For iterative work, start `pnpm --filter=next dev` in a separate session before
editing. It rebuilds in one to two seconds per change instead of about sixty, and
it removes the class of mistakes this rule exists to prevent. If you are changing
source or integration tests and choose not to run it, say why — docs-only,
read-only investigation, or CI-only analysis are the documented reasons.

If Turbopack behaves strangely after a branch switch or pull, suspect a stale
native binary at `packages/next-swc/native/*.node` rather than your change.

*Evidence:*

- `AGENTS.md` lines 441-447 — **authoritative**
- `AGENTS.md` lines 105-112 — **authoritative**

*Proof gap:* Nothing detects that `packages/next/dist/` is stale relative to `src/`. The tests run happily against the previous build, so the failure mode is not an error but a passing or failing result that describes code you are no longer editing.

**Decision:**

#### `run-nextjs-lint-gate` — Run the repository lint gate before handoff

quality · directive `always` · deterministic · 82/100 (very-high) · confidence high

Lenses: `base` · Scope: `**/*`

`pnpm lint` is a single command that fans out to seven parallel checks: type
checking, TypeScript lint, a prettier format check, eslint, the ast-grep pattern
rules, a language check, and unused-turbo-task detection. Run it rather than
picking individual checks, because a change can pass eslint and still fail
ast-grep or the language check.

Two faster paths exist for tight loops and are worth knowing: `pnpm types` for
type errors alone, and `pnpm prettier-fix` for formatting alone. Neither is a
substitute for the gate before handoff.

To avoid the slow pre-commit path, the repository documents running exactly what
the hook runs on just your changed files, which is substantially cheaper than a
failed `lint-staged` cycle.

This standard maps the existing gate; `ssb` did not execute it, and its presence
here is not a passing result.

*Evidence:*

- `AGENTS.md` lines 227-234 — **authoritative**

*Mapped command:* `pnpm lint` — coverage `full`, not executed by `ssb`.

*Proves:* Type checks, TypeScript lint, prettier formatting, eslint, ast-grep pattern rules, language checks, and unused-turbo-task detection all pass.

**Decision:**

#### `wire-feature-flags-end-to-end` — Wire a new feature flag through every layer

architecture · directive `always` · guidance · 80/100 (very-high) · confidence high

Lenses: `language:typescript` · Scope: `packages/next/src/**`

A feature flag is not a single declaration. Adding one means:

- adding its type in `config-shared.ts`;
- adding its schema entry in `config-schema.ts`; and
- adding it to `define-env.ts` when user-bundled code reads it.

If the flag is consumed inside pre-compiled runtime internals, that is not enough:
wire the runtime env values as well, in `next-server.ts` or `export/worker.ts` as
needed. `define-env.ts` governs user bundling only and does not control pre-compiled
runtime bundle internals — this is the distinction that makes a partially wired flag
look like a working one that is simply disabled.

For edge builds, force flags gating Node-only imports to `false` in
`define-env.ts`, so the Node-only branch is eliminated rather than bundled.

The `$flags` skill in `.agents/skills/flags/SKILL.md` expands this with verification
steps and examples.

*Evidence:*

- `AGENTS.md` lines 464-470 — **authoritative**

*Proof gap:* No check asserts that a flag reached every layer it needs. A flag wired into `define-env.ts` but not into the runtime env is simply undefined inside pre-compiled internals, which reads as the feature being off rather than as a wiring bug.

**Decision:**

#### `never-expose-secret-values` — Never reveal environment secret values

security · directive `never` · guidance · 78/100 (high) · confidence high

Lenses: `base` · Scope: `**/*`

Treat every environment variable value as sensitive unless it is a known test-mode
flag. Specifically:

- never print or paste token, API key, or cookie values into responses, commits, or
  shared logs;
- mirror CI environment variable **names and modes** exactly, but do not inline
  literal secret values into commands;
- when a required secret is missing locally, stop and ask rather than inventing a
  placeholder credential;
- never commit a local secret file, and use placeholder-only examples when
  documenting environment setup; and
- summarize and redact sensitive-looking values when sharing command output.

The distinction that matters in practice is between a variable's **name** and its
**value**. Reproducing a CI failure requires matching names and modes — for example
`IS_WEBPACK_TEST=1` — and that is expected and safe. Echoing the value of a token to
show what was set is not.

Inventing a plausible placeholder credential is the failure mode worth calling out:
it produces a run that looks legitimate but tests nothing, and it can mask the fact
that the environment was never configured.

*Evidence:*

- `AGENTS.md` lines 350-358 — **authoritative**

*Proof gap:* Whether an output leaked a secret is a judgment about the value's meaning, not a pattern in the diff. No repository check observes chat responses or shared logs at all.

**Decision:**

#### `keep-require-dce-safe` — Keep require calls eliminable by dead-code elimination

compatibility · directive `always` · guidance · 76/100 (high) · confidence high

Lenses: `language:typescript` · Scope: `packages/next/src/**`

Keep `require()` calls inside compile-time `if`/`else` branches so the bundler can
eliminate the unused side. Avoid early-return and throw patterns around a
`require()`: they leave the call reachable and the module gets bundled.

Never gate a `require()` on `typeof window`. This is enforced with `severity: error`
by the `no-typeof-window-require` ast-grep rule, because gating on `typeof window`
bundles the server branch into the browser bundle. The documented alternative is to
split the module into `<name>.ts` (the default, used on the server and unbundled
runtimes) and `<name>.browser.ts`; the browser bundle is aliased to the `.browser`
sibling automatically.

The mapped command proves only the `typeof window` case. The broader obligation —
that DCE can actually eliminate the branch — is not checked, and in edge builds it
depends on flags gating Node-only imports being forced to `false` in
`define-env.ts`.

`ssb` did not execute the mapped command, and a passing ast-grep scan does not
establish the rest of this rule.

*Evidence:*

- `AGENTS.md` lines 464-470 — **authoritative**
- `.config/ast-grep/rules/no-typeof-window-require.yml` lines 13-33 — **authoritative**

*Mapped command:* `pnpm lint-ast-grep` — coverage `partial`, not executed by `ssb`.

*Proves:* No `require()` under `packages/next/src/**` is gated on a `typeof window` condition, in either TypeScript or TSX files.

**Decision:**

#### `filter-internal-request-headers` — Treat non-standard request headers as attacker-controlled

security · directive `ask-first` · guidance · 74/100 (high) · confidence high

Lenses: `task:security`, `task:review` · Scope: `packages/next/src/server/**`

Next.js strips internal headers from incoming requests through
`filterInternalHeaders()` in `packages/next/src/server/lib/server-ipc/utils.ts`,
called at the entry point in `router-server.ts` before any server code runs. It
strips **only** the names listed in the `INTERNAL_HEADERS` array.

That makes the array the trust boundary. Any header not in it arrives exactly as the
client sent it, so server code reading a non-standard header is reading
attacker-controlled input unless that name is on the list.

Before adding code that reads a request header which is not a standard HTTP header —
standard here meaning `content-type`, `accept`, `user-agent`, `host`,
`authorization`, `cookie`, and similar — stop and get it looked at as a security
question. Either the name belongs in `INTERNAL_HEADERS`, or the value must be
treated as untrusted. Do not decide this silently in passing; the repository asks
reviewers to flag exactly this pattern.

*Evidence:*

- `AGENTS.md` lines 508-512 — **authoritative**
- `packages/next/src/server/lib/server-ipc/utils.ts` lines 42-42
- `packages/next/src/server/lib/router-server.ts` lines 245-245

*Proof gap:* Nothing links a header read to the filter list. `filterInternalHeaders` strips only the names in `INTERNAL_HEADERS`, and no check detects server code reading a non-standard header absent from it, so a forgeable-header bug is invisible until review or a report.

**Decision:**

#### `follow-agent-pr-conduct-limits` — Respect the limits on agent-authored PR and issue content

compliance · directive `never` · guidance · 74/100 (high) · confidence high

Lenses: `base` · Scope: `**/*`

This repository places explicit limits on what an agent may author, and they differ
by pull request origin.

Never write a full description for a **fork** pull request whose merge target is
`vercel/next.js`. Fork pull requests are external contributions, and the repository
requires that you tell the user you are not permitted to write the description. You
may still offer to review their description, supply technical detail, help translate
to English, or provide the link for them to open it themselves. For **branch**
pull requests — those on `vercel/next.js` itself — descriptions are allowed, as are
titles and messages for local commits.

The same restriction covers issues, discussions, and comments: only members of the
`vercel` or `vercel-labs` organizations may have an agent create them, and
membership is checkable through the GitHub API or CLI. Commenting on the user's own
pull request is among the documented exceptions.

Two additional prohibitions apply to every change here:

- never add "Generated with Claude Code" or co-author footers to commits or pull
  requests; and
- never mark a pull request ready for review with `gh pr ready` — leave it in draft
  and let the user decide.

*Evidence:*

- `AGENTS.md` lines 279-291 — **authoritative**
- `AGENTS.md` lines 419-424 — **authoritative**

*Proof gap:* These are constraints on actions taken in the GitHub interface, not properties of the repository. No repository check can observe a description that was written or a pull request that was marked ready.

**Decision:**

#### `keep-react-server-imports-in-entry-base` — Keep vendored React server imports in entry-base

architecture · directive `always` · guidance · 72/100 (high) · confidence high

Lenses: `language:typescript` · Scope: `packages/next/src/**`

`react-server-dom-webpack/*` imports must stay in `entry-base.ts`. Elsewhere,
consume them through component module exports rather than importing them directly.

The same boundary explains a related constraint. `app-page.ts` is a build template
compiled by the *user's* bundler, so every `require()` in it is traced by webpack or
Turbopack at `next build` time in the user's project. Internal modules cannot be
required by relative path there, because those paths do not resolve from a user's
project. Export the helper from `entry-base.ts` and reach it as `entryBase.*` in the
template instead.

This is the failure mode worth remembering: the mistake works locally and breaks
only for users. `entry-base.ts` is the single place where the vendored React surface
is allowed to enter, which is what keeps that resolution predictable.

The `$react-vendoring` skill in `.agents/skills/react-vendoring/SKILL.md` covers the
boundaries in detail.

*Evidence:*

- `AGENTS.md` lines 471-471 — **authoritative**
- `AGENTS.md` lines 480-480 — **authoritative**

*Proof gap:* No check asserts the boundary. A relative `require()` added to a build template resolves correctly in this repository and fails only in a user's project at `next build`, where the path does not exist.

**Decision:**

#### `use-retry-not-check-in-tests` — Never poll with setTimeout or the deprecated check helper

testability · directive `never` · guidance · 72/100 (high) · confidence high

Lenses: `task:testing` · Scope: `test/**`

Never wait on asynchronous behavior with `await new Promise(resolve =>
setTimeout(resolve, ...))`, and never use `check()` from `next-test-utils` — it is
deprecated.

Use `retry()` from `next-test-utils` with `expect()` inside it:

```typescript
import { retry } from 'next-test-utils'
await retry(async () => {
  const text = await browser.elementByCss('p').text()
  expect(text).toBe('expected value')
})
```

A fixed sleep encodes a guess about timing, which is the main source of flakiness
in this suite: too short and it fails under load, too long and every run pays for
it. `retry()` polls until the assertion passes and reports the real assertion
failure when it does not.

`check()` is the older form of the same idea and still exists, so nothing stops you
from using it. Prefer `retry()` plus `expect()` in new tests, and when you are
already editing a test that uses `check()`, converting it is a reasonable
incidental improvement.

*Evidence:*

- `AGENTS.md` lines 184-209 — **authoritative**

*Proof gap:* Verified against the repository's ast-grep rule set: none of the six rules in `.config/ast-grep/rules/` covers `check()` or `setTimeout`. `check()` remains exported from `next-test-utils` and callable, so neither the deprecated helper nor a bare sleep fails any existing check.

**Decision:**

#### `generate-tests-with-new-test` — Generate new test suites with the repository generator

testability · directive `always` · guidance · 70/100 (high) · confidence high

Lenses: `task:testing` · Scope: `test/**`

Create new test suites with `pnpm new-test` rather than by hand. The repository
states the generator is mandatory, and it produces the directory layout and fixture
files the harness expects.

For non-interactive use, forward arguments through `--`:

```bash
pnpm new-test -- --args <appDir> <name> <type>
```

`appDir` is `true` or `false`, `name` is the test name, and `type` is one of
`e2e`, `production`, `development`, or `unit`.

The generator also produces the fixture-directory shape the repository prefers.
Point `nextTestSetup` at a real directory with `files: __dirname` rather than
defining files inline; inline `files` objects are harder to maintain and the
generated structure already avoids them.

*Evidence:*

- `AGENTS.md` lines 150-162 — **authoritative**
- `package.json` lines 10-10 — **authoritative**

*Proof gap:* No check asserts that a test suite was scaffolded by the generator. A hand-written suite missing the expected fixture structure fails later and for reasons that do not point back to the omission.

**Decision:**

#### `keep-ast-grep-rule-pairs-in-sync` — Keep paired TypeScript and TSX ast-grep rules in sync

maintainability · directive `always` · guidance · 66/100 (high) · confidence high

Lenses: `task:implementation` · Scope: `.config/ast-grep/rules/**`

An ast-grep rule matches a single language, so a pattern that must apply to both
`.ts` and `.tsx` sources exists as two files. `no-typeof-window-require.yml` and
`no-typeof-window-require-tsx.yml` are such a pair, and each file's header says so:
"ast-grep rules are single-language; keep the two `rule` bodies in sync."

When editing the pattern in one, make the same edit in its counterpart. When adding
a new rule that should cover both file types, add both files and cross-reference
them in comments the way the existing pair does.

Only genuinely paired rules are in scope. Most rules in this directory are
single-language on purpose — the Rust-oriented ones such as `no-context.yml` and
`no-err-anyhow.yml` have no TSX counterpart and need none. The signal that a rule is
half of a pair is the explicit note in its header, not merely that it targets
TypeScript.

*Evidence:*

- `.config/ast-grep/rules/no-typeof-window-require.yml` lines 3-4 — **authoritative**
- `.config/ast-grep/rules/no-typeof-window-require-tsx.yml` lines 3-4 — **authoritative**

*Proof gap:* Nothing compares the two `rule` bodies. Editing one and not the other leaves the scan passing while coverage silently drops for the other file type, which is the same outcome as having no rule.

**Decision:**

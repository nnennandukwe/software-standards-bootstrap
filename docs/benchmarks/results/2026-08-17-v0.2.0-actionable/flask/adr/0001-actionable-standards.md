# ADR 0001: Adopt actionable repository standards

- Status: Proposed
- Baseline commit: `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`
- Manifest: `.software-standards/manifest.yaml`
- Inventory: `.software-standards/inventory.json`
- Report: `.software-standards/report.md`

## Context

The repository was inspected at the pinned baseline above. The developer retained the following evidence-backed actionable artifacts after review. Verification recipes are recorded here but were not executed by SSB.

## Semantic rules

### Deprecate public APIs before removal (`deprecate-public-apis-before-removal`)

- Source: `.software-standards/rules/deprecate-public-apis-before-removal.md`
- Scope: `src/flask/**/*.py`
- Lenses: `language:python`
- Directive: `always`
- Category: `compatibility`
- Derivation: `inferred`
- Confidence: `high`
- Utility: `very-high` (92/100, `ssb-utility-v1`)
- Evidence: `CHANGES.rst:6-16` (`demonstrates`), `src/flask/app.py:300-308` (`demonstrates`), `src/flask/ctx.py:528-538` (`demonstrates`), `src/flask/globals.py:65-75` (`demonstrates`), `docs/api.rst:309-317` (`demonstrates`)

Before removing a public API, announce and document a deprecation period and removal version, and emit a `DeprecationWarning`. When a replacement exists, name it in the warning and keep the old use working through a compatibility alias or adapter during that period. Remove the compatibility path only after the announced period.

### Keep change evidence synchronized (`keep-change-evidence-synchronized`)

- Source: `.software-standards/rules/keep-change-evidence-synchronized.md`
- Scope: `**/*`
- Lenses: `base`
- Directive: `always`
- Category: `quality`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (93/100, `ssb-utility-v1`)
- Evidence: `.github/pull_request_template.md:17-24` (`declares`)

For every behavior change, add tests that demonstrate the correct behavior and fail without the change. Update the relevant documentation in `docs` and in code, add a `CHANGES.rst` entry that summarizes the change and links its issue, and add `.. versionchanged::` entries to affected code documentation.

### Keep Sans-IO free of I/O and Flask globals (`keep-sansio-free-of-io-and-globals`)

- Source: `.software-standards/rules/keep-sansio-free-of-io-and-globals.md`
- Scope: `src/flask/sansio/**/*.py`
- Lenses: `language:python`
- Directive: `never`
- Category: `architecture`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (89/100, `ssb-utility-v1`)
- Evidence: `src/flask/sansio/README.md:3-6` (`declares`)

Keep `src/flask/sansio` reusable by alternative Flask implementations. Do not perform I/O, enter a likely I/O path, or access Flask globals from this layer. Put I/O-bound and Flask-context behavior in the concrete layer outside `sansio`.

## Consequences

- `AGENTS.md` is a derived projection; the manifest, inventory, human report, and canonical artifact source files remain editable.
- Verification recipes remain deliberately invoked repository procedures; this record does not claim their commands passed.
- The developer-created pull request and its merge constitute adoption; this ADR remains Proposed until then.

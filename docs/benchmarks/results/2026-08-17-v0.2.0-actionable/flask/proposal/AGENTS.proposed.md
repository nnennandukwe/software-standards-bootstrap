<!-- software-standards-bootstrap:start -->
<!-- source-digest: sha256:af86995fa2dbddfde27f460f2ad24b7a38df5f83817680ed76779d7ec4789fe7 -->
<!-- content-digest: sha256:8d18558410b8dc61f77b58d60a6673e56d7b499a7da2d63669a65a7e67652bf7 -->
## Software Standards Bootstrap

This managed section is derived from retained canonical sources. An unmerged generated change is a proposal; repository review and merge are the adoption decision. File presence alone does not prove adoption.

Generated from `.software-standards/manifest.yaml`, `.software-standards/inventory.json`, `.software-standards/report.md`, `.software-standards/orientation.yaml` and the manifest-listed artifacts by `ssb render`. Edit canonical sources and the manifest together, then rerun the command.

Baseline: `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`

SSB did not stage, commit, push, open a pull request, execute any displayed recipe command, or activate another system. Recipe presence and expected results are not execution evidence.

### Repository orientation

Flask is a lightweight WSGI web application framework designed to scale from quick starts to complex applications while leaving dependency and project-layout choices to application developers.

- Evidence: `README.md:3-14 (declares)`

#### Important areas

- `src/flask/sansio` - Reusable code for alternative Flask implementations that must remain outside I/O paths and independent of Flask globals.
  - Evidence: `src/flask/sansio/README.md:3-6 (declares)`

### How routing works

- **Host-specific:** `AGENTS.md` discovery, directory placement, and nested-file precedence depend on the active host; consult that host's documented behavior.
- **SSB generator-defined:** Scopes and lenses are SSB routing metadata, not native `AGENTS.md` glob activation. A semantic rule applies when its affected path scope matches; contextual artifacts also require every represented lens dimension to match, with values inside one dimension treated as alternatives.
- If the language, framework, task, or affected path is uncertain, load the potentially relevant rule, recipe, or skill instead of excluding it.
- **SSB generator-defined:** Directives mean: `never` is prohibited, `ask-first` requires developer authorization, `always` is required, and `prefer` is the default when no documented exception or explicit user direction applies.
- Linked artifact files are canonical. This projection is a concise router, not a replacement for their complete content.

### Standing orders

#### Always

##### Keep change evidence synchronized (`keep-change-evidence-synchronized`)

> For every behavior change, add tests that demonstrate the correct behavior and fail without the change. Update the relevant documentation in `docs` and in code, add a `CHANGES.rst` entry that summarizes the change and links its issue, and add `.. versionchanged::` entries to affected code documentation.

- Applies to: `**/*`
- Category: `quality`
- Canonical rule: [.software-standards/rules/keep-change-evidence-synchronized.md](.software-standards/rules/keep-change-evidence-synchronized.md)
- Evidence: `.github/pull_request_template.md:17-24`

### Contextual semantic rules

#### [Keep Sans-IO free of I/O and Flask globals](.software-standards/rules/keep-sansio-free-of-io-and-globals.md) (`keep-sansio-free-of-io-and-globals`) - `never`

- Load when: `language:python`
- Applies to: `src/flask/sansio/**/*.py`
- Category: `architecture`
- Canonical rule: [.software-standards/rules/keep-sansio-free-of-io-and-globals.md](.software-standards/rules/keep-sansio-free-of-io-and-globals.md)
- Evidence: `src/flask/sansio/README.md:3-6`

#### [Deprecate public APIs before removal](.software-standards/rules/deprecate-public-apis-before-removal.md) (`deprecate-public-apis-before-removal`) - `always`

- Load when: `language:python`
- Applies to: `src/flask/**/*.py`
- Category: `compatibility`
- Canonical rule: [.software-standards/rules/deprecate-public-apis-before-removal.md](.software-standards/rules/deprecate-public-apis-before-removal.md)
- Evidence: `CHANGES.rst:6-16`, `src/flask/app.py:300-308`, `src/flask/ctx.py:528-538`, `src/flask/globals.py:65-75`, `docs/api.rst:309-317`
<!-- software-standards-bootstrap:end -->

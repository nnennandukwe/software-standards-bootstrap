<!-- software-standards-bootstrap:start -->
<!-- source-digest: sha256:9031b05ab3cccbc2d9feff8ad16d17d563e600e488036a6f3230e5fdf4bde8de -->
<!-- content-digest: sha256:68d7ad02a1fbc9d60962c92f8dc426a5b5644ba6c2f6b815b7e4b157cb1e9f5c -->
## Software Standards Bootstrap

This managed section is derived from retained canonical sources. An unmerged generated change is a proposal; repository review and merge are the adoption decision. File presence alone does not prove adoption.

Generated from `.software-standards/manifest.yaml`, `.software-standards/inventory.json`, `.software-standards/report.md`, `.software-standards/orientation.yaml` and the manifest-listed artifacts by `ssb render`. Edit canonical sources and the manifest together, then rerun the command.

Baseline: `adbc8813901bba65827259daa8e22ff94ec1f30e`

SSB did not stage, commit, push, open a pull request, execute any displayed recipe command, or activate another system. Recipe presence and expected results are not execution evidence.

### Repository orientation

Cobra is a Go library for creating modern CLI applications.

- Evidence: `README.md:7-14 (declares)`

#### Important areas

- `doc` - Generates man, Markdown, reStructuredText, and YAML command documentation.
  - Evidence: `doc/man_docs.go:33-48 (declares)`, `doc/md_docs.go:51-57 (declares)`, `doc/rest_docs.go:56-62 (declares)`, `doc/yaml_docs.go:48-60 (declares)`

#### Related standards

- Related skill: [Change shell completion behavior](.agents/skills/change-shell-completion-behavior/SKILL.md)
- Related skill: [Change document generation](.agents/skills/change-document-generation/SKILL.md)

### How routing works

- **Host-specific:** `AGENTS.md` discovery, directory placement, and nested-file precedence depend on the active host; consult that host's documented behavior.
- **SSB generator-defined:** Scopes and lenses are SSB routing metadata, not native `AGENTS.md` glob activation. A semantic rule applies when its affected path scope matches; contextual artifacts also require every represented lens dimension to match, with values inside one dimension treated as alternatives.
- If the language, framework, task, or affected path is uncertain, load the potentially relevant rule, recipe, or skill instead of excluding it.
- **SSB generator-defined:** Directives mean: `never` is prohibited, `ask-first` requires developer authorization, `always` is required, and `prefer` is the default when no documented exception or explicit user direction applies.
- Linked artifact files are canonical. This projection is a concise router, not a replacement for their complete content.

### Standing orders

#### Always

##### Cover Go code changes with tests (`cover-go-code-changes-with-tests`)

> When submitting a Go feature or behavior change, add adequate tests that exercise its intended behavior.

- Applies to: `**/*.go`
- Category: `testability`
- Canonical rule: [.software-standards/rules/cover-go-code-changes-with-tests.md](.software-standards/rules/cover-go-code-changes-with-tests.md)
- Evidence: `CONTRIBUTING.md:33-34`

### Contextual semantic rules

#### [Preserve compatibility APIs until Cobra v2](.software-standards/rules/preserve-compatibility-apis-until-cobra-v2.md) (`preserve-compatibility-apis-until-cobra-v2`) - `never`

- Load when: `language:go`
- Applies to: `**/*.go`
- Related skill: [Change shell completion behavior](.agents/skills/change-shell-completion-behavior/SKILL.md)
- Category: `compatibility`
- Canonical rule: [.software-standards/rules/preserve-compatibility-apis-until-cobra-v2.md](.software-standards/rules/preserve-compatibility-apis-until-cobra-v2.md)
- Evidence: `cobra.go:109-114`, `cobra.go:141-144`, `command.go:590-618`

#### [Keep build constraints compatible with Go 1.15](.software-standards/rules/keep-build-constraints-compatible-with-go-1-15.md) (`keep-build-constraints-compatible-with-go-1-15`) - `always`

- Load when: `language:go`
- Applies to: `**/*.go`
- Category: `compatibility`
- Canonical rule: [.software-standards/rules/keep-build-constraints-compatible-with-go-1-15.md](.software-standards/rules/keep-build-constraints-compatible-with-go-1-15.md)
- Evidence: `.golangci.yml:75-81`, `command_notwin.go:15-18`, `command_win.go:15-18`

### Agent Skills

#### [Change document generation](.agents/skills/change-document-generation/SKILL.md) (`change-document-generation`)

Coordinate Cobra documentation-generation changes across man, Markdown, reStructuredText, and YAML implementations, tests, and public guidance.

- Use when: `task:planning`, `task:implementation`
- Applies to: `doc/*.go`, `site/content/docgen/**`
- Category: `documentation`
- Evidence: `site/content/docgen/_index.md:1-13`, `doc/man_docs.go:33-48`, `doc/md_docs.go:51-57`, `doc/rest_docs.go:56-62`, `doc/yaml_docs.go:48-60`, `doc/man_docs_test.go:39-67`, `doc/md_docs_test.go:26-42`, `doc/rest_docs_test.go:26-41`, `doc/yaml_docs_test.go:27-42`

#### [Change shell completion behavior](.agents/skills/change-shell-completion-behavior/SKILL.md) (`change-shell-completion-behavior`)

Coordinate shared and shell-specific Cobra completion changes across implementations, tests, compatibility surfaces, and user documentation.

- Use when: `task:planning`, `task:implementation`
- Applies to: `*completions*.go`, `active_help*.go`, `site/content/completions/**`
- Related rule: [Preserve compatibility APIs until Cobra v2](.software-standards/rules/preserve-compatibility-apis-until-cobra-v2.md)
- Category: `compatibility`
- Evidence: `site/content/completions/_index.md:1-8`, `site/content/completions/_index.md:458-474`, `site/content/completions/_index.md:491-539`, `active_help.go:22-39`, `bash_completionsV2.go:469-483`, `zsh_completions.go:24-43`, `fish_completions.go:275-291`, `powershell_completions.go:330-349`, `bash_completionsV2_test.go:23-32`, `zsh_completions_test.go:23-32`, `fish_completions_test.go:26-56`, `powershell_completions_test.go:23-32`
<!-- software-standards-bootstrap:end -->

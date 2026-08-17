# ADR 0001: Adopt actionable repository standards

- Status: Proposed
- Baseline commit: `adbc8813901bba65827259daa8e22ff94ec1f30e`
- Manifest: `.software-standards/manifest.yaml`
- Inventory: `.software-standards/inventory.json`
- Report: `.software-standards/report.md`

## Context

The repository was inspected at the pinned baseline above. The developer retained the following evidence-backed actionable artifacts after review. Verification recipes are recorded here but were not executed by SSB.

## Semantic rules

### Cover Go code changes with tests (`cover-go-code-changes-with-tests`)

- Source: `.software-standards/rules/cover-go-code-changes-with-tests.md`
- Scope: `**/*.go`
- Lenses: `base`
- Directive: `always`
- Category: `testability`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `high` (67/100, `ssb-utility-v1`)
- Evidence: `CONTRIBUTING.md:33-34` (`declares`)

When submitting a Go feature or behavior change, add adequate tests that exercise its intended behavior.

### Keep build constraints compatible with Go 1.15 (`keep-build-constraints-compatible-with-go-1-15`)

- Source: `.software-standards/rules/keep-build-constraints-compatible-with-go-1-15.md`
- Scope: `**/*.go`
- Lenses: `language:go`
- Directive: `always`
- Category: `compatibility`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `high` (74/100, `ssb-utility-v1`)
- Evidence: `.golangci.yml:75-81` (`declares`), `command_notwin.go:15-18` (`demonstrates`), `command_win.go:15-18` (`demonstrates`)

On Go files that use build constraints, retain both the `//go:build` form and the equivalent `// +build` form while the module remains at Go 1.15. Remove the legacy form only when the repository raises its required Go version to 1.17 or later.

### Preserve compatibility APIs until Cobra v2 (`preserve-compatibility-apis-until-cobra-v2`)

- Source: `.software-standards/rules/preserve-compatibility-apis-until-cobra-v2.md`
- Scope: `**/*.go`
- Lenses: `language:go`
- Directive: `never`
- Category: `compatibility`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (89/100, `ssb-utility-v1`)
- Evidence: `cobra.go:109-114` (`declares`), `cobra.go:141-144` (`declares`), `command.go:590-618` (`declares`)

Do not remove functions or methods documented as retained for compatibility before a Cobra v2 release.

## Agent Skills

### Change document generation (`change-document-generation`)

- Source: `.agents/skills/change-document-generation/SKILL.md`
- Description: Coordinate Cobra documentation-generation changes across man, Markdown, reStructuredText, and YAML implementations, tests, and public guidance.
- Scope: `doc/*.go`, `site/content/docgen/**`
- Lenses: `task:planning`, `task:implementation`
- Category: `documentation`
- Derivation: `inferred`
- Confidence: `high`
- Utility: `high` (76/100, `ssb-utility-v1`)
- Evidence: `site/content/docgen/_index.md:1-13` (`declares`), `doc/man_docs.go:33-48` (`demonstrates`), `doc/md_docs.go:51-57` (`demonstrates`), `doc/rest_docs.go:56-62` (`demonstrates`), `doc/yaml_docs.go:48-60` (`demonstrates`), `doc/man_docs_test.go:39-67` (`demonstrates`), `doc/md_docs_test.go:26-42` (`demonstrates`), `doc/rest_docs_test.go:26-41` (`demonstrates`), `doc/yaml_docs_test.go:27-42` (`demonstrates`)

### Change shell completion behavior (`change-shell-completion-behavior`)

- Source: `.agents/skills/change-shell-completion-behavior/SKILL.md`
- Description: Coordinate shared and shell-specific Cobra completion changes across implementations, tests, compatibility surfaces, and user documentation.
- Scope: `*completions*.go`, `active_help*.go`, `site/content/completions/**`
- Lenses: `task:planning`, `task:implementation`
- Category: `compatibility`
- Derivation: `inferred`
- Confidence: `high`
- Utility: `very-high` (82/100, `ssb-utility-v1`)
- Evidence: `site/content/completions/_index.md:1-8` (`declares`), `site/content/completions/_index.md:458-474` (`declares`), `site/content/completions/_index.md:491-539` (`declares`), `active_help.go:22-39` (`declares`), `bash_completionsV2.go:469-483` (`demonstrates`), `zsh_completions.go:24-43` (`demonstrates`), `fish_completions.go:275-291` (`demonstrates`), `powershell_completions.go:330-349` (`demonstrates`), `bash_completionsV2_test.go:23-32` (`demonstrates`), `zsh_completions_test.go:23-32` (`demonstrates`), `fish_completions_test.go:26-56` (`demonstrates`), `powershell_completions_test.go:23-32` (`demonstrates`)

## Consequences

- `AGENTS.md` is a derived projection; the manifest, inventory, human report, and canonical artifact source files remain editable.
- Verification recipes remain deliberately invoked repository procedures; this record does not claim their commands passed.
- The developer-created pull request and its merge constitute adoption; this ADR remains Proposed until then.

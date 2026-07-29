# Cobra / Claude Code proposal record

Proposal generated on 2026-07-23. Developer retention decisions were recorded
on 2026-07-26; see [Developer review](#developer-review). All proposed rules
and the related skill were approved as Keep. Edit/delete/rerender propagation
and the explicitly requested ADR remain unverified.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.191
- Model: `claude-sonnet-4-6`
- Reasoning profile: `medium`
- Repository: `github.com/spf13/cobra`
- Baseline commit: `adbc8813901bba65827259daa8e22ff94ec1f30e`
- Evaluation branch: `ssb-claude-evaluation`
- Project skill location:
  `.claude/skills/software-standards-bootstrap/SKILL.md`

## Inventory

- Contract: `ssb-inventory-v1`
- Safe tracked files: 63
- Indexed bytes: 631,792
- Limits: 20,000 files; 25 MiB total; 1 MiB per file
- Truncated: no
- Excluded: 1 binary; all other exclusion categories 0

## Proposal

Validation passed with 6 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `apache-license-header` | compliance | deterministic | 93 | very-high | Pending |
| `nolint-must-name-linter` | quality | deterministic | 75 | high | Pending |
| `flag-annotation-key-constant` | maintainability | guidance | 74 | high | Pending |
| `completion-generator-requires-test` | testability | guidance | 69 | high | Pending |
| `dual-build-tag-syntax` | compatibility | guidance | 66 | high | Pending |
| `doc-generator-requires-test` | testability | guidance | 62 | medium | Pending |

Related skill: `add-shell-completion-generator` (`testability`).

The structural review covered all five required dimensions:

- package/dependency boundaries: recorded as architecture context;
- parallel families: completion and documentation generator/test pairings
  emitted; site-documentation pairing kept assessment-only;
- platform/configuration seams: dual build tags emitted;
- public compatibility: annotation-key constants emitted; inconsistent
  deprecation markers kept assessment-only; and
- source/test/documentation symmetry: narrow family symmetry emitted and
  generic Go test colocation rejected.

`AGENTS.md` rendered with source digest
`sha256:2b6905453d7c9ff23ba8153f3ef2100e55c1ddfab453c25ebd59838ba50aa16f`
and content digest
`sha256:245d1828171ae937e74f5aaeca13384198876b4479225e2e7340f6e0b4816f50`.

## Validation repair loop

The first generated procedural skill used unsupported top-level `id` and
`title` fields and omitted the core `name` field. `ssb validate` rejected it.
Claude removed the unsupported fields, added
`name: add-shell-completion-generator`, selected `metadata.topic: testability`,
and reran validation successfully before rendering.

The initial generation process was interrupted after creating proposal sources
but before validation. A second Claude Code turn was explicitly scoped to
recover those immediately preceding consumer-created files, complete
validation, and render. The interrupted turn is not counted as a passing run;
the validated recovery result is the recorded proposal gate.

The recovery assessment initially copied 63 safe tracked files instead of the
65 in the authoritative `ssb inspect` result. A targeted follow-up corrected
the assessment, revalidated the unchanged pack, previewed the render, and
confirmed that `AGENTS.md` remained byte-stable. The incorrect count is not
used as release evidence.

## Developer review

Decisions below are the developer's judgment, not generator output. On
2026-07-26, the developer approved every pending rule and the related skill as
**Keep**.

**High-band retention: 5 of 5 (100%).** The medium-band rule and related skill
were also kept.

Evidence paths are relative to Cobra baseline `adbc8813901bba65827259daa8e22ff94ec1f30e`.

### High-band rules (count toward the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `apache-license-header` | 93 | `cobra.go:1-13`, `.github/workflows/test.yml:17-32` | Yes (1 of 2) | `docker run -v $(pwd):/wrk -w /wrk ghc…` | Keep | Approved as written by the developer on 2026-07-26. |
| `nolint-must-name-linter` | 75 | `.golangci.yml:47-48`, `.github/workflows/test.yml:35-53` | Yes (all) | `golangci-lint run -v` | Keep | Approved as written by the developer on 2026-07-26. |
| `flag-annotation-key-constant` | 74 | `command.go:33-35`, `flag_groups.go:25-29`, `bash_completions.go:29-34` | **No — source-only** | none | Keep | Approved as written by the developer on 2026-07-26. |
| `completion-generator-requires-test` | 69 | `CONTRIBUTING.md:32-34`, `bash_completions_test.go:83-83`, `zsh_completions_test.go:23-23`, `fish_completions_test.go:26-26` | Yes (1 of 4) | none | Keep | Approved as written by the developer on 2026-07-26. |
| `dual-build-tag-syntax` | 66 | `.golangci.yml:77-81`, `command_notwin.go:15-16`, `command_win.go:15-16` | Yes (1 of 3) | none | Keep | Approved as written by the developer on 2026-07-26. |

### Medium-band rules (outside the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `doc-generator-requires-test` | 62 | `CONTRIBUTING.md:32-34`, `doc/man_docs_test.go:39-39`, `doc/md_docs_test.go:26-26`, `doc/yaml_docs_test.go:27-27` | Yes (1 of 4) | none | Keep | Approved as written by the developer on 2026-07-26. |

### Related skill

| Artifact | Decision | Rationale |
|---|---|---|
| `add-shell-completion-generator` | Keep | Approved as written by the developer on 2026-07-26. |

Generated at `.agents/skills/add-shell-completion-generator/SKILL.md`.

### What each rule asks

- **`apache-license-header`** (93, very-high) — Every `.go` file in this repository must begin with the Apache 2.0 license header exactly as it appears in the existing sources: ``` // Copyright 2013-2023 The Cobra Authors // // Licensed under the Apache License, Version 2.0 (the "License"); // you may not use this file except in compliance with the License. // You may obtain a copy of the License at // // http://www.apache.org/licenses/LICENSE-2.0 // // Unless required by applicable law or agreed to in writing, software // distributed under the License is distributed on an "AS IS" BASIS, // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. // See the License for the specific language governing permissions and // limitations under the License. ``` The CI `lic-headers` job runs `addlicense -check` against every commit and pull request and will fail if any `.go` file is missing the header. The year range and owner string must match the configured values (`2013-2023`, `The Cobra Authors`). Files under `.github/` are excluded from the check.
- **`nolint-must-name-linter`** (75, high) — Every `//nolint` comment must name the specific linter being suppressed: **Correct:** ```go _ = riskyFunc() //nolint:errcheck // reason: return value intentionally discarded ``` **Incorrect — bare nolint suppresses all linters:** ```go _ = riskyFunc() //nolint ``` **Why:** The `nolintlint` linter is enabled in `.golangci.yml` (line 48) and is enforced by the CI `golangci-lint` job. A bare `//nolint` directive suppresses every enabled linter on that line, hiding unrelated findings and silently degrading the quality signal for future contributors. **Exception noted in `.golangci.yml`:** Test files (paths matching `_test.go`) are exempt from the `nolintlint` check that rejects unused `//nolint:staticcheck` directives (`.golangci.yml:70-74`). This accommodates a matrix incompatibility where some Go versions do not trigger certain `staticcheck` warnings in test files while others do. This exemption is narrow and deliberate; it does not permit bare `//nolint` in test files.
- **`flag-annotation-key-constant`** (74, high) — Any string used as a `pflag.Flag.Annotations` key must be declared as a named `const`, not written as an inline string literal at the call site. **Correct — declare once as a constant:** ```go const ( myAnnotationKey = "cobra_annotation_my_feature" ) // Use the constant: flags.SetAnnotation(name, myAnnotationKey, []string{"value"}) ``` **Incorrect — inline literal at the call site:** ```go flags.SetAnnotation(name, "cobra_annotation_my_feature", []string{"value"}) ``` **Why:** Annotation keys are used as map keys inside `pflag.Flag.Annotations`. A typo in a string literal used in one place (`SetAnnotation`) but not in another (`Lookup` / `VisitAll`) produces a silent mismatch: the annotation is set but never read, or vice versa. Declaring the key as a constant makes the compiler catch all such mismatches. A runtime mismatch in `MarkFlagsRequiredTogether` or `ValidateFlagGroups` can cause a panic. All existing annotation key sets follow this pattern: | File | Constants | |---|---| | `command.go:33–35` | `FlagSetByCobraAnnotation`, `CommandDisplayNameAnnotation` | | `flag_groups.go:25–29` | `requiredAsGroupAnnotation`, `oneRequiredAnnotation`, `mutuallyExclusiveAnnotation` | | `bash_completions.go:29–34` | `BashCompFilenameExt`, `BashCompCustom`, `BashCompOneRequiredFlag`, `BashCompSubdirsInDir` | Name the constant with a descriptive `cobra_annotation_` prefix to avoid collisions with user-defined annotation keys.
- **`completion-generator-requires-test`** (69, high) — Every shell completion generator file (e.g., `bash_completions.go`, `zsh_completions.go`, `fish_completions.go`, `powershell_completions.go`, `bash_completionsV2.go`) must have a corresponding `*_test.go` file in the same package that exercises its output. All five existing generators satisfy this invariant: | Generator | Test file | |---|---| | `bash_completions.go` | `bash_completions_test.go` | | `bash_completionsV2.go` | `bash_completionsV2_test.go` | | `zsh_completions.go` | `zsh_completions_test.go` | | `fish_completions.go` | `fish_completions_test.go` | | `powershell_completions.go` | `powershell_completions_test.go` | **Why:** Shell completion scripts are highly sensitive to formatting details. A regression in the generated script typically produces silent failures in the user's shell rather than a visible error, making manual verification impractical. **When adding a new shell:** Create the `<shell>_completions.go` implementation and a `<shell>_completions_test.go` that verifies at minimum the generated script structure and at least one completion scenario. See the `add-shell-completion-generator` skill for the complete workflow.
- **`dual-build-tag-syntax`** (66, high) — Any `.go` file that uses a build constraint to target a specific platform or environment must include **both** the modern `//go:build` directive and the legacy `// +build` directive on consecutive lines. Example (from `command_win.go`): ```go //go:build windows // +build windows ``` **Why:** The repository's minimum Go version is 1.15. The `//go:build` form was introduced in Go 1.17. Go 1.15 and 1.16 only recognise `// +build`; omitting it causes those versions to ignore the constraint entirely, silently compiling platform-specific code on all targets. The `.golangci.yml` `govet` configuration explicitly disables the `buildtag` checker (lines 77–81) to allow both forms to coexist without a lint warning. This is intentional, not an oversight. **Scope:** This rule applies to the specific `_win.go` and `_notwin.go` naming convention used by this repository. If new platform-conditional files are added under a different naming scheme, apply the same requirement. **When Go 1.17 becomes the minimum:** The `.golangci.yml` comment at line 79 notes this can be removed once Cobra requires Go 1.17 or higher. At that point, re-enable the `buildtag` govet check and drop the legacy `// +build` lines.
- **`doc-generator-requires-test`** (62, medium) — Every documentation generator in the `doc/` package (`man_docs.go`, `md_docs.go`, `rest_docs.go`, `yaml_docs.go`) must have a corresponding `*_test.go` file in the same directory that exercises its output. All four existing generators satisfy this invariant: | Generator | Test file | |---|---| | `doc/man_docs.go` | `doc/man_docs_test.go` | | `doc/md_docs.go` | `doc/md_docs_test.go` | | `doc/rest_docs.go` | `doc/rest_docs_test.go` | | `doc/yaml_docs.go` | `doc/yaml_docs_test.go` | **Why:** Documentation generators produce structured text consumed by external tooling (man pages, static-site generators, YAML parsers). Subtle changes to formatting or structure can break downstream pipelines silently; tests catch these regressions before release. **When adding a new format:** Create both the `doc/<format>_docs.go` implementation and a `doc/<format>_docs_test.go` that verifies the generated output against expected structure.

### Open judgment questions

Auto-flagged by evidence profile. These are prompts for review, not verdicts.

1. **Source-only evidence (1 rule).** `flag-annotation-key-constant` (74) cite only code or config that SSB did not mark authoritative — no maintainer-written policy document. For each: is this an invariant the maintainers would defend, or a pattern the generator inferred from repetition?
   - `flag-annotation-key-constant` — `command.go`, `flag_groups.go`, `bash_completions.go`
2. **Lowest-scored rule.** `doc-generator-requires-test` (62) sits at the bottom of this pack.

## Changed and untracked paths

```text
.agents/skills/add-shell-completion-generator/SKILL.md
.claude/skills/software-standards-bootstrap
.software-standards/assessment.md
.software-standards/rules/apache-license-header.md
.software-standards/rules/completion-generator-requires-test.md
.software-standards/rules/doc-generator-requires-test.md
.software-standards/rules/dual-build-tag-syntax.md
.software-standards/rules/flag-annotation-key-constant.md
.software-standards/rules/nolint-must-name-linter.md
AGENTS.md
```

The `.claude/skills/software-standards-bootstrap` path is the evaluator's
uncommitted project-skill harness, not generated repository policy.

## Safety and review boundary

- No Cobra code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index and tracked source tree remained
  unchanged.
- No Git mutation occurred after evaluator setup.
- The proposal remained uncommitted. Rules and skills are editable sources;
  `AGENTS.md` is derived.
- No ADR was previewed or created.

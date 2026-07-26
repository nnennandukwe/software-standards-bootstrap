# Cobra / Claude Code inventory-v2 proposal record

Generated on 2026-07-26. This record is at the mandatory developer-review gate;
it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.220
- Model: reported by the session environment as Opus 5 (1M context), model id
  `claude-opus-5[1m]`. This is **not independently observable** from
  `claude --version`, which prints only the CLI version. Recorded as
  self-reported rather than verified.
- Observable configuration: `learning` output style active. No reasoning-effort
  setting is exposed by the CLI, so none is claimed.
- Host: macOS 15.7.3 build 24G419 (`arm64`)
- Git: 2.39.5 (Apple Git-154)
- Repository: `github.com/spf13/cobra`
- Baseline commit: `adbc8813901bba65827259daa8e22ff94ec1f30e`
- Evaluation branch: `ssb-claude-v2-evaluation`
- SSB source commit: `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Evaluator binary SHA-256:
  `c9d93a2aeef27249a1fc828fef025e77f6f0dd742d847bc4927cd8c538fd6fba`
- Raw inventory output SHA-256:
  `98e65eea07a092693e3de5a5cc945996bd13b8a970a5370dbc631760ac43ddf3`

This raw inventory digest is byte-identical to the digest recorded in
[`codex-cobra.md`](codex-cobra.md). Both runs used the same evaluator binary
against the same pin, so identical output is the expected determinism result and
is not evidence of shared work.

## Independence limitation

This is the one record in the Claude Code set where independence is qualified.
Before this evaluation ran, the operating session had already read the Codex
inventory-v1 Cobra rule bodies in detail and the inventory-v2
[`codex-cobra.md`](codex-cobra.md) record, which lists all seven Codex rule ids.
Every rule below was derived from a direct reading of the pinned Cobra files,
and no Codex proposal source was opened or copied during generation, but this
run cannot be described as blind.

The Flask, Django, and Next.js Codex inventory-v2 packs were **not** read before
their corresponding Claude evaluations, so those three records carry no
equivalent qualification.

## Inventory

- Contract: `ssb-inventory-v2`, schema 2
- Candidate: 66 files, 705,271 bytes
- Scanned: 66 files, 705,271 bytes
- Indexed: 65 files, 631,792 bytes
- Remaining: 0 files, 0 bytes
- Truncated: no
- Limits: 40,000 candidate files; 134,217,728 candidate bytes; 1,048,576 bytes
  per file
- Excluded: 1 binary; all other categories 0

## Pending proposal

`ssb validate` passed with 8 rules, 1 related skill, and 100% evidence
resolution. Validation passed on the first attempt with no corrections.

| Rule | Primary topic | Classification | Score | Decision |
|---|---|---|---:|---|
| `verify-go-changes` | correctness | deterministic | 89 | Pending |
| `preserve-license-headers` | compliance | deterministic | 84 | Pending |
| `preserve-go-support-boundary` | compatibility | guidance | 84 | Pending |
| `keep-shell-completion-family-aligned` | compatibility | guidance | 81 | Pending |
| `coordinate-security-fixes-privately` | security | guidance | 76 | Pending |
| `preserve-dual-build-tag-syntax` | compatibility | guidance | 74 | Pending |
| `preserve-explicit-compatibility-shims` | compatibility | guidance | 73 | Pending |
| `keep-doc-generator-autogen-tag-aligned` | maintainability | guidance | 62 | Pending |

Proposed skill: `add-shell-completion-generator` (`compatibility`) — **Pending
review**.

## Structural review and candidate dispositions

All five dimensions were reviewed and recorded in
`.software-standards/assessment.md`.

| Dimension | Disposition |
|---|---|
| Package and dependency boundaries | Assessment-only. The one-way `doc` to `cobra` dependency is stable with no observed near-violation. |
| Parallel implementation families | Emitted `keep-shell-completion-family-aligned` and `keep-doc-generator-autogen-tag-aligned`; the completion family also produced the proposed skill. |
| Platform and configuration seams | Emitted `preserve-dual-build-tag-syntax` and `preserve-go-support-boundary`. The `preExecHookFn` seam is assessment-only for failing the three-occurrence threshold. |
| Public compatibility surfaces | Emitted `preserve-explicit-compatibility-shims`. Plain `Deprecated:` markers excluded as an ordinary Go convention. |
| Source, test, and documentation symmetry | Represented inside the two family rules. Routine `_test.go` colocation rejected as a generic convention. |

Two findings are worth the reviewer's attention because they were verified
rather than assumed:

- `yaml_docs.go` emits no auto-generated tag at all, so `DisableAutoGenTag` is
  vacuous there rather than drifted. The generator family rule therefore covers
  only the three generators that emit the tag.
- The CI matrix's lowest tested Go version is 1.17 while `go.mod` declares
  `go 1.15`, so the declared floor is asserted but never built or tested. This is
  recorded as the proof gap on `preserve-go-support-boundary`.

## Rendered output

The derived `AGENTS.md` was created fresh; Cobra had no pre-existing
`AGENTS.md`. Digests:

- source digest:
  `sha256:c0d98e5c1e81a84d1299f0700f27b633153b0ccb9c076519143c9e839e2312f4`
- content digest:
  `sha256:109edfddeddcdae87e7fc121ca5f07a767f821d5a1661300f4ff36dcac739f38`

`ssb validate` was run again after rendering and passed.

## Proposed paths

```text
.agents/skills/add-shell-completion-generator/SKILL.md
.software-standards/assessment.md
.software-standards/rules/coordinate-security-fixes-privately.md
.software-standards/rules/keep-doc-generator-autogen-tag-aligned.md
.software-standards/rules/keep-shell-completion-family-aligned.md
.software-standards/rules/preserve-dual-build-tag-syntax.md
.software-standards/rules/preserve-explicit-compatibility-shims.md
.software-standards/rules/preserve-go-support-boundary.md
.software-standards/rules/preserve-license-headers.md
.software-standards/rules/verify-go-changes.md
AGENTS.md
```

The untracked `.claude/skills/software-standards-bootstrap` symlink is the
evaluator's own skill attachment. It is not part of the proposal.

## Safety and review boundary

- The clone was fresh, created for this run, and `HEAD` remained at
  `adbc8813901bba65827259daa8e22ff94ec1f30e` throughout.
- No Cobra code, test, hook, build script, linter, formatter, package manager,
  or cited verification command was executed. `make all`,
  `golangci-lint`, and the `addlicense` check are cited only.
- No Cobra dependency was installed.
- The Git index is clean: 0 staged paths and 0 modified tracked files. All
  proposal paths are untracked.
- No Git mutation beyond creating the initial attached evaluation branch.
- No ADR was previewed or created, and `ssb adr` was never run.

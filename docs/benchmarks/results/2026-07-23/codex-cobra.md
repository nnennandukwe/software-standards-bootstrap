# Cobra / Codex proposal record

Proposal generated on 2026-07-23. Developer retention decisions recorded on
2026-07-24; see [Developer review](#developer-review). The edit, delete,
rerender, and explicitly requested ADR steps for this pack are not yet
performed, so this record is not yet a completed end-to-end benchmark judgment.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.715.71837 (build 5702)
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/spf13/cobra`
- Baseline commit: `adbc8813901bba65827259daa8e22ff94ec1f30e`
- Evaluation branch: `ssb-evaluation`

## Inventory

- Contract: `ssb-inventory-v1`
- Safe tracked files: 65
- Indexed bytes: 631,792
- Limits: 20,000 files; 25 MiB total; 1 MiB per file
- Truncated: no
- Excluded: 1 binary; all other exclusion categories 0

## Proposal

Validation passed with 8 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `verify-go-changes` | correctness | deterministic | 90 | very-high | Keep |
| `preserve-go-compatibility` | compatibility | guidance | 80 | very-high | Edit and keep |
| `preserve-explicit-compatibility-shims` | compatibility | guidance | 78 | high | Keep |
| `keep-completions-portable` | compatibility | guidance | 75 | high | Keep |
| `preserve-license-headers` | compliance | deterministic | 70 | high | Keep |
| `coordinate-security-fixes-privately` | security | guidance | 70 | high | Keep |
| `keep-documentation-generators-aligned` | maintainability | guidance | 64 | medium | Reject |
| `preserve-platform-execution-hook` | compatibility | guidance | 57 | medium | Reject |

Related skill: `maintain-shell-completions` (`compatibility`).

The structural review covered all five required dimensions:

- package/dependency boundaries: root package and `doc` package recorded;
- parallel families: completion and documentation generators evaluated, with
  documentation alignment emitted;
- platform seams: Windows/non-Windows execution hook emitted;
- public compatibility: explicit shims emitted at a narrow scope; and
- source/test/documentation symmetry: completion portability emitted and
  generic colocation rejected.

`AGENTS.md` rendered with source digest
`sha256:d7292856179e565fb255e96a387d086a7970e7aed046137ba5c31c15f8e328f3`
and content digest
`sha256:7f05d7c0af4a52f120207a0fb50fee7da6b7af99658fab54581156a230d957c6`.

## Developer review

Decisions below are the developer's judgment, not generator output. Recorded
2026-07-24 by the repository owner.

**High-band retention: 6 of 6 (100%)** — five Keep and one Edit and keep,
against a 70% threshold. Both rejections are medium-band and are outside the
threshold population.

Evidence paths are relative to Cobra baseline
`adbc8813901bba65827259daa8e22ff94ec1f30e`.

### High-band rules (count toward the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `verify-go-changes` | 90 | `CONTRIBUTING.md:29-36`, `Makefile:12-24` | Yes (both) | `make all` | Keep | Authoritative contribution contract and exact `make all` formatting/test gate. Confirmed 2026-07-24. |
| `preserve-go-compatibility` | 80 | `go.mod:1-3`, `.golangci.yml:76-81`, `.github/workflows/test.yml:56-86` | Yes (all) | none | Edit and keep | Authoritative minimum-Go declaration with matching lint and CI matrix. Edit from the earlier review is reaffirmed. |
| `preserve-explicit-compatibility-shims` | 78 | `command.go:584-622`, `cobra.go:100-170` | Yes (both) | none | Keep | Retained. Both cited shims are explicit authoritative in-source compatibility commitments; the developer accepts the rule's generalized scope as written. |
| `keep-completions-portable` | 75 | `README.md:42-52`, `site/content/completions/_index.md:513-520`, `:539-546` | Yes (all) | none | Keep | Supported four-shell surface, documented compatibility differences, matching implementation/test/doc families, honest proof gap. Confirmed 2026-07-24. |
| `preserve-license-headers` | 70 | `.github/workflows/test.yml:17-32` | Yes | `addlicense -check` | Keep | Exact CI-owned Apache header checker with the `.github/**` exclusion preserved in the rule body. Confirmed 2026-07-24. |
| `coordinate-security-fixes-privately` | 70 | `SECURITY.md:9-23` | Yes | none | Keep | Exact authoritative private disclosure and maintainer-coordination policy. Confirmed 2026-07-24. |

### Medium-band rules (outside the threshold)

| Rule | Score | Evidence | Authoritative | Verification | Decision | Rationale |
|---|---:|---|---|---|---|---|
| `keep-documentation-generators-aligned` | 64 | `site/content/docgen/_index.md:1-13` | Yes (single 13-line source) | none | Reject | Medium band. The score does not clear the developer's bar for a retained standard, and the evidence base is a single short documentation index. |
| `preserve-platform-execution-hook` | 57 | `command.go:1094-1097`, `command_notwin.go:15-20`, `command_win.go:15-40` | **No — source-only** | none | Reject | Medium band and the lowest score in the pack. The score does not clear the developer's bar for a retained standard. |

### Related skill

| Artifact | Topic | Decision | Rationale |
|---|---|---|---|
| `maintain-shell-completions` | compatibility | Edit and keep | Tessl fresh review 71 → 99 after `tessl review fix`; added use trigger, exact Cobra paths, per-shell checks, and a `make all` retry loop. All added paths exist at baseline. Confirmed 2026-07-24. |

### Retained pack

Six rules and one related skill are retained: `verify-go-changes`,
`preserve-go-compatibility`, `preserve-explicit-compatibility-shims`,
`keep-completions-portable`, `preserve-license-headers`, and
`coordinate-security-fixes-privately`, with the
`maintain-shell-completions` skill.

`keep-documentation-generators-aligned` and `preserve-platform-execution-hook`
are deleted. Both rejections are medium-band; high-band retention is unaffected
at 6 of 6.

Both rejections were made on band and score rather than on a specific evidence
defect. Recorded as a calibration signal: the medium band did not produce a
retained standard for this repository.

## Changed and untracked paths

```text
.agents/skills/maintain-shell-completions/SKILL.md
.software-standards/assessment.md
.software-standards/rules/coordinate-security-fixes-privately.md
.software-standards/rules/keep-completions-portable.md
.software-standards/rules/keep-documentation-generators-aligned.md
.software-standards/rules/preserve-explicit-compatibility-shims.md
.software-standards/rules/preserve-go-compatibility.md
.software-standards/rules/preserve-license-headers.md
.software-standards/rules/preserve-platform-execution-hook.md
.software-standards/rules/verify-go-changes.md
AGENTS.md
```

## Safety and review boundary

- No Cobra code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index and tracked source tree remained
  unchanged.
- SSB performed no network or Git mutation.
- The proposal remained uncommitted. Rules and skills are editable sources;
  `AGENTS.md` is derived.
- No ADR was previewed or created.

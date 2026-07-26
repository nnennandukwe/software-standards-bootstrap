# Cobra / Codex inventory-v2 proposal record

Generated on 2026-07-26. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.721.31836 (build 5828), Codex CLI
  0.146.0-alpha.3.1
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/spf13/cobra`
- Baseline commit: `adbc8813901bba65827259daa8e22ff94ec1f30e`
- Evaluation branch: `ssb-codex-v2-evaluation`
- SSB source commit: `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Raw inventory output SHA-256:
  `98e65eea07a092693e3de5a5cc945996bd13b8a970a5370dbc631760ac43ddf3`

## Inventory

- Contract: `ssb-inventory-v2`, schema 2
- Candidate and scanned coverage: 66 files, 705,271 bytes
- Indexed coverage: 65 files, 631,792 bytes
- Limits: 40,000 files; 128 MiB total; 1 MiB per file
- Remaining: 0 files, 0 bytes
- Truncated: no
- Excluded: 1 binary; all other categories 0

## Pending proposal

Validation passed with 7 rules, 1 new procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Decision |
|---|---|---|---:|---|
| `verify-go-changes` | correctness | deterministic | 90 | Pending |
| `preserve-go-support-boundary` | compatibility | guidance | 86 | Pending |
| `preserve-license-headers` | compliance | deterministic | 82 | Pending |
| `keep-shell-completions-portable` | compatibility | guidance | 79 | Pending |
| `coordinate-security-fixes-privately` | security | guidance | 75 | Pending |
| `preserve-explicit-compatibility-shims` | compatibility | guidance | 72 | Pending |
| `keep-shell-completion-family-aligned` | maintainability | guidance | 70 | Pending |

Proposed skill: `maintain-shell-completions` — **Pending review**.

The assessment recorded all five structural dimensions. It emitted the public
command compatibility shims, shell-generator family, Go/build-tag support seam,
completion portability surface, and source/test/documentation symmetry.
Platform-hook and documentation-generator candidates remained
assessment-only.

The derived `AGENTS.md` has source digest
`sha256:a8cdc466f0f872bbb2b7a9de0b6da713399ae31d2352fa77ba3bf98d946ea2df`
and content digest
`sha256:811285883f04ea3c3e32231e34d3cb1728b76ab8a72d5c336bdb63ba2b78af5e`.

## Proposed paths

```text
.software-standards/assessment.md
.software-standards/rules/coordinate-security-fixes-privately.md
.software-standards/rules/keep-shell-completion-family-aligned.md
.software-standards/rules/keep-shell-completions-portable.md
.software-standards/rules/preserve-explicit-compatibility-shims.md
.software-standards/rules/preserve-go-support-boundary.md
.software-standards/rules/preserve-license-headers.md
.software-standards/rules/verify-go-changes.md
.agents/skills/maintain-shell-completions/SKILL.md
AGENTS.md
```

The evaluator-only `software-standards-bootstrap` skill attachment is untracked
but is not part of the proposal.

## Safety and review boundary

- The clone was fresh and `HEAD` stayed at the benchmark pin.
- No Cobra code, test, hook, build script, linter, package manager, or cited
  verification command was executed.
- Proposal sources and derived output remain uncommitted; the index is clean.
- No ADR was previewed or created.

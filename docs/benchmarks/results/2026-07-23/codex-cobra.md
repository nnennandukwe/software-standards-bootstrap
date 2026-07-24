# Cobra / Codex proposal record

Generated on 2026-07-23. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

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
| `verify-go-changes` | correctness | deterministic | 90 | very-high | Pending |
| `preserve-go-compatibility` | compatibility | guidance | 80 | very-high | Pending |
| `preserve-explicit-compatibility-shims` | compatibility | guidance | 78 | high | Pending |
| `keep-completions-portable` | compatibility | guidance | 75 | high | Pending |
| `preserve-license-headers` | compliance | deterministic | 70 | high | Pending |
| `coordinate-security-fixes-privately` | security | guidance | 70 | high | Pending |
| `keep-documentation-generators-aligned` | maintainability | guidance | 64 | medium | Pending |
| `preserve-platform-execution-hook` | compatibility | guidance | 57 | medium | Pending |

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

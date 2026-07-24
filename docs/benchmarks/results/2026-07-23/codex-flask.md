# Flask / Codex proposal record

Generated on 2026-07-23. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.715.71837 (build 5702)
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/pallets/flask`
- Baseline commit: `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`
- Evaluation branch: `ssb-evaluation`

## Inventory

- Contract: `ssb-inventory-v1`
- Safe tracked files: 230
- Indexed bytes: 1,474,850
- Limits: 20,000 files; 25 MiB total; 1 MiB per file
- Truncated: no
- Excluded: 5 binary; 1 secret-like; all other exclusion categories 0

## Proposal

Validation passed with 7 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `verify-flask-runtime-matrix` | correctness | deterministic | 95 | very-high | Pending |
| `keep-sansio-boundary-pure` | architecture | guidance | 85 | very-high | Pending |
| `preserve-python-support-boundary` | compatibility | guidance | 83 | very-high | Pending |
| `preserve-async-wsgi-bridge` | compatibility | guidance | 78 | high | Pending |
| `keep-dual-type-checkers-green` | correctness | deterministic | 75 | high | Pending |
| `run-repository-precommit-checks` | quality | deterministic | 75 | high | Pending |
| `document-public-behavior-changes` | documentation | guidance | 70 | high | Pending |

Related skill: `maintain-async-wsgi-bridge` (`compatibility`).

The structural review covered all five required dimensions. It emitted the
sans-I/O architecture boundary and async-to-WSGI compatibility seam, recorded
the Python/platform configuration boundary, evaluated the public behavior
surface, and retained only repository-specific source/test/documentation
symmetry.

`AGENTS.md` rendered with source digest
`sha256:ac9e8cd1a7dd45b58f5944292cd42b87ba0352e085b512793a1c66d81b1bbd12`
and content digest
`sha256:8960eb2d7831dff4668fc1bb8fa8c604736ff968cebc118c1c14fa2eb6722e51`.

## Changed and untracked paths

```text
.agents/skills/maintain-async-wsgi-bridge/SKILL.md
.software-standards/assessment.md
.software-standards/rules/document-public-behavior-changes.md
.software-standards/rules/keep-dual-type-checkers-green.md
.software-standards/rules/keep-sansio-boundary-pure.md
.software-standards/rules/preserve-async-wsgi-bridge.md
.software-standards/rules/preserve-python-support-boundary.md
.software-standards/rules/run-repository-precommit-checks.md
.software-standards/rules/verify-flask-runtime-matrix.md
AGENTS.md
```

## Safety and review boundary

- No Flask code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index and tracked source tree remained
  unchanged.
- SSB performed no network or Git mutation.
- The proposal remained uncommitted. Rules and skills are editable sources;
  `AGENTS.md` is derived.
- No ADR was previewed or created.

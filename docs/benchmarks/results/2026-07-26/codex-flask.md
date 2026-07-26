# Flask / Codex inventory-v2 proposal record

Generated on 2026-07-26. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Codex desktop 26.721.31836 (build 5828), Codex CLI
  0.146.0-alpha.3.1
- Model: `gpt-5.6-sol`
- Reasoning profile: `xhigh`
- Repository: `github.com/pallets/flask`
- Baseline commit: `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`
- Evaluation branch: `ssb-codex-v2-evaluation`
- SSB source commit: `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Raw inventory output SHA-256:
  `ed8013e208958ef189a329644ea1ddfcc4d73657c17c9a27f25576d4cf7e84ef`

## Inventory

- Contract: `ssb-inventory-v2`, schema 2
- Candidate and scanned coverage: 235 files, 1,814,782 bytes
- Indexed coverage: 230 files, 1,474,850 bytes
- Limits: 40,000 files; 128 MiB total; 1 MiB per file
- Remaining: 0 files, 0 bytes
- Truncated: no
- Excluded: 5 binary and 1 secret-like file; all other categories 0

## Pending proposal

Validation passed with 7 rules, 1 new procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Decision |
|---|---|---|---:|---|
| `verify-flask-runtime-matrix` | correctness | deterministic | 92 | Pending |
| `preserve-python-support-boundary` | compatibility | guidance | 87 | Pending |
| `keep-dual-type-checkers-green` | correctness | deterministic | 84 | Pending |
| `run-repository-precommit-checks` | correctness | deterministic | 82 | Pending |
| `keep-sansio-boundary-pure` | architecture | guidance | 80 | Pending |
| `preserve-async-wsgi-bridge` | compatibility | guidance | 78 | Pending |
| `document-public-behavior-changes` | documentation | guidance | 72 | Pending |

Proposed skill: `maintain-async-wsgi-bridge` — **Pending review**.

The assessment recorded all five structural dimensions. It emitted the
sans-I/O package boundary, runtime and type-checker families, Python support
seam, async-to-WSGI public compatibility bridge, and required
test/documentation/changelog symmetry. A separate operating-system rule
remained assessment-only.

The derived `AGENTS.md` has source digest
`sha256:30ffc5bd9f5b160b2450dcff4471b9cc0ece412257431610ef18630944253948`
and content digest
`sha256:bc24d05ee133484a37aa888a52636839c2f7feb8cede937f50f4622006944762`.

## Proposed paths

```text
.software-standards/assessment.md
.software-standards/rules/document-public-behavior-changes.md
.software-standards/rules/keep-dual-type-checkers-green.md
.software-standards/rules/keep-sansio-boundary-pure.md
.software-standards/rules/preserve-async-wsgi-bridge.md
.software-standards/rules/preserve-python-support-boundary.md
.software-standards/rules/run-repository-precommit-checks.md
.software-standards/rules/verify-flask-runtime-matrix.md
.agents/skills/maintain-async-wsgi-bridge/SKILL.md
AGENTS.md
```

The evaluator-only `software-standards-bootstrap` skill attachment is untracked
but is not part of the proposal.

## Safety and review boundary

- The clone was fresh and `HEAD` stayed at the benchmark pin.
- No Flask code, test, hook, build script, linter, package manager, or cited
  verification command was executed.
- Proposal sources and derived output remain uncommitted; the index is clean.
- No ADR was previewed or created.

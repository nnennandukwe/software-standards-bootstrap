# Flask / Claude Code inventory-v2 proposal record

Generated on 2026-07-26. This record is at the mandatory developer-review gate;
it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.220
- Model: reported by the session environment as Opus 5 (1M context), model id
  `claude-opus-5[1m]`. **Not independently observable** from `claude --version`;
  recorded as self-reported rather than verified.
- Observable configuration: `learning` output style active. No reasoning-effort
  setting is exposed by the CLI, so none is claimed.
- Host: macOS 15.7.3 build 24G419 (`arm64`)
- Git: 2.39.5 (Apple Git-154)
- Repository: `github.com/pallets/flask`
- Baseline commit: `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`
- Evaluation branch: `ssb-claude-v2-evaluation`
- SSB source commit: `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Evaluator binary SHA-256:
  `c9d93a2aeef27249a1fc828fef025e77f6f0dd742d847bc4927cd8c538fd6fba`
- Raw inventory output SHA-256:
  `ed8013e208958ef189a329644ea1ddfcc4d73657c17c9a27f25576d4cf7e84ef`

The Codex inventory-v2 Flask pack was not read before this evaluation. This run
is independent.

## Inventory

- Contract: `ssb-inventory-v2`, schema 2
- Candidate: 235 files, 1,814,782 bytes
- Scanned: 235 files, 1,814,782 bytes
- Indexed: 230 files, 1,474,850 bytes
- Remaining: 0 files, 0 bytes
- Truncated: no
- Limits: 40,000 candidate files; 134,217,728 candidate bytes; 1,048,576 bytes
  per file
- Excluded: 5 binary, 1 secret-like; all other categories 0

## Pending proposal

`ssb validate` passed with 8 rules, 1 related skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Decision |
|---|---|---|---:|---|
| `verify-flask-runtime-matrix` | correctness | deterministic | 88 | Pending |
| `preserve-python-support-floor` | compatibility | guidance | 86 | Pending |
| `harden-github-actions-workflows` | security | guidance | 86 | Pending |
| `keep-sansio-layer-protocol-agnostic` | architecture | guidance | 84 | Pending |
| `keep-dual-type-checkers-green` | correctness | deterministic | 78 | Pending |
| `pin-and-freeze-pre-commit-hooks` | security | guidance | 76 | Pending |
| `document-public-changes-in-changes-rst` | documentation | guidance | 68 | Pending |
| `keep-test-warnings-fatal` | maintainability | deterministic | 66 | Pending |

Proposed skill: `add-supported-python-version` (`compatibility`) — **Pending
review**.

## Validation correction loop

The first `ssb validate` run failed with one diagnostic: a
`go-yaml` scanner error in
`.software-standards/rules/pin-and-freeze-pre-commit-hooks.md` at line 32,
because an unquoted `proof_gap` value contained a colon-space sequence. Only the
proposal source was edited — the value was rewritten as a quoted scalar — and
validation passed on rerun. No evidence, score, or rule body semantics changed.

## Structural review and candidate dispositions

All five dimensions were reviewed and recorded in
`.software-standards/assessment.md`.

| Dimension | Disposition |
|---|---|
| Package and dependency boundaries | Emitted `keep-sansio-layer-protocol-agnostic`. |
| Parallel implementation families | No family-alignment rule; Flask has no large sibling-generator set. The `tests/type_check` pairing is expressed as an obligation inside `keep-dual-type-checkers-green` instead. |
| Platform and configuration seams | Emitted `verify-flask-runtime-matrix` and `preserve-python-support-floor`, plus the proposed skill. |
| Public compatibility surfaces | Emitted `keep-dual-type-checkers-green` and `document-public-changes-in-changes-rst`. The `__init__.py` alias re-export style is assessment-only. |
| Source, test, and documentation symmetry | Represented inside the two rules above. Routine test colocation rejected as a generic Python convention. |

Two findings were verified rather than assumed:

- `import os` appears in three sans-I/O modules. Reading them confirmed the
  imports support pure path computation, not I/O, so this is consistent with the
  stated boundary. The rule text says so explicitly to prevent a future reviewer
  misreading the import as a violation.
- `docs/contributing.rst` is a six-line pointer to the external Pallets guide, so
  the in-repository contribution contract is `.github/pull_request_template.md`.
  `document-public-changes-in-changes-rst` cites the template for that reason.

## Rendered output

The derived `AGENTS.md` was created fresh; Flask had no pre-existing
`AGENTS.md`. Digests:

- source digest:
  `sha256:83ea738d86c2f035139de4c36d8710115ec19f80688d903700cbcfba97b06dc9`
- content digest:
  `sha256:1483c77845aa2c110e8d31133f4e87afb08ec0ff20205bcb422224fd1d8153dd`

`ssb validate` was run again after rendering and passed.

## Proposed paths

```text
.agents/skills/add-supported-python-version/SKILL.md
.software-standards/assessment.md
.software-standards/rules/document-public-changes-in-changes-rst.md
.software-standards/rules/harden-github-actions-workflows.md
.software-standards/rules/keep-dual-type-checkers-green.md
.software-standards/rules/keep-sansio-layer-protocol-agnostic.md
.software-standards/rules/keep-test-warnings-fatal.md
.software-standards/rules/pin-and-freeze-pre-commit-hooks.md
.software-standards/rules/preserve-python-support-floor.md
.software-standards/rules/verify-flask-runtime-matrix.md
AGENTS.md
```

The untracked `.claude/skills/software-standards-bootstrap` symlink is the
evaluator's own skill attachment and is not part of the proposal.

## Safety and review boundary

- The clone was fresh and `HEAD` remained at
  `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81` throughout.
- No Flask code, test, hook, linter, formatter, or package manager was executed.
  `tox run`, `tox run -e typing`, `pytest`, `pre-commit run --all-files`, and the
  `zizmor` scan are cited only.
- No Flask dependency was installed; no virtual environment was created.
- The Git index is clean: 0 staged paths and 0 modified tracked files. All
  proposal paths are untracked.
- No Git mutation beyond creating the initial attached evaluation branch.
- No ADR was previewed or created, and `ssb adr` was never run.

# Django / Claude Code proposal record

Generated on 2026-07-23. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.191
- Model: `claude-sonnet-4-6`
- Reasoning profile: `medium`
- Repository: `github.com/django/django`
- Baseline commit: `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`
- Evaluation branch: `ssb-claude-evaluation`
- Project skill location:
  `.claude/skills/software-standards-bootstrap/SKILL.md`

## Inventory

- Contract: `ssb-inventory-v1`
- Safe tracked files: 3,730
- Indexed bytes: 26,184,396
- Limits: 20,000 files; 25 MiB total; 1 MiB per file
- Truncated: yes, at the total-byte limit
- Excluded: 1,267 binary; 4 symlinks; 72 vendor/generated-tree; all other
  exclusion categories 0
- Indexed coverage ends at `tests/delete/models.py`; later paths were not used
  for positive or negative conclusions.

## Proposal

Validation passed with 10 rules, 1 procedural skill, and 100% evidence
resolution.

| Rule | Primary topic | Classification | Score | Band | Decision |
|---|---|---|---:|---|---|
| `python-black-formatting` | maintainability | deterministic | 83 | very-high | Pending |
| `import-order-isort` | maintainability | deterministic | 81 | very-high | Pending |
| `flake8-line-length` | maintainability | deterministic | 78 | high | Pending |
| `test-use-assertraisesmessage` | testability | guidance | 75 | high | Pending |
| `python-requires-3-12-plus` | compatibility | guidance | 74 | high | Pending |
| `no-settings-at-module-level` | correctness | guidance | 73 | high | Pending |
| `deprecation-warning-class` | compatibility | guidance | 71 | high | Pending |
| `f-string-no-i18n` | correctness | guidance | 71 | high | Pending |
| `model-field-lowercase` | maintainability | guidance | 71 | high | Pending |
| `pr-trac-ticket-required` | developer-experience | guidance | 64 | medium | Pending |

Related skill: `add-deprecation` (`compatibility`).

The structural review covered all five required dimensions. It recorded
database-backend package boundaries, evaluated database/cache implementation
families and test symmetry, mapped Python/platform CI seams, emitted typed
deprecation compatibility guidance, and kept broad release-note symmetry
assessment-only.

`AGENTS.md` rendered with source digest
`sha256:ee46c9914007a2bc73b9a8d9766fcfcbbaa673742880cb4450cd281a02124246`
and content digest
`sha256:c0683806f3121202881e08b461936591dbecf541bf465b3356b7dcea97d9e66a`.

## Assessment correction

The initial assessment disclosed truncation and exclusions but omitted the
3,730 safe-file and 26,184,396 indexed-byte totals. An evaluator audit caught
the omission. A targeted Claude Code recovery turn added the authoritative
inventory values, revalidated the unchanged rule pack, previewed the render,
and rerendered it. The incomplete assessment is not counted as final evidence.

## Changed and untracked paths

```text
.agents/skills/add-deprecation/SKILL.md
.claude/skills/software-standards-bootstrap
.software-standards/assessment.md
.software-standards/rules/deprecation-warning-class.md
.software-standards/rules/f-string-no-i18n.md
.software-standards/rules/flake8-line-length.md
.software-standards/rules/import-order-isort.md
.software-standards/rules/model-field-lowercase.md
.software-standards/rules/no-settings-at-module-level.md
.software-standards/rules/pr-trac-ticket-required.md
.software-standards/rules/python-black-formatting.md
.software-standards/rules/python-requires-3-12-plus.md
.software-standards/rules/test-use-assertraisesmessage.md
AGENTS.md
```

The `.claude/skills/software-standards-bootstrap` path is the evaluator's
uncommitted project-skill harness, not generated repository policy.

## Safety and review boundary

- No Django code, hook, build script, test, linter, package manager, or cited
  verification command was executed.
- `HEAD` stayed at the pin; the index and tracked source tree remained
  unchanged.
- No Git mutation occurred after evaluator setup.
- The proposal remained uncommitted. Rules and skills are editable sources;
  `AGENTS.md` is derived.
- No ADR was previewed or created.

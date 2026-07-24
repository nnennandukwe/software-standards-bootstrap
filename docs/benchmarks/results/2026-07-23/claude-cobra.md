# Cobra / Claude Code proposal record

Generated on 2026-07-23. This record is at the mandatory developer-review
gate; it is not a completed benchmark judgment and contains no ADR.

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

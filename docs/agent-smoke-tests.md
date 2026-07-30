# Agent-host behavioral conformance tests

This suite is integration evidence for one observed host version and fixture.
It does not establish behavior for other hosts or versions.

A conforming host can expose the portable skill, read the report and canonical
artifacts, invoke developer-authorized `ssb` commands, and disclose active
artifacts, file activity, command activity, and results.

## Generation behavior

In a fresh clone at a pinned fixture:

1. Confirm attached `HEAD` and a clean tracked/staged state.
2. Expose `software-standards-bootstrap` through the host's documented project
   skill mechanism.
3. Ask it to generate an evidence-backed actionable standards pack.
4. Verify it runs `ssb inspect` before writing files and stops on incomplete
   inventory without `--allow-partial`.
5. Verify targeted reads resolve to the inventory baseline and no repository
   code or command is executed.
6. Verify all five structural dimensions and existing automatic enforcement
   are reviewed before candidate routing.
7. Verify each candidate is routed to exactly one semantic rule, verification
   recipe, Agent Skill, or automation proposal, or is rejected and discarded.
8. Verify accepted artifacts have exact evidence, one category, activation
   metadata, `medium` or `high` confidence, and utility of at least 45.
9. Verify rules contain no command/check metadata; recipes contain existing
   commands with `enforces` references; procedural decisions become skills;
   and proposed checks remain automation proposals.
10. Verify the final report contains complete inventory and accepted outputs
    but no rejected candidates, reasons, or counts.
11. Verify `ssb validate` and `ssb render` succeed.
12. Verify all changed and untracked paths are disclosed and no Git mutation
    occurred.
13. Edit one artifact, delete another with its manifest entry, rerender, and
    verify only surviving adoptable artifacts appear.
14. Ask explicitly for the ADR after review.
15. Verify `AGENTS.md` inlines base rules, links contextual rules and recipes,
    indexes skills, shows related recipe/skill links, and omits automation.
16. Verify the ADR records rules, recipes, and skills with category,
    derivation, confidence, utility, and evidence; excludes automation; and
    remains `Proposed`.
17. Verify zero-artifact and automation-only packs create no managed section
    and no ADR.

## Existing-pack selection matrix

Use fixtures with base and contextual artifacts whose bodies contain unique
sentinels.

| Request | Expected task lens | Required behavior |
|---|---|---|
| Plan a bounded change | `planning` | Load matching base, path, language, framework, and planning artifacts. |
| Implement a bounded change | `implementation` | Load matching implementation artifacts and no unrelated planning-only bodies. |
| Verify a change | `verification` | Load matching recipes and verification skills; run commands only with separate authorization. |

For every row, verify the host:

- does not inspect, rewrite, or rerender the existing pack;
- reconciles report entries, managed-index entries, and canonical sources;
- reports active artifact IDs;
- requires matching scopes and every represented lens dimension;
- loads conservatively when context is uncertain;
- does not read irrelevant contextual bodies;
- does not treat automation proposals as active; and
- never reports a recipe command as executed or passed without execution
  evidence.

## Acceptance

Require 100% resolvable evidence and at least 70% developer keep or
edit-and-keep across all final artifact kinds.

Record the host, version, fixture pin, inventory, artifact counts by kind,
evidence result, developer decisions, complete changed paths, and forbidden
actions.

Codex may expose the skill at
`.agents/skills/software-standards-bootstrap/SKILL.md`. Claude Code may expose
the same portable content at
`.claude/skills/software-standards-bootstrap/SKILL.md`. Other hosts use their
documented project mechanism without adding host-specific fields to the
portable source.

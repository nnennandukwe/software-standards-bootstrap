# Codex and Claude Code behavioral smoke tests

These are release acceptance tests, not automated claims. Record the consumer version, operating system, fixture commit, and complete changed-path list with each result.

## Shared acceptance behavior

In a fresh clone of a pinned fixture:

1. Confirm attached `HEAD` and a clean tracked/staged state.
2. Expose the portable `software-standards-bootstrap` skill through the consumer's documented project skill location.
3. Invoke:

   ```text
   Use the software-standards-bootstrap skill to analyze this repository
   and generate an evidence-backed rules pack.
   ```

4. Verify the host runs `ssb inspect` before writing proposal files.
5. Verify targeted reads resolve to the inventory baseline and repository code is not executed.
6. Verify the structural-pattern review covers all five required dimensions and records a disposition for every plausible structural candidate.
7. Verify emitted rules have resolvable evidence, score arithmetic, scope, confidence, and honest proof classification.
8. Verify genuinely procedural work becomes a portable skill instead of a long rule.
9. Verify `ssb validate` and `ssb render` succeed.
10. Verify the host reports every changed and untracked path and performs no Git mutation.
11. Edit one rule, delete another, rerender, and verify only surviving sources appear.
12. Ask explicitly for the ADR only after reviewing retained files.
13. Verify the ADR contains only surviving artifacts and remains `Proposed`.

Acceptance requires 100% resolvable evidence and at least 70% developer “keep” or “edit and keep” judgment among high and very-high candidates.

## Codex

Place or link the skill at:

```text
.agents/skills/software-standards-bootstrap/SKILL.md
```

Test both explicit invocation and description-based selection. Do not add Codex-only `agents/openai.yaml` metadata to the portable contract.

## Claude Code

Expose the same `SKILL.md` content at Claude Code's documented project location:

```text
.claude/skills/software-standards-bootstrap/SKILL.md
```

Do not infer that Claude Code discovers `.agents/skills`. Do not add Claude-only substitutions or frontmatter to the portable source.

## Result record

Use this table in a release issue or pull request:

| Consumer/version | Fixture/commit | OS | Evidence resolution | High-band retention | Git mutation | Result |
|---|---|---|---:|---:|---|---|
| Codex | | | | | None required | |
| Claude Code | | | | | None required | |

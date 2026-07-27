# Agent-host behavioral conformance tests

This suite is release and integration evidence. It is not part of normal
developer usage. A developer uses `ssb` by exposing the portable skill to one
compatible agent host and asking that host to generate, consume, maintain, or
record the repository's standards pack.

A conforming agent host can:

- expose the portable `SKILL.md` through a documented installation, discovery,
  or explicit-import mechanism;
- read repository guidance and canonical rule sources;
- invoke developer-authorized `ssb` commands; and
- disclose active rule IDs, file activity, command activity, and results.

Run this suite once for every host compatibility claim. A passing record proves
the observed host version and fixture only; it does not establish behavior for
other hosts or versions. Maintainers may use Codex and Claude Code as reference
adapters, but neither is a prerequisite for a developer using `ssb` through a
different conforming host.

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
5. Verify inventory coverage is complete. Exit `4` or `truncated: true` must
   stop the workflow, and the host must not pass `--allow-partial`.
6. Verify targeted reads resolve to the inventory baseline and repository code is not executed.
7. Verify the structural-pattern review covers all five required dimensions and records a disposition for every plausible structural candidate.
8. Verify newly emitted rules use `ssb.dev/rule/v2` and have resolvable
   evidence, score arithmetic, scope, confidence, exactly one supported
   primary topic, valid lenses and directive, and honest proof coverage.
9. Verify genuinely procedural work becomes a portable skill instead of a long rule, and that the skill declares its supported primary topic in `metadata.topic`.
10. Verify `ssb validate` and `ssb render` succeed.
11. Verify the host reports every changed and untracked path and performs no Git mutation.
12. Edit one rule, delete another, rerender, and verify only surviving sources appear.
13. Ask explicitly for the ADR only after reviewing retained files.
14. Verify `AGENTS.md` places base standing orders inline in
    a legacy v1 group when needed, followed by `Never`, `Ask first`, `Always`,
    `Prefer` order for v2, and links contextual rules without copying their
    bodies.
15. Verify its verification section deduplicates commands and labels them
    “mapped, not executed by ssb”.
16. Verify `AGENTS.md` exposes every retained rule's primary topic and the ADR
    exposes primary topics for both surviving rules and referenced skills.
17. Verify the ADR exposes lenses, directive, verification coverage, and the
    bounded property proved for each retained v2 rule.
18. Verify the ADR contains only surviving artifacts and remains `Proposed`.

## Existing-pack progressive-selection matrix

Run these prompts after a valid pack exists. Include fixtures with at least one
base rule and language, framework, and task rules whose bodies contain unique
sentinel text.

| Request | Expected task lens | Required behavior |
|---|---|---|
| Implement a bounded change | `implementation` | Load matching base, path, language, framework, and implementation rules. |
| Review the change | `review` | Load matching base and review rules; do not load implementation-only bodies. |
| Add or run tests | `testing` | Load matching base and testing rules; do not load unrelated security bodies. |
| Perform a security assessment | `security` | Load matching base and security rules; do not load unrelated testing bodies. |

For every row, verify the agent:

- does not run `ssb inspect`, rewrite the pack, or render new output;
- reconciles every canonical rule ID and its selection frontmatter with all of
  its managed-index occurrences before reporting a complete active set;
- reports the active rule IDs;
- loads any rule only when its path scope matches, and loads a contextual rule
  only when every represented lens dimension also matches, treating
  alternatives within a dimension as OR;
- loads conservatively when relevant context cannot be determined;
- does not read irrelevant contextual rule bodies; and
- never reports a mapped verification command as executed or passed without
  separate execution evidence.

Separately verify that an explicit developer request to validate/rerender
edited sources uses reviewed-pack maintenance mode, and that an explicit
post-review ADR request previews and creates only the `Proposed` ADR. Neither
continuation may rerun inspection or rewrite canonical sources.

Acceptance requires 100% resolvable evidence and at least 70% developer “keep” or “edit and keep” judgment among high and very-high candidates.

## Reference adapters

These examples document known skill-exposure mechanisms. They do not change
the shared conformance contract above.

### Codex

Place or link the skill at:

```text
.agents/skills/software-standards-bootstrap/SKILL.md
```

Test both explicit invocation and description-based selection. Do not add
Codex-only `agents/openai.yaml` metadata to the portable contract.

### Claude Code

Expose the same `SKILL.md` content at Claude Code's documented project location:

```text
.claude/skills/software-standards-bootstrap/SKILL.md
```

Do not infer that Claude Code discovers `.agents/skills`. Do not add Claude-only substitutions or frontmatter to the portable source.

### Other hosts

Use the host's documented project installation, discovery, or explicit-import
mechanism. Record that mechanism without adding host-specific fields to the
portable source. Run the same shared generation and progressive-selection
behavior; do not weaken acceptance because invocation syntax or discovery
location differs.

## Result record

Use this table in a release issue or pull request:

| Host/version | Skill exposure method | Fixture/commit | OS | Evidence resolution | Context selection | Active IDs disclosed | Proof claim | Git mutation | Result |
|---|---|---|---|---:|---|---|---|---|---|
| | | | | | | | Mapped only | None required | |

One completed row demonstrates conformance for that host version. Every host
named as behaviorally supported needs its own completed row. Multiple
independent host records support the project's portability claim; ordinary
users do not need to reproduce them before using the tool.

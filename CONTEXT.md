# Software Standards Bootstrap

> Turn evidence from a pinned repository snapshot into a developer-reviewed standards proposal.

## Domain Language

### Primary topic

**Definition**: The single controlled software-engineering concern that best explains a proposed rule's risk or a procedural skill's intended outcome.

**Use when**: Classifying every retained rule and generated Agent Skill for review, projection, and adoption records.

**Avoid**: Treating the topic as a list of tags, deriving it from the file type, or confusing it with proof classification, importance, confidence, or path scope.

**Examples**:

- A rule preserving the repository's supported Go version has `compatibility` as its primary topic.
- A skill that keeps generated shell completions aligned across supported shells has `compatibility` as its primary topic.

**Related terms**: Rule, Agent Skill, classification, importance, confidence, scope

---

### Activation lens

**Definition**: A v2 selection dimension that tells an agent when to load a
rule. A rule is either repository-wide `base` guidance or uses one or more
`language`, `framework`, and `task` lenses.

**Use when**: Routing retained rules progressively from the root `AGENTS.md`
index to canonical rule bodies.

**Avoid**: Treating the primary topic as a loader selector, duplicating one
cross-dimensional rule into several files, or excluding a potentially relevant
rule when context is uncertain.

**Related terms**: Scope, directive, primary topic, progressive disclosure

---

### Directive

**Definition**: The rule's behavioral modality: `always`, `ask-first`, `never`,
or `prefer`.

**Use when**: Distinguishing obligations, approval boundaries, prohibitions,
and defaults without encoding that intent only in prose.

**Avoid**: Inferring importance or proof coverage from the directive.

**Related terms**: Importance, classification, scope

---

### Verification coverage

**Definition**: Whether a cited repository command proves the complete rule
(`full`) or only a bounded part of guidance (`partial`) when that command
passes.

**Use when**: Recording `verification.proves` for a mapped command.

**Avoid**: Treating a mapped command as executed, a command's presence as a
passing result, or a broad build as proof of every behavior.

**Related terms**: Classification, proof gap, deterministic rule

---

## Last Updated

2026-07-27

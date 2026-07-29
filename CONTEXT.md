# Software Standards Bootstrap

> Turn evidence from a pinned repository snapshot into a developer-reviewed
> actionable standards proposal.

## Domain language

### Artifact kind

**Definition**: The candidate's one primary developer-action destination:
semantic rule, verification recipe, Agent Skill, or automation proposal.

**Use when**: Separating implementation conditions, existing commands,
procedural work, and proposed checks.

**Avoid**: Mixing command/check metadata into a semantic rule or treating an
automation proposal as implemented behavior.

---

### Category

**Definition**: The single controlled engineering concern that best describes
an accepted artifact.

**Use when**: Reviewing and projecting every accepted artifact.

**Avoid**: Treating category as activation, artifact kind, confidence, utility,
derivation, or scope.

---

### Activation lens

**Definition**: A selection dimension that tells an agent when to load an
artifact. An artifact is either `base` or uses `language`, `framework`, and
`task` lenses.

**Use when**: Routing active rules, recipes, and Agent Skills from `AGENTS.md`.

**Avoid**: Using category as a loader selector or excluding a potentially
relevant artifact when context is uncertain.

---

### Derivation

**Definition**: Whether an artifact is `extracted` from a repository-maintained
declaration/enforcement or `inferred` from repeated implementation evidence.

**Use when**: Applying the correct exact-evidence threshold.

**Avoid**: Treating derivation, confidence, utility, or an evidence role as
interchangeable.

---

### Verification recipe

**Definition**: An ordered, deliberately invoked sequence of existing commands
with exact `enforces` evidence, an application condition, and expected
successful results.

**Use when**: A developer benefits from running an existing command.

**Avoid**: Treating a recorded command as executed or passed, adding semantic
judgment or branching, or placing the command in a semantic rule.

---

### Automation proposal

**Definition**: A reviewable design for a valuable check that does not exist.

**Use when**: Earlier automatic feedback would add utility beyond existing
instructions and mechanisms.

**Avoid**: Treating the proposal as generated code, active policy, or
ADR-adopted behavior.

## Last updated

2026-07-29

# Structural-pattern workflow

Use this pass to discover repository-specific structural invariants before
routing and scoring candidates. It complements authority, risk, command, and
automation discovery; it does not turn ordinary ecosystem conventions into
artifacts automatically.

## Required review dimensions

Review every dimension against inventory-listed files:

1. **Package and dependency boundaries:** identify layers, ownership
   boundaries, and repeated dependency direction.
2. **Parallel implementation families:** identify sibling generators,
   backends, adapters, providers, serializers, commands, or handlers and their
   paired tests or documentation.
3. **Platform and configuration seams:** identify build tags, operating-system
   files, feature flags, environment adapters, and CI matrices.
4. **Public compatibility surfaces:** identify exported APIs, deprecations,
   aliases, shims, schemas, formats, and migration behavior.
5. **Source, test, and documentation symmetry:** identify repository-specific
   pairings that make behavior reviewable. Do not promote routine colocation.

## Candidate decisions

For each plausible candidate:

- cite exact files and lines establishing the shape or boundary;
- state the future planning, implementation, or verification action and its
  regression risk;
- narrow scope to the supported package, family, seam, or public surface;
- use conditional wording such as "every affected generator" when appropriate;
- classify derivation as extracted or inferred and apply its evidence threshold;
- inspect whether an existing automatic mechanism already handles the condition;
- route remaining value to one semantic rule, verification recipe, Agent Skill,
  or automation proposal; and
- reject descriptions that provide no developer action.

Lack of explicit prose authority is not a rejection reason when three
consistent occurrences across two files establish an inferred invariant.
Conversely, repetition without shared behavior, risk, or developer action is
not an artifact.

## Report output

Complete all five dimensions before writing accepted outputs. Record only
run-wide limitations and accepted-output summaries in
`.software-standards/report.md`.

Do not preserve rejected candidates, reasons, or counts. If a dimension yields
no accepted artifact, do not invent one. Do not impose a fixed count.

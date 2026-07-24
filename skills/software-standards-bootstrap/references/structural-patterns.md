# Structural-pattern workflow

Use this pass to discover repository-specific structural invariants before
scoring candidates. It complements policy, risk, and checker discovery; it does
not turn ordinary language or framework conventions into rules automatically.

## Required review dimensions

Review every dimension against inventory-listed files:

1. **Package and dependency boundaries:** identify modules, packages, layers,
   ownership boundaries, and repeated dependency direction. Look for seams a
   future change could accidentally cross.
2. **Parallel implementation families:** identify sibling generators,
   backends, adapters, providers, serializers, commands, or handlers and their
   paired tests or documentation. Determine which shared behaviors require
   affected family members to remain aligned.
3. **Platform and configuration seams:** identify build tags, operating-system
   files, feature flags, environment-specific adapters, and CI matrix coverage.
   Look for shared code that intentionally delegates through those seams.
4. **Public compatibility surfaces:** identify exported APIs, deprecations,
   aliases, shims, schemas, file formats, and documented migration behavior.
   Prefer a narrow compatibility rule when the evidence supports one surface
   but not a repository-wide promise.
5. **Source, test, and documentation symmetry:** identify repository-specific
   pairings that make behavior reviewable, such as one test family per backend
   or one user document per supported integration. Do not promote routine file
   colocation that is merely an ordinary ecosystem convention.

## Candidate decisions

For each plausible structural candidate:

- cite the exact files and lines that establish the repeated shape or boundary;
- state the future change and regression risk the candidate would govern;
- narrow the scope to the supported package, family, platform seam, or public
  surface instead of discarding the candidate because it is not repository-wide;
- use conditional wording such as "every affected generator" when not every
  family member must change on every edit;
- apply the existing authority threshold or the alternative threshold of three
  consistent occurrences across at least two files; and
- distinguish actionable guidance from architecture description that belongs
  only in the assessment.

Lack of an authoritative prose policy is not, by itself, a rejection reason
when the occurrence threshold is satisfied. Conversely, repetition without a
shared behavior, risk boundary, or developer action is repository context, not
a rule.

## Assessment output

Add a `Structural pattern review` section to
`.software-standards/assessment.md`. Record all five dimensions, the evidence
examined, and one disposition for each plausible candidate:

- emitted as a scored rule;
- represented by a procedural Agent Skill;
- assessment-only with the unmet threshold or unclear boundary; or
- rejected as a generic convention or accidental similarity.

If a dimension yields no candidate, say so. Do not silently omit a completed
dimension, and do not impose a fixed candidate count.

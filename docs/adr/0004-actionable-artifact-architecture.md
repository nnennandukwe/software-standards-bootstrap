# 0004: Adopt actionable artifact architecture

- Status: Accepted
- Date: 2026-07-29

## Context

Putting semantic guidance, existing commands, procedural workflows, and
proposed checks into one proof-oriented rule format blurred distinct developer
actions. It also made command metadata remote from its operational context and
treated rejected candidates as durable output.

The project is pre-release and has no public schema consumer, so a single
cutover is safer than a compatibility layer.

## Decision

Use `.software-standards/report.md` as the accepted artifact index and run
narrative. Support four primary output kinds:

- semantic rules for implementation conditions;
- verification recipes for deliberately invoked existing commands;
- portable Agent Skills for multi-step procedures; and
- automation proposals for valuable checks that do not yet exist.

Keep category, activation, derivation, and exact evidence adjacent to native
artifacts. Store portable-skill provenance in the report. Store confidence,
utility, and relationships in the report for every accepted artifact.

The host agent owns semantic eligibility, naming, routing, confidence, and
utility judgments. Offline Go code owns strict schemas, exact inventory replay,
evidence hashes and thresholds, confidence and utility gates, relationship
integrity, projection, and ADR output.

Reject and discard low-confidence, low-utility, unsupported, and already
fully-automated candidates. Do not persist rejection reasons or counts.

`AGENTS.md` projects rules, recipe links, and Agent Skill indexes but omits
automation proposals. ADRs record rules, recipes, and skills but exclude
automation proposals.

## Consequences

- Commands no longer appear in semantic rules.
- Proposed checks are not mistaken for implemented enforcement.
- Agent Skills are primary outputs instead of rule attachments.
- Empty and automation-only packs validate without creating empty derived
  output.
- The rewritten semantic rule contract has no compatibility parser.
- Validated JSON remains response schema 2 with its unreleased shape changed
  in place.
- Runtime model, network, checker execution, and checker generation remain out
  of scope.

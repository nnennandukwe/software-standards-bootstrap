# Changelog

All notable changes are documented here. The project follows semantic versioning.

## [Unreleased]

### Added

- Added the actionable report, semantic rule, verification recipe, portable
  Agent Skill, and automation proposal contracts.
- Added exact inventory replay, confidence and utility gates, cross-artifact
  relationships, and zero-artifact/automation-only behavior.
- Added the governed `ssb prune` lifecycle-review protocol with pinned local
  capability profiles, explicit provenance, complete rule/skill inventory,
  evidence-backed semantic proposals, digest-bound approval, dry-run-first
  application and recovery, separate rerender/ADR states, and external
  content-addressed verification receipts.

### Changed

- Replaced arbitrary indexed-output caps with measured inventory-v2 candidate
  work budgets: 40,000 files, 128 MiB total, and a fixed 1 MiB per file.
- Incomplete inventory coverage now returns exit `4` unless explicitly allowed
  for diagnostic use, and reports complete candidate/scanned/remaining
  accounting.
- Git blob batching now uses the measured 512-entry, 4 MiB policy.
- Rewrote `ssb.dev/rule/v2` as a semantic-rule contract with local derivation
  and exact evidence, removing pre-release compatibility and command/check
  metadata.
- Changed the managed `AGENTS.md` section into a progressive router with
  base semantic rules, contextual rule and recipe links, Agent Skill indexes,
  and no automation proposals.
- Advanced `ssb validate --format json` to response schema 2 and include the
  normalized pack only for valid results.
- Added existing-pack progressive selection to the portable Agent Skill.
- Advanced the portable skill metadata version to `0.4.0` for actionable
  candidate routing and existing-pack selection.

## [0.1.0] - 2026-07-23

### Added

- Git-safe `inspect` inventory with stable text and JSON output.
- Strict evidence-backed rule-pack validation.
- Drift-detecting bounded `AGENTS.md` rendering.
- Proposed ADR generation from retained artifacts.
- Portable `software-standards-bootstrap` Agent Skill.
- Mandatory structural-pattern discovery and benchmark acceptance coverage.
- Evidence-backed engineering categorization for generated artifacts.
- Cross-platform CI and release provenance configuration.
- Public benchmark evidence for Cobra, Flask, Django, and Next.js across Codex
  and Claude Code consumer workflows.

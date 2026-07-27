# Changelog

All notable changes are documented here. The project follows semantic versioning.

## [Unreleased]

### Changed

- Replaced arbitrary indexed-output caps with measured inventory-v2 candidate
  work budgets: 40,000 files, 128 MiB total, and a fixed 1 MiB per file.
- Incomplete inventory coverage now returns exit `4` unless explicitly allowed
  for diagnostic use, and reports complete candidate/scanned/remaining
  accounting.
- Git blob batching now uses the measured 512-entry, 4 MiB policy.
- Added `ssb.dev/rule/v2` activation lenses, behavioral directives, and
  explicit full/partial proof coverage while retaining v1 read compatibility.
- Changed the managed `AGENTS.md` section into a progressive router with
  scoped base standing orders, contextual rule links, and mapped-command
  disclosure.
- Advanced `ssb validate --format json` to response schema 2 and include the
  normalized pack only for valid results.
- Added existing-pack progressive selection to the portable Agent Skill.
- Advanced the portable skill metadata version to `0.2.0` for the v2
  generation and existing-pack selection contract.

## [0.1.0] - 2026-07-23

### Added

- Git-safe `inspect` inventory with stable text and JSON output.
- Strict evidence-backed rule-pack validation.
- Drift-detecting bounded `AGENTS.md` rendering.
- Proposed ADR generation from retained artifacts.
- Portable `software-standards-bootstrap` Agent Skill.
- Mandatory structural-pattern discovery and benchmark acceptance coverage.
- Required primary-topic metadata for rules and skills, with AGENTS and ADR projection.
- Cross-platform CI and release provenance configuration.
- Public benchmark evidence for Cobra, Flask, Django, and Next.js across Codex
  and Claude Code consumer workflows.

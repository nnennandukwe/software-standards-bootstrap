# Changelog

All notable changes are documented here. The project follows semantic versioning.

## Unreleased

### Added

- A checksum-verifying `install.ps1` for Windows, matching the `install.sh`
  guarantees: architecture and release-tag validation before any download,
  SHA-256 verification before the binary reaches its destination, refusal to
  replace an existing Agent Skill, and no partial binary left behind on
  failure or interruption.

## [0.1.1] - 2026-07-31

### Added

- A checksum-verifying `install.sh` for macOS and Linux, with latest-release
  discovery, exact-version and install-directory options, and explicit
  repository-local Agent Skill installation from the matching release tag.
- An actual-archive release gate that checks all six platform and architecture
  targets for a complete Agent Skill matching the tagged source.
- Source, author, and Apache-2.0 creator attribution in the bundled Agent Skill.

### Fixed

- Complete Agent Skill packaging in release archives. The v0.1.0 archives
  omitted the root `SKILL.md` while including its reference files; v0.1.1
  includes both the entrypoint and every reference file.

### Documentation

- Clarified that retaining an automation proposal preserves a design for
  separate review and does not authorize implementation or activation.

## [0.1.0] - 2026-07-31

First public release. Set the release date in this heading when the signed tag
is created. Earlier pre-release iterations of the rule, inventory, and
projection contracts were never published, so this section describes the
shipping contract in full rather than a delta.

### Added

- Git-safe `inspect` inventory with stable text and JSON output, measured
  inventory-v2 work budgets of 40,000 candidate files, 128 MiB total, and a
  fixed 1 MiB per file, and 512-entry / 4 MiB Git blob batching.
- Exit `4` on incomplete inventory coverage unless explicitly allowed for
  diagnostic use, with complete candidate, scanned, and remaining accounting.
- The `ssb.dev/report/v1` actionable report, `ssb.dev/rule/v2` semantic rule,
  verification recipe, portable Agent Skill, and automation proposal contracts.
- Strict evidence-backed pack validation, exact inventory replay, confidence
  and utility gates, cross-artifact relationships, and zero-artifact and
  automation-only behavior.
- `ssb validate --format json` response schema 2, which includes the normalized
  pack only for valid results.
- Drift-detecting bounded `AGENTS.md` rendering as a progressive router with
  inlined base semantic rules, contextual rule and recipe links, Agent Skill
  indexes, and no automation proposals.
- Proposed ADR generation from retained artifacts, on explicit request only.
- Mandatory structural-pattern discovery and evidence-backed engineering
  categorization for generated artifacts.
- The portable `software-standards-bootstrap` Agent Skill at metadata version
  `0.4.0`, covering actionable candidate routing and existing-pack progressive
  selection.
- The governed `ssb prune` lifecycle-review protocol with pinned local
  capability profiles, explicit provenance, complete rule and skill inventory,
  evidence-backed semantic proposals, digest-bound approval, dry-run-first
  application and recovery, separate rerender and ADR states, and external
  content-addressed verification receipts.
- Cross-platform CI and release provenance configuration.

# 2026-07-29 actionable-artifact benchmark gate

This record separates the fresh offline inventory run from proposal generation,
developer retention, rendering, ADR creation, and release evidence. It does not
upgrade an earlier state into a later one.

## Run identity

- SSB branch: `agent/actionable-artifact-architecture`
- SSB commit: `f5675a6`
- Host: macOS 15.7.3, `arm64`
- SSB and repository-tool network use: none
- Attempted consumer: Claude Code 2.1.220 with `claude-sonnet-5`;
  model-service transport was used
- Repository-code execution: none
- Fixture source: fresh local clones on an attached
  `actionable-benchmark` branch
- Historical result directories: read-only and not reused as acceptance

## Fresh inventory results

All four fixtures were inspected at the exact pins in
[`testdata/benchmarks.yaml`](../../../../testdata/benchmarks.yaml) with schema 2
defaults. Every run completed without truncation.

| Repository | Candidate files | Candidate bytes | Scanned files | Scanned bytes | Indexed files | Indexed bytes | Remaining files | Remaining bytes |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Cobra | 66 | 705,271 | 66 | 705,271 | 65 | 631,792 | 0 | 0 |
| Flask | 235 | 1,814,782 | 235 | 1,814,782 | 230 | 1,474,850 | 0 | 0 |
| Django | 7,001 | 45,506,636 | 7,001 | 45,506,636 | 5,619 | 36,820,618 | 0 | 0 |
| Next.js | 29,073 | 111,110,455 | 29,073 | 111,110,455 | 28,403 | 88,643,646 | 0 | 0 |

Exclusion accounting:

Tree-level exclusions (`oversized`, `secret-like`, `symlink`, `submodule`,
`vendor/generated tree`, and `non-regular`) are removed before candidate
accounting. Scan-level `binary` and `generated` exclusions explain the
candidate-to-indexed delta. The table therefore must not be summed as though
every exclusion were part of the candidate set.

| Repository | Binary | Generated | Oversized | Secret-like | Symlink | Submodule | Vendor/generated tree | Non-regular |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Cobra | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| Flask | 5 | 0 | 0 | 1 | 0 | 0 | 0 | 0 |
| Django | 1,382 | 0 | 0 | 0 | 4 | 0 | 72 | 0 |
| Next.js | 652 | 18 | 21 | 113 | 29 | 0 | 1,060 | 0 |

## State ledger

| State | Evidence |
|---|---|
| Fresh inventory | Complete for all four exact pins; no truncation |
| Proposal generation | Not complete |
| Evidence resolution | Not assessed because no final artifacts were emitted |
| Developer retention | Not performed |
| Rendering | Not performed |
| ADR creation | Not performed |
| Release evidence | Not performed |

A locked-down Claude Code consumer run was attempted on Cobra after inventory.
The first attempts correctly refused unavailable evaluator inputs or binary
execution. A final locally scoped attempt produced no artifact before its
bounded run was terminated. It is recorded as an incomplete consumer run and
counts as neither success nor failure of any artifact.

## Acceptance

Inventory completeness passes. Actionable-artifact acceptance remains open
until fresh final artifacts exist for all four fixtures, all evidence resolves,
and a developer records at least 70% aggregate `keep` or `edit-and-keep`.

# DX audit

**Scope:** Greenfield v0.1 CLI, README, rule schema, Agent Skill, recovery messages, generated artifact paths, and contributor/release documentation.

**Dimensions audited:** Setup and re-entry, CLI ergonomics, output clarity, recovery guidance, artifact visibility, debugging, testability, and docs/help parity.

## P1

No findings.

## P2

No findings.

## P3

No findings.

## Resolved during audit

- Subcommands initially treated `--help` as a usage failure. Each of `inspect`, `validate`, `render`, and `adr` now returns exit `0` with focused flag and default guidance, protected by a CLI behavior test.
- Text inventory paths initially printed control characters literally. Paths are now quoted, so newlines and shell metacharacters remain unambiguous.

## Evidence

- Root and focused help were exercised through the built binary.
- A fresh committed copy of `testdata/fixtures/go-service` completed `ssb inspect` without executing its executable or repository-owned checks.
- CLI tests assert exit codes, stdout/stderr routing, recovery text, JSON shape, and no-write behavior.
- README and Agent Skill command forms are parity-tested against the canonical contract.
- Every successful write reports the exact artifact path and next review gate.

## Residual release risks

- A GitHub release has not yet been published and installed from its archive.
- Public benchmark and Codex/Claude acceptance records require external consumer runs at release time.
- Windows and Linux binaries are cross-built locally and configured in CI; execution on hosted Windows/Linux runners begins after the repository is pushed.

Summary: 0 open findings.

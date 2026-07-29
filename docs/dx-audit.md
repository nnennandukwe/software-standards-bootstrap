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
- Governed-prune recovery initially assumed every crash-left lock had an
  application journal. `recover --clear-stale-lock` now handles review-owned
  locks independently and names the exact review-scoped command.
- A fully rejected review initially looked indefinitely incomplete. Status and
  apply output now expose `no-changes-approved` as a terminal reviewed outcome
  that does not manufacture application or verification events.
- Dry-run output now prints the canonical application-plan digest. The same
  digest is recorded by an explicitly approved write, so a reviewer can bind
  the preview to the later mutation.
- Dirty-worktree guidance from prune now names `ssb prune inspect`, and
  structured unable-to-determine actions identify the exact artifact and
  missing evidence category.
- Ignored governed files now block inspection instead of disappearing from the
  inventory. Skill provenance reports `unknown` for partial bundles and
  `mixed` for complete bundles with different declared origins.
- Portable-path diagnostics now reject Windows-reserved names and invalid
  filename components consistently on POSIX and Windows, including tracked
  baseline skill support and case-fold collisions.
- A rerender or ADR event that was durably recorded is no longer rolled back
  when only lock cleanup fails. The artifact and event remain aligned, and the
  error names `recover --clear-stale-lock`.
- Review-aware render and ADR flows validate adopted rules against their
  recorded historical baselines, while ordinary pack commands retain the
  current-HEAD contract. Missing baselines fail with fetch-or-review guidance.
- Windows rejects approved changes involving an executable prestate during
  application planning, before a dry run can imply the change is writable.
- Rerender verification now binds the complete `AGENTS.md` output rather than
  comparing the whole file to its managed-section digest. It also rejects
  symlinked output and binds optional render events on skill-only reviews.
- Review roots and durable capability/provenance snapshots now reject
  symlinked directory components before locks, recovery, event writes, or
  snapshot reads.
- Historical evidence commits must be ancestors of current history, and Git
  operational failures remain distinguishable from ancestry rejection.
- Relative capability-profile and receipt paths now resolve from the process
  working directory instead of being misreported as bundle escapes.
- Artifact removal previews the paired `report.md` index update in the same
  dry-run plan. Zero-rule results are valid; canonical ID/path replacement is
  blocked with guidance to create a new actionable pack because lifecycle
  review cannot invent fresh manifest provenance, confidence, or utility.
- Review publication, event replacement, transition and mutation locks, and
  application journals now use repository-root-anchored filesystem operations;
  swap tests prove that neither an external symlink, an in-repository sibling
  symlink, nor a real-directory replacement can enter a locked transition.
- Atomic event replacement reports both the replacement failure and any
  temporary-file cleanup failure, including the exact residual path and safe
  manual-cleanup guidance.
- Skill-only reviews may omit rerendering, but cannot add an unbound render
  after verification has completed.
- `prune validate` now checks replacement rule and skill contracts plus the
  proposed resulting graph. Approval repeats the graph preflight for the exact
  accepted actions and records no event when it fails.
- Validate, approval, and apply now share one candidate-operation builder, with
  parity tests for candidate digest drift across all three entry points.

## Evidence

- Root and focused help were exercised through the built binary.
- Every `ssb prune` subcommand's focused help returned exit `0` from the built
  binary. A disposable committed repository completed `prune inspect`; text
  status then reported the seven lifecycle states separately.
- A fresh committed copy of `testdata/fixtures/go-service` completed `ssb inspect` without executing its executable or repository-owned checks.
- CLI tests assert exit codes, stdout/stderr routing, recovery text, JSON shape,
  plan identity, no-change behavior, and no-write behavior.
- README and Agent Skill command forms are parity-tested against the canonical contract.
- Every successful write reports the exact artifact path and next review gate.
- The complete Go test suite, race suite, vet, native build, Windows and Linux
  cross-builds, vulnerability scan, and Agent Skills format validation passed
  on 2026-07-29. Statement coverage was 72.4% overall and 71.4% for
  `internal/prune`.

## Residual release risks

- A GitHub release has not yet been published and installed from its archive.
- Public benchmark and Codex/Claude acceptance records require external consumer runs at release time.
- Windows and Linux binaries are cross-built locally and configured in CI; execution on hosted Windows/Linux runners begins after the repository is pushed.

Summary: 0 open findings.

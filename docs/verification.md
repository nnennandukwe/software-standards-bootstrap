# v0.1 verification record

Verified locally on 2026-07-23 with Go 1.26.5 and Git 2.39.5.

## Automated gates

| Gate | Result |
|---|---|
| `go test ./...` | Pass |
| `go test -race ./...` | Pass |
| `go vet ./...` | Pass |
| `go build ./cmd/ssb` | Pass |
| `go tool govulncheck ./...` | Pass — no vulnerabilities found |
| GoReleaser v2.17.0 configuration check | Pass |
| GitHub workflow actionlint v1.7.12 | Pass |
| Agent Skills reference validation | Pass |

The Agent Skill validation command used was:

```bash
uvx --from skills-ref agentskills validate skills/software-standards-bootstrap
```

## Cross-builds

CGO-free binaries compiled successfully for:

- Darwin amd64 and arm64;
- Linux amd64 and arm64; and
- Windows amd64 and arm64.

The output formats were Mach-O, statically linked ELF, and PE32+ respectively.

## Failure-path guarantees

Tests verify:

- dirty, staged, detached, unborn, non-Git, changed-`HEAD`, and existing-pack inspection failures;
- untracked-only and nested-worktree starts;
- literal spaces, Unicode, newlines, and shell metacharacters without shell execution;
- binary, oversized, generated/vendor, secret-like, symlink, and submodule exclusion;
- stable inventory plus explicit truncation;
- strict schema, score, evidence, classification, skill-reference, and baseline failures;
- new/existing `AGENTS.md`, direct drift, malformed/duplicate markers, symlink targets, dry runs, source edits, and deletions;
- staged-render write failure without replacement or temporary-file residue;
- default, existing, ambiguous, traversal, symlink, submodule, collision-safe, and dry-run ADR behavior;
- ADR write failure without partial output; and
- no mutation from `inspect`, `validate`, dry runs, or failed commands.

## Release-only acceptance still required

These gates cannot be truthfully completed in the local greenfield repository:

- publish and install verified GitHub release assets;
- run CI on hosted macOS, Linux, and Windows;
- complete the four pinned public benchmark workflows;
- record Codex and Claude Code behavior at exact versions;
- meet the 70% high-band retention threshold; and
- confirm 100% evidence resolution in those external runs.

The release runbook blocks `v0.1.0` until those records exist.

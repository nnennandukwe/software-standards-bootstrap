# v0.1.0 verification record

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

## Hosted CI

[CI run 30057004619](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/30057004619)
passed on source commit
`a426d419cb129541f4bf69497ed7232eff926c46`:

| Hosted gate | Result |
|---|---|
| macOS / Go 1.26.5 | Pass |
| Ubuntu / Go 1.26.5 | Pass |
| Windows / Go 1.26.5 | Pass |
| Race detector | Pass |
| `go vet` | Pass |
| `govulncheck` | Pass |

## Failure-path guarantees

Tests verify:

- dirty, staged, detached, unborn, non-Git, changed-`HEAD`, and existing-pack inspection failures;
- untracked-only and nested-worktree starts;
- literal spaces, Unicode, newlines, and shell metacharacters without shell execution;
- binary, oversized, generated/vendor, secret-like, symlink, and submodule exclusion;
- stable inventory-v2 candidate accounting, exact resource boundaries, and
  fail-closed incomplete coverage;
- strict v1/v2 schema, score, evidence, primary-topic, classification,
  activation-lens, directive, proof-coverage, skill-reference, and baseline
  failures;
- v1 compatibility and v2 progressive projection: base standing orders are
  inline, contextual rule bodies remain in canonical files, commands are
  deduplicated, and proof is labeled as mapped rather than executed;
- primary-topic projection for rules in `AGENTS.md` and lenses, directive,
  proof coverage, and primary topics for rules and skills in the ADR;
- valid JSON interchange output containing a normalized pack, with invalid
  output limited to diagnostics;
- new/existing `AGENTS.md`, direct drift, malformed/duplicate markers, symlink targets, dry runs, source edits, and deletions;
- staged-render write failure without replacement or temporary-file residue;
- default, existing, ambiguous, traversal, symlink, submodule, collision-safe, and dry-run ADR behavior;
- ADR write failure without partial output; and
- no mutation from `inspect`, `validate`, dry runs, or failed commands.

## Public benchmark status

The durable proposal records are in
[`docs/benchmarks/results/2026-07-23/`](benchmarks/results/2026-07-23/README.md).

| Consumer/version | Fixture coverage | Evidence resolution | Developer retention | End-to-end result |
|---|---:|---:|---:|---|
| Codex desktop 26.715.71837 (build 5702), `gpt-5.6-sol` | 4/4 proposals | 100% | Pending | Review gate |
| Claude Code 2.1.191, `claude-sonnet-4-6` | 4/4 proposals | 100% | Pending | Review gate |

All recorded proposals include the five-dimension structural-pattern pass.
Django and Next.js disclose inventory truncation. No evaluated repository code
or cited verification command was executed, and SSB performed no Git mutation.
Evaluator review found and corrected inaccurate or incomplete inventory
summaries in the initial Claude assessments before they became release
evidence. Next.js also exercised the validation recovery path for an
unsupported relationship to a target-owned skill; no pre-existing target file
was edited.

These proposal records used `ssb-inventory-v1`. They are historical pre-change
evidence, not acceptance for inventory v2. The inventory-v2 resource harness
completes all four pins on macOS arm64; Linux resource-envelope evidence and
fresh Codex/Claude proposal records remain release gates.

## Release controls

| Control | Result |
|---|---|
| GitHub immutable releases | Enabled and verified before release creation |
| `CHANGELOG.md` v0.1.0 notes | Prepared |
| Signed `v0.1.0` tag | Pending |
| Six release archives | Pending tag workflow |
| Checksums and SPDX SBOMs | Pending tag workflow |
| Build-provenance and SBOM attestations | Pending tag workflow |
| Clean archive installation | Pending published assets |

## Remaining release gates

The release runbook still blocks `v0.1.0` until:

- the benchmark and consumer records are reviewed and land on `main`;
- inventory-v2 Linux resource-envelope evidence is recorded;
- fresh inventory-v2 Codex and Claude proposal records replace the historical
  v1 release evidence;
- a developer records keep, edit-and-keep, defer, or reject decisions;
- at least 70% of high and very-high candidates are kept or edit-and-kept;
- the edit/delete/rerender and explicitly requested ADR behavior is verified
  for both consumers;
- a configured signing identity creates the signed tag; and
- published archives, checksums, SBOMs, attestations, and a clean installation
  are independently verified.

No pending item is represented as complete.

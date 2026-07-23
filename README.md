# Software Standards Bootstrap

Software Standards Bootstrap (`ssb`) is an Apache-2.0 command-line tool and portable Agent Skill for turning evidence from a real Git repository into a reviewable standards proposal.

The host agent performs semantic analysis. The `ssb` binary supplies deterministic safety, inventory, validation, rendering, and ADR contracts. It never calls a model, sends telemetry, makes network requests, executes repository code, or performs Git mutations.

## What v0.1 produces

```text
.software-standards/
├── assessment.md
└── rules/
    └── <rule-id>.md
.agents/skills/<skill-name>/SKILL.md
AGENTS.md
docs/adr/NNNN-agentic-rules.md
```

The rule files and skills are editable proposal sources. Root `AGENTS.md` is a lossy managed projection. The ADR is generated only after a developer has reviewed, edited, or deleted the proposed files, and it remains `Proposed` until the developer-created pull request is merged.

## Requirements

- Git 2.39 or newer
- A commit-backed, attached `HEAD`
- Go 1.26.5 only when building from source

`ssb inspect` requires no tracked or staged changes. Untracked-only files are allowed. The other commands operate on the uncommitted proposal files created by the Agent Skill.

## Install

### Verified release archive

After a release is published, download the archive for your operating system and the accompanying `checksums.txt` file from GitHub Releases. Verify both the checksum and GitHub artifact attestation before installing:

```bash
shasum -a 256 --check checksums.txt
gh attestation verify <archive> --repo nnennandukwe/software-standards-bootstrap
```

Windows users can verify SHA-256 with `Get-FileHash` and compare it with `checksums.txt`. Release archives cover macOS, Linux, and Windows on amd64 and arm64 where Go supports the target.

### Build from source

```bash
go install github.com/nnennandukwe/software-standards-bootstrap/cmd/ssb@v0.1.0
```

## CLI

```text
ssb inspect  [--repo PATH] [--format text|json]
ssb validate [--repo PATH] [--format text|json]
ssb render   [--repo PATH] [--dry-run]
ssb adr      [--repo PATH] [--adr-dir PATH] [--dry-run]
```

Exit codes:

- `0`: success
- `1`: rule-pack validation failure
- `2`: usage or repository precondition failure
- `3`: unexpected internal failure

`inspect` and `validate` are read-only. `render` may change only the bounded Software Standards Bootstrap section in root `AGENTS.md`. `adr` exclusively creates one new record and refuses overwrite or path escape.

## Agent Skill workflow

Install or expose [the portable skill](skills/software-standards-bootstrap/SKILL.md) to a compatible host, then ask:

```text
Use the software-standards-bootstrap skill to analyze this repository
and generate an evidence-backed rules pack.
```

The skill:

1. runs `ssb inspect`;
2. performs targeted semantic reads without executing repository code;
3. writes an assessment, dynamically ranked rule files, and genuinely procedural skills;
4. runs `ssb validate` and `ssb render`;
5. lists every changed and untracked path; and
6. waits for the developer to edit or delete proposal sources before an ADR is requested.

The workflow has no fixed candidate count. Candidates below 25 remain assessment-only. A rule requires one authoritative source or three consistent occurrences across two files.

Codex discovers repository skills under `.agents/skills`. Claude Code uses a different documented project discovery path, so the same portable skill content must be exposed through `.claude/skills` for its behavioral smoke test. Other consumers are format-compatible only until tested.

## Evidence and proof boundary

Every rule records:

- schema and stable ID;
- repository-relative scopes;
- `guidance` or `deterministic` classification;
- `ssb-score-v1` factors, total, and importance band;
- confidence and baseline commit;
- exact evidence line ranges and SHA-256 excerpt hashes;
- an existing verification command with its repository source, or an explicit proof gap; and
- related Agent Skill IDs when the standard is procedural.

A rule is `deterministic` only when it cites an existing repository check. `ssb` records that mapping but does not run the command or claim it passed.

See [the rule format](docs/rule-format.md), [architecture](docs/architecture.md).

## Safety model

- Inventory reads the pinned Git tree and blob objects, not worktree files.
- Only tracked regular files are considered.
- Binaries, oversized files, secret-like paths, generated/vendor trees, symlinks, and submodules are excluded.
- Paths are passed directly to Git without a shell and are NUL-delimited where applicable.
- `HEAD` and tracked state are rechecked before an inventory is returned.
- Resource truncation is disclosed in text and JSON output.
- Existing packs, malformed markers, projection drift, target collisions, symlink escapes, submodule targets, and ADR ambiguity fail closed.
- Writes are bounded, staged locally, and left uncommitted.

## Non-goals

v0.1 does not provide a generic rules catalog, configuration synchronization, tool-specific rule projections, checker generation, hosted services, direct model APIs, telemetry, hooks, automatic refresh, or downstream product activation.

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/ssb
go tool govulncheck ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution and behavior-test expectations.

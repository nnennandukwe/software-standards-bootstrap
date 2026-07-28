# Software Standards Bootstrap

Software Standards Bootstrap (ssb) is a CLI tool and agent skill for developing repo rules, agent skills, and AGENTS.md inside greenfield repos to make your repos more agent friendly.
Agentifying your repos in this manner helps developers get even better use out of AI tools that can extract well documented patterns in your code quality and conventions per repo, formatted into markdown files that AI agents can easily ingest and interpret.

When you run this tool from inside your coding agent, it will get instructions to perform semantic analysis of your repo. `ssb` is solely meant to provide deterministic safety, inventory, validation, rendering, and ADR contracts for the results of the repo analysis. Your coding agent will analyze your repo and extract conventions your code follows based on programming language, frameworks, and your code quality practices. From there, it'll propose candidates for official coding rules, AGENTS.md content, and agent skills to be added to your repo.

As a developer, you then have the ability to approve, edit, or reject the creation of the agent-native artifacts it generates. And finally, you'll have the option to create an official ADR (architecture decision record) for the addition of these artifacts into your repo.

(`ssb` never calls a model, sends telemetry, makes network requests, executes repository code, or performs Git mutations. We leave that to your coding agent.)

## Scaffolding that `ssb` generates

```text
.software-standards/
├── assessment.md
└── rules/
    └── <rule-id>.md
.agents/skills/<skill-name>/SKILL.md
AGENTS.md
docs/adr/NNNN-agentic-rules.md
```

The rule files and skills are editable proposals for updates to your codebase.
`AGENTS.md` is a managed router file that links generated coding
rules by language, framework, and coding task. An ADR is generated only after a
developer has reviewed, edited, or deleted the proposed files, and it remains
`Proposed` until the developer-created pull request is merged.

## Requirements

- Git 2.39 or newer
- A commit-backed, attached `HEAD`
- Go 1.26.5 **only when building from source**

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
ssb inspect  [--repo PATH] [--format text|json] [resource limits]
ssb validate [--repo PATH] [--format text|json]
ssb render   [--repo PATH] [--dry-run]
ssb adr      [--repo PATH] [--adr-dir PATH] [--dry-run]
```

`inspect` accepts `--max-candidate-files N` and
`--max-candidate-bytes N`. The defaults are 40,000 candidate files and
128 MiB of candidate content. `--allow-partial` changes incomplete coverage
from a blocked result into an explicitly diagnostic success; it does not make
the inventory complete and must not be used to generate a proposal.

Exit codes:

- `0`: success
- `1`: rule-pack validation failure
- `2`: usage or repository precondition failure
- `3`: unexpected internal failure
- `4`: inventory coverage incomplete

`inspect` and `validate` are read-only. Valid JSON from `validate` includes the
normalized pack as a local interchange envelope; invalid output omits the pack
and reports diagnostics. `render` may change only the bounded Software
Standards Bootstrap section in root `AGENTS.md`. `adr` exclusively creates one
new record and refuses overwrite or path escape.

## Agent Skill workflow

Install or expose [the portable skill](skills/software-standards-bootstrap/SKILL.md) to a compatible host, then ask:

```text
Use the software-standards-bootstrap skill to analyze this repository
and generate an evidence-backed rules pack.
```

When no pack exists, the skill:

1. runs `ssb inspect`;
2. performs targeted authority, risk, and structural-pattern review without executing repository code;
3. writes an assessment, dynamically ranked and topically classified rule files, and genuinely procedural skills;
4. runs `ssb validate` and `ssb render`;
5. lists every changed and untracked path; and
6. waits for the developer to edit or delete proposal sources before an ADR is requested.

When a pack already exists, the skill does not reinspect, validate, render, or
rewrite it. It reconciles the managed router against every canonical rule's
selection frontmatter, selects base rules whose scopes match the affected
paths plus contextual rules matching scope and each represented language,
framework, and task dimension, reports the active rule IDs, and treats
uncertain context conservatively by loading potentially relevant rules.

That prohibition applies to rule consumption, not the documented review
continuation. An explicit request to validate or rerender developer-edited
sources uses a bounded maintenance mode, and an explicit post-review ADR
request previews and creates only the `Proposed` record. Neither mode
reinspects or rewrites canonical sources.

The workflow has no fixed candidate count. Candidates below 25 remain assessment-only. A rule requires one authoritative source or three consistent occurrences across two files.

The structural-pattern review explicitly covers package and dependency
boundaries, parallel implementation families, platform and configuration seams,
public compatibility surfaces, and source/test/documentation symmetry. The
assessment records a disposition for each plausible candidate instead of
silently dropping patterns that lack a prose policy or repository-wide scope.

Codex discovers repository skills under `.agents/skills`. Claude Code uses a different documented project discovery path, so the same portable skill content must be exposed through `.claude/skills` for its behavioral smoke test. Other consumers are format-compatible only until tested.

## Evidence and proof boundary

Every rule records:

- schema and stable ID;
- exactly one primary topic from the controlled software-engineering taxonomy;
- one base lens or contextual language, framework, and task lenses;
- an `always`, `ask-first`, `never`, or `prefer` directive;
- repository-relative scopes;
- `guidance` or `deterministic` classification;
- `ssb-score-v1` factors, total, and importance band;
- confidence and baseline commit;
- exact evidence line ranges and SHA-256 excerpt hashes;
- an existing verification command with its repository source, coverage, and
  bounded proved property, or an explicit proof gap; and
- related Agent Skill IDs when the standard is procedural.

Referenced skills carry the same required primary-topic metadata, based on the workflow's intended engineering outcome. Generated `AGENTS.md` guidance and the Proposed ADR expose these topics so reviewers can see whether the retained standards primarily concern correctness, compatibility, maintainability, performance, security, or another supported concern. Prefer a narrow topic; `quality` is the fallback only when no narrower topic fits.

New proposals use `ssb.dev/rule/v2`; existing v1 packs remain valid and
renderable. A v2 rule is `deterministic` only when a cited repository check
fully covers the rule and the source records the exact bounded property proved
when that command passes. Guidance may map partial proof or record a proof gap.
`ssb` records that mapping but does not run the command or claim it passed.

The v2 metadata is suitable for later catalog ingestion, but stable rule IDs
remain repository-local. A separate catalog must assign its own namespace,
version, import review, and lifecycle.

See [the rule format](docs/rule-format.md), [topic taxonomy](skills/software-standards-bootstrap/references/topics.md), [architecture](docs/architecture.md).

## Safety model

- Inventory reads the pinned Git tree and blob objects, not worktree files.
- Only tracked regular files are considered.
- Binaries, oversized files, secret-like paths, generated/vendor trees, symlinks, and submodules are excluded.
- Paths are passed directly to Git without a shell and are NUL-delimited where applicable.
- `HEAD` and tracked state are rechecked before an inventory is returned.
- Candidate file and byte budgets are applied before blob reads.
- Incomplete coverage is disclosed in text and JSON output and returns exit
  `4` unless the caller explicitly requests diagnostic partial output.
- Existing packs, malformed markers, projection drift, target collisions, symlink escapes, submodule targets, and ADR ambiguity fail closed.
- Writes are bounded, staged locally, and left uncommitted.

## Non-goals

`ssb` does not provide a generic rules catalog, catalog import, configuration
synchronization, tool-specific rule projections, checker generation, hosted
services, direct model APIs, telemetry, hooks, automatic refresh, or downstream
product activation.

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/ssb
go tool govulncheck ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution and behavior-test expectations.

# Architecture and trust boundaries

## Design objective

Software Standards Bootstrap turns a pinned repository snapshot into a Git-reviewable proposal. The semantic and deterministic responsibilities are deliberately separate.

```text
host agent semantic reads
          │
          ▼
assessment + rule/skill source files
          │
          ├── ssb validate ── exact evidence, score, proof, schema checks
          ├── ssb render ──── bounded AGENTS.md projection
          └── ssb adr ─────── Proposed record from retained sources
```

Git is the selection and approval surface. The CLI does not stage, commit, branch, push, open a pull request, or activate a rule in another system.

## Modules

### Workspace

The workspace module resolves one non-bare worktree, requires Git 2.39 or newer, checks attached and commit-backed `HEAD`, and owns literal Git invocation. Inspection additionally requires no tracked or staged changes and refuses an existing `.software-standards` pack.

All Git subprocess arguments bypass a shell. `GIT_OPTIONAL_LOCKS=0`, `GIT_LITERAL_PATHSPECS=1`, and `LC_ALL=C` keep read behavior explicit and machine-stable.

### Inventory

The inventory module enumerates `HEAD` with `git ls-tree -r -z -l --full-tree` and reads accepted blob object IDs with `git cat-file`. It accepts only `100644` and `100755` blobs.

The default bounds are:

- 20,000 indexed files;
- 25 MiB total indexed text; and
- 1 MiB per file.

Reaching a file-count or total-byte limit marks the report as truncated. Oversized files are counted as exclusions. The inventory contains no timestamps or absolute paths, so output remains stable for the same commit and limits.

Before success, the module rechecks attached `HEAD`, the exact commit, tracked/staged status, and the absence of a pre-existing pack.

### Rule pack

The rule-pack module parses strict YAML with known-field and unique-key validation. It owns:

- schema and ID validation;
- score factor bounds, arithmetic, and importance bands;
- candidate evidence thresholds;
- exact line-range hashing against baseline blobs;
- the same binary, size, secret, generated/vendor, symlink, and submodule eligibility boundary used by inspection;
- required primary-topic validation for rules and referenced skills;
- classification and existing-proof mapping;
- related Agent Skill path and frontmatter validation; and
- symlink and traversal rejection.

It never interprets whether a candidate is a good engineering standard. That remains host-agent judgment reviewed by a developer.

### Renderer

The renderer sorts rules by stable ID and creates one marked root `AGENTS.md` section. It preserves every pre-existing byte outside the section.

The section stores:

- a digest of the current baseline and rule sources; and
- a self-verifying digest over the recorded source digest and generated body.

A source edit leaves the old section internally valid and allows deterministic replacement. A direct section edit breaks the self-digest and is reported as drift. Missing, reversed, or duplicate markers are not repaired automatically.

### ADR

The ADR module preserves one existing convention among `docs/adr`, `docs/adrs`, `adr`, or `adrs`; otherwise it defaults to `docs/adr`. Multiple conventions require `--adr-dir`.

It rejects paths outside the repository, symlink components, and submodule prefixes. The next numeric filename is created with exclusive-create semantics. Existing files are never replaced. Content includes only rules and referenced skills that survive developer review, exposes each artifact's primary topic, and always has `Proposed` status.

## Canonical versus derived state

| Artifact | Role | Editable | Proof |
|---|---|---:|---:|
| `.software-standards/assessment.md` | Repository context and discarded candidates | Yes | No |
| `.software-standards/rules/*.md` | Canonical proposed rule sources | Yes | Evidence mapping only |
| `.agents/skills/*/SKILL.md` | Canonical proposed procedural workflows | Yes | No |
| Root `AGENTS.md` managed section | Derived guidance projection | No | No |
| Proposed ADR | Adoption record from retained sources | New record only | No |
| Existing repository checker | Repository-owned deterministic mechanism | Outside ssb | Only when run elsewhere |

## No network contract

The runtime imports only the Go standard library and `go.yaml.in/yaml/v4`. No package opens sockets or performs update checks. Network access in release workflows and optional public benchmark evaluation is outside the runtime.

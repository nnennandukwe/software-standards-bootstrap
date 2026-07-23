# Competitive and Format Research

Status: implementation input for Software Standards Bootstrap v0.1
Researched: 2026-07-23
Method: primary sources only (official specifications, product documentation, and project repositories)

## Executive conclusion

Software Standards Bootstrap should not compete as another rules catalog or cross-tool configuration synchronizer. Existing projects already curate rules, select rules by stack or prompt, and fan a shared source into tool-specific files. The defensible v0.1 boundary is repository-specific discovery: inspect a pinned Git snapshot, attach exact evidence and transparent scores, separate declarative rules from procedural Agent Skills, map rules to checks that already exist, let developers select the result through ordinary Git changes, and record the retained decisions in an ADR.

Two portability claims must remain separate:

1. A `SKILL.md` can conform to the open Agent Skills file format.
2. A particular agent will discover, invoke, and interpret that skill from a particular path.

The first is specified; the second is product-specific and must be tested.

## Agent Skills: open format and compatibility boundary

### Portable core

The [Agent Skills specification](https://agentskills.io/specification) defines a skill as a directory containing a required `SKILL.md`, with optional `scripts/`, `references/`, and `assets/`. `SKILL.md` is YAML frontmatter followed by Markdown instructions.

For the portable v0.1 skill:

- `name` and `description` are required by the specification.
- `name` must be 1-64 lowercase alphanumeric or hyphen characters, cannot begin or end with a hyphen or contain consecutive hyphens, and must match the parent directory.
- `description` is 1-1024 characters and should describe both behavior and when to use it.
- `license`, `compatibility`, and string-to-string `metadata` are optional.
- The Markdown body has no required section schema.
- `allowed-tools` is explicitly experimental and support may vary across implementations.
- References should be relative to the skill root; the specification recommends progressive disclosure rather than placing all supporting material in `SKILL.md`.
- The reference package supplies the `agentskills validate <skill-directory>` format validator (for example, `uvx --from skills-ref agentskills validate ...`).

The official [Agent Skills quickstart](https://agentskills.io/skill-creation/quickstart) calls the format open, uses `.agents/skills/` in its VS Code example, and says the same skill works in compatible agents including Claude Code and OpenAI Codex. That is evidence for shared content format, not proof that every product scans the same repository path or implements every optional field identically.

The specification's [client implementation guidance](https://agentskills.io/client-implementation/adding-skills-support) makes that boundary explicit: the specification defines what is inside a skill, while installation/discovery locations, activation, tool plumbing, trust, and permissions belong to the client. It describes `.agents/skills/` as a cross-client convention rather than a required specification path. Project-loaded skills are untrusted repository input and must remain subject to the host's normal permission model.

### Consumer-specific behavior

| Consumer | Primary-source behavior | Compatibility implication |
|---|---|---|
| OpenAI Codex | Codex says its skills build on the open standard, discovers repository skills from `.agents/skills` between the current directory and repository root, supports explicit `$skill`/`/skills` invocation and implicit description matching, and adds optional Codex-only `agents/openai.yaml` metadata. See [Build skills](https://developers.openai.com/codex/build-skills). | `.agents/skills/<name>/SKILL.md` is a documented Codex project location. `agents/openai.yaml` must not be presented as part of the cross-agent specification. |
| Claude Code | Claude Code documents project skills at `.claude/skills/<name>/SKILL.md`, personal skills at `~/.claude/skills/`, and plugin skills under a plugin's `skills/`. It also supports implementation-specific fields and substitutions such as `disable-model-invocation`, `context`, `$ARGUMENTS`, and `${CLAUDE_SKILL_DIR}`. See [Extend Claude with skills](https://code.claude.com/docs/en/skills). | The same spec-conforming `SKILL.md` content may be usable, but official Claude Code documentation does not establish `.agents/skills/` as an auto-discovery path. A Claude smoke test must install or expose the skill through a documented Claude location and must not rely on Claude-only frontmatter for the portable contract. |

This project should therefore:

- generate the standards skill with only the specification's portable core fields;
- put repository output in `.agents/skills/` as the documented Codex/open-format location;
- describe Claude Code as **file-format compatible and behavior-tested through its documented skill location**, not as proven to auto-discover `.agents/skills/`;
- test invocation, changed-path reporting, validation failure, and the “leave changes uncommitted” instruction independently in Codex and Claude Code;
- describe other consumers as specification-compatible but unverified until a product-specific smoke test exists.

## `AGENTS.md`: format and ecosystem

The official [AGENTS.md site](https://agents.md/) describes `AGENTS.md` as a simple open format and a README-like place for coding-agent instructions. Its relevant contracts are deliberately small:

- it is standard Markdown with no required fields or headings;
- a root file supplies repository guidance;
- nested files can supply subproject guidance, and the closest file takes precedence when instructions conflict;
- explicit user prompts override file guidance;
- the ecosystem page lists Codex, Jules, Factory, Aider, goose, OpenCode, VS Code, Cursor, Gemini CLI, GitHub Copilot coding agent, and other consumers;
- the format is stewarded by the Agentic AI Foundation under the Linux Foundation.

Consumer discovery details still vary. Codex's [AGENTS.md documentation](https://developers.openai.com/codex/agent-configuration/agents-md) constructs a chain from global guidance, then project root, then directories down to the current working directory; closer files appear later and override broader guidance. Codex also has product-specific `AGENTS.override.md`, fallback-name, and combined-size behavior. Those are Codex contracts, not fields in the open Markdown format.

Implications for the renderer:

- Treat root `AGENTS.md` as a lossy guidance projection, not the canonical rule store or proof mechanism.
- Preserve every byte outside one unambiguous managed section.
- Reject duplicate or malformed markers rather than guessing at document structure.
- Render concise human-readable Markdown; there is no portable frontmatter or machine schema to populate.
- Do not claim that a root managed section overrides nested `AGENTS.md`.
- Do not use `AGENTS.md` rendering as proof of Claude Code compatibility; Claude Code is not listed among the supported agents on the current AGENTS.md ecosystem page, and its official project-memory documentation is a separate product contract.

## Comparable projects and differentiation

Repository heads below were resolved on 2026-07-23 and linked to immutable commits.

| Project | What the primary source shows | Boundary relative to this project |
|---|---|---|
| [PaulDuvall/centralized-rules at `151b249`](https://github.com/PaulDuvall/centralized-rules/blob/151b2499708a0019a0e621738368c87797049477/README.md) | A curated base/language/framework/cloud rules repository with progressive disclosure, project-marker and prompt-keyword detection, multi-tool support, install/update scripts, local overrides, hooks, and pre-commit quality gates. | Strong catalog, selection, installation, and synchronization precedent. Software Standards Bootstrap must not ship another generic catalog or hook-based updater. It differs by deriving candidates from exact target-repository evidence, scoring them transparently, recording baseline and excerpt hashes, mapping existing proof without running it, and requiring Git review before adoption. |
| [dyoshikawa/rulesync at `d102f38`](https://github.com/dyoshikawa/rulesync/blob/d102f38b3600c055dd863c395ebb84f46540eb9e/README.md) | A CLI with a `.rulesync` source of truth that imports, converts, and generates rules, commands, MCP configuration, subagents, and skills for many tools and open formats. | It solves cross-tool configuration generation and conversion. Software Standards Bootstrap should emit only its repo-native rule pack, portable skills, root managed section, and ADR; tool-specific fan-out is out of scope. |
| [intellectronica/ruler at `a53cfae`](https://github.com/intellectronica/ruler/blob/a53cfae1fa26ee8d4f3b16796f2779f69eca57c7/README.md) | A `.ruler/` source of truth that distributes instructions to agent-specific files, supports nested rule loading, propagates MCP settings, and automates generated-file ignore behavior. Its compatibility table also demonstrates that skill locations vary by consumer. | It solves centralized authoring and distribution. Software Standards Bootstrap should neither centralize organization policy nor keep outputs synchronized; it should make a one-time, evidence-backed proposal in the analyzed repository and leave lifecycle control to normal Git review. |

The differentiation claim should remain narrow:

- **Input:** a real repository snapshot, not a generic standards library.
- **Evidence:** path, location, baseline commit, excerpt hash, confidence, and disclosed truncation.
- **Judgment:** a versioned score with visible factors, not opaque “best practice” selection.
- **Classification:** repository context stays in the assessment, durable guidance becomes a rule, and genuinely procedural work becomes an Agent Skill.
- **Proof:** cite an existing command or record an explicit proof gap; do not invent or claim execution of a checker.
- **Selection:** developers edit or delete source files and inspect the Git diff.
- **Adoption record:** generate a `Proposed` ADR from the files that survive review.
- **No synchronization claim:** no hooks, refresh daemon, central catalog, model API, staged changes, commits, branches, pushes, PRs, or downstream activation.

## Git behavior the implementation can rely on

### Resolve and pin the repository

- Resolve the repository root with `git rev-parse --show-toplevel`.
- Require `git rev-parse --verify --end-of-options 'HEAD^{commit}'` to succeed and record the full returned object ID as the baseline. Git documents `^{commit}` as commit-ish verification in [git-rev-parse](https://git-scm.com/docs/git-rev-parse).
- If v0.1 rejects detached `HEAD`, check it separately with `git symbolic-ref --quiet HEAD`; a detached `HEAD` can still resolve to a valid commit, so commit verification alone cannot enforce that precondition. See [git-symbolic-ref](https://git-scm.com/docs/git-symbolic-ref).

### Detect tracked and staged dirt without rejecting untracked output

[`git status --porcelain=v1`](https://git-scm.com/docs/git-status) is intended for scripts and is guaranteed not to change incompatibly across Git versions or with user configuration. `-z` uses NUL-delimited, unquoted pathnames.

A suitable explicit query is:

```text
git --no-optional-locks status --porcelain=v1 -z --untracked-files=no --ignore-submodules=untracked
```

Treat any record as a dirty precondition failure. `--untracked-files=no` allows the documented untracked-only starting state. `--ignore-submodules=untracked` ignores untracked-only content inside a submodule while still reporting modified tracked content or a checked-out commit that differs from the superproject's gitlink. If product policy instead intends submodule-untracked files to block inspection, use `--ignore-submodules=none`; do not inherit a repository's `diff.ignoreSubmodules` setting accidentally. The distinction and all four modes are documented by [`git status`](https://git-scm.com/docs/git-status).

The leading `--no-optional-locks` matters to the read-only contract: Git documents that background/scripted status calls can otherwise perform an optional index refresh. It must be present on both the initial and final status queries.

### Enumerate tracked regular files without following links

Use:

```text
git ls-tree -r -z --full-tree -l HEAD
```

[`git ls-tree`](https://git-scm.com/docs/git-ls-tree) enumerates the committed tree and can return mode, object type, object ID, blob size, and path. `-z` makes the path verbatim and NUL-terminated. This supports paths containing whitespace, newlines, Unicode, and shell metacharacters without shell parsing, supplies the size needed for the oversized-file cutoff, and makes the inventory source the recorded baseline commit rather than the mutable index or worktree.

The [Git data model](https://git-scm.com/docs/gitdatamodel) identifies the relevant tree modes:

- `100644` and `100755`: regular files;
- `120000`: symbolic links;
- `160000`: gitlinks (submodules).

Accept only regular blob entries. Skip `120000` and `160000`; do not descend into gitlinks. Git's [submodule documentation](https://git-scm.com/docs/gitsubmodules) confirms that the superproject records a submodule as a gitlink containing the expected submodule commit.

[`git ls-files --cached --stage -z`](https://git-scm.com/docs/git-ls-files) remains useful for tests that need to inspect index modes or merge stages, but it should not be the canonical inventory source: v0.1 evidence is tied to `HEAD`, and the clean precondition already rejects a staged tree that differs from `HEAD`.

For the deterministic inventory, prefer reading the recorded blob object by object ID with [`git cat-file`](https://git-scm.com/docs/git-cat-file), rather than opening the worktree path. Do not request `--filters`, `--textconv`, or `--follow-symlinks`: those options deliberately invoke configured content transforms or link resolution. Object reads make the inventory correspond to the pinned Git data, avoid following worktree symlinks, and avoid executing repository code.

Before returning success, re-read `HEAD` and the dirty-state query. If either differs from the recorded start state, report concurrent repository change and discard the inventory result. This final recheck is a project safety invariant built from the Git primitives above, not a transaction guarantee supplied by Git.

### Write-path implication

Index modes protect inspection, not later writes. Before `render`, `adr`, or Agent Skill file creation, resolve the intended path beneath the repository root, reject `..`, and inspect each existing path component without following symlinks. Reject symlink, submodule, and collision targets. Inspection commands should continue to read Git objects; mutation commands need a separate filesystem containment check.

## Go release and distribution security

The Go release history records [Go 1.26.5](https://go.dev/doc/devel/release) as released on 2026-07-07 with security fixes, so the plan's toolchain version was current on the research date.

### Checksums and SBOMs

[GoReleaser checksums](https://goreleaser.com/customization/package/checksum/) generate and upload a checksum manifest for published artifacts; SHA-256 is the documented default. Publish that manifest with every release and include all binary archives.

[GoReleaser SBOM support](https://goreleaser.com/customization/sbom/) can generate SBOMs for release artifacts and defaults to using Syft for each archive. It supports SPDX JSON and CycloneDX JSON output. Generate one machine-readable SBOM per distributable archive and publish the files as release assets.

Checksums alone let a consumer compare bytes with a manifest, but they do not establish who produced the manifest. Bind the artifacts and digests to the GitHub Actions build identity with signed attestations.

### Provenance and SBOM attestations

The current [`actions/attest`](https://github.com/actions/attest) action:

- creates signed in-toto attestations using short-lived Sigstore certificates;
- generates SLSA build provenance when given an artifact subject;
- accepts a GoReleaser-style checksum file through `subject-checksums`;
- accepts SPDX or CycloneDX JSON through `sbom-path` for SBOM attestations;
- stores attestations in GitHub's attestations API; and
- supports verification with `gh attestation verify`.

The action requires `id-token: write` to obtain the signing certificate and `attestations: write` to persist the attestation. Keep `contents: read`; add `artifact-metadata: write` only when creating the optional artifact storage record. GitHub's [artifact-attestation overview](https://docs.github.com/en/actions/concepts/security/artifact-attestations) explains that provenance records the workflow, repository, organization, commit, event, and related OIDC claims.

For this public OSS repository, GitHub uses Sigstore's public-good instance and public transparency log. [Sigstore's Rekor documentation](https://docs.sigstore.dev/logging/overview/) describes that log as immutable, tamper-resistant, and append-only. No separate Rekor upload is needed for the GitHub-generated public attestation. An attestation proves the artifact's identity and build provenance; it is not evidence that the program is vulnerability-free or behaviorally correct.

Release workflow implications:

1. Build the macOS, Linux, and Windows artifacts in GitHub Actions from the release tag.
2. Generate the SHA-256 manifest and per-archive SBOMs before publishing.
3. Create provenance over the exact subjects named by the checksum manifest.
4. Create SBOM attestations that bind each archive to its corresponding SBOM.
5. Upload archives, checksum manifest, and SBOMs to the draft release.
6. Document both `sha256sum`/platform-equivalent verification and `gh attestation verify`.

GitHub's [secure-use reference](https://docs.github.com/en/actions/reference/security/secure-use) says pinning actions to a full-length commit SHA is the only immutable way to consume an action, so release workflows should pin GoReleaser and attestation actions to reviewed full SHAs rather than mutable tags.

GitHub also supports [immutable releases](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/prevent-release-changes). Enable the setting before `v0.1.0`, attach every asset to a draft, and publish only after the asset set is complete; immutability applies only to future releases and prevents later asset or tag replacement.

## v0.1 boundaries confirmed by this research

- Conform to the Agent Skills core specification; test discovery and behavior per consumer.
- Render a bounded root `AGENTS.md` section without treating Markdown as a structured policy database.
- Do not build a catalog, central-policy manager, tool-specific converter, or synchronization layer.
- Read the pinned Git snapshot through committed-tree metadata and Git objects; skip symlinks and gitlinks.
- Allow untracked-only starts while rejecting tracked/staged change, and recheck the repository before reporting success.
- Publish cross-platform archives with SHA-256 checksums, per-archive SBOMs, signed provenance and SBOM attestations, pinned workflow actions, and immutable GitHub releases.

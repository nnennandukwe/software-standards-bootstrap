# Software Standards Bootstrap

Software Standards Bootstrap (`ssb`) generates four actionable artifact kinds
for an existing Git repository:

1. **Semantic rules** for evidence-backed implementation conditions.
2. **Verification recipes** for deliberately invoked existing commands.
3. **Agent Skills** for multi-step engineering workflows.
4. **Automation proposals** for valuable checks that do not yet exist.

A required run report indexes accepted artifacts. A derived root `AGENTS.md`
routes future coding agents to active rules, recipes, and skills.

Run `ssb` in a repository whose engineering conventions are not yet documented for AI tools. A compatible coding agent analyzes the committed repository and proposes these files. Developers review, edit, delete, or approve them before adoption.

The coding agent performs semantic analysis. The offline `ssb` CLI pins the
input commit, builds and replays a safe inventory, validates schemas and exact
evidence, renders `AGENTS.md`, and creates an optional Proposed ADR.

## What it generates

### Semantic rules

Each proposed rule is stored at:

```text
.software-standards/rules/<rule-id>.md
```

Rules tell future AI tools what to `always`, `never`, or `prefer`, and when to
ask a developer before proceeding.

Every rule carries its engineering category, activation lenses, path scopes,
derivation, exact repository evidence, and actionable body. Commands and check
metadata never live in a semantic rule.

For example:

> Always use the service interface when code under `internal/api/` accesses persistence. Do not call the database package directly.

The generated rule would cite the repository files and line ranges that establish that boundary.

`ssb` does not turn generic best practices into repository rules. Extracted
rules need repository authority or enforcement. Inferred rules need three
supporting occurrences across at least two files. Rejected candidates are
discarded.

Before proposing artifacts, the agent reviews dependency boundaries, parallel
implementations, platform seams, compatibility surfaces,
source/test/documentation symmetry, and existing automatic enforcement.

### Verification recipes

Existing commands that deliver value when deliberately invoked are recorded at:

```text
.software-standards/verification/<recipe-id>.yaml
```

A recipe states when it applies, its ordered commands, exact `enforces`
evidence, and the expected successful result. `ssb` records but never executes
recipe commands.

### Agent Skills

Multi-step procedures are generated as reusable Agent Skills:

```text
.agents/skills/<skill-name>/SKILL.md
```

A skill can tell an AI coding tool how to plan a cross-boundary change, update
generated code, modify a schema and its dependents, or recover a multi-step
engineering workflow.

Rules define semantic conditions. Recipes record existing commands. Skills
define procedures.

### Automation proposals

Valuable automatic checks that do not exist are described at:

```text
.software-standards/automation/<proposal-id>.yaml
```

These are reviewable designs, not implemented checks or active standards.

### Optimized `AGENTS.md`

`ssb` renders the proposed guidance into a managed section in the repository's root `AGENTS.md`.

Base rules become standing orders. Contextual rules and recipes are links.
Primary Agent Skills are indexed by activation context. Automation proposals
are omitted because they are not active behavior.

The report and canonical artifact files are editable sources. `AGENTS.md` is
derived.

### Report and ADR

Every pack requires:

```text
.software-standards/report.md
```

The report records the complete inventory, accepted artifact index, confidence,
utility, relationships, run-wide limitations, and accepted-output summaries.
It contains no rejected candidates or reasons. A zero-artifact report is valid.

After developer review, `ssb` can create an optional ADR:

```text
docs/adr/NNNN-actionable-standards.md
```

The ADR remains `Proposed` until the developer-created pull request is merged.

## How future AI work uses the output

Once reviewed and merged, compatible AI coding tools can use the generated files throughout the software development lifecycle.

| AI-assisted activity | Repository context supplied |
|---|---|
| Planning | Architectural boundaries, affected components, required workflows, approval points, and constraints |
| Code generation | Coding conventions, preferred patterns, prohibited patterns, file scopes, and implementation guidance |
| Testing | Repository-specific verification recipes and expected successful results |
| Code review | The repository's correctness, maintainability, security, compatibility, and review requirements |
| Maintenance | Established patterns for changing code without breaking repository structure or conventions |

## Human review and deterministic guardrails

Generation creates an uncommitted proposal. Developers decide which artifacts
become part of the repository through normal Git review.

Semantic rules, command recipes, procedures, and proposed automation stay
separate. `ssb` does not execute a recipe, implement a proposed checker, or
claim that any external check passed.

## Who it is for

`ssb` is for teams making an existing repository's SDLC more AI-integrated while preserving engineering judgment, repository conventions, and review responsibility. It fits repositories that:

- use or plan to use AI tools for software development;
- contain coding or SDLC conventions that AI tools need to follow;
- need documented rules, reusable Agent Skills, or a clearer root `AGENTS.md`;
- want generated guidance backed by inspectable repository evidence; and
- require developer approval before that guidance is adopted.

It does not provide a generic rules catalog, invent standards without repository evidence, or replace developer review.

## Quick start

### Requirements

- Git 2.39 or newer
- A repository with at least one commit on an attached branch
- No tracked or staged working-tree changes
- Go 1.26.5 when building `ssb` from source
- `GOBIN` or `$(go env GOPATH)/bin` on `PATH`
- A compatible coding agent that can use the bundled Agent Skill

Untracked files are allowed during inspection.

### 1. Install the CLI

```bash
git clone https://github.com/nnennandukwe/software-standards-bootstrap.git
cd software-standards-bootstrap
go install ./cmd/ssb
ssb --help
```

### 2. Expose the bootstrap skill

From this checkout, copy the bundled skill into the target repository.

For Codex:

```bash
mkdir -p /path/to/repository/.agents/skills
cp -R skills/software-standards-bootstrap /path/to/repository/.agents/skills/
```

For Claude Code:

```bash
mkdir -p /path/to/repository/.claude/skills
cp -R skills/software-standards-bootstrap /path/to/repository/.claude/skills/
```

### 3. Generate the proposal

From the clean target repository, ask the coding agent:

```text
Use the software-standards-bootstrap skill to analyze this repository
and generate evidence-backed actionable artifacts and AGENTS.md guidance.
```

The agent runs the inventory, analyzes repository evidence, writes the proposal, validates it, renders `AGENTS.md`, and reports every changed or untracked path.

### 4. Review and rerender

Review the report, every canonical artifact, and the generated `AGENTS.md`
section. Do not edit the managed section directly. Edit canonical sources and
the report together, then rerun validation and rendering.

```bash
ssb validate --repo .
ssb render --repo . --dry-run
ssb render --repo .
```

Review the complete uncommitted diff before creating a pull request.

### 5. Create an optional ADR

Only after reviewing the retained proposal:

```bash
ssb adr --repo . --dry-run
ssb adr --repo .
```

The developer-created pull request and its merge are the adoption decision.

## CLI commands

```text
ssb inspect  [--repo PATH] [--format text|json] [resource limits]
ssb validate [--repo PATH] [--format text|json]
ssb render   [--repo PATH] [--review ID] [--dry-run]
ssb adr      [--repo PATH] [--review ID] [--adr-dir PATH] [--dry-run]
ssb prune    <inspect|validate|approve|apply|recover|status|verify> [options]
```

- `inspect` creates a safe inventory of one committed repository snapshot.
- `validate` checks the report, all four artifact schemas, inventory, evidence,
  confidence, utility, scopes, and relationships.
- `render` updates only the managed Software Standards Bootstrap section of root `AGENTS.md`, removing that section when no rule, recipe, or skill is active.
- `adr` creates one new Proposed ADR from retained rules, recipes, and skills.

`inspect` supports `--max-candidate-files` and `--max-candidate-bytes`. `--allow-partial` permits diagnostic output from an incomplete inventory, but that output cannot be used to generate a proposal. Exit code `4`: inventory coverage incomplete.

- `0`: success
- `1`: actionable-pack or prune-proposal validation failure
- `2`: usage or repository precondition failure
- `3`: unexpected internal failure
- `4`: inventory coverage incomplete

Run `ssb <command> --help` for complete command options.

## Governed lifecycle review

`prune` is the current name for a governed lifecycle review of an adopted
pack. It does not mean automatic cleanup. The workflow compares every rule and
repository Agent Skill with a developer-selected, point-in-time host/model
capability profile and proposes `keep`, `update`, `consolidate`, `remove`, or
`unable-to-determine`. Every disposition requires evidence and rationale.
Actionable dispositions bind repository and capability evidence; an
unable-to-determine disposition records a structured evidence gap against an
exact artifact. Unknown provenance remains unable to determine.
Skill provenance covers the complete tracked bundle; a partially declared
skill remains unknown, and ignored untracked governed files block inspection.

Each review is stored under
`.software-standards/reviews/<review-id>/` with its immutable context,
proposal, candidate inputs, evidence snapshots, and digest-chained events.

Prune inspection fails closed on incomplete inventory and writes only an
immutable review context. The Agent Skill writes the semantic proposal. The
CLI validates it, records one digest-bound human approval, shows application
as a dry run by default, and applies only after `--write`. Application,
rerendering, optional ADR creation, and receipt-backed verification are
separate events. One canonical application-plan digest binds dry run, mutation,
recovery, and verification. If no changes are approved, status reports a
terminal no-change outcome without inventing application or verification.
See [the prune protocol](docs/prune.md).

## Safety

The `ssb` CLI:

- does not call an AI model;
- does not make network requests or send telemetry;
- does not execute repository code, tests, hooks, or recipe commands;
- does not stage, commit, branch, push, or open pull requests;
- reads inspection input from the committed Git tree rather than worktree files;
- stops proposal generation when inventory coverage is incomplete;
- leaves generated files local and uncommitted for developer review;
- pins prune capability evidence and preserves unknown provenance as unknown;
- keeps prune application dry-run by default and journals approved writes for
  recovery;
- binds artifact removals and their `report.md` index updates into one
  recoverable plan;
- rejects portable-path escapes and case-fold collisions before mutation; and
- binds verification to the exact application, governed poststate, rerender,
  and external check receipts.

## Detailed documentation

- [Rule format](docs/rule-format.md)
- [Category taxonomy](skills/software-standards-bootstrap/references/categories.md)
- [Architecture and trust boundaries](docs/architecture.md)
- [Agent workflow tests](docs/agent-smoke-tests.md)
- [Governed prune protocol](docs/prune.md)
- [Verification record](docs/verification.md)
- [Contributing](CONTRIBUTING.md)

## Non-goals

`ssb` does not provide a generic rules catalog, automatic synchronization with
an online model registry, vendor-release-note trust, tool-specific rule
projections, checker generation, hosted services, direct model APIs, telemetry,
hooks, automatic refresh, unreviewed rewriting or deletion, or downstream
product activation. The `prune` name and command architecture remain revisable.

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/ssb
go tool govulncheck ./...
```

Licensed under Apache-2.0.

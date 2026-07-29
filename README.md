# Software Standards Bootstrap

Software Standards Bootstrap (`ssb`) generates three kinds of AI-facing guidance for an existing Git repository:

1. **Coding-convention rules** that explain how code should be structured, implemented, tested, reviewed, and maintained.
2. **Agent Skills** that give AI coding tools step-by-step instructions for repository-specific engineering workflows.
3. **An optimized root `AGENTS.md`** that routes AI coding tools to the rules and related skills relevant to their current files and task.

Together, these files give future AI-assisted work repository-specific context for planning changes, generating code, testing, maintenance, code review, and release work.

Run `ssb` in a repository that already contains engineering conventions but does not yet document them clearly for AI coding tools. A compatible coding agent analyzes the committed repository and proposes rules, skills, and `AGENTS.md` guidance. Developers review, edit, delete, or approve those files before they become part of the repository.

The coding agent performs the semantic analysis. The `ssb` CLI pins the input commit, builds a safe inventory, validates the generated evidence, renders `AGENTS.md`, and creates an optional Proposed ADR.

## What it generates

### Coding-convention rules

Each proposed rule is stored at:

```text
.software-standards/rules/<rule-id>.md
```

Rules tell future AI tools what to `always`, `never`, or `prefer`, and when to ask a developer before proceeding. They can cover code structure, architectural boundaries, implementation patterns, testing, review, security, documentation, and release work.

Every rule records where it applies, the exact repository evidence supporting it, the commit that was analyzed, and whether an existing repository check verifies all or part of it.

For example:

> Always use the service interface when code under `internal/api/` accesses persistence. Do not call the database package directly.

The generated rule would cite the repository files and line ranges that establish that boundary.

`ssb` does not turn generic best practices into repository rules. A rule needs an authoritative repository source or a consistent pattern repeated across the repository. Findings without enough evidence remain in the assessment.

Before proposing rules, the agent performs a structural-pattern review of dependency boundaries, parallel implementations, platform seams, compatibility surfaces, and source/test/documentation symmetry. Every retained rule also names one primary topic, such as architecture, correctness, maintainability, security, or testability, so reviewers can see the engineering concern it protects.

### Agent Skills

Multi-step procedures are generated as reusable Agent Skills:

```text
.agents/skills/<skill-name>/SKILL.md
```

A skill can tell an AI coding tool how to plan a cross-boundary change, update generated code, modify a schema and its dependents, or follow an existing review or release workflow.

Rules define constraints. Skills define procedures.

### Optimized `AGENTS.md`

`ssb` renders the proposed guidance into a managed section in the repository's root `AGENTS.md`.

Base rules become standing orders. Contextual rules are routed by affected path, language, framework, and task. Rules reference procedural skills when needed. This gives an AI tool the relevant repository guidance without loading every rule for every task.

The rule and skill files are the editable sources. `AGENTS.md` is generated from them.

### Assessment and ADR

The initial workflow also creates:

```text
.software-standards/assessment.md
```

The assessment records what was analyzed, what became a rule or skill, and which findings remained assessment-only.

After developer review, `ssb` can create an optional ADR:

```text
docs/adr/NNNN-agentic-rules.md
```

The ADR remains `Proposed` until the developer-created pull request is merged.

## How future AI work uses the output

Once reviewed and merged, compatible AI coding tools can use the generated files throughout the software development lifecycle.

| AI-assisted activity | Repository context supplied |
|---|---|
| Planning | Architectural boundaries, affected components, required workflows, approval points, and constraints |
| Code generation | Coding conventions, preferred patterns, prohibited patterns, file scopes, and implementation guidance |
| Testing | Repository-specific testing expectations and existing verification commands |
| Code review | The repository's correctness, maintainability, security, compatibility, and review requirements |
| Maintenance | Established patterns for changing code without breaking repository structure or conventions |
| Release work | Relevant release rules, procedures, checks, and ask-first boundaries |

The goal is to give AI tools the same repository-specific context a responsible developer would need before making or reviewing a change.

## Human review and deterministic guardrails

Generation creates an uncommitted proposal. Developers decide which rules and skills become part of the repository through normal Git review.

A rule is **guidance** when repository evidence supports the instruction but no existing command fully enforces it.

A rule is **deterministic** only when the repository already contains a command that fully checks it. `ssb` records where the command is defined and what it proves when it passes.

`ssb` does not generate a checker, run the mapped command, or claim that it passed. Partial checks are recorded as partial coverage rather than complete enforcement.

## Who it is for

`ssb` is for repositories that:

- use or plan to use AI tools for software development;
- contain coding or SDLC conventions that AI tools need to follow;
- need documented rules, reusable Agent Skills, or a clearer root `AGENTS.md`;
- want generated guidance backed by inspectable repository evidence; and
- require developer approval before that guidance is adopted.

It is designed for teams making their SDLC more AI-integrated while preserving engineering judgment, repository conventions, and review responsibility.

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
and generate evidence-backed coding rules, Agent Skills, and AGENTS.md guidance.
```

The agent runs the inventory, analyzes repository evidence, writes the proposal, validates it, renders `AGENTS.md`, and reports every changed or untracked path.

### 4. Review and rerender

Review the assessment, every proposed rule and skill, and the generated `AGENTS.md` section. Edit or delete anything you do not want to adopt.

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
ssb render   [--repo PATH] [--dry-run]
ssb adr      [--repo PATH] [--adr-dir PATH] [--dry-run]
```

- `inspect` creates a safe inventory of one committed repository snapshot.
- `validate` checks proposed rules, evidence, scopes, proof mappings, and related skills.
- `render` updates only the managed Software Standards Bootstrap section of root `AGENTS.md`.
- `adr` creates one new Proposed ADR from the rules and skills that survived review.

`inspect` supports `--max-candidate-files` and `--max-candidate-bytes`. `--allow-partial` permits diagnostic output from an incomplete inventory, but that output cannot be used to generate a proposal. Exit code `4`: inventory coverage incomplete.

Run `ssb <command> --help` for complete command options.

## Safety

The `ssb` CLI:

- does not call an AI model;
- does not make network requests or send telemetry;
- does not execute repository code, tests, hooks, or mapped verification commands;
- does not stage, commit, branch, push, or open pull requests;
- reads inspection input from the committed Git tree rather than worktree files;
- stops proposal generation when inventory coverage is incomplete; and
- leaves generated files local and uncommitted for developer review.

The coding agent performs the semantic analysis and writes the proposal. The CLI provides the inventory, validation, rendering, and ADR safety boundaries.

## Detailed documentation

- [Rule format](docs/rule-format.md)
- [Topic taxonomy](skills/software-standards-bootstrap/references/topics.md)
- [Architecture and trust boundaries](docs/architecture.md)
- [Agent workflow tests](docs/agent-smoke-tests.md)
- [Verification record](docs/verification.md)
- [Contributing](CONTRIBUTING.md)

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/ssb
go tool govulncheck ./...
```

Licensed under Apache-2.0.

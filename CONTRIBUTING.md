# Contributing

Thank you for helping improve Software Standards Bootstrap.

## Development setup

Install Go 1.26.5 and Git 2.39 or newer, then run:

```bash
go mod download
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/ssb
go tool govulncheck ./...
```

## Behavior-first changes

Every behavior change starts with a focused failing test. Prefer the lowest layer that proves the user-visible contract:

- workspace and inventory integration tests for Git states and object reads;
- rule-pack tests for schema, scoring, evidence, and proof mapping;
- renderer tests for byte preservation, drift, markers, and atomic failure;
- ADR tests for convention, containment, ambiguity, and collision; and
- CLI tests for exit code, stdout, stderr, recovery text, and filesystem effects.

Every write path needs a happy path, a blocked path, recovery guidance, and a no-partial-output assertion.

## Scope guardrails

Do not add:

- runtime network behavior, telemetry, API keys, or model clients;
- generic rule catalogs or tool-specific synchronization;
- repository code execution during inspection;
- generated checkers;
- automatic Git actions; or
- silent overwrite or repair behavior.

Proposals that change these boundaries need an explicit architecture decision before implementation.

## Pull requests

Keep changes behavior-closed. Include:

- scenarios added;
- the red test observed first;
- the production change that reached green;
- full verification results;
- documentation or Agent Skill parity updates; and
- residual manual acceptance gaps.

Do not claim Codex, Claude Code, release, or public-benchmark acceptance without a recorded run against the exact version and commit.

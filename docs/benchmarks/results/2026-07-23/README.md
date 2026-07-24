# 2026-07-23 benchmark records

These records capture the v0.1 benchmark proposal gate against the four public
repositories pinned in [`testdata/benchmarks.yaml`](../../../../testdata/benchmarks.yaml).
They are release evidence, not adopted policy for the evaluated repositories.

## Immutable evaluator input

- SSB source commit:
  `a426d419cb129541f4bf69497ed7232eff926c46`
- Evaluator binary SHA-256:
  `3b26fde24b92a61fce308a55a96668523cbdbe4fb7f833fd8a6a5b0d3768dcaf`
- Host: macOS 15.7.3 (`arm64`)
- Git: 2.39.5 (Apple Git-154)
- Go: 1.26.5 (`darwin/arm64`)

## Consumer matrix

| Consumer | Exact version | Cobra | Flask | Django | Next.js |
|---|---|---|---|---|---|
| Codex desktop | 26.715.71837 (build 5702), `gpt-5.6-sol`, `xhigh` | Proposal recorded | Proposal recorded | Proposal recorded | Proposal recorded |
| Claude Code | 2.1.191, `claude-sonnet-4-6` | Proposal recorded | Proposal recorded | Proposal recorded | Proposal recorded |

Every proposal record stops at the mandatory developer-review gate. A proposal
is not a passing end-to-end smoke test until a developer has made explicit
retention decisions, one rule has been edited, another deleted, the derived
output has been rerendered, and an ADR has been explicitly requested and
verified.

## Records

- [Cobra / Codex](codex-cobra.md)
- [Flask / Codex](codex-flask.md)
- [Django / Codex](codex-django.md)
- [Next.js / Codex](codex-nextjs.md)
- [Cobra / Claude Code](claude-cobra.md)
- [Flask / Claude Code](claude-flask.md)
- [Django / Claude Code](claude-django.md)
- [Next.js / Claude Code](claude-nextjs.md)

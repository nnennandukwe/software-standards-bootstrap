# Next.js / Codex actionable-artifact acceptance

Generated and reviewed on 2026-07-31. The developer explicitly kept every
final artifact. The pack remains uncommitted in the disposable fixture; its
ADR remains `Proposed`.

## Runtime and immutable input

- Consumer: `codex-cli 0.145.0`
- Session: `019fb896-7889-7c62-81d7-06c9fac58db3`
- Model and reasoning: `gpt-5.6-sol`, `xhigh`
- Repository: `github.com/vercel/next.js`
- Baseline: `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`
- Attached branch: `ssb-evaluation`
- Inventory: schema 2; 29,073 candidate and scanned files; 28,403 indexed
  files; 111,110,455 candidate/scanned bytes; 88,643,646 indexed bytes
- Exclusions: 652 binary, 18 generated, 21 oversized, 113 secret-like,
  29 symlink, and 1,060 vendor/generated-tree; submodule and non-regular 0
- Truncated: no; remaining candidate files and bytes: 0

## Final artifacts and decisions

All eleven artifacts resolved their exact evidence and passed validation.

| Artifact | Kind | Category | Confidence | Utility | Decision |
|---|---|---|---|---:|---|
| `read-local-readmes-before-editing` | Rule | developer-experience | high | 57 | Keep |
| `keep-internal-request-headers-unforgeable` | Rule | security | high | 83 | Keep |
| `wire-feature-flags-across-consumption-boundaries` | Rule | correctness | high | 81 | Keep |
| `keep-conditional-requires-dead-code-eliminable` | Rule | compatibility | high | 81 | Keep |
| `route-app-page-runtime-imports-through-entry-base` | Rule | architecture | high | 70 | Keep |
| `distinguish-dev-server-only-behavior` | Rule | correctness | high | 65 | Keep |
| `verify-repository-lint-suite` | Recipe | quality | high | 69 | Keep |
| `verify-next-package-types` | Recipe | correctness | high | 60 | Keep |
| `verify-edge-bundling` | Recipe | compatibility | high | 75 | Keep |
| `iterate-next-core-change` | Agent Skill | developer-experience | high | 78 | Keep |
| `automate-internal-header-registration` | Automation proposal | security | medium | 76 | Keep |

Retention is 11/11 (100%), exceeding the required exact per-fixture threshold
of 70%.

The structural review covered repository and package guidance, parallel
feature-configuration surfaces, the router/header security seam, Edge and
webpack compatibility, the app-page runtime boundary, source/test symmetry,
existing lint/type/bundling commands, and existing automatic enforcement. The
new header-registration check remains a proposal.

## Projection and ADR

- `ssb validate`: pass, 6 rules, 3 recipes, 1 Agent Skill, 1 automation proposal
- `ssb render --dry-run`: current
- ADR: `docs/adr/0001-actionable-standards.md`
- ADR status: `Proposed`
- ADR SHA-256:
  `fae98b2e37cd327438f56bbbf0da280f10b08b5f9821bd55d2fe0c5505021923`
- ADR contents: 6 rules, 3 recipes, and 1 Agent Skill; automation omitted

## Complete changed and untracked paths

```text
.agents/skills/iterate-next-core-change/SKILL.md
.agents/skills/software-standards-bootstrap/SKILL.md
.agents/skills/software-standards-bootstrap/references/categories.md
.agents/skills/software-standards-bootstrap/references/evidence-workflow.md
.agents/skills/software-standards-bootstrap/references/prune-review.md
.agents/skills/software-standards-bootstrap/references/rule-schema.md
.agents/skills/software-standards-bootstrap/references/structural-patterns.md
.software-standards/automation/automate-internal-header-registration.yaml
.software-standards/report.md
.software-standards/rules/distinguish-dev-server-only-behavior.md
.software-standards/rules/keep-conditional-requires-dead-code-eliminable.md
.software-standards/rules/keep-internal-request-headers-unforgeable.md
.software-standards/rules/read-local-readmes-before-editing.md
.software-standards/rules/route-app-page-runtime-imports-through-entry-base.md
.software-standards/rules/wire-feature-flags-across-consumption-boundaries.md
.software-standards/verification/verify-edge-bundling.yaml
.software-standards/verification/verify-next-package-types.yaml
.software-standards/verification/verify-repository-lint-suite.yaml
AGENTS.md
docs/adr/0001-actionable-standards.md
```

## Safety boundary

No Next.js code or recorded recipe command was executed. No automation was
implemented. `HEAD` stayed at the pin, the index remained empty, and no Git or
repository-tool network mutation occurred. Worktree-only inventory assembly
files were removed before handoff.

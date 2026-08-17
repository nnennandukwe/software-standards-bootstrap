# ADR 0001: Adopt actionable repository standards

- Status: Proposed
- Baseline commit: `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd`
- Manifest: `.software-standards/manifest.yaml`
- Inventory: `.software-standards/inventory.json`
- Report: `.software-standards/report.md`

## Context

The repository was inspected at the pinned baseline above. The developer retained the following evidence-backed actionable artifacts after review. Verification recipes are recorded here but were not executed by SSB.

## Semantic rules

### Keep pnpm security settings synchronized (`keep-pnpm-security-settings-synchronized`)

- Source: `.software-standards/rules/keep-pnpm-security-settings-synchronized.md`
- Scope: `pnpm-workspace.yaml`, `scripts/install-native.mjs`, `test/lib/pnpm-security-settings.js`, `test/lib/create-next-install.js`
- Lenses: `task:implementation`
- Directive: `always`
- Category: `security`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `high` (75/100, `ssb-utility-v1`)
- Evidence: `pnpm-workspace.yaml:33-39` (`declares`), `scripts/install-native.mjs:77-94` (`demonstrates`), `test/lib/pnpm-security-settings.js:1-3` (`declares`), `test/lib/create-next-install.js:64-93` (`demonstrates`)

When changing the root `blockExoticSubdeps`, `minimumReleaseAge`, or `minimumReleaseAgeExclude` settings, make the same change to the literal `pnpm-workspace.yaml` emitted by `scripts/install-native.mjs`. Keep isolated test fixtures reading and copying the root values through `test/lib/pnpm-security-settings.js` instead of adding another duplicated value set.

### Match Next.js test mode and bundler (`match-next-test-mode-and-bundler`)

- Source: `.software-standards/rules/match-next-test-mode-and-bundler.md`
- Scope: `packages/next/**`, `test/**`, `turbopack/**`
- Lenses: `framework:nextjs`, `task:verification`
- Directive: `always`
- Category: `testability`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `high` (78/100, `ssb-utility-v1`)
- Evidence: `AGENTS.md:73-105` (`declares`), `package.json:23-38` (`enforces`), `AGENTS.md:441-447` (`declares`)

Run focused Next.js integration tests with the root `test-dev-*` or `test-start-*` script matching the required mode and bundler (Webpack, Turbopack, or Rspack). Rebuild before integration tests after source changes: after bootstrap, core-only changes may use `pnpm --filter=next build`; use `pnpm build-all` after a branch switch/bootstrap, before a CI push, or for cross-package or Rust/Turbopack changes.

### Read path-local READMEs before editing (`read-path-readmes-before-editing`)

- Source: `.software-standards/rules/read-path-readmes-before-editing.md`
- Scope: `**/*`
- Lenses: `base`
- Directive: `always`
- Category: `developer-experience`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `high` (70/100, `ssb-utility-v1`)
- Evidence: `AGENTS.md:44-53` (`declares`)

Before editing or creating a file in a subdirectory, read every `README.md` on the directory path from the repository root through the target directory. Apply the closest documented conventions to the change.

### Treat unfiltered internal request headers as forgeable (`treat-unfiltered-internal-request-headers-as-forgeable`)

- Source: `.software-standards/rules/treat-unfiltered-internal-request-headers-as-forgeable.md`
- Scope: `packages/next/src/server/**`
- Lenses: `framework:nextjs`
- Directive: `always`
- Category: `security`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (90/100, `ssb-utility-v1`)
- Evidence: `AGENTS.md:508-512` (`declares`)

When server code reads a nonstandard request header, treat that header as externally forgeable unless the router entry point strips it through `INTERNAL_HEADERS`. Flag the change for security review and verify the ingress filtering before relying on the header for an internal trust decision.

### Use Next.js test fixtures and polling conventions (`use-next-test-fixtures-and-polling-conventions`)

- Source: `.software-standards/rules/use-next-test-fixtures-and-polling-conventions.md`
- Scope: `test/**`
- Lenses: `framework:nextjs`, `task:implementation`
- Directive: `always`
- Category: `testability`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (81/100, `ssb-utility-v1`)
- Evidence: `AGENTS.md:150-161` (`declares`), `AGENTS.md:180-184` (`declares`), `AGENTS.md:198-225` (`declares`)

Create new test suites non-interactively with `pnpm new-test -- --args <appDir> <name> <type>` so they use repository templates; use `true|false` for `<appDir>` and `e2e|production|development|unit` for `<type>`. Prefer real fixture directories over inline `files` objects, and wait with `retry()` plus `expect()` instead of fixed `setTimeout` delays or deprecated `check()`.

### Wire Next.js runtime flags across all consumers (`wire-next-runtime-flags-across-all-consumers`)

- Source: `.software-standards/rules/wire-next-runtime-flags-across-all-consumers.md`
- Scope: `packages/next/src/**`
- Lenses: `framework:nextjs`, `task:implementation`
- Directive: `always`
- Category: `correctness`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `very-high` (88/100, `ssb-utility-v1`)
- Evidence: `AGENTS.md:464-470` (`declares`)

For every new or changed Next.js feature flag, keep its config type and schema synchronized. Define it for user-bundled code when that code consumes it, wire runtime environment values for precompiled server or export-worker consumers, and force flags guarding Node-only imports to `false` in edge builds.

## Verification recipes

### Verify Next.js package types (`verify-next-package-types`)

- Source: `.software-standards/verification/verify-next-package-types.yaml`
- Scope: `packages/next/src/**/*.ts`, `packages/next/src/**/*.tsx`
- Lenses: `framework:nextjs`, `task:verification`
- When: Use to verify declaration compilation after changing non-test, non-story TypeScript under `packages/next/src`; it does not cover `*.test.*`, `*.stories.tsx`, or Storybook files.
- Category: `correctness`
- Derivation: `extracted`
- Confidence: `high`
- Utility: `high` (65/100, `ssb-utility-v1`)
- Evidence: `packages/next/package.json:86-86` (`enforces`), `AGENTS.md:103-105` (`declares`), `packages/next/tsconfig.build.json:4-22` (`enforces`)

## Consequences

- `AGENTS.md` is a derived projection; the manifest, inventory, human report, and canonical artifact source files remain editable.
- Verification recipes remain deliberately invoked repository procedures; this record does not claim their commands passed.
- The developer-created pull request and its merge constitute adoption; this ADR remains Proposed until then.

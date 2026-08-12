<!-- software-standards-bootstrap:start -->
<!-- source-digest: sha256:8e7ac51d033b957b03b4752bc1cf262a35c0e80198ab81a5283ea555f7774708 -->
<!-- content-digest: sha256:8a2a57b23ca8dfd076cf1b8d1d0d8c77ffb508a02c89d3b53245307d40b084e6 -->
## Software Standards Bootstrap

This managed section is derived from retained canonical sources. An unmerged generated change is a proposal; repository review and merge are the adoption decision. File presence alone does not prove adoption.

Generated from `.software-standards/manifest.yaml`, `.software-standards/inventory.json`, `.software-standards/report.md`, `.software-standards/orientation.yaml` and the accepted artifacts by `ssb render`. Edit canonical sources and the manifest together, then rerun the command.

Baseline: `3e0091b89fdaa3912f4f8e7b33fbb3104e47d71e`

SSB did not stage, commit, push, open a pull request, execute any command, or activate another system. Recipe presence and expected results are not execution evidence.

### Repository orientation

Hoop is an open-source layer 7 gateway that governs access to infrastructure at the wire-protocol boundary.

- Evidence: `README.md:33-43 (declares)`

#### Important areas

- `gateway` — Hosts the HTTP and gRPC gateway, business services, persistence, and transport plugins.
  - Evidence: `CLAUDE.md:30-50 (declares)`

- `agent` — Hosts the long-lived agent connection and protocol-specific packet handlers.
  - Evidence: `CLAUDE.md:52-68 (declares)`

#### Prerequisites

- Go 1.26.2
  - Evidence: `go.work:1-3 (declares)`

#### Canonical documents

- [Repository architecture and conventions](CLAUDE.md)
  - Evidence: `CLAUDE.md:3-15 (declares)`

- [Development and verification workflow](DEV.md)
  - Evidence: `DEV.md:149-180 (declares)`

#### Related standards
- Related recipe: [Verify Go changes](.software-standards/verification/verify-go-changes.yaml)
- Related skill: [Change gateway api route](.agents/skills/change-gateway-api-route/SKILL.md)

#### Task guidance

- **Planning:** Identify the affected module boundary and its public protocol or API contracts before proposing a change.
  - Evidence: `CLAUDE.md:30-50 (declares)`

- **Implementation:** Preserve the documented library, gateway, agent, and shared-protocol boundaries while changing behavior.
  - Evidence: `CLAUDE.md:52-68 (declares)`

- **Verification:** Use the documented focused verification for the affected surface before handoff.
  - Evidence: `DEV.md:149-180 (declares)`

- **Handoff:** Describe the change, tests performed, and any remaining concerns for reviewers.
  - Evidence: `.github/PULL_REQUEST_TEMPLATE.md:40-52 (declares)`

### How routing works

- Directory placement and nearest-file precedence are host-level `AGENTS.md` behavior.
- Scopes and lenses are SSB's agent-readable routing contract, not native `AGENTS.md` glob activation. A semantic rule applies when its affected path scope matches; contextual artifacts also require every represented lens dimension to match, with values inside one dimension treated as alternatives.
- If the language, framework, task, or affected path is uncertain, load the potentially relevant rule, recipe, or skill instead of excluding it.
- Directives mean: `never` is prohibited, `ask-first` requires developer authorization, `always` is required, and `prefer` is the default when no documented exception or explicit user direction applies.
- Linked artifact files are canonical. This projection is a concise router, not a replacement for their complete content.

### Standing orders

#### Never

##### Keep libhoop independent (`keep-libhoop-independent`)

Do not import packages from `gateway/`, `agent/`, `client/`, or `common/` into `_libhoop/`. Bridge across the library boundary with standard-library types.

- Applies to: `_libhoop/**/*.go`
- Category: `architecture`
- Canonical rule: [.software-standards/rules/keep-libhoop-independent.md](.software-standards/rules/keep-libhoop-independent.md)
- Evidence: `CLAUDE.md:66-68`

#### Always

##### Preserve analytics event continuity (`preserve-analytics-event-continuity`)

When a route is added, duplicated, or superseded for an already tracked action, preserve its analytics event or document an intentional successor. Use constants from `gateway/analytics/events.go`; never replace them with event-name string literals or silently remove the final emission site.

- Applies to: `gateway/api/**/*.go`, `gateway/analytics/**/*.go`
- Related skill: [Change gateway api route](.agents/skills/change-gateway-api-route/SKILL.md)
- Category: `correctness`
- Canonical rule: [.software-standards/rules/preserve-analytics-event-continuity.md](.software-standards/rules/preserve-analytics-event-continuity.md)
- Evidence: `CLAUDE.md:194-199`

##### Synchronize environment configuration (`synchronize-environment-configuration`)

Keep runtime environment-variable reads synchronized with the matching gateway or agent Helm values, secret pass-through, user-facing chart documentation, and `.env.sample` in the same change.

- Applies to: `.env.sample`, `gateway/**/*.go`, `agent/**/*.go`, `deploy/helm-chart/chart/gateway/**/*`, `deploy/helm-chart/chart/agent/**/*`
- Category: `operability`
- Canonical rule: [.software-standards/rules/synchronize-environment-configuration.md](.software-standards/rules/synchronize-environment-configuration.md)
- Evidence: `CLAUDE.md:201-208`

##### Preserve transport plugin order (`preserve-transport-plugin-order`)

Preserve transport plugin registration in this order: review, audit, DLP, access control, webhooks, then Slack. Treat any reordering as a behavior change that requires explicit lifecycle analysis.

- Applies to: `gateway/main.go`
- Category: `correctness`
- Canonical rule: [.software-standards/rules/preserve-transport-plugin-order.md](.software-standards/rules/preserve-transport-plugin-order.md)
- Evidence: `CLAUDE.md:83-93`, `gateway/main.go:186-194`

##### Use protocol packet constants (`use-protocol-packet-constants`)

Represent protocol packet types with constants from `common/proto/{agent,client,gateway,system}`. Extend the appropriate constant package instead of introducing packet-type string literals.

- Applies to: `agent/**/*.go`, `client/**/*.go`, `gateway/**/*.go`, `common/**/*.go`
- Category: `compatibility`
- Canonical rule: [.software-standards/rules/use-protocol-packet-constants.md](.software-standards/rules/use-protocol-packet-constants.md)
- Evidence: `CLAUDE.md:59-63`

### Verification commands

#### [Regenerate OpenAPI artifacts](.software-standards/verification/regenerate-openapi-artifacts.yaml) (`regenerate-openapi-artifacts`)

- When: After changing an API route, handler annotation, or schema and before handoff.
- Route when: `task:verification`
- Applies to: `gateway/api/**/*.go`, `gateway/api/openapi/**/*`
- Related skill: [Change gateway api route](.agents/skills/change-gateway-api-route/SKILL.md)
- Category: `documentation`
- Canonical recipe: [.software-standards/verification/regenerate-openapi-artifacts.yaml](.software-standards/verification/regenerate-openapi-artifacts.yaml)
- Evidence: `Makefile:148-150`

##### Step 1

```
make generate-openapi-docs
```

Expected result: The generated OpenAPI v2 and v3 artifacts reflect the current gateway API definitions.

#### [Verify Go changes](.software-standards/verification/verify-go-changes.yaml) (`verify-go-changes`)

- When: Before handing off a change to Go code or module metadata.
- Route when: `task:verification`
- Applies to: `**/*.go`, `**/go.mod`, `go.work`
- Category: `testability`
- Canonical recipe: [.software-standards/verification/verify-go-changes.yaml](.software-standards/verification/verify-go-changes.yaml)
- Evidence: `Makefile:80-87`

##### Step 1

```
make test-oss
```

Expected result: The OSS Go test target completes successfully after preparing the libhoop mapping and WASM artifact.

#### [Verify React webapp changes](.software-standards/verification/verify-react-webapp.yaml) (`verify-react-webapp`)

- When: Before handing off a change beneath webapp\_v2.
- Route when: `task:verification`
- Applies to: `webapp_v2/**/*`
- Category: `testability`
- Canonical recipe: [.software-standards/verification/verify-react-webapp.yaml](.software-standards/verification/verify-react-webapp.yaml)
- Evidence: `webapp_v2/package.json:6-12`

##### Step 1

Working directory: `webapp_v2`

```
npm run lint
```

Expected result: ESLint completes without errors.

##### Step 2

Working directory: `webapp_v2`

```
npm run build
```

Expected result: Vite produces the production webapp build without errors.

### Agent Skills

#### [Add feature flag](.agents/skills/add-feature-flag/SKILL.md) (`add-feature-flag`)

Add and apply a Hoop feature flag without exposing new behavior by default.

- Use when: `task:implementation`
- Applies to: `common/featureflag/**/*.go`, `gateway/**/*.go`, `agent/**/*.go`, `client/**/*.go`, `webapp/**/*.cljs`, `webapp_v2/**/*.js`, `webapp_v2/**/*.jsx`
- Related recipe: [Verify Go changes](.software-standards/verification/verify-go-changes.yaml)
- Related recipe: [Verify React webapp changes](.software-standards/verification/verify-react-webapp.yaml)
- Category: `reliability`
- Evidence: `CLAUDE.md:150-168`, `DEV.md:282-339`

#### [Change gateway api route](.agents/skills/change-gateway-api-route/SKILL.md) (`change-gateway-api-route`)

Change a Hoop gateway API route while preserving access, analytics, and generated API contracts.

- Use when: `task:implementation`
- Applies to: `gateway/api/**/*.go`, `gateway/analytics/**/*.go`
- Related rule: [Preserve analytics event continuity](.software-standards/rules/preserve-analytics-event-continuity.md)
- Related recipe: [Regenerate OpenAPI artifacts](.software-standards/verification/regenerate-openapi-artifacts.yaml)
- Related recipe: [Verify Go changes](.software-standards/verification/verify-go-changes.yaml)
- Category: `compatibility`
- Evidence: `CLAUDE.md:38-48`, `CLAUDE.md:189-199`, `Makefile:148-150`
<!-- software-standards-bootstrap:end -->

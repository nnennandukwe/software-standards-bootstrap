<!-- software-standards-bootstrap:start -->
<!-- source-digest: sha256:63f4e7ad2cf949f0af4b777e42a2b990e85b4dfaedcdf31146f1d4c3fd86ba23 -->
<!-- content-digest: sha256:794cbebfe92ccca722008a9b0f384bb62adf7c31993e55644955d93149b3a790 -->
## Software Standards Bootstrap

Generated from `.software-standards/report.md` and its accepted artifacts by `ssb render`. Edit canonical sources and the manifest together, then rerun the command.

Baseline: `3e0091b89fdaa3912f4f8e7b33fbb3104e47d71e`

### How to apply these standards

- A semantic rule is active only when its affected path scope matches. For contextual artifacts, every represented lens dimension must also match; values within one dimension are alternatives.
- If the language, framework, task, or affected path is uncertain, load the potentially relevant rule, recipe, or skill instead of excluding it.
- Directives mean: `never` is prohibited, `ask-first` requires developer authorization, `always` is required, and `prefer` is the default when no documented exception or explicit user direction applies.
- Linked artifact files are canonical. This projection is a concise router, not a replacement for their complete content.
- Verification recipes record existing commands for deliberate use. `ssb` did not execute them.

### Standing orders

#### Never

##### Keep libhoop independent (`keep-libhoop-independent`)

- Source: [.software-standards/rules/keep-libhoop-independent.md](.software-standards/rules/keep-libhoop-independent.md)
- Scope: `_libhoop/**/*.go`
- Category: `architecture`
- Evidence: `CLAUDE.md:66-68`

Do not import packages from `gateway/`, `agent/`, `client/`, or `common/` into `_libhoop/`. Bridge across the library boundary with standard-library types.

#### Always

##### Preserve analytics event continuity (`preserve-analytics-event-continuity`)

- Source: [.software-standards/rules/preserve-analytics-event-continuity.md](.software-standards/rules/preserve-analytics-event-continuity.md)
- Scope: `gateway/api/**/*.go`, `gateway/analytics/**/*.go`
- Category: `correctness`
- Evidence: `CLAUDE.md:194-199`
- Related skill: [Change gateway api route](.agents/skills/change-gateway-api-route/SKILL.md)

When a route is added, duplicated, or superseded for an already tracked action, preserve its analytics event or document an intentional successor. Use constants from `gateway/analytics/events.go`; never replace them with event-name string literals or silently remove the final emission site.

##### Synchronize environment configuration (`synchronize-environment-configuration`)

- Source: [.software-standards/rules/synchronize-environment-configuration.md](.software-standards/rules/synchronize-environment-configuration.md)
- Scope: `.env.sample`, `gateway/**/*.go`, `agent/**/*.go`, `deploy/helm-chart/chart/gateway/**/*`, `deploy/helm-chart/chart/agent/**/*`
- Category: `operability`
- Evidence: `CLAUDE.md:201-208`

Keep runtime environment-variable reads synchronized with the matching gateway or agent Helm values, secret pass-through, user-facing chart documentation, and `.env.sample` in the same change.

##### Preserve transport plugin order (`preserve-transport-plugin-order`)

- Source: [.software-standards/rules/preserve-transport-plugin-order.md](.software-standards/rules/preserve-transport-plugin-order.md)
- Scope: `gateway/main.go`
- Category: `correctness`
- Evidence: `CLAUDE.md:83-93`, `gateway/main.go:186-194`

Preserve transport plugin registration in this order: review, audit, DLP, access control, webhooks, then Slack. Treat any reordering as a behavior change that requires explicit lifecycle analysis.

##### Use protocol packet constants (`use-protocol-packet-constants`)

- Source: [.software-standards/rules/use-protocol-packet-constants.md](.software-standards/rules/use-protocol-packet-constants.md)
- Scope: `agent/**/*.go`, `client/**/*.go`, `gateway/**/*.go`, `common/**/*.go`
- Category: `compatibility`
- Evidence: `CLAUDE.md:59-63`

Represent protocol packet types with constants from `common/proto/{agent,client,gateway,system}`. Extend the appropriate constant package instead of introducing packet-type string literals.

### Verification recipes

- [Regenerate OpenAPI artifacts](.software-standards/verification/regenerate-openapi-artifacts.yaml) (`regenerate-openapi-artifacts`) — category: `documentation`; lenses: `task:verification`; scope: `gateway/api/**/*.go`, `gateway/api/openapi/**/*`
  - When: After changing an API route, handler annotation, or schema and before handoff.
  - Evidence: `Makefile:148-150`
  - Related skill: [Change gateway api route](.agents/skills/change-gateway-api-route/SKILL.md)

- [Verify Go changes](.software-standards/verification/verify-go-changes.yaml) (`verify-go-changes`) — category: `testability`; lenses: `task:verification`; scope: `**/*.go`, `**/go.mod`, `go.work`
  - When: Before handing off a change to Go code or module metadata.
  - Evidence: `Makefile:80-87`

- [Verify React webapp changes](.software-standards/verification/verify-react-webapp.yaml) (`verify-react-webapp`) — category: `testability`; lenses: `task:verification`; scope: `webapp_v2/**/*`
  - When: Before handing off a change beneath webapp_v2.
  - Evidence: `webapp_v2/package.json:6-12`

### Agent Skills

- [Add feature flag](.agents/skills/add-feature-flag/SKILL.md) — description: Add and apply a Hoop feature flag without exposing new behavior by default.; category: `reliability`; lenses: `task:implementation`; scope: `common/featureflag/**/*.go`, `gateway/**/*.go`, `agent/**/*.go`, `client/**/*.go`, `webapp/**/*.cljs`, `webapp_v2/**/*.js`, `webapp_v2/**/*.jsx`
  - Evidence: `CLAUDE.md:150-168`, `DEV.md:282-339`
  - Related recipe: [Verify go changes](.software-standards/verification/verify-go-changes.yaml)
  - Related recipe: [Verify react webapp](.software-standards/verification/verify-react-webapp.yaml)

- [Change gateway api route](.agents/skills/change-gateway-api-route/SKILL.md) — description: Change a Hoop gateway API route while preserving access, analytics, and generated API contracts.; category: `compatibility`; lenses: `task:implementation`; scope: `gateway/api/**/*.go`, `gateway/analytics/**/*.go`
  - Evidence: `CLAUDE.md:38-48`, `CLAUDE.md:189-199`, `Makefile:148-150`
  - Related recipe: [Regenerate openapi artifacts](.software-standards/verification/regenerate-openapi-artifacts.yaml)
  - Related recipe: [Verify go changes](.software-standards/verification/verify-go-changes.yaml)
<!-- software-standards-bootstrap:end -->

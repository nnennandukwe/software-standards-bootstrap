---
schema: ssb.dev/rule/v2
id: preserve-analytics-event-continuity
title: Preserve analytics event continuity
category: correctness
lenses:
  - kind: base
directive: always
scopes:
  - "gateway/api/**/*.go"
  - "gateway/analytics/**/*.go"
derivation: extracted
evidence:
  - role: declares
    path: CLAUDE.md
    lines: 194-199
    excerpt_sha256: sha256:464543a2049637e81c764f90b0e61b2aabe5082857e58a004476419f0e7464a7
---
When a route is added, duplicated, or superseded for an already tracked action, preserve its analytics event or document an intentional successor. Use constants from `gateway/analytics/events.go`; never replace them with event-name string literals or silently remove the final emission site.

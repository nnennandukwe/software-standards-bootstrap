---
schema: ssb.dev/rule/v2
id: synchronize-environment-configuration
title: Synchronize environment configuration
category: operability
lenses:
  - kind: base
directive: always
scopes:
  - ".env.sample"
  - "gateway/**/*.go"
  - "agent/**/*.go"
  - "deploy/helm-chart/chart/gateway/**/*"
  - "deploy/helm-chart/chart/agent/**/*"
derivation: extracted
evidence:
  - role: declares
    path: CLAUDE.md
    lines: 201-208
    excerpt_sha256: sha256:ca672d5725a89a6d76fe9102c3006273294ac73dbd8f333baeaabac474f02c74
---
Keep runtime environment-variable reads synchronized with the matching gateway or agent Helm values, secret pass-through, user-facing chart documentation, and `.env.sample` in the same change.

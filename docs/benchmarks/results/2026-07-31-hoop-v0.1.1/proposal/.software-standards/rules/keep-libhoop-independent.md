---
schema: ssb.dev/rule/v2
id: keep-libhoop-independent
title: Keep libhoop independent
category: architecture
lenses:
  - kind: base
directive: never
scopes:
  - "_libhoop/**/*.go"
derivation: extracted
evidence:
  - role: declares
    path: CLAUDE.md
    lines: 66-68
    excerpt_sha256: sha256:12b6d4208a4065eff938b31047d238da39fba486a4f3e796d98a0b66e5ba2b30
---
Do not import packages from `gateway/`, `agent/`, `client/`, or `common/` into `_libhoop/`. Bridge across the library boundary with standard-library types.

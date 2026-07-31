---
schema: ssb.dev/rule/v2
id: use-protocol-packet-constants
title: Use protocol packet constants
category: compatibility
lenses:
  - kind: base
directive: always
scopes:
  - "agent/**/*.go"
  - "client/**/*.go"
  - "gateway/**/*.go"
  - "common/**/*.go"
derivation: extracted
evidence:
  - role: declares
    path: CLAUDE.md
    lines: 59-63
    excerpt_sha256: sha256:a42ec603b3e6dfef5a6e34d49db07088acb9e16a7797df8815b2620dd1c398bd
---
Represent protocol packet types with constants from `common/proto/{agent,client,gateway,system}`. Extend the appropriate constant package instead of introducing packet-type string literals.

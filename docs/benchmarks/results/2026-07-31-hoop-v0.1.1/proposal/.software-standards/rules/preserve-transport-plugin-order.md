---
schema: ssb.dev/rule/v2
id: preserve-transport-plugin-order
title: Preserve transport plugin order
category: correctness
lenses:
  - kind: base
directive: always
scopes:
  - "gateway/main.go"
derivation: extracted
evidence:
  - role: declares
    path: CLAUDE.md
    lines: 83-93
    excerpt_sha256: sha256:bf08e79e6078cac03b6cab38f78b568c8332a0ecbb17f9b03c4a8bd1db8efa45
  - role: demonstrates
    path: gateway/main.go
    lines: 186-194
    excerpt_sha256: sha256:cec9d932feb286f9b6256137f56b960a2945a86556d7ddd776b35150eccc15b6
---
Preserve transport plugin registration in this order: review, audit, DLP, access control, webhooks, then Slack. Treat any reordering as a behavior change that requires explicit lifecycle analysis.

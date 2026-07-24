# Rule schema quick reference

Required frontmatter:

```yaml
schema: ssb.dev/rule/v1
id: lower-kebab-case
title: Developer-facing title
topic: correctness
scopes:
  - "repository/relative/**"
classification: guidance # or deterministic
importance: high
score:
  method: ssb-score-v1
  total: 70
  factors:
    prevalence: 15
    consistency: 15
    authority: 15
    risk: 15
    applicability: 10
confidence: high
baseline_commit: <exact inspect baseline>
evidence:
  - path: tracked/file
    lines: 10-14
    excerpt_sha256: sha256:<64 lowercase hex>
    authoritative: true
verification:
  command: existing command
  source:
    path: tracked/check-definition
    lines: 1-4
    excerpt_sha256: sha256:<64 lowercase hex>
related_skills:
  - procedural-skill
```

`topic` is required and must be exactly one value from: `architecture`, `compatibility`, `compliance`, `correctness`, `developer-experience`, `documentation`, `maintainability`, `operability`, `performance`, `quality`, `reliability`, `security`, or `testability`. Choose the concern that best explains the rule's engineering risk or change obligation. Use `quality` only when no narrower topic fits.

For guidance without an existing check:

```yaml
verification:
  proof_gap: Explain the precise unautomated boundary.
```

Factor ranges are 0–25 prevalence, 0–20 consistency, 0–20 authority, 0–20 risk, and 0–15 applicability. Bands are 80–100 very-high, 65–79 high, 45–64 medium, and 25–44 low.

Referenced Agent Skills must set `metadata.topic` to one value from the same taxonomy, based on the workflow's primary engineering outcome.

The Markdown after the closing `---` is the exact rule body. Unknown fields, duplicate keys, score mismatch, stale baseline, missing evidence, hash mismatch, unsafe scopes, unsupported topic or classification, and missing related skills fail validation.

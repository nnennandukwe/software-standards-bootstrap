# Governed lifecycle review

This workflow proposes lifecycle decisions for an adopted pack. It is not
automatic cleanup. The CLI provides deterministic inventory and validation;
you provide bounded semantic analysis.

## Input gate

Use only the exact `context.json` created by `ssb prune inspect`. Stop when:

- inventory is truncated or missing;
- any canonical rule or repository skill cannot be read from the inventory;
- the capability profile is absent, invalid, or lacks conformance evidence for
  a claimed supported/unsupported capability;
- the baseline changed; or
- evidence needed for a decision is absent.

Never infer provenance. `unknown` means the only valid disposition is
`unable-to-determine`.

## Evaluation passes

First evaluate rules independently:

- Does repository evidence still support the stated scope and directive?
- Does the rule duplicate or contradict another rule?
- Does its mapped verification still exist, and what does it actually prove?
- Does the pinned host/model capability make the guidance redundant or
  unsupported according to conformance evidence?

Then evaluate skills independently:

- Is the workflow still discoverable by the pinned host?
- Is it genuinely procedural and repository-specific?
- Is it stale, duplicated, unsupported, or orphaned?
- Is an apparent orphan intentionally user-authored?

Finally evaluate relationships:

- rule-to-skill references and unreferenced skills;
- the full tracked bundle beneath each skill directory, including references,
  scripts, and assets;
- rule pairs that should be consolidated;
- rule and skill changes that must be approved together; and
- derived `AGENTS.md` impact.

## Proposal contract

Write `.software-standards/reviews/<review-id>/proposal.yaml` with schema
`ssb.dev/prune-proposal/v1`. Cover every context artifact exactly once.

```yaml
schema: ssb.dev/prune-proposal/v1
review_id: example-review
context_digest: sha256:<digest from context.json>
actions:
  - id: keep-example
    disposition: keep
    sources:
      - kind: rule
        id: example
        path: .software-standards/rules/example.md
        sha256: sha256:<artifact digest>
    rationale: Repository and pinned capability evidence still support it.
    confidence: high
    repository_evidence:
      - path: README.md
        lines: 10-15
        sha256: sha256:<full file digest from context inventory>
    capability_refs: [repository-instruction-discovery]
    required_verification:
      - id: repository-tests
        command: go test ./...
```

Allowed dispositions:

- `keep`: exactly one source, no target;
- `update`: exactly one source and one complete target;
- `consolidate`: at least two same-kind sources and one complete target;
- `remove`: exactly one source, no target; and
- `unable-to-determine`: exactly one source, no target, and one or more
  `unresolved_questions`.

For update and consolidation, write the complete replacement below
`candidates/<action-id>/` and provide:

```yaml
target:
  kind: rule
  id: replacement-id
  target_path: .software-standards/rules/replacement-id.md
  source_path: candidates/<action-id>/replacement-id.md
  sha256: sha256:<complete candidate digest>
  mode: "100644" # use "100755" only when the entrypoint is executable
```

For a skill replacement, `target` must also list every file in the complete
replacement bundle:

```yaml
supporting_files:
  - target_path: .agents/skills/replacement/references/workflow.md
    source_path: candidates/<action-id>/references/workflow.md
    sha256: sha256:<complete supporting file digest>
    mode: "100644"
```

Omitted old support is removed. Do not propose an actionable disposition when
any tracked skill file is absent from the bounded inventory.

Every actionable disposition requires non-empty rationale, an honest
`low`/`medium`/`high` confidence band, at least one repository evidence
reference, and at least one non-unknown capability reference. Use
`dependencies` when actions must be approved together.

Every actionable proposal action must map at least one exact external check
under `required_verification`; do not execute it or fabricate its later
receipt.

Do not edit `context.json` or append `events.jsonl`. Do not approve a proposal,
apply its files, rerender `AGENTS.md`, create an ADR, execute verification
commands, fabricate receipts, or perform Git mutations.

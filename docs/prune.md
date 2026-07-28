# Governed rule and skill lifecycle review

`prune` is a working product term, not a promise that cleanup is automatic or
that the current command layout is permanent. The implemented protocol makes
stale-rule discovery tangible while keeping semantic judgment and destructive
changes under review.

## Trust model

A review starts from:

- one attached, clean, commit-backed `HEAD`;
- a complete bounded inventory;
- every `.software-standards/rules/*.md` file;
- every `.agents/skills/*/SKILL.md` file, referenced or not;
- a local `ssb.dev/capability-profile/v1` document; and
- an optional local `ssb.dev/artifact-provenance/v1` declaration.

The capability profile is a point-in-time snapshot. It must name the exact host
version and provider/model ID. A supported or unsupported capability needs
content-addressed conformance evidence stored beside the profile. Release notes
alone are not conformance evidence. The CLI does not download profiles, query a
registry, infer “latest,” verify vendor signatures, or expire profiles.

Expiry and supersession remain open product questions. Version 1 profiles do
not expire automatically, and a reviewer must explicitly select a different
snapshot. This behavior is not evidence that an older profile remains current.

Inspection copies the exact profile, every referenced evidence file, and the
optional provenance manifest beneath `reviews/<id>/inputs/`. `context.json`
binds those copies by path and digest, so the review remains auditable after
the caller's original input directory is removed.

Provenance is explicit and digest-bound. An absent declaration is `unknown`;
the tool never guesses that a file was generated or user-authored. Unknown
provenance forces `unable-to-determine`.

## State and command contract

```text
ssb prune inspect --repo . --review <id> --capabilities <profile> [--provenance <manifest>]
# Host agent writes reviews/<id>/proposal.yaml and complete candidate files.
ssb prune validate --repo . --review <id>
ssb prune approve --repo . --review <id> --approve <ids> --reject <ids>
ssb prune apply --repo . --review <id>              # dry run
ssb prune apply --repo . --review <id> --write
# Only after confirming the prior process is gone and a journal exists:
ssb prune recover --repo . --review <id> [--clear-stale-lock]
ssb render --repo . --review <id>                   # when rules changed
ssb adr --repo . --review <id>                      # optional
ssb prune verify --repo . --review <id> --receipts <directory>
ssb prune status --repo . --review <id>
```

Validation and status keep routine output compact: counts are grouped by
artifact kind and disposition, followed by one concise row per configuration.
The full context, rationale, evidence, candidates, and event payloads remain in
the review bundle for on-demand inspection.

Every action is exactly one of `keep`, `update`, `consolidate`, `remove`, or
`unable-to-determine`. Every artifact appears in exactly one action. Update and
consolidate targets point to complete candidate files beneath the review
bundle; no fuzzy patch is trusted.

Approval is a single event that lists every action as approved or rejected and
binds the exact proposal digest. Dependencies must be approved together.
Unable-to-determine cannot be approved. Application refuses a changed `HEAD`,
tracked/staged drift, changed sources, changed candidates, path collisions, and
a result with no rules. It writes an application recovery journal before any
file operation.

A skill disposition covers its tracked bundle: `SKILL.md` plus tracked files
beneath the same skill directory. Dry-run output lists every affected file.
Moving, consolidating, or removing a skill therefore cannot silently leave
stale references, scripts, or assets behind.

Skill update and consolidation candidates are bundles too. The target lists a
complete replacement `SKILL.md` and every supporting file to install; omitted
old supporting files are removed. Each candidate file is individually
path-bound, mode-bound, size-bounded, and content-addressed. The complete
candidate set also stays within the review's file/byte budget.

Candidate modes use Git tree values `100644` and `100755`. POSIX hosts can
materialize either mode. Windows can materialize `100644`, but rejects
`100755` during proposal validation because the executable bit would require
an index-only Git change. `ssb` never stages that change or silently downgrades
the approved mode. Apply executable candidates from a POSIX host instead.

All tracked skill files must be eligible for the bounded text inventory.
Binary, generated, secret-like, vendored, non-regular, or oversized support
fails inspection closed instead of being read outside the inventory boundary.

Rerender, ADR recording, and verification are independent events. An ADR is
optional. Rule-changing applications must record rerendering before
verification.

## Evidence contracts

Repository evidence uses an inventory-listed path, a one-based line range, and
the exact full-file digest from `context.json`. Capability references name
observed entries in the pinned profile. Every actionable disposition requires
both kinds plus rationale and a confidence band.

An actionable proposal must also map at least one exact external check through
`required_verification`. Without a passing, content-addressed receipt for every
approved required check, the review cannot enter the verified state.

External checks are never executed by `ssb`. Each required check uses a receipt
named `<check-id>.yaml`:

```yaml
schema: ssb.dev/prune-check-receipt/v1
review_id: example-review
proposal_digest: sha256:<64 lowercase hex characters>
check_id: go-test
command: go test ./...
status: passed
observed_at: 2026-07-27T18:00:00Z
evidence:
  - path: logs/go-test.txt
    sha256: sha256:<64 lowercase hex characters>
```

Receipt evidence paths are relative to the receipt directory and must match
their bytes.

## Capability profile example

```yaml
schema: ssb.dev/capability-profile/v1
id: codex-example
host: {name: codex, version: 1.2.3}
model: {provider: openai, id: gpt-example}
observed_at: 2026-07-27T18:00:00Z
evidence:
  - id: skill-discovery-run
    kind: conformance
    path: skill-discovery.json
    sha256: sha256:<64 lowercase hex characters>
capabilities:
  - id: repository-skill-discovery
    status: supported
    evidence_ids: [skill-discovery-run]
```

## Evaluation matrix

The protocol should be exercised against:

| Repository case | Host case | Expected demonstration |
|---|---|---|
| Existing pack with a stale generated rule | Pinned Codex profile | Evidence-backed update/remove proposal and reviewed diff |
| Existing pack with an unreferenced user skill | Pinned Claude Code profile | Skill remains visible and cannot be silently deleted |
| Contradictory rule pair | Two independently pinned host profiles | Consolidation rationale changes only with observed evidence |
| Missing provenance | Any host | Unable-to-determine and blocked approval |
| Truncated large repository inventory | Any host | Exit 4 and no review bundle |
| Greenfield repository with no adopted pack | Any host | Prune inspection blocked; generation remains the entry point |

The concrete value demonstration is a before/after Git diff whose every changed
file traces through context evidence, a validated disposition, explicit human
approval, application, rerendering, and independent check receipts.

## Non-goals

- Automatic synchronization with an online model registry.
- Treating vendor release notes as sufficient evidence.
- Unreviewed deletion or rewriting.
- Replacing architectural decisions with an unconstrained LLM prompt.
- Treating the `prune` name or current CLI/implementation architecture as final.

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

Provenance is explicit and digest-bound. A rule needs one declaration for its
exact bytes. A skill needs one declaration for `SKILL.md` and every tracked
supporting file in its bundle. A partial declaration makes the whole skill
`unknown`; differing complete declarations derive `mixed`. The tool never
guesses that a file was generated or user-authored. Unknown provenance forces
`unable-to-determine`.

Ignored files do not escape this boundary. Inspection checks all untracked
paths below the governed rule and skill roots, including paths matched by Git
exclude rules, and stops until they are committed, moved, or removed.

## State and command contract

```text
ssb prune inspect --repo . --review <id> --capabilities <profile> [--provenance <manifest>]
# Host agent writes reviews/<id>/proposal.yaml and complete candidate files.
ssb prune validate --repo . --review <id>
ssb prune approve --repo . --review <id> --approve <ids> --reject <ids>
ssb prune apply --repo . --review <id>              # dry run
ssb prune apply --repo . --review <id> --write
# Only after confirming the prior process is gone:
ssb prune recover --repo . --review <id> [--clear-stale-lock]
ssb render --repo . --review <id>                   # when rules changed
ssb adr --repo . --review <id>                      # optional
ssb prune verify --repo . --review <id> --receipts <directory>
ssb prune status --repo . --review <id>
```

Capability profiles, provenance manifests, and receipt directories may be
absolute or relative to the process working directory. Their referenced
evidence remains relative to the containing profile, manifest, or receipt
directory.

Validation and status keep routine output compact: counts are grouped by
artifact kind and disposition, followed by one concise row per configuration.
The full context, rationale, evidence, candidates, and event payloads remain in
the review bundle for on-demand inspection.

Every action is exactly one of `keep`, `update`, `consolidate`, `remove`, or
`unable-to-determine`. Every artifact appears in exactly one action. Update and
consolidate targets point to complete candidate files beneath the review
bundle; no fuzzy patch is trusted. Validation checks candidate rule/skill
contracts and the proposed resulting rule-skill graph. Approval repeats that
preflight for the exact accepted decision set before it can record an event.

Approval is a single event that lists every action as approved or rejected and
binds the exact proposal digest. Dependencies must be approved together.
Unable-to-determine cannot be approved. Approval is not recorded when its
decisions cannot produce a safe application plan. Removing an artifact also
removes its accepted index entry and inbound relationships in the same plan.
In a manifest-layout pack this updates `.software-standards/manifest.yaml`; in
an embedded-layout pack it updates `.software-standards/report.md`. Manifest
updates refresh the exact primary-file digest while leaving category,
directive, scopes, derivation, evidence, confidence, and utility unchanged. A
valid skill-only or zero-artifact result is allowed. Replacement actions that
change a canonical artifact ID or path are rejected because lifecycle review
cannot invent fresh provenance, confidence, or utility for the new manifest
entry.
Application refuses a changed `HEAD`, tracked/staged drift, changed sources,
changed candidates, and path collisions. It writes an application recovery
journal before any file operation.

Dry run and write derive the same canonical application plan. Each operation
contains the exact prestate and poststate, including presence, digest, and
mode. The plan digest also binds the baseline, context, proposal, and approval
event. Event replay, journaling, recovery, and verification consume that same
plan rather than reconstructing separate action projections.

When no applicable changes are approved, `apply --write` records no
application event. Status reports `no_changes_approved`; rerendering and
verification are not applicable.

A skill disposition covers its tracked bundle: `SKILL.md` plus tracked files
beneath the same skill directory. Dry-run output lists every affected file.
Moving, consolidating, or removing a skill therefore cannot silently leave
stale references, scripts, or assets behind.

Skill update and consolidation candidates are bundles too. The target lists a
complete replacement `SKILL.md` and every supporting file to install; omitted
old supporting files are removed. Each candidate file is individually
path-bound, mode-bound, size-bounded, and content-addressed. The complete
candidate set also stays within the review's file/byte budget.

Every tracked governed path, plus review-bundle, evidence, candidate, and
target paths, uses one portable slash-separated spelling. Validation rejects
traversal, backslashes, drive or stream colons, Windows-reserved device names,
invalid Windows filename characters, case-fold collisions, and components
ending in a dot or space before filesystem access. The review directory and
durable capability/provenance snapshot paths must also remain real
non-symlink directories inside the repository. Lifecycle operations pin the
expected review and parent directory identities so a symlink to an external or
in-repository sibling cannot redirect publication, events, locks, or journals.

Candidate modes use Git tree values `100644` and `100755`. POSIX hosts can
materialize either mode. Windows can materialize `100644`, but rejects
`100755` during proposal validation because the executable bit would require
an index-only Git change. `ssb` never stages that change or silently downgrades
the approved mode. Application planning also rejects removal or replacement of
an existing `100755` file on Windows, before reporting a writable dry run.
Apply any review that changes an executable file from a POSIX host instead.

All tracked skill files must be eligible for the bounded text inventory.
Binary, generated, secret-like, vendored, non-regular, or oversized support
fails inspection closed instead of being read outside the inventory boundary.

Rerender, ADR recording, and verification are independent events. An ADR is
optional. Rule-changing applications must record rerendering before
verification. Review-aware rerendering and ADR creation validate adopted rules
against each rule's recorded historical baseline; missing historical commits
and commits outside current history fail closed. Verification binds the
rerender event to the complete rendered
`AGENTS.md` bytes and rejects symlinked output. A skill-only review does not
require rerendering, but if a render event is explicitly recorded, receipts
and verification bind it too. Rerendering cannot be added after verification;
additional rendered changes require a new review.

## Evidence contracts

Repository evidence uses an inventory-listed path, a one-based line range, and
the exact full-file digest from `context.json`. Capability references name
observed entries in the pinned profile. Every actionable disposition requires
both kinds plus rationale and a confidence band.

Unable-to-determine requires at least one `evidence_gaps` entry. Each gap names
the missing evidence category, an exact source artifact path, and the missing
fact. An unresolved question without this structured context is insufficient.
For example:

```yaml
evidence_gaps:
  - kind: provenance
    artifact_path: .agents/skills/example/SKILL.md
    detail: No provenance declaration matches the inventoried bytes.
```

An actionable proposal must also map at least one exact external check through
`required_verification`. Without a passing, content-addressed receipt for every
approved required check, the review cannot enter the verified state.

External checks are never executed by `ssb`. Each required check uses a receipt
named `<check-id>.yaml`:

```yaml
schema: ssb.dev/prune-check-receipt/v1
review_id: example-review
proposal_digest: sha256:<64 lowercase hex characters>
application_event_digest: sha256:<exact applied event digest>
plan_digest: sha256:<canonical application plan digest>
render_event_digest: sha256:<required whenever a render event was recorded>
check_id: go-test
command: go test ./...
status: passed
observed_at: 2026-07-27T18:00:00Z
evidence:
  - path: logs/go-test.txt
    sha256: sha256:<64 lowercase hex characters>
```

Receipt evidence paths are relative to the receipt directory and must match
their bytes. A receipt must be observed no earlier than its exact application
event. Verification recomputes the complete governed rule/skill poststate,
checks removals and modes, rejects additional configuration paths, and checks
the recorded `AGENTS.md` render bytes when rules changed.

## Recovery

Application retains its journal until both the review lock and repository
mutation lock are released. Cleanup failures are returned with an exact
recovery command. After confirming no process owns the review, use
`--clear-stale-lock`; this may clear review-owned locks even when a crash
occurred after journal removal. Recovery never treats third-party bytes as an
approved prestate or poststate.

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

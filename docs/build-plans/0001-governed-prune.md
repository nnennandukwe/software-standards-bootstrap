# Build plan 0001: Governed rule and skill lifecycle review

- Status: Approved for draft implementation
- Date: 2026-07-27
- Issue: [#10](https://github.com/nnennandukwe/software-standards-bootstrap/issues/10)

## Decision boundary

This plan authorizes a reviewable draft implementation that tests the
architecture proposed in issue #10. It does not select `prune` as the final
product name, approve merge or release, or settle the long-term command and
trust architecture. Those decisions depend on evaluation evidence.

## Evidence used

- The existing Go CLI separates inventory, validation, rendering, and ADR
  generation.
- Rule schemas v1 and v2 already bind repository evidence and baseline commits.
- The repository Agent Skill contract is a complete `SKILL.md` plus optional
  tracked supporting files.
- Inventory v2 provides a bounded, commit-backed input and reports truncation.
- Issue #10 requires fail-closed evidence, dry-run-first behavior, explicit
  approval, and separate lifecycle states.

## Command contract

Add a namespaced experimental workflow:

1. `ssb prune inspect` creates an immutable review context.
2. A host agent writes a complete proposal and any complete candidates.
3. `validate` checks coverage, evidence, candidates, and verification mappings.
4. `approve` records an explicit decision for every action.
5. `apply` defaults to dry run; `--write` applies only approved changes.
6. Existing `render` and `adr` commands may record their own review states.
7. `verify` accepts external receipts without executing their commands.
8. `status` reports every state independently; `recover` restores interrupted
   application bytes.

Help text and exit codes are part of the draft contract. Evaluation may justify
renaming or restructuring it before release.

## Trust model

Use local, point-in-time capability profiles naming exact host and model
versions. Bind supported and unsupported claims to content-addressed
conformance evidence and copy the inputs into the review bundle. Do not query
an online registry, infer a latest model, or treat release notes as sufficient
evidence. Unknown provenance or capability evidence yields
`unable-to-determine`.

Signed imported snapshots, profile expiry, supersession, and other distribution
models remain research questions; the draft deliberately does not imply trust
in them.

## Intermediate representations

Introduce separately versioned schemas for:

- immutable review context and artifact inventory;
- capability profile and optional provenance manifest;
- proposal actions and complete candidates;
- digest-chained lifecycle events;
- recovery journal; and
- external verification receipts.

Keep rule schemas v1 and v2 unchanged. Treat a skill as `SKILL.md` plus every
tracked, bounded-inventory-eligible file beneath its skill directory for
application and recovery. Replacement skill candidates carry an explicit,
complete entrypoint and supporting-file set.

## Compatibility and migration

Existing inspect, validate, render, and ADR behavior remains available without
a review ID. Existing rule packs need no schema migration. Repositories without
provenance may inspect, but unknown artifacts cannot receive an actionable
disposition. Repositories without an adopted pack remain on the generation
workflow.

## Failure modes

Fail closed for incomplete inventory, invalid or missing capability evidence,
unknown provenance, stale baseline, tracked worktree drift, changed source or
candidate bytes, path collisions, incomplete decisions, dependency mismatch,
invalid resulting rule-skill graphs, interrupted writes, malformed event
history, and missing or mismatched verification receipts.

Use exclusive review creation, transition locks, atomic event-log replacement,
a pre-mutation recovery journal, exact pre/post-state recovery checks, and
explicit stale-lock recovery.

## Evaluation matrix

Evaluate at least:

- a stale generated rule on a pinned Codex host profile;
- an unreferenced user-authored skill on a pinned Claude Code profile;
- contradictory rules across two independently pinned host profiles;
- missing provenance;
- truncated large-repository inventory; and
- a greenfield repository with no adopted pack.

The tangible demonstration is a before/after diff where every changed file
traces to pinned context, rationale and evidence, an explicit human decision,
application, any required rerender, and external check receipts.

## Tests and documentation

Cover schema strictness, inventory completeness, proposal cardinality,
provenance and capability failure paths, dry-run behavior, approval binding,
candidate and graph validation, skill bundles, atomic events, concurrency,
recovery, state ordering, receipt verification, CLI parity, exit codes, and
rollback.

Document the trust model, state machine, schemas, recovery procedure,
evaluation gaps, Agent Skill workflow, and non-goals. Run unit tests, race
tests, vet, build, diff checks, Agent Skill validation, and independent
correctness and quality reviews before opening the draft PR.

## Approval

The draft implementation may proceed for architectural review. Merge,
distribution, or material follow-on work requires explicit approval based on
the evaluation evidence and a plan updated for the chosen public contract.

# ADR 0003: Governed rule and skill lifecycle review protocol

- Status: Proposed
- Date: 2026-07-27

## Context

Adopted rules and Agent Skills can become stale, redundant, contradictory, or
unsupported as repositories, agent hosts, and model capabilities change.
Inferring obsolescence from a model name or applying free-form model edits
would collapse evidence, judgment, approval, and mutation into one unsafe step.

## Decision

Introduce a locally governed lifecycle protocol under the working name
`prune`. Pin complete repository inventory, explicit provenance declarations,
and a versioned point-in-time host/model capability profile in an immutable
review context. Let the host agent produce evidence-backed semantic
dispositions, while the Go CLI owns strict validation, digest-bound approval,
dry-run-first application, recovery, transition recording, and external
receipt verification.

Keep rule v1/v2 unchanged. Lifecycle context, proposal, event, capability,
provenance, and receipt documents have separate versioned schemas. Keep
proposal, approval, application, rerendering, optional ADR creation, and
verification as distinct states.

Derive one canonical, content-addressed application plan after approval. The
plan contains exact file prestates and poststates plus the complete final
governed configuration. Use it for dry run, mutation, event validation,
recovery, and verification. External receipts bind the exact application
event and plan; rule-changing reviews also bind the rerender event.

An unable-to-determine disposition records a structured evidence gap for an
exact artifact. A review with no applicable approved changes reaches a derived
no-change outcome and does not create application or verification events.

## Consequences

The workflow is reviewable and works without network or registry trust.
Generated and user-authored artifacts remain visible; unknown provenance fails
closed. The host agent remains responsible for semantic quality, so evaluation
must cover representative repositories and pinned host profiles. The public
name and command architecture may change after that evaluation.

For skills, provenance covers `SKILL.md` and every tracked supporting file.
Partial declarations yield unknown provenance; complete declarations with
different origins yield mixed provenance. Ignored untracked governed paths
block inventory rather than disappearing through Git exclude rules.

Application is intentionally conservative: one review binds one baseline and
one proposal, all decisions are explicit, complete candidate files replace
fuzzy patches, unrelated tracked changes block writes, and recovery data is
created before mutation. Portable paths are validated independently of the
current operating system. Recovery data remains until locks are released, and
explicit stale-lock recovery covers journal-free crash windows.

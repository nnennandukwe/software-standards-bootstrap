# 0001: Evidence-derived inventory resource limits

- Status: Proposed
- Date: 2026-07-24

## Context

Inventory v1 limited output to 20,000 indexed files and 25 MiB of indexed
text. The repository described those values only as conservative. They did not
bound actual work because binary and generated blobs were read before being
excluded and therefore did not consume either limit.

The pinned release corpus also disproved the coverage assumption. Django
contains 35.11 MiB of eligible text and Next.js contains 84.54 MiB, so the
25 MiB limit retained approximately 71% and 29% of their eligible bytes.
Because Git-tree ordering determines the retained prefix, later source and test
trees were systematically unavailable.

Inspection feeds repository-wide semantic review. Incomplete coverage must not
silently advance that workflow.

## Decision

Inventory v2 will budget candidate work before blob reads.

- The defaults are 40,000 candidate files, 128 MiB of candidate content, and a
  fixed 1 MiB per file.
- Candidate totals are calculated after mode, path, secret-like, vendor, and
  oversized exclusions but before binary/generated content classification.
- Every candidate read consumes file and byte budget.
- Reports distinguish candidate, scanned, indexed, and remaining work.
- Incomplete coverage returns exit `4` after emitting the report.
- `--allow-partial` is available only for explicitly diagnostic inventory; the
  bundled proposal workflow never uses it.
- Candidate file and byte defaults may be raised through CLI flags. The
  per-file boundary remains fixed because validation uses the same evidence
  eligibility contract.
- Git blob reads use at most 512 entries and 4 MiB per batch. This was the
  lowest-memory measured configuration within 10% of the fastest configuration
  in the required sweep.

Defaults are reassessed whenever the supported corpus, scanner implementation,
minimum platform, or release benchmark set changes. The aggregate defaults
must cover at least 120% of the largest supported candidate workload, rounded
upward to a binary power of two for bytes and the next 10,000 for files.

## Consequences

- Inventory JSON advances to schema 2 and `ssb-inventory-v2`.
- Inventory-v1 consumers must update; no alias preserves the old misleading
  semantics.
- The four pinned repositories complete under the new defaults.
- A repository larger than the supported envelope fails closed with exact
  recovery information instead of producing an apparently successful partial
  proposal.
- Historical inventory-v1 benchmark records remain immutable but cannot serve
  as inventory-v2 release acceptance.
- Fresh consumer evidence is required before the first signed release.

## Alternatives considered

- Keep 25 MiB and document it: rejected because it fails two supported
  repository classes and has no workload basis.
- Remove all limits: rejected because adversarial or accidental repository
  contents could cause unbounded Git decompression and hashing work.
- Sample or prioritize directories: deferred because it changes deterministic
  coverage semantics and introduces semantic selection policy into the CLI.
- Expose the per-file limit: deferred until inspection limits can be persisted
  and validation can apply the same policy.

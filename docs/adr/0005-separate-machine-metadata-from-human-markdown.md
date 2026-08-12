# ADR 0005: Separate machine metadata from human-facing Markdown

- Status: Accepted
- Date: 2026-08-12

## Context

Published v0.1.1 packs embed the complete inventory and accepted-artifact
index in `report.md`, and embed routing and provenance frontmatter in semantic
rules. That representation is deterministic but makes the files developers
open first lead with machine data. Large repositories make the report
especially difficult to review.

The runtime still needs strict offline replay, exact provenance, portable
paths, and compatibility with packs already published as
`ssb.dev/report/v1` and `ssb.dev/rule/v2`.

## Decision

New packs use the `split-v1` layout. `ssb.dev/manifest/v1` owns the pinned
baseline, exact references to `inventory.json` and `report.md`, the accepted
artifact index, selection and provenance metadata, confidence, utility,
relationships, and SHA-256 digests of each primary file's raw bytes.

`inventory.json` is the complete, unedited schema 2 inspection response.
`report.md` has no frontmatter, starts with `# Software standards report`, and
contains narrative plus links to both machine files. Semantic rules have no
frontmatter: one opening H1 supplies the title and actionable text follows.
Portable Agent Skills retain their standard frontmatter but omit SSB-owned
category metadata.

The presence of a safe regular `manifest.yaml` selects `split-v1`. An invalid
split manifest never falls back to legacy parsing. If no manifest exists, a
frontmatter-based `ssb.dev/report/v1` pack selects `legacy-v1`. Both layouts
normalize into one internal pack. Validation never rewrites or migrates them.

Validation JSON moves to response schema 3 and names the detected layout and
its separate paths. Rendering and ADR generation consume the normalized pack.
Governed prune updates split primary-file digests and removes entries and
dangling relationships within its existing journal and rollback boundary; it
does not change manifest-owned semantic metadata.

## Consequences

- Human-facing Markdown opens on the information a developer reviews.
- Inventory size no longer determines report size or reading order.
- Exact raw-byte digests make line-ending and substitution changes explicit.
- Published v0.1.1 packs remain usable across the full lifecycle without
  conversion.
- Editing a digest-bound split source requires a matching manifest update.
- There is no migration command. Generating a fresh pack is the review path
  for changing legacy representation or manifest-owned semantic metadata.
- This decision does not redesign root `AGENTS.md`; that work remains separate.

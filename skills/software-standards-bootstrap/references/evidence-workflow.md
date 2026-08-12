# Evidence workflow

## Canonical bytes

Evidence is tied to the manifest's `baseline_commit`. Hash exact bytes from the
inclusive one-based line range, preserving line endings. Do not hash copied
prose, normalized whitespace, worktree-only edits, or model summaries.

`ssb validate` reads the same range from the pinned Git blob and reports the
expected digest when a hash is stale. It also rebuilds the recorded complete
inventory from the pinned baseline and limits.

The split manifest also hashes every complete primary artifact plus the
inventory and human report. Those digests cover raw bytes, including line
endings, rather than normalized YAML, JSON, or Markdown.

## Roles and derivation

Use evidence roles literally:

- `declares`: an explicit repository-maintained obligation;
- `demonstrates`: a concrete implementation occurrence supporting an inferred
  invariant; and
- `enforces`: a repository mechanism that actively checks a condition.

An `extracted` artifact needs at least one `declares` or `enforces` citation.
An `inferred` artifact needs at least three distinct `demonstrates` citations
across at least two files. Similar-looking code with different state or risk
boundaries is not consistent evidence.

The inventory eligibility boundary also applies during validation. Do not cite
binary, oversized, secret-like, generated/vendor, symlink, submodule, or other
excluded paths.

## Existing commands

Inspect existing commands, their invocation sites and triggers, and the exact
condition they enforce. Never run a command during generation and never claim
that it passed.

- If an existing automatic mechanism completely handles the condition, emit
  nothing.
- If developer value comes from deliberately invoking an existing command,
  emit a verification recipe with exact `enforces` evidence.
- If value comes from a semantic implementation condition, emit a semantic
  rule without command metadata.
- If an automatic check would be valuable but does not exist, emit an
  automation proposal rather than inventing a command or checker.

## Preferred examples and counterexamples

Point to a preferred example only when authority or repeated evidence
establishes it as the shape to follow. Point to a counterexample only when
repository authority identifies it as deprecated, unsafe, or intentionally
avoided. Cite locations rather than copying large blocks into `AGENTS.md`.

## Rejected candidates

Reject and discard candidates whose evidence threshold is unmet, scope is
unclear, confidence is below medium, utility is below 45, or destination is not
one of the four artifact kinds. Do not persist the candidate, reason, or count.

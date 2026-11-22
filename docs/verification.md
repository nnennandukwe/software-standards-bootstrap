# Verification contract

Release and pull-request evidence must keep proposal generation, developer
retention, projection, ADR creation, and release verification as separate
states. A file, draft, command declaration, or green summary is not proof that
a later state completed.

## Automated gates

Run:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/ssb
go tool govulncheck ./...
go run github.com/goreleaser/goreleaser/v2@v2.17.0 check
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
make verify-release-archives
uvx --from skills-ref==0.1.1 agentskills validate skills/software-standards-bootstrap
make verify
```

## Contract coverage

Tests cover:

- dirty, staged, detached, unborn, non-Git, changed-`HEAD`, and existing-pack
  inspection failures;
- literal paths, safe Git invocation, binary/generated/resource exclusions,
  and complete inventory accounting;
- exact replay of the report inventory at its pinned baseline and limits;
- strict report, semantic-rule, Agent Skill, and automation-proposal schemas;
- strict `ssb.dev/verification/v1` compatibility, rejection of v2-only fields
  in v1, and `ssb.dev/verification/v2` `working_directory` validation for root,
  nested, missing, file-valued, submodule, traversal, absolute, alternate
  separator, and Windows-volume paths;
- bounded `ssb.dev/orientation/v1` summary, areas, prerequisites, documents,
  task guidance, authoritative evidence, exact digest binding, relationship
  targets, duplicates, symlinks, paths, and schema-only behavior;
- global IDs, canonical paths, category, lenses, scopes, derivation, exact
  evidence, confidence, utility, and relationships;
- extracted and inferred evidence thresholds;
- recipe step references to exact `enforces` evidence;
- rejection of prior rule contracts and rule-owned command/check metadata;
- normalized response-schema-3 JSON containing orientation and all four
  artifact kinds, additive root/catalog measurements and warnings, normalized
  routing groups, and `working_directory: .` for verification/v1 steps;
- optional `root_core` limits and exact global-base eligibility;
- a 300-contextual-rule fixture whose unchanged root core stays below 16 KiB
  while every rule remains discoverable through exact-selector bundles;
- deterministic finite-root, catalog, and bundle bytes; root-to-tree and
  catalog-to-bundle digests; operational non-root rules and inert recipes;
  Agent Skill links; and automation omission;
- ADR inclusion of rules, recipes, and skills and exclusion of automation
  proposals;
- zero-artifact, orientation-only, and automation-only no-write behavior;
- orientation exclusion from artifact denominators and ADR eligibility;
- root or routing drift, reserved or malformed markers, dynamically safe
  Markdown fences, unsafe targets, collision-safe ADR creation, complete
  write-set dry runs, transactional write failures, and old rerender-event
  replay; and
- no mutation from inspection, validation, dry runs, or failed operations.

## Benchmark evidence

Run the blind pinned-fixture workflow in [benchmarks.md](benchmarks.md). Record
every emitted rule, verification recipe, Agent Skill, and automation proposal.
Require complete inventory, resolvable evidence, and at least 70% developer
keep or edit-and-keep across final artifacts in each pinned repository
independently. One conforming agent host satisfies the consumer gate; record
its exact version. A pooled cross-repository average is not acceptance. Use
every final artifact emitted before developer review as that fixture's
denominator; deferred and rejected artifacts remain in it, and the exact
fraction must meet the threshold without rounding.

Orientation is repository context, not an actionable artifact. It does not
enter the benchmark denominator or ADR eligibility. Contract snapshots for a
generated root file use `proposal/AGENTS.proposed.md`; that inert filename
prevents a retained benchmark fixture from becoming host instructions.

For portable routing acceptance, run the implementation, verification,
renderer, rulepack, and release scenarios in
[agent-smoke-tests.md](agent-smoke-tests.md). Retain compact evidence with the
exact Codex version and source commit, selected and excluded bundle/artifact
IDs, file hashes, and command-execution state. A generated fixture or green
unit test is not agent-host conformance evidence.

The current [v0.2.0 actionable-artifact record](benchmarks/results/2026-08-17-v0.2.0-actionable/README.md)
retains exactly one run record, final inert projection, and `Proposed` ADR per
fixture, plus the result README. Its contract test enforces the exact 13-file
tree; fixture pins and inventory/exclusion counts; all 21 artifact IDs,
decisions, and pre-review/final hashes; source and projection digests; SHA-256
and Git-blob bindings for retained files; per-fixture retention arithmetic;
and orientation/automation exclusion from ADRs. Raw inventories, manifests,
reports, orientations, canonical artifacts, transcripts, logs, and disposable
propagation files remain outside Git. The record is benchmark evidence, not a
release or fixture-adoption claim.

Historical result files under
[`docs/benchmarks/results/2026-07-23/`](benchmarks/results/2026-07-23/README.md)
remain immutable evidence for their recorded contract. They are not acceptance
for this actionable-artifact cutover.

## Release controls

Record the reviewed source, hosted checks, signature, public release body,
checksums, archive contents, SPDX documents, attestations and fresh
installations. Run the public installer on a native Windows machine as well as
macOS or Linux. Keep the source and release evidence tied to their exact
commits. Follow the [public release verification procedure](releasing.md#verify-the-published-release)
for commands, expected results and failure handling.

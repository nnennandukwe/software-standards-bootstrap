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
make verify
```

Validate the bundled portable skill with the repository's pinned Agent Skills
reference check.

## Contract coverage

Tests cover:

- dirty, staged, detached, unborn, non-Git, changed-`HEAD`, and existing-pack
  inspection failures;
- literal paths, safe Git invocation, binary/generated/resource exclusions,
  and complete inventory accounting;
- exact replay of the report inventory at its pinned baseline and limits;
- strict report, semantic-rule, verification-recipe, Agent Skill, and
  automation-proposal schemas;
- global IDs, canonical paths, category, lenses, scopes, derivation, exact
  evidence, confidence, utility, and relationships;
- extracted and inferred evidence thresholds;
- recipe step references to exact `enforces` evidence;
- rejection of prior rule contracts and rule-owned command/check metadata;
- normalized JSON containing all four artifact kinds;
- base-rule inlining, contextual rule/recipe links, Agent Skill indexing, and
  automation omission in `AGENTS.md`;
- ADR inclusion of rules, recipes, and skills and exclusion of automation
  proposals;
- zero-artifact and automation-only no-write behavior;
- drift, malformed markers, unsafe targets, collision-safe ADR creation, dry
  runs, and atomic write failures; and
- no mutation from inspection, validation, dry runs, or failed operations.

## Benchmark evidence

Run the blind pinned-fixture workflow in [benchmarks.md](benchmarks.md). Record
every emitted rule, verification recipe, Agent Skill, and automation proposal.
Require complete inventory, resolvable evidence, and at least 70% developer
keep or edit-and-keep across all final artifacts.

Historical result files under
[`docs/benchmarks/results/2026-07-23/`](benchmarks/results/2026-07-23/README.md)
remain immutable evidence for their recorded contract. They are not acceptance
for this actionable-artifact cutover.

## Release controls

Do not mark a release complete until the source commit, hosted checks, signed
tag, archives, checksums, SBOMs, attestations, and clean installation are each
verified independently. Pending evidence remains pending.

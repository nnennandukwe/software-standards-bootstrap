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
  artifact kinds, with `working_directory: .` for verification/v1 steps;
- lifecycle-first orientation, action-first base rules, contextual links,
  exact inert verification commands, working directories, expected results,
  relationship labels, Agent Skill indexing, and automation omission in
  `AGENTS.md`;
- ADR inclusion of rules, recipes, and skills and exclusion of automation
  proposals;
- zero-artifact, orientation-only, and automation-only no-write behavior;
- orientation exclusion from artifact denominators and ADR eligibility;
- drift, reserved or malformed markers, dynamically safe Markdown fences,
  unsafe targets, collision-safe ADR creation, dry runs, and atomic write
  failures; and
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

Historical result files under
[`docs/benchmarks/results/2026-07-23/`](benchmarks/results/2026-07-23/README.md)
remain immutable evidence for their recorded contract. They are not acceptance
for this actionable-artifact cutover.

## Release controls

Do not mark a release complete until the source commit, hosted checks, signed
tag, archives, checksums, SBOMs, attestations, and clean installation are each
verified independently. Pending evidence remains pending.

Verify the public installer separately from the release workflow in an
isolated destination:

```bash
install_root=$(mktemp -d) || exit 1
./install.sh --version v0.1.1 --install-dir "$install_root/bin"
"$install_root/bin/ssb" --help
rm -rf "$install_root"
```

On Windows, exercise the PowerShell installer the same way:

```powershell
$installRoot = Join-Path $env:TEMP ([System.IO.Path]::GetRandomFileName())
.\install.ps1 -Version v0.1.1 -InstallDir "$installRoot\bin"
& "$installRoot\bin\ssb.exe" --help
Remove-Item -LiteralPath $installRoot -Recurse -Force
```

Installer scripts are served from `main` and shipped inside each release
archive. They are outside the attestation boundary, so both installers must
enforce the same guarantees and are verified by running them, not by
attestation.

Verify the published archive contents against the tagged source separately
from checksums, SBOMs, and attestations:

```bash
release_root=$(mktemp -d) || exit 1
gh release download v0.1.1 --repo nnennandukwe/software-standards-bootstrap --dir "$release_root"
SSB_RELEASE_ARCHIVE_DIR="$release_root" SSB_RELEASE_SOURCE_REF=v0.1.1 go test ./internal/releaseconfig -run '^TestGeneratedReleaseArchivesContainCompleteSkill$'
rm -rf "$release_root"
```

This archive gate requires all six target archives and byte-for-byte copies of
every regular file beneath `skills/software-standards-bootstrap`. The v0.1.0
archives remain historical incomplete-skill artifacts because they omit the
root `SKILL.md`.

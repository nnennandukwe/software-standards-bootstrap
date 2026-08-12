# Architecture and trust boundaries

## Design objective

Software Standards Bootstrap turns a pinned repository snapshot into a Git-reviewable proposal. The semantic and deterministic responsibilities are deliberately separate.

```text
host agent semantic reads
          │
          ▼
split manifest + inventory + human report + four actionable artifact types
          │
          ├── ssb validate ── schema, inventory, evidence, confidence,
          │                   utility, and relationship checks
          ├── ssb render ──── bounded AGENTS.md projection
          └── ssb adr ─────── Proposed record from adoptable sources
```

For an adopted pack, the governed lifecycle protocol adds another explicit
state machine:

```text
pinned inventory + capability profile + provenance declarations
          │
          ▼
immutable context → semantic proposal → human approval → application
                                                        │
                                                        ├→ rerender
                                                        ├→ optional ADR
                                                        └→ receipt verification
```

No arrow is inferred from the presence of a later artifact. Each transition is
validated and recorded separately.

Git is the selection and approval surface. The CLI does not stage, commit, branch, push, open a pull request, or activate a rule in another system.

## Modules

### Workspace

The workspace module resolves one non-bare worktree, requires Git 2.39 or newer, checks attached and commit-backed `HEAD`, and owns literal Git invocation. Inspection additionally requires no tracked or staged changes and refuses an existing `.software-standards` pack.

All Git subprocess arguments bypass a shell. `GIT_OPTIONAL_LOCKS=0`, `GIT_LITERAL_PATHSPECS=1`, and `LC_ALL=C` keep read behavior explicit and machine-stable.

### Inventory

The inventory module enumerates `HEAD` with `git ls-tree -r -z -l --full-tree` and reads accepted blob object IDs with `git cat-file`. It accepts only `100644` and `100755` blobs.

The default bounds are:

- 40,000 candidate files;
- 128 MiB total candidate content; and
- 1 MiB per file.

Candidate limits apply before blob reads, so binary and generated candidates
consume the same work budget as indexed text. The complete candidate set and
the scanned prefix are reported separately. Reaching either limit marks the
report as truncated, retains exact remaining candidate counts and bytes, and
causes the CLI to return exit `4` unless diagnostic partial output was
explicitly allowed. Oversized files are counted as pre-read exclusions.

Git blob reads use batches of at most 512 entries and 4 MiB. This policy was
selected from the pinned benchmark corpus as the lowest-memory configuration
within 10% of the fastest measured configuration. The inventory contains no
timestamps or absolute paths, so output remains stable for the same commit and
limits.

Before success, the module rechecks attached `HEAD`, the exact commit, tracked/staged status, and the absence of a pre-existing pack.

Prune inventory uses the same bounded blob protocol but requires an existing
committed pack. It rechecks the exact commit and clean tracked state without
misclassifying that pack as a generation collision.

### Governed lifecycle review

The prune module inventories all canonical rules and every repository Agent
Skill, including unreferenced skills and each skill's bounded, tracked support
files. A support file excluded by the inventory safety/resource policy blocks
the review. It embeds a local capability profile that names an exact host
version, exact model/provider identity, observation time, and content-addressed
conformance evidence. There is no implicit “latest” profile and no network
lookup.

Review schemas are separate from actionable artifact schemas. Context and
proposal files are immutable inputs once approved. Candidate replacements are
complete files. Events form a digest chain binding the context, proposal,
baseline, decision, and results.

An approved removal binds the artifact bytes and the corresponding pack index
update into one application plan, journal, and rollback boundary. For
`split-v1`, prune updates `.software-standards/manifest.yaml`, refreshes
primary-file digests, and removes dangling relationships without changing
semantic metadata. For `legacy-v1`, it updates
`.software-standards/report.md`. Both preserve valid zero-rule and
zero-artifact packs without leaving a dangling accepted-artifact entry.

Proposal validation parses replacement rule and skill contracts and evaluates
the proposed resulting rule-skill graph. Approval repeats that preflight for
the exact accepted decision set and records no event when it fails.

The review bundle retains exact copies of the selected capability profile,
all referenced conformance evidence, and the optional provenance declaration
under `inputs/`. Context loading rechecks those durable copies. Ignored
untracked files under governed roots block inspection. Skill provenance is
complete-bundle provenance: partial declarations produce `unknown`, while
fully declared files with different origins produce `mixed`.

Application accepts only approved actions, verifies current source and
candidate digests, refuses unrelated tracked changes, and derives one
content-addressed application plan containing exact prestate, poststate, and
the complete final governed configuration. Dry run, event validation,
journaling, recovery, and verification share this plan. Application writes a
recovery journal before mutation. Replacement skill targets enumerate the complete
entrypoint/support bundle, including intended regular-file modes. Existing
paths are atomically moved to a phase-specific private
claim before replacement; exclusive creation prevents a concurrent repository
writer from being overwritten. A repository-wide mutation lock separates
claims owned by different reviews. After the application event commits, the
journal remains until both locks are released successfully, then is removed
through the still-open identity-pinned review descriptor. Explicit stale-lock
recovery covers journal-free lock cleanup windows. Rollback and recovery restore
only recognized pre/poststate bytes and remove directories created by the
failed transition.

Git executable mode is a platform capability, not a best-effort hint. Windows
proposal validation rejects `100755` candidates because the filesystem cannot
materialize that Git tree mode without changing the index, and the CLI does not
stage. Application planning also rejects a Windows change whose existing
prestate is `100755`, including removal, before dry-run output can imply that
the plan is writable. It does not report an application state after silently
writing a non-executable file.

Ordinary `validate`, `render`, and `adr` commands continue to require rule
evidence pinned to current `HEAD`. After an approved prune application,
review-aware `render --review` and `adr --review` instead validate resulting
artifacts against the pack's recorded historical baseline. A baseline that
is not a reachable ancestor of current `HEAD` is a hard evidence
failure, even if its Git object remains locally resolvable. The rerender event binds
both the managed-section digest and the complete `AGENTS.md` output digest;
verification rechecks the complete regular, non-symlink output. Rerendering is
optional for skill-only reviews, but an explicitly recorded render event is
still bound into their verification receipts and event and therefore must
precede verification.

Portable paths are validated independently of the current operating system.
Traversal, alternate separators, drive or stream syntax, Windows-reserved
device names, invalid Windows filename characters, and trailing dots or spaces
are rejected before filesystem access. The same validation and case-fold
collision check applies to tracked baseline rules and complete skill bundles,
not only proposed candidates. Review roots and durable capability/provenance
snapshots reject symlinked path components before lifecycle reads, locks,
recovery, or event writes.

Dry run is the default. Unknown provenance can only produce
`unable-to-determine`, and that disposition cannot be approved for application.
It must record a structured evidence gap for an exact source artifact. A review
with no approved applicable operations derives a terminal no-change outcome
without an application event.

Verification consumes externally produced, content-addressed receipts. It
does not run the commands named by those receipts. Receipts bind the exact
application event and plan, any required rerender event, and an observation
time after application. Verification rechecks the complete current governed
poststate before recording its event.

### Actionable pack

`rulepack.Validate` detects the source layout once and is the single public
validation seam over all pack files and four artifact types. Presence of a
safe regular `.software-standards/manifest.yaml` selects `split-v1`; an
invalid split manifest never falls back. Without it, `ssb.dev/report/v1`
frontmatter selects `legacy-v1`. Strict YAML and JSON reject unknown and
duplicate fields. Both layouts normalize into one pack consumed by rendering,
JSON, ADR creation, and governed prune.

Split loading checks file type, symlinks, and canonical paths before parsing.
`manifest.yaml` is limited to 1 MiB and `inventory.json` to 128 MiB. The
manifest binds the exact raw bytes of the inventory, human report, and each
primary artifact, including line endings. Human report and rule presentation
are validated rather than silently normalized.

The module owns:

- `ssb.dev/manifest/v1` and `ssb.dev/report/v1` schemas, accepted index, global
  IDs, kinds, canonical paths, and exact raw-byte digests;
- exact replay of the complete schema 2 inventory at the pinned baseline;
- confidence gates, utility factor bounds, arithmetic, and threshold;
- category, lenses, scopes, directive, and derivation validation;
- evidence roles, thresholds, exact line-range hashing, and inventory
  eligibility;
- recipe step-to-`enforces` evidence references;
- portable Agent Skill frontmatter and manifest-owned SSB metadata;
- automation proposal completeness;
- cross-artifact relationship integrity; and
- symlink and traversal rejection.

It never decides whether a candidate is valuable or whether a semantic name is
good. Those remain host-agent judgments reviewed by a developer. It never
executes a recipe or implements an automation proposal.

Valid `ssb validate --format json` output includes the normalized pack in
response schema 3. It reports `pack.format` as `split-v1` or `legacy-v1` and
exposes separate manifest, inventory, and report paths only when they exist.
Invalid output omits the pack. This is a local interchange boundary, not a
catalog import or synchronization mechanism.

### Renderer

The renderer creates one marked root `AGENTS.md` section and preserves every
pre-existing byte outside it. It orders base semantic rules by directive,
utility, and stable ID. It inlines base rule bodies, links contextual semantic
rules and verification recipes, and indexes primary Agent Skills by activation
context. It shows related recipe and skill links and omits automation proposals.

An empty or automation-only pack does not create or rewrite an unprojected
`AGENTS.md`; if a generated managed section remains from an earlier pack, the
renderer removes that stale section and preserves all surrounding bytes.

The section stores:

- a digest of the current baseline and renderable canonical sources; and
- a self-verifying digest over the recorded source digest and generated body.

A source edit leaves the old section internally valid and allows deterministic
replacement. A direct section edit breaks the self-digest and is reported as
drift. Missing, reversed, or duplicate markers are not repaired automatically.

### ADR

The ADR module preserves one existing convention among `docs/adr`,
`docs/adrs`, `adr`, or `adrs`; otherwise it defaults to `docs/adr`.
Multiple conventions require `--adr-dir`.

It rejects paths outside the repository, symlink components, and submodule
prefixes. The next numeric filename is created with exclusive-create semantics.
Existing files are never replaced. Content includes only semantic rules,
verification recipes, and Agent Skills that survive developer review. It
records category, derivation, confidence, utility, and concise evidence
sources. Automation proposals are excluded. An empty or automation-only pack
fails before creating directories or files. Every created ADR has `Proposed`
status.

## Canonical versus derived state

| Artifact | Role | Editable | Evidence state |
|---|---|---:|---:|
| `.software-standards/manifest.yaml` | Split-pack accepted index, selection metadata, provenance, relationships, and primary-file digests | Yes | Exact file and evidence binding |
| `.software-standards/inventory.json` | Complete unedited inspection response | Yes | Exact baseline inventory accounting |
| `.software-standards/report.md` | Human limitations and accepted-output narrative; legacy packs also retain their `ssb.dev/report/v1` machine contract here | Yes | Split: digest-bound narrative; legacy: inventory and evidence index |
| `.software-standards/rules/*.md` | Human-first semantic rules; legacy files retain `ssb.dev/rule/v2` frontmatter | Yes | Manifest-owned evidence mapping |
| `.software-standards/verification/*.yaml` | Canonical existing-command recipes | Yes | Records commands, never a run result |
| `.agents/skills/*/SKILL.md` | Canonical procedural workflows | Yes | No |
| `.software-standards/automation/*.yaml` | Reviewable proposed-check designs | Yes | Not implemented or adopted |
| Root `AGENTS.md` managed section | Derived rule, recipe, and skill router | No | No |
| Proposed ADR | Adoption record from rules, recipes, and skills | New record only | No |
| Review `context.json` | Complete pinned lifecycle input | No | Inventory and capability evidence |
| Review `proposal.yaml` | Semantic dispositions, structured evidence gaps, and complete candidates | Host-authored before approval | Evidence mapping only |
| Canonical application plan | Exact pre/poststate and complete final governed configuration | Derived only | Shared plan digest |
| Review `events.jsonl` | Digest-chained human/tool transitions | Append only | Transition and application-plan integrity |
| Existing repository checker | Repository-owned mechanism referenced by a recipe | Outside ssb | Only when run elsewhere |

## No network contract

The runtime imports only the Go standard library and `go.yaml.in/yaml/v4`. No
package opens sockets or performs update checks. Actionable lenses and valid-pack JSON
make later catalog ingestion possible without adding catalog fetching,
namespacing, synchronization, or activation to the runtime. Network access in
release workflows and optional public benchmark evaluation is outside the
runtime.

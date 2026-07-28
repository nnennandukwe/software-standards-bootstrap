# Architecture and trust boundaries

## Design objective

Software Standards Bootstrap turns a pinned repository snapshot into a Git-reviewable proposal. The semantic and deterministic responsibilities are deliberately separate.

```text
host agent semantic reads
          │
          ▼
assessment + rule/skill source files
          │
          ├── ssb validate ── exact evidence, score, proof, schema checks
          ├── ssb render ──── bounded AGENTS.md projection
          └── ssb adr ─────── Proposed record from retained sources
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

Review schemas are separate from `ssb.dev/rule/v1` and `/v2`; lifecycle
metadata does not require a rule-v3 migration. Context and proposal files are
immutable inputs once approved. Candidate replacements are complete files.
Events form a digest chain binding the context, proposal, baseline, decision,
and results.

The review bundle retains exact copies of the selected capability profile,
all referenced conformance evidence, and the optional provenance declaration
under `inputs/`. Context loading rechecks those durable copies.

Application accepts only approved actions, verifies current source and
candidate digests, refuses unrelated tracked changes, and writes a recovery
journal before mutation. Replacement skill targets enumerate the complete
entrypoint/support bundle, including intended regular-file modes. Existing
paths are atomically moved to a phase-specific private
claim before replacement; exclusive creation prevents a concurrent repository
writer from being overwritten. A repository-wide mutation lock separates
claims owned by different reviews. Rollback and recovery restore only
recognized pre/poststate bytes and remove directories created by the failed transition.

Git executable mode is a platform capability, not a best-effort hint. Windows
proposal validation rejects `100755` candidates because the filesystem cannot
materialize that Git tree mode without changing the index, and the CLI does not
stage. It does not report an application state after silently writing a
non-executable file.
Dry run is the default. Unknown provenance can only produce
`unable-to-determine`, and that disposition cannot be approved for application.

Verification consumes externally produced, content-addressed receipts. It
does not run the commands named by those receipts.

### Rule pack

The rule-pack module parses strict YAML with known-field and unique-key validation. It owns:

- schema and ID validation;
- score factor bounds, arithmetic, and importance bands;
- candidate evidence thresholds;
- exact line-range hashing against baseline blobs;
- the same binary, size, secret, generated/vendor, symlink, and submodule eligibility boundary used by inspection;
- required primary-topic validation for rules and referenced skills;
- v2 activation-lens and directive validation with v1 read compatibility;
- classification and existing-proof mapping;
- full versus partial verification coverage and bounded proved-property validation;
- related Agent Skill path and frontmatter validation; and
- symlink and traversal rejection.

It never interprets whether a candidate is a good engineering standard. That remains host-agent judgment reviewed by a developer.

Valid `ssb validate --format json` output includes the normalized pack in
response schema 2. Invalid output omits the pack. This is a local interchange
boundary, not a catalog import or synchronization mechanism.

### Renderer

The renderer creates one marked root `AGENTS.md` section and preserves every
pre-existing byte outside it. It orders base standing orders by directive
severity, importance, and stable ID; deduplicates mapped verification commands;
and groups contextual source links by language, framework, and task. Contextual
rule bodies remain only in their canonical source files for progressive
loading.

Rule v1 has no lens or directive fields, so the renderer keeps retained v1
rules in a separate base group labeled “directive not recorded” while
preserving their original classification. This preserves visibility without
inventing v2 semantics.

The section stores:

- a digest of the current baseline and rule sources; and
- a self-verifying digest over the recorded source digest and generated body.

A source edit leaves the old section internally valid and allows deterministic replacement. A direct section edit breaks the self-digest and is reported as drift. Missing, reversed, or duplicate markers are not repaired automatically.

### ADR

The ADR module preserves one existing convention among `docs/adr`, `docs/adrs`, `adr`, or `adrs`; otherwise it defaults to `docs/adr`. Multiple conventions require `--adr-dir`.

It rejects paths outside the repository, symlink components, and submodule
prefixes. The next numeric filename is created with exclusive-create
semantics. Existing files are never replaced. Content includes only rules and
referenced skills that survive developer review, exposes each artifact's
primary topic, and records v2 lenses, directive, proof coverage, and bounded
proved property. It always has `Proposed` status.

## Canonical versus derived state

| Artifact | Role | Editable | Proof |
|---|---|---:|---:|
| `.software-standards/assessment.md` | Repository context and discarded candidates | Yes | No |
| `.software-standards/rules/*.md` | Canonical proposed rule sources | Yes | Evidence mapping only |
| `.agents/skills/*/SKILL.md` | Canonical proposed procedural workflows | Yes | No |
| Root `AGENTS.md` managed section | Derived standing orders and contextual rule router | No | No |
| Proposed ADR | Adoption record from retained sources | New record only | No |
| Review `context.json` | Complete pinned lifecycle input | No | Inventory and capability evidence |
| Review `proposal.yaml` | Semantic dispositions and complete candidates | Host-authored before approval | Evidence mapping only |
| Review `events.jsonl` | Digest-chained human/tool transitions | Append only | Transition integrity |
| Existing repository checker | Repository-owned deterministic mechanism | Outside ssb | Only when run elsewhere |

## No network contract

The runtime imports only the Go standard library and `go.yaml.in/yaml/v4`. No
package opens sockets or performs update checks. V2 lenses and valid-pack JSON
make later catalog ingestion possible without adding catalog fetching,
namespacing, synchronization, or activation to the runtime. Network access in
release workflows and optional public benchmark evaluation is outside the
runtime.

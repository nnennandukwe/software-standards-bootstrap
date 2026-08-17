# 2026-08-17 v0.2.0 actionable-artifact acceptance

This record closes the fresh actionable-artifact benchmark gate for the four
public repositories pinned in
[`testdata/benchmarks.yaml`](../../../../testdata/benchmarks.yaml). Generation
and semantic review completed on 2026-08-14. On 2026-08-17, the developer
approved all 21 recommended keep/edit-and-keep decisions and the generated-doc
cleanup; the approved projections and four `Proposed` ADRs were then retained.

This is benchmark acceptance evidence. It does not claim that the fixture
repositories adopted the proposals, or that software-standards-bootstrap
`v0.2.0` was merged, tagged, published, installed, or adopted.

## Run identity

- Generation source commit:
  `a51431ef0faa417c6b9160d0bf9793b0f0db5538`
- Generation binary SHA-256:
  `78318c6ac51bb12a42776522a47011e1e0a752d8ad1606167373a4eb10ee8981`
- Approved projection and ADR source commit:
  `908f990ab63d775ca31c0be3fc1632acde9b843f`
- Projection and ADR binary SHA-256:
  `7ff82f5429ce565da648ad2124870e680c35542b5886256897704fd61acb2783`
- Consumer: Codex CLI 0.145.0, model `gpt-5.6-sol`, reasoning profile
  `xhigh`, approval policy `never`
- Host: macOS 15.7.3 (`24G419`), `arm64`
- Git: 2.39.5 (Apple Git-154)
- Go: 1.26.5 (`darwin/arm64`)
- SSB and repository-tool network use: none
- Repository-code and recorded recipe execution: none

Each fixture record binds the exact Codex session identifier exposed in its
canonical final-run header and the local run name
`ssb-v020-acceptance-<fixture>-a51431e`.

## Complete inventory results

All four schema-2 inventories completed at their exact pins under the default
limits without truncation. The raw inspection bytes matched the fixture
inventory bytes and the normalized validation inventory in every run.

| Repository | Exact pin | Candidate files | Candidate bytes | Scanned files | Scanned bytes | Indexed files | Indexed bytes | Remaining files | Remaining bytes |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Cobra | `adbc8813901bba65827259daa8e22ff94ec1f30e` | 66 | 705,271 | 66 | 705,271 | 65 | 631,792 | 0 | 0 |
| Flask | `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81` | 235 | 1,814,782 | 235 | 1,814,782 | 230 | 1,474,850 | 0 | 0 |
| Django | `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f` | 7,001 | 45,506,636 | 7,001 | 45,506,636 | 5,619 | 36,820,618 | 0 | 0 |
| Next.js | `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd` | 29,073 | 111,110,455 | 29,073 | 111,110,455 | 28,403 | 88,643,646 | 0 | 0 |

| Repository | Binary | Generated | Oversized | Secret-like | Symlink | Submodule | Vendor/generated tree | Non-regular |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Cobra | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| Flask | 5 | 0 | 0 | 1 | 0 | 0 | 0 | 0 |
| Django | 1,382 | 0 | 0 | 0 | 4 | 0 | 72 | 0 |
| Next.js | 652 | 18 | 21 | 113 | 29 | 0 | 1,060 | 0 |

Tree-level exclusions (`oversized`, `secret-like`, `symlink`, `submodule`,
`vendor/generated tree`, and `non-regular`) are removed before candidate
accounting. Scan-level `binary` and `generated` exclusions explain the
candidate-to-indexed delta.

## Approved artifact retention

Every emitted actionable artifact had resolvable evidence. Retention was
evaluated independently per fixture using every final artifact emitted before
developer review as the denominator. Orientation was recorded as repository
context but was excluded from both the denominator and ADR eligibility.

| Repository | Rules | Recipes | Skills | Automation | Keep | Edit and keep | Exact retention | Result |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| Cobra | 3 | 0 | 2 | 0 | 3 | 2 | 5/5 (100%) | Pass |
| Flask | 3 | 0 | 0 | 0 | 2 | 1 | 3/3 (100%) | Pass |
| Django | 3 | 2 | 1 | 0 | 3 | 3 | 6/6 (100%) | Pass |
| Next.js | 6 | 1 | 0 | 0 | 2 | 5 | 7/7 (100%) | Pass |
| **Total** | **15** | **3** | **3** | **0** | **10** | **11** | **21/21 (100%)** | **Pass** |

The per-fixture records bind every artifact ID to its kind, canonical path,
decision, pre-review SHA-256, final SHA-256, and approved edit summary:

- [Cobra](cobra/run.yaml)
- [Flask](flask/run.yaml)
- [Django](django/run.yaml)
- [Next.js](nextjs/run.yaml)

Three edit-and-keep decisions changed only manifest metadata while retaining
the canonical artifact bytes: Django database-feature evidence, Next.js pnpm
consumer scope/evidence, and Next.js header-review routing. The other eight
edit-and-keep decisions changed canonical artifact bytes. Generated reports and
projections were then refreshed so their lifecycle wording says `proposed`,
`retained`, or `manifest-listed` rather than implying acceptance or adoption.

## Projection, ADR, and readback

The approved packs passed schema-3 validation with empty diagnostics. The
projection dry-run payload matched the written projection byte for byte after
removing the command wrapper, and the final replay reported that no write would
occur. Each retained ADR has `Proposed` status, includes every retained rule,
recipe, and Agent Skill ID, and contains no orientation or automation entry.

The record keeps exactly 13 files: this README plus, for each fixture,
`run.yaml`, `proposal/AGENTS.proposed.md`, and
`adr/0001-actionable-standards.md`. It deliberately retains no raw inventory,
manifest, report, orientation, canonical artifact, transcript, command log, or
replay bundle. Each `run.yaml` carries SHA-256 and Git-blob bindings for the two
retained fixture files, plus source/content/output projection digests and the
non-retained canonical-source hashes needed to audit provenance.

## Edit/delete propagation check

A disposable derivative of the approved Cobra pack verified propagation
without changing the accepted fixture or its retention decisions. The check:

- added sentinel `SSB-PROPAGATION-EDIT-20260817` to
  `cover-go-code-changes-with-tests`;
- deleted `change-document-generation` and its manifest/orientation
  relationship;
- validated 4 artifacts (3 rules, 1 skill, 0 recipes, 0 automation);
- rendered, then passed a no-write dry run;
- created a `Proposed` ADR with four artifact headings;
- found the edit sentinel in both projection and ADR, and the deleted ID in
  neither; and
- produced projection SHA-256
  `90c2b7d74fffbc4b3d1669624464ac352336bf189f70dc77fc0c8bef0291b498`
  and ADR SHA-256
  `d9987b87d2a87ade019a113059e90a674e022ac9b4eb1b29ca0c1a9489390304`.

The derivative files and command logs are not retained.

## State ledger

| State | Evidence |
|---|---|
| Fresh inventory | Complete at all four exact pins; no truncation |
| Proposal generation | Complete on 2026-08-14 in four isolated local host runs |
| Evidence resolution | 21/21 final artifacts resolvable |
| Semantic review | Complete on 2026-08-14; 10 keep and 11 edit-and-keep recommendations |
| Developer retention | Explicitly approved on 2026-08-17; 21/21 retained and every per-fixture threshold passes |
| Projection | Four approved final projections retained and digest-bound |
| ADR creation | Four `Proposed` ADRs retained; no adoption claim |
| Edit/delete propagation | Passed in a disposable Cobra derivative; derivative not retained |
| Release evidence | Not complete; this benchmark record does not imply merge, tag, publication, installation, or adoption |

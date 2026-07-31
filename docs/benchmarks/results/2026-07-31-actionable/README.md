# 2026-07-31 actionable-artifact host acceptance

This record closes the fresh actionable-artifact benchmark gate for the four
public repositories pinned in
[`testdata/benchmarks.yaml`](../../../../testdata/benchmarks.yaml). It records
proposal generation, developer retention, rendering, and ADR creation as
separate states. It does not imply that `v0.1.0` was tagged or published.

The developer reviewed the final proposals on 2026-07-31 and explicitly chose
`keep` for every emitted artifact. Automation proposals count in the retention
denominator, but remain proposed check designs: they are omitted from
`AGENTS.md` and the ADR and were not implemented.

## Run identity

- SSB source commit: `735355125c3b9f8af125e22bd167ff8077fce9cf`
- Evaluator binary SHA-256:
  `26e53e145abd26805ceee7e8e75687beab76a1873566bc578882ab892c36ea91`
- Consumer: `codex-cli 0.145.0`, model `gpt-5.6-sol`, reasoning profile
  `xhigh`, approval policy `never`
- Host: macOS 15.7.3 (`24G419`), `arm64`
- Git: 2.39.5 (Apple Git-154)
- Go: 1.26.5 (`darwin/arm64`)
- Fixture branch: attached `ssb-evaluation` at each exact pin
- SSB and repository-tool network use: none
- Repository-code and recorded recipe execution: none

Codex attempted its normal remote installed-plugin synchronization before each
repository task. An unrelated Calendly plugin bundle failed name validation.
That host-startup attempt supplied no repository evidence and did not cause
`ssb` or a repository tool to use the network.

## Consumer sessions

| Repository | Session ID | Exact pin |
|---|---|---|
| Cobra | `019fb874-3a3f-7400-b6c7-9baa5ad514ba` | `adbc8813901bba65827259daa8e22ff94ec1f30e` |
| Flask | `019fb87d-9961-7b11-b6e2-a8284fc2ba2f` | `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81` |
| Django | `019fb88c-17b1-7f93-88b3-f1b60aa76085` | `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f` |
| Next.js | `019fb896-7889-7c62-81d7-06c9fac58db3` | `6c6d1632e1e1f03e32db1dd8dc0a63dee49d68cd` |

Each session ran against only one fixture clone. Command-log review found no
cross-fixture or parent-directory repository read in these four accepted runs.

## Complete inventory results

All four schema-2 inventories completed under the default limits without
truncation.

| Repository | Candidate files | Candidate bytes | Scanned files | Scanned bytes | Indexed files | Indexed bytes | Remaining files | Remaining bytes |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Cobra | 66 | 705,271 | 66 | 705,271 | 65 | 631,792 | 0 | 0 |
| Flask | 235 | 1,814,782 | 235 | 1,814,782 | 230 | 1,474,850 | 0 | 0 |
| Django | 7,001 | 45,506,636 | 7,001 | 45,506,636 | 5,619 | 36,820,618 | 0 | 0 |
| Next.js | 29,073 | 111,110,455 | 29,073 | 111,110,455 | 28,403 | 88,643,646 | 0 | 0 |

Exclusion accounting:

| Repository | Binary | Generated | Oversized | Secret-like | Symlink | Submodule | Vendor/generated tree | Non-regular |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Cobra | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| Flask | 5 | 0 | 0 | 1 | 0 | 0 | 0 | 0 |
| Django | 1,382 | 0 | 0 | 0 | 4 | 0 | 72 | 0 |
| Next.js | 652 | 18 | 21 | 113 | 29 | 0 | 1,060 | 0 |

## Artifact retention

Acceptance is evaluated independently for each fixture. Every final artifact
had resolvable evidence and received an explicit `keep` decision.

| Repository | Rules | Recipes | Skills | Automation proposals | Keep | Total | Exact retention | Result |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| Cobra | 4 | 1 | 1 | 1 | 7 | 7 | 7/7 (100%) | Pass |
| Flask | 3 | 3 | 1 | 1 | 8 | 8 | 8/8 (100%) | Pass |
| Django | 5 | 1 | 1 | 0 | 7 | 7 | 7/7 (100%) | Pass |
| Next.js | 6 | 3 | 1 | 1 | 11 | 11 | 11/11 (100%) | Pass |

The fixture records contain the per-artifact category, confidence, utility,
and decision:

- [Cobra](codex-cobra.md)
- [Flask](codex-flask.md)
- [Django](codex-django.md)
- [Next.js](codex-nextjs.md)

## Structural and enforcement review

The host followed the generation workflow's required review of dependency and
package boundaries, parallel implementations and families, platform seams,
compatibility surfaces, source/test/documentation symmetry, existing commands,
and existing automatic enforcement. The accepted outputs reflect fixture-
specific findings rather than a shared catalog. Every candidate was routed to
one artifact kind or discarded before the final report.

## Rendering, ADRs, and propagation

All four accepted packs pass `ssb validate`; `ssb render --dry-run` reports
their generated worktree projections current. After the explicit developer
request, `ssb adr` created a `Proposed` adoption record in each fixture:

| Repository | Adoptable entries | Automation omitted | ADR SHA-256 |
|---|---:|---:|---|
| Cobra | 6 | 1 | `d3fb9927dbc9cd9da576ef5a8a65e3ba7f2e6bff0f5a34fa717bb9596ea51a8b` |
| Flask | 7 | 1 | `c9a4f4e723deb710dd369369fdf4af410f605b3a228ea702f1c14bc6f643c628` |
| Django | 7 | 0 | `c0fbb8580a62ca4aa9750e36c97ed896ca07ca4f5d63b98f91667d59914c330d` |
| Next.js | 10 | 1 | `fae98b2e37cd327438f56bbbf0da280f10b08b5f9821bd55d2fe0c5505021923` |

Each ADR is `docs/adr/0001-actionable-standards.md` inside its disposable
fixture clone. A consistency check confirmed that every retained rule, recipe,
and Agent Skill appears with category, derivation, confidence, utility, and
evidence, while automation IDs do not appear.

Edit/delete propagation was verified separately without changing the approved
packs. In a disposable derivative of Cobra, the body of
`cover-go-changes-with-tests` was edited and
`update-affected-document-generators-together` was deleted together with its
manifest entry and relationship. Validation then reported six artifacts;
rerendering and ADR creation propagated the edited body and omitted the deleted
rule. The derivative stayed at the Cobra pin with an empty index and was not
used for retention.

## Safety and excluded diagnostics

- No repository code, hook, build script, test, linter, formatter, package
  manager, or recorded verification recipe was executed.
- No automation proposal was implemented.
- Fixture `HEAD` values stayed at their pins; no file was staged, committed,
  branched, pushed, or tagged by a consumer session.
- A Flask temporary-directory cleanup attempt was blocked by Codex execution
  policy before execution. It deleted nothing and did not affect the fixture.
- Temporary inventory files used during Next.js report assembly were removed
  before handoff and are not part of the changed-path set.
- The seven `software-standards-bootstrap` skill files listed by each fixture
  were evaluator-provided host input, not generated artifact decisions.
- Earlier incomplete, write-restricted, or cross-fixture diagnostic attempts
  were excluded. In particular, a shared-parent Django diagnostic observed
  sibling output and was terminated; only the later repository-isolated
  Django session above counts.

## State ledger

| State | Evidence |
|---|---|
| Fresh inventory | Complete at all four exact pins; no truncation |
| Proposal generation | Complete in four isolated Codex sessions |
| Evidence resolution | 33/33 final artifacts resolvable |
| Developer retention | Explicit keep for 33/33; per-fixture threshold passes |
| Rendering | Valid and current for all four accepted packs |
| Edit/delete propagation | Passed in a disposable Cobra derivative |
| ADR creation | Four `Proposed` ADRs created and checked; automation omitted |
| Release evidence | This dated record; it does not imply a tag or publication |

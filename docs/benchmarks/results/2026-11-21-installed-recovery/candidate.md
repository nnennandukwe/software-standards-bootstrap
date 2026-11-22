# Interrupted generation with the installed v0.3.0 release candidate

On November 21, 2026, a Codex CLI host assessed a pinned Flask fixture, wrote a semantic rule and received SIGINT before it wrote the manifest. A second host recovered the proposal with the installed candidate CLI and skill. Its final Proposed ADR matched the August developer-reviewed ADR byte for byte.

## Reproduce the setup

| Component | Observed identity |
| --- | --- |
| Candidate | `v0.3.0-rc` from `c69bfd6913fd10e6ea08415dca07539388ca468f` |
| Installed CLI | macOS arm64, SHA-256 `e3cbd617a7de30d9967ec0836f0c4f05fdfe5eab66c58ac15665423e74a4f7e4` |
| Installed skill | Version 0.7.0, six files, Git tree `0f0afc5f18aafd7dffd578ae58c66a575b448516`, SKILL.md SHA-256 `a231f1a029261c66cf745116b58338f753a2e286f0654e1c140594f189e4ac23` |
| Fixture | pallets/flask at `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`, attached branch `ssb-evaluation` |
| Host | Codex CLI 0.163.0, `gpt-6.1-sol`, medium reasoning, macOS 26.6 (25G72), arm64 |
| Generation and continuation sessions | `01a200d9-5d2c-7b13-9609-97331dad4c92` and `01a200da-5f78-7591-a91b-07909b59b3fc` |

The operator ran the candidate's shell installer with its download transport mapped to the archived CI build and checksum manifest. The installer verified the archive and wrote the binary and skill to isolated destinations. The macOS arm64 archive's SHA-256 was `57400117d12d7a20b9dbe24e523257bdd9ca4be6b22374b3b543c11f9d6f96d8`. [Candidate build](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/44970855699)

The operator directed the first host to examine Flask's Sans-IO boundary. Before opening the historical rule, the host checked pinned Git blobs and proposed a scoped architecture rule with utility 87/100. It then compared the recommendation with the historically reviewed rule, scored at 89/100, and persisted those reviewed bytes. The operator sent SIGINT to the enclosing host after verifying the inventory, first rule and absence of a manifest. The host exited 1. An earlier model-selection attempt returned HTTP 400 before any fixture operations.

## Installed commands and outcomes

All repository commands below used `--repo .`. Inspection, assessment and validation used `--format json`. The recorder captured stdout, stderr, exit status, raw file hashes, HEAD and index bytes for each invocation.

| Step | Command | Exit | Observed result |
| --- | --- | --- | --- |
| 1 | `ssb inspect` | 0 | Complete inventory, 235 candidates and 230 indexed files. The recorder saved the original stdout bytes. |
| 2 | `ssb resume` | 0 | Verified baseline and retained rule. Inventory reusable. |
| 3 | `ssb resume` | 0 | Repeated assessment returned the same state. |
| 4 | `ssb version` | 0 | Installed tag and full candidate commit matched. |
| 5 | `ssb validate` | 0 | Three-rule pre-review proposal valid. |
| 6 | `ssb render --dry-run` | 0 | Previewed the initial projection without writes. |
| 7 | `ssb render` | 0 | Wrote the managed root, catalog and three bundles. |
| 8 | `ssb resume` | 1 | Two reviewed edits left old digest bindings. Inventory reusable, proposal needs correction. |
| 9 | `ssb render` | 1 | Rejected stale inputs. Every fixture file stayed byte-identical. |
| 10 | `ssb resume` | 0 | Corrected manifest matched the reviewed bytes. Proposal ready to render. |
| 11 | `ssb validate` | 0 | Three reviewed rules valid. |
| 12 | `ssb render --dry-run` | 0 | Previewed the corrected projection without writes. |
| 13 | `ssb render` | 0 | Updated the root, catalog and one changed bundle. |
| 14 | `ssb resume` | 0 | Reported complete with developer review as the remaining stage. |
| 15 | `ssb render` | 0 | Projection current. Zero files changed. |
| 16 | `ssb adr --dry-run` | 0 | Preview matched the historical Proposed ADR. |
| 17 | `ssb adr` | 0 | Wrote the matching Proposed ADR. |

## What the bytes establish

The operator reconstructed the three rules from the retained August Proposed ADR. The August review transcript supplied the numbered original manifest, orientation and report, and its cleanup verification supplied the final report text. Updating the two approved digest entries reproduced the full reviewed manifest, including every evidence reference and utility field. The release candidate generated the inventory anew and matched its historical digest. [August Flask decisions and file identities](../2026-08-17-v0.2.0-actionable/flask/run.yaml)

| File or check | SHA-256 |
| --- | --- |
| Inventory | `ed8013e208958ef189a329644ea1ddfcc4d73657c17c9a27f25576d4cf7e84ef` |
| Reviewed manifest | `f8b563509dd0f69dc986829ca08560685d8675ec86564b790b3b869dfb51e40e` |
| Corrected root | `47e5912f89338ba7bbb76fed5e52e68dc017aa07d22155b50ffaefe8e21783bb` |
| Corrected catalog | `aad6c2ea1de139b6e099090e05442c4ea2131e8f1932d0de567a12a7910514dc` |
| Generated Proposed ADR | `0a68f2e52d028e22733534e701e2d0c1027bbae08f47edbe1b716edf4e49638f` |
| Git index before and after both sessions | `28d270c2b20f0778bf68b18020feae45b076e83f06a4c7d41d96e3378508a524` |
| Preserved manual root prefix | `5294c4f44bb13ca4501d5627e4d8322f73916e3c3012ff30884db51621898198` |
| Preserved manual skill | `64f10e1a5ffc16fcd7149d71754f5aa2290b6dc74db8e31f3c0a64f6dc2906eb` |
| Preserved unrelated local note | `a17a2480bcc12ec41ab632942ed3a6c43ac90d94017b2ffdb9b6c6a75a779dcf` |

The independent check compared all 244 preexisting files other than the root, including the installed skill, and checked that the original root text appeared exactly once as its prefix. Both sessions preserved HEAD and the raw Git index. The host read Flask source for evidence and ran the installed SSB commands. Flask code and recipe commands stayed unexecuted.

This is a directed recovery exercise on a disposable Flask clone. The historical decision covers the exact August canonical inputs. The exercise produced a proposed change for review. Its results are scoped to the candidate binary, skill and host identified above.

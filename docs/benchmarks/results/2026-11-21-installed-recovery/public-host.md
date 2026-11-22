# Public installation and interrupted generation on macOS

The public macOS arm64 installer supplied the v0.3.0 binary and skill 0.7.0 to an independent Flask checkout. Codex CLI 0.163.0 used them to resume interrupted generation, apply the recorded developer edits, rerender and produce the exact August Proposed ADR. [Public release](https://github.com/nnennandukwe/software-standards-bootstrap/releases/tag/v0.3.0)

| Identity | Verified value |
| --- | --- |
| Source | `e4c7f5754f4aee4d18683364f29c5d4dd12a4796` |
| Installed binary SHA-256 | `cd6884d79265081e10ff3356b65c32d3cd1b5125d0e7a133296cc611e965e989` |
| Installed SKILL.md SHA-256 | `a231f1a029261c66cf745116b58338f753a2e286f0654e1c140594f189e4ac23`, 6 installed skill files |
| Fixture | pallets/flask at `36e4a824f340fdee7ed50937ba8e7f6bc7d17f81`, attached branch `ssb-evaluation` |
| Host | macOS 26.6 (25G72) arm64, Codex CLI 0.163.0, `gpt-6.1-sol`, medium reasoning |
| Generation | Session `01a200f7-f9ba-7f53-8b82-c22432d10a39`, SIGINT to the enclosing host, exit 1 before the manifest |
| Continuation | Session `01a200f9-0abb-7983-9994-e2290b6af1d7`, exit 0 |

The operator directed the generation host to inspect Flask's Sans-IO boundary. It wrote its assessment before opening the historical rule. When the verified inventory and first reviewed rule reached disk, the operator interrupted the host. A fresh host used the installed recovery command and finished the historical review workflow at the same fixture pin. The August decision applies to those exact canonical bytes. [August review ledger](../2026-08-17-v0.2.0-actionable/flask/run.yaml)

## Public command evidence

The operator recorded the installed program's actual stdout, stderr, exit code and fixture write set. Repository operations used `--repo .`, and inspection, validation and recovery used `--format json`. The inspection recorder saved stdout unedited as the inventory after the CLI exited.

| Step | Installed command | Exit | Outcome | Fixture files the CLI changed |
| --- | --- | --- | --- | --- |
| 1 | `ssb inspect` | 0 | 235 candidates, 230 indexed | 0 |
| 2 | `ssb resume` | 0 | needs_proposal | 0 |
| 3 | `ssb resume` | 0 | needs_proposal | 0 |
| 4 | `ssb version` | 0 | ssb v0.3.0 (commit e4c7f5754f4aee4d18683364f29c5d4dd12a4796) | 0 |
| 5 | `ssb validate` | 0 | valid | 0 |
| 6 | `ssb render --dry-run` | 0 | projection preview | 0 |
| 7 | `ssb render` | 0 | projection written | 5 |
| 8 | `ssb resume` | 1 | needs_correction | 0 |
| 9 | `ssb render` | 1 | rejected | 0 |
| 10 | `ssb resume` | 0 | ready_to_render | 0 |
| 11 | `ssb validate` | 0 | valid | 0 |
| 12 | `ssb render --dry-run` | 0 | projection preview | 0 |
| 13 | `ssb render` | 0 | projection written | 3 |
| 14 | `ssb resume` | 0 | complete | 0 |
| 15 | `ssb render` | 0 | current projection | 0 |
| 16 | `ssb adr --dry-run` | 0 | ADR preview | 0 |
| 17 | `ssb adr` | 0 | Proposed ADR | 1 |

The independent comparison checked all 244 preexisting files outside the root and matched the manual root prefix exactly once. It also matched the historical reviewed manifest, report, orientation and every rule. The blocked render changed zero files. The corrected render wrote the projection, and its replay changed zero files.

| SHA-256 check | Observed digest |
| --- | --- |
| Reviewed manifest | `f8b563509dd0f69dc986829ca08560685d8675ec86564b790b3b869dfb51e40e` |
| Final root | `47e5912f89338ba7bbb76fed5e52e68dc017aa07d22155b50ffaefe8e21783bb` |
| Final catalog | `aad6c2ea1de139b6e099090e05442c4ea2131e8f1932d0de567a12a7910514dc` |
| Raw Git index before and after both sessions | `5ac875cd0fdb9f25ad22570d386cd0cefb18386254c361126e7ea8037945cbe1` |
| Manual root prefix | `5294c4f44bb13ca4501d5627e4d8322f73916e3c3012ff30884db51621898198` |
| Manual skill | `64f10e1a5ffc16fcd7149d71754f5aa2290b6dc74db8e31f3c0a64f6dc2906eb` |
| Unrelated note | `a17a2480bcc12ec41ab632942ed3a6c43ac90d94017b2ffdb9b6c6a75a779dcf` |
| Resulting Proposed ADR | `0a68f2e52d028e22733534e701e2d0c1027bbae08f47edbe1b716edf4e49638f` |

The observations cover this public Darwin arm64 binary, matching skill and named host in a disposable checkout. The generated proposal remained local. Flask code and recipe commands stayed unexecuted.

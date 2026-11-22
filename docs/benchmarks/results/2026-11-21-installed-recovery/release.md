# Public verification of SSB v0.3.0

SSB v0.3.0 and its portable Agent Skill 0.7.0 were published on November 21, 2026, at 5:54:43 p.m. Pacific. An independent download verified all 13 assets from the immutable public release. Fresh native installations on all six targets checked binary identity, skill bytes and interrupted-generation recovery. [Public release](https://github.com/nnennandukwe/software-standards-bootstrap/releases/tag/v0.3.0) · [Repeat the verification](../../../releasing.md#verify-the-published-release)

## Source, review and publication

| Stage | Identity and result |
| --- | --- |
| Candidate host exercise | Codex CLI 0.163.0 exercised 17 commands with the installed candidate from `c69bfd6913fd10e6ea08415dca07539388ca468f` against the pinned Flask fixture. [Candidate record](candidate.md) |
| Reviewed PR head | `b34656b64c89d76a06c6f1365efc91ab5b2cae6d`, based on `cf5acd22f9cec31bef6d45fe79eaa716a5b0ec19`. Local Qodo checkpoint `8d2234db2c8442aebf233c33e2b97c0c` and completed hosted review `2090868` covered that head. The three local findings and both hosted findings had automatic `implemented` attribution. [Source PR #50](https://github.com/nnennandukwe/software-standards-bootstrap/pull/50) |
| Tested candidate | Local `make verify` passed on Go 1.26.9. All five CI jobs passed on the candidate branch and PR, including macOS, Linux, native Windows, race/vet/vulnerability checks and the release archive preflight. [Candidate CI](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/44971633764) · [PR CI](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/44972370078) |
| Merged and tagged source | Merge commit `e4c7f5754f4aee4d18683364f29c5d4dd12a4796` has the same tree as the reviewed head, `577014364425e7747d750d05be44c0b41350c497`. All five main CI jobs passed. The signed tag object is `27c07bf81036d9356ecfbf8fbdafd5410cdd3d03`. Local Git verified the signature and GitHub reported `valid`. [Main CI](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/44972971234) · [Signed tag](https://api.github.com/repos/nnennandukwe/software-standards-bootstrap/git/tags/27c07bf81036d9356ecfbf8fbdafd5410cdd3d03) |
| Published release | GitHub release ID `454126018` at `2026-11-22T01:54:43Z`, with `draft: false` and `immutable: true`. The public body matched the rendered tag's changelog bytes, SHA-256 `b26c0af49d43c2da0374f4cf0c06fb9553abcc5e5003c47ea7b8490a3464b312`. [Public metadata](https://api.github.com/repos/nnennandukwe/software-standards-bootstrap/releases/tags/v0.3.0) · [Release workflow](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/44973482465) |

The source scan reported zero reachable vulnerabilities and zero affected imported packages. It reported two standard-library module advisories on Go 1.26.9. GO-2026-6859 concerns an HTTP server race fixed in Go 1.26.11. GO-2026-6809 concerns certificate trust fixed in Go 1.26.10. [Go advisory GO-2026-6859](https://pkg.go.dev/vuln/GO-2026-6859) · [Go advisory GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809)

## What developers receive

The download contained six archives, six corresponding `.spdx.json` documents and `checksums.txt`. All twelve entries in that manifest matched the downloaded SHA-256 values. The manifest itself hashed to `9f7c2b683601f5f3d5dced86ceac20471f04a14f5ed7e21b9d138ef720f29dca`. [Published checksum manifest](https://github.com/nnennandukwe/software-standards-bootstrap/releases/download/v0.3.0/checksums.txt)

| Public archive | Executable header | Archive SHA-256 |
| --- | --- | --- |
| `ssb_v0.3.0_darwin_amd64.tar.gz` | Mach-O64, amd64 | `f0adefba71ba788b06b46e793a814e7380b3d69f771529f6b8fef620f881401d` |
| `ssb_v0.3.0_darwin_arm64.tar.gz` | Mach-O64, arm64 | `16efd7cc21b93f5367b2ad1b5cb0631c0845e8b1176887f358205ca7005feb13` |
| `ssb_v0.3.0_linux_amd64.tar.gz` | ELF64, amd64 | `d7905ad7954caf9f2eb76b802f622d3c87fd12140f06f54bcebe27bcde7185c8` |
| `ssb_v0.3.0_linux_arm64.tar.gz` | ELF64, arm64 | `b728d4a3c16c1b8e438bfb4bc8182f7461fb90a972f309bfc084b7f6ef89594d` |
| `ssb_v0.3.0_windows_amd64.zip` | PE32+, amd64 | `fd63cfcbdd41c7d0392c869a05edf49492ec58307d2736c116dae95185a7ae62` |
| `ssb_v0.3.0_windows_arm64.zip` | PE32+, arm64 | `cd797291e26cec82f5155b50b37d5f7b70af688bad8e692f05c0a4147fbd5df2` |

Each executable's header, Go build settings and embedded revision identified its target, Go 1.26.9, CGO disabled, the v0.3.0 module and the clean tagged source commit. In every archive, all 14 other files matched their tagged Git blobs, including both installers, the recovery guide and all six skill files. The skill tree was `0f0afc5f18aafd7dffd578ae58c66a575b448516`. Its `SKILL.md` SHA-256 was `a231f1a029261c66cf745116b58338f753a2e286f0654e1c140594f189e4ac23`. [Tagged skill](https://github.com/nnennandukwe/software-standards-bootstrap/tree/v0.3.0/skills/software-standards-bootstrap)

For each of the twelve checksum subjects, `gh attestation verify` verified SLSA v1 provenance against repository `nnennandukwe/software-standards-bootstrap`, source ref `refs/tags/v0.3.0`, the full tagged commit and signer workflow `.github/workflows/release.yml`. Each archive also passed SPDX 2.3 attestation verification under the same constraints. Each signed SPDX predicate matched the downloaded JSON document, which named the archive and binary and bound both SHA-256 values. All certificates identified attempt 1 of release run `44973482465`. [Release workflow attempt](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/44973482465/attempts/1)

## Native public installations

The public shell and PowerShell installers ran on all six targets. Each installed binary's SHA-256 matched the executable extracted from its independently verified archive. Each reported `ssb v0.3.0 (commit e4c7f5754f4aee4d18683364f29c5d4dd12a4796)` and matched all six installed skill files to Git. macOS arm64 ran locally. GitHub hosted the other five native runs. [Windows amd64 release job](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/44973482465/job/137058573608) · [Four additional native jobs](https://github.com/nnennandukwe/software-standards-bootstrap/actions/runs/44973824820) · [Tagged installation check](https://github.com/nnennandukwe/software-standards-bootstrap/blob/v0.3.0/scripts/verify-public-install.py)

| Native system | Installed binary SHA-256 |
| --- | --- |
| macOS 15.7.9 amd64 | `c40ca729363598e9a444f95ffe45c285db802a587a11bc8e7a26e7b343b2dc7a` |
| macOS 26.6 arm64 | `cd6884d79265081e10ff3356b65c32d3cd1b5125d0e7a133296cc611e965e989` |
| Ubuntu 24.04 amd64 | `014ab54df18d033ec9763fce9aec17628e6340ffbcf57469efc5a8fcfcb3ea81` |
| Ubuntu 24.04 arm64 | `4a30d316cf726628e42001b6cbf90831bea087ef4a2bc102e868027668cb5f8a` |
| Windows Server 2025 amd64 | `ebd3779ddbc3b47af2087d75689062c035d9e75a244fe80c546c99a82b762633` |
| Windows 11 arm64 | `441b0f43f5cf3c77e8370067ba31bdd562bf2df4dd717bd8873bd98027b38244` |

All six installations resumed the same incomplete fixture at `ad75ff2f632cc5e3fe1977ac4ee0baabc3eb746d` as `needs_proposal`, preserving its files, index and HEAD. The inventory SHA-256 was `8ae4ce4f7aeac8d4a271d423ba5ba963ad80cb939b20b056276deb468a3b0566`.

The companion record follows the public binary and skill through an interrupted proposal on a disposable Flask clone. [Installed public host exercise](public-host.md)

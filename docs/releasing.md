# Release runbook

## Prerequisites

- All four behavior slices are reviewed on `main`.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/ssb`, and `go tool govulncheck ./...` pass on Go 1.26.5.
- The pinned release-configuration preflight below passes.
- CI has built and tested macOS, Linux, and Windows.
- Pinned public benchmark results meet the evidence and retention thresholds.
- At least one agent-host smoke-test record identifies its exact consumer version and fixture commits.
- GitHub immutable releases are enabled for future releases.
- `CHANGELOG.md` contains the release notes, and the version heading carries the
  tag date instead of `unreleased`.
- The portable skill's `metadata.version` matches the contract being released.

Do not tag a release from an unreviewed working tree.

Validate the workflow and GoReleaser configuration with the exact tool versions
used for this release contract:

```bash
go run github.com/goreleaser/goreleaser/v2@v2.17.0 check
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
```

## Tag and workflow

Create and push a signed semantic-version tag after every gate passes:

```bash
git tag -s v0.1.0 -m "Software Standards Bootstrap v0.1.0"
git push origin v0.1.0
```

The release workflow:

1. reruns tests, race tests, vet, and vulnerability scanning;
2. builds CGO-free macOS, Linux, and Windows archives with Go 1.26.5;
3. generates `checksums.txt` with SHA-256;
4. generates one SPDX JSON SBOM per archive with pinned Syft;
5. uploads all assets to a draft GitHub release;
6. creates signed provenance over the checksum subjects;
7. creates an SBOM attestation for each archive; and
8. publishes the draft only after all attestations succeed.

Every third-party action is pinned to a full commit SHA. GoReleaser and Syft versions are exact.

## Verify the published release

From the repository checkout, exercise the public installer against an isolated
destination:

```bash
install_root=$(mktemp -d) || exit 1
./install.sh --version v0.1.0 --install-dir "$install_root/bin"
"$install_root/bin/ssb" --help
rm -rf "$install_root"
```

In a clean directory:

```bash
gh release download v0.1.0 --repo nnennandukwe/software-standards-bootstrap
shasum -a 256 --check checksums.txt
gh attestation verify ssb_v0.1.0_darwin_arm64.tar.gz --repo nnennandukwe/software-standards-bootstrap
```

Inspect the matching `.spdx.json` asset and confirm all six archives are present:

- Darwin amd64 and arm64;
- Linux amd64 and arm64; and
- Windows amd64 and arm64.

Only then mark the release verification gate complete.

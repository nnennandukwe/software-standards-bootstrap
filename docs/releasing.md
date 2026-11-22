# Release runbook

## Prerequisites

- Every change listed for the release is reviewed and merged to `main`.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/ssb`, and `go tool govulncheck ./...` pass on Go 1.26.9.
- The pinned release-configuration preflight below passes.
- CI has built and tested macOS, Linux, and Windows.
- Pinned public benchmark results meet the evidence and retention thresholds.
- At least one agent-host smoke-test record identifies its exact consumer version and fixture commits.
- GitHub immutable releases are enabled for future releases.
- `CHANGELOG.md` contains the release notes, and the version heading carries the
  tag date instead of `unreleased`.
- The portable skill's `metadata.version` matches the contract being released.

Check the repository setting with an authenticated GitHub CLI session. The
command must return `true` before you tag.

```bash
gh api repos/nnennandukwe/software-standards-bootstrap/immutable-releases --jq .enabled
```

Validate the workflow and GoReleaser configuration with the exact tool versions
used for this release contract:

```bash
go run github.com/goreleaser/goreleaser/v2@v2.17.0 check
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
make verify-release-archives
uvx --from skills-ref==0.1.1 agentskills validate skills/software-standards-bootstrap
```

## Tag and workflow

The release linker flags inject `.Tag` and `.FullCommit` into the binary.
Run `ssb version` or `ssb --version` to read that identity as
`ssb <version> (commit <full-commit>)`. Local builds use `dev` and `unknown`.
Version output doesn't include a wall-clock build date.

After all prerequisite checks pass, use a clean checkout of the reviewed
merge. Verify the changelog heading carries the tag date and that the tested
commit matches the current remote main before signing.

```bash
set -e
git fetch origin main
test -z "$(git status --porcelain)"
test "$(git rev-parse HEAD)" = "$(git rev-parse FETCH_HEAD)"
git tag -s v0.3.0 -m "Software Standards Bootstrap v0.3.0"
git verify-tag v0.3.0
git push origin refs/tags/v0.3.0
```

The release workflow:

1. reruns tests, race tests, vet, and vulnerability scanning;
2. extracts the tagged version's section from `CHANGELOG.md` as curated release notes;
3. builds CGO-free macOS, Linux, and Windows archives with Go 1.26.9;
4. generates `checksums.txt` with SHA-256;
5. generates one SPDX JSON SBOM per archive with pinned Syft;
6. uploads all assets and the curated notes to a draft GitHub release;
7. verifies the byte-exact GoReleaser-rendered draft body and complete,
   source-matching Agent Skill;
8. creates signed provenance over the checksum subjects;
9. creates an SBOM attestation for each archive; and
10. publishes the draft only after the release-note, archive, and attestation gates succeed.

After publication, the Windows job installs from the public tag and exercises
recovery. Retain that job's result separately from the draft verification.
Third-party actions use full commit pins. GoReleaser and Syft use exact versions.

## Verify the published release

Use a checkout of the signed tag and a fresh download directory. Run these
commands from the checkout's root in Bash. You'll need `gh` with attestation
support, `jq`, `shasum`, Git and the repository's pinned Go toolchain.

### Verify the tag, downloads and published notes

```bash
set -euo pipefail
tag=v0.3.0
repo=nnennandukwe/software-standards-bootstrap
workflow="$repo/.github/workflows/release.yml"
git verify-tag "$tag"
commit=$(git rev-parse "$tag^{commit}")
test "$(git rev-parse HEAD)" = "$commit"
gh api "repos/$repo/releases/tags/$tag" | jq -e --arg tag "$tag" \
  '.tag_name == $tag and .draft == false and .immutable == true'

release_root=$(mktemp -d)
mkdir "$release_root/assets"
gh release download "$tag" --repo "$repo" --dir "$release_root/assets"
gh release view "$tag" --repo "$repo" --json body --template '{{.body}}' > "$release_root/public.md"
git show "$tag:CHANGELOG.md" > "$release_root/changelog.md"
sh scripts/release-notes.sh "$tag" "$release_root/changelog.md" > "$release_root/expected.md"
printf '\n' >> "$release_root/expected.md"
cmp "$release_root/expected.md" "$release_root/public.md"
```

GoReleaser v2.17.0 adds the final line feed. The comparison checks the entire
published body, including that byte.

### Verify all six archives and their attestations

Continue in the same shell. The checksum manifest covers each archive and its
SPDX JSON document. Check that the download directory and checksum subjects
contain the full sets before verifying the hashes.

```bash
for os in darwin linux windows; do
  ext=tar.gz
  if [ "$os" = windows ]; then ext=zip; fi
  for arch in amd64 arm64; do
    archive="ssb_${tag}_${os}_${arch}.$ext"
    printf '%s\n' "$archive" "$archive.spdx.json"
  done
done | LC_ALL=C sort > "$release_root/subjects.txt"

(
  cd "$release_root/assets"
  awk '{print $2}' checksums.txt | LC_ALL=C sort > "$release_root/checksum-names.txt"
  cmp "$release_root/subjects.txt" "$release_root/checksum-names.txt"
  { cat "$release_root/subjects.txt"; echo checksums.txt; } | LC_ALL=C sort > "$release_root/expected-assets.txt"
  printf '%s\n' * | LC_ALL=C sort > "$release_root/downloaded.txt"
  cmp "$release_root/expected-assets.txt" "$release_root/downloaded.txt"
  shasum -a 256 --check checksums.txt
)
SSB_RELEASE_ARCHIVE_DIR="$release_root/assets" SSB_RELEASE_SOURCE_REF="$tag" \
  go test -count=1 ./internal/releaseconfig -run '^TestGeneratedReleaseArchivesContainCompleteSkill$'

trust=(--repo "$repo" --source-ref "refs/tags/$tag" --source-digest "$commit" --signer-workflow "$workflow")
while IFS= read -r name; do
  gh attestation verify "$release_root/assets/$name" "${trust[@]}" --predicate-type https://slsa.dev/provenance/v1
done < "$release_root/subjects.txt"

for archive in "$release_root"/assets/*.tar.gz "$release_root"/assets/*.zip; do
  gh attestation verify "$archive" "${trust[@]}" --predicate-type https://spdx.dev/Document/v2.3 \
    --format json > "$release_root/$(basename "$archive").attestation.json"
  jq -e --slurpfile sbom "$archive.spdx.json" \
    'any(.[]; .verificationResult.statement.predicate == $sbom[0])' \
    "$release_root/$(basename "$archive").attestation.json"
done
```

This checks twelve provenance subjects, the full skill tree in each archive
and six signed SPDX predicates against their public JSON documents. Keep the
download directory until you've recorded the results. The historical v0.1.0
archives omit the skill's root `SKILL.md`.

### Exercise the public installers

Run the tagged verification script on macOS or Linux and on a native Windows
machine. The script downloads the public installer into a fresh directory,
checks it against the tagged source and runs it.

```bash
python3 scripts/verify-public-install.py v0.3.0
```

```powershell
python scripts/verify-public-install.py v0.3.0
```

The installed binary must report the tag and full commit. Compare **every
installed skill path and its raw bytes** with the Git blobs from that tag.
Git checkout filters can change line endings on Windows. The verification
script checks the unfiltered bytes and exercises read-only recovery in an
isolated repository. Record the host OS, architecture, source commit, command
results and corresponding release workflow run. If a public check fails,
retain the failing release and its evidence while preparing a reviewed fix.

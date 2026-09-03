# 2026-09-01 portable-routing Codex conformance

This record binds issue 35's generated root, catalog, and nine routing bundles
to one observed Codex CLI 0.145.0 session. The host passed five independent
selection requests covering ordinary Go implementation and verification,
renderer work, rulepack work, and release verification.

This is exact-host conformance evidence for the recorded bytes. It does not
establish behavior for another host version, prove that any displayed recipe
ran, or claim merge, release, publication, or adoption.

## Bound implementation and bytes

The fixture was generated from projection commit
`f83f5c3c337819565207e910ce24df5392db3bdb`. After transaction hardening at
`3a4c4763ba9750cc967577d75b0c1764ea1abd5b`, the disposable generator was
replayed and the complete fixture tree was byte-identical. The host result is
therefore bound to the exact output bytes of the hardened implementation.

- Root output SHA-256:
  `a2db2625e39dce5b8486e28ac09878bf585b795405c993edb67184eefbc7bf1e`
- Root content digest:
  `sha256:9044cf6eb6a918d65b4ce454ba639f8756cbc81540cd6538bd79c177c571b0d9`
- Catalog output SHA-256:
  `c857b55cbb7be4ea8949c19e3a00d7d535e31225983a25714a774d4b8bd7ea54`
- Catalog content digest:
  `sha256:191d91c4d3294caa71508c73045cca9396aa6d516f0612db2d81ae000183d42d`
- Root routing-tree digest:
  `sha256:52f42cac777003c426ac2797b72bb7c9a6908956fa6dd9249aa683092256d682`

Every root, catalog, and bundle self-digest passed. Every catalog-to-bundle
file binding passed, and the path-sorted compact-JSON tree digest matched the
root binding. The planning-only bundle body was not read; it was accessed only
through integrity hashing.

## Selection results

| Request | Selected artifacts | Explicitly excluded contextual artifacts |
|---|---|---|
| Implement `app/service.go` | `global-safety`, `app-boundary`, `go-contract`, `implementation-contract` | planning, renderer, rulepack, verification, release |
| Verify `app/service_test.go` | `global-safety`, `app-boundary`, `go-contract`, `verify-change` | implementation, planning, renderer, rulepack, release |
| Implement `internal/render/render.go` | `global-safety`, `go-contract`, `implementation-contract`, `renderer-purity` | app, planning, rulepack, verification, release |
| Implement `internal/rulepack/rulepack.go` | `global-safety`, `go-contract`, `implementation-contract`, `rulepack-validation-seam` | app, planning, renderer, verification, release |
| Verify `.github/workflows/release.yml` | `global-safety`, `verify-change`, `release-contract` | app, Go, implementation, planning, renderer, rulepack |

The release request followed the linked skill through EOF. `go test ./...` was
displayed by the selected verification recipe but was not executed or reported
as passed. The host ran only read-only local inspection and integrity hashing;
it did not run `ssb`, repository code, or Git mutation.

The compact machine-readable identity, all bundle hashes, and full
selected/excluded artifact sets are in [run.yaml](run.yaml). The disposable
fixture and raw host transcript are intentionally not retained.

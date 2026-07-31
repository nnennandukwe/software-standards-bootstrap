# Governance

Software Standards Bootstrap is maintained in the open under Apache-2.0.

## Decision making

Routine changes are accepted through reviewed pull requests. Maintainers evaluate changes against:

- repository-specific evidence rather than generic catalog growth;
- deterministic safety and failure behavior;
- portable format claims separated from consumer-specific behavior;
- no hidden network, execution, Git mutation, or downstream activation; and
- reviewable, uncommitted developer ownership.

Changes to the canonical CLI, schema, scoring method, trust boundary, or adoption lifecycle require an ADR in this repository.

## Releases

Maintainers cut releases from reviewed tags after CI, race tests, vet, vulnerability scanning, cross-platform builds, fixture evaluation, and a recorded smoke test on at least one agent host. Release assets include checksums, SBOMs, and attestations.

## Roles

Maintainers merge changes and publish releases. Contributors propose changes and participate in review. The project may add maintainers based on sustained, constructive technical contributions and demonstrated care for the safety boundary.

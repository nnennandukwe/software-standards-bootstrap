Active artifact IDs: `use-protocol-packet-constants`, `preserve-analytics-event-continuity`, `synchronize-environment-configuration`, `regenerate-openapi-artifacts`, `verify-go-changes`.

- Repository root (`.`): `make generate-openapi-docs`
  - Expected: Generated OpenAPI v2 and v3 artifacts reflect current gateway API definitions.
- Repository root (`.`): `make test-oss`
  - Expected: OSS Go tests complete successfully after preparing the libhoop mapping and WASM artifact.

Orientation guidance: use focused verification for the affected surface before handoff; report the change, tests performed, and remaining concerns.

No command ran.
# Ledger Service fixture

This small public fixture exists to exercise Software Standards Bootstrap against a realistic repository shape. It is not a catalog of recommended rules.

## Architecture

Domain packages return contextual errors to callers. Command entry points translate those errors at the process boundary. Storage writes accept a request-scoped idempotency key.

## Verification

Contributors use `make verify` before review.

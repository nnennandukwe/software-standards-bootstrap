# Evaluation expectations

This file records human evaluation targets; it is not input to a generic rules catalog.

- `CONTRIBUTING.md` and `Makefile` provide authoritative evidence for the existing `make verify` workflow.
- Error wrapping occurs three times across two domain files and is a candidate only after scope and risk review.
- Request-scoped idempotency keys are visible in both payment and refund storage boundaries, but the fixture does not contain enough authoritative behavior to claim a complete payment invariant.
- `generated/` and `vendor/` are excluded from the inventory.
- `.env.example` is safe template evidence and is not treated as a secret.
- A host agent must not run `make verify`; it may only map that existing command.

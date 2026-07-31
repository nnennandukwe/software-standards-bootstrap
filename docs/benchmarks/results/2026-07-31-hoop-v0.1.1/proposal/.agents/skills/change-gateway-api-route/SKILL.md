---
name: change-gateway-api-route
description: Change a Hoop gateway API route while preserving access, analytics, and generated API contracts.
metadata:
  category: compatibility
---

# Change a gateway API route

Use this procedure when adding, replacing, or materially changing a route under `gateway/api/`.

1. Locate the current route registration and handler domain. Identify the role middleware, authentication, audit behavior, analytics event, handler, and existing clients before editing.
2. Preserve the repository's route-registration order: path, optional role restriction, authentication, optional audit middleware, optional analytics tracking, then handler. Make any intentional access or audit change explicit in the handoff.
3. If the route performs an already tracked action, carry forward the same analytics constant or document the deliberate successor. Never use an event-name string literal.
4. Add or update Swagger annotations on the handler and any request or response schema affected by the route.
5. Regenerate the OpenAPI artifacts with the repository recipe and inspect the resulting diff for the intended route and schema changes.
6. Add focused handler or service tests, then run the Go verification recipe.
7. Summarize compatibility, authorization, audit, analytics, and OpenAPI effects in the handoff.

If the old route must remain for compatibility, keep both paths wired to equivalent access, audit, and analytics behavior until removal is explicitly approved.

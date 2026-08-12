---
name: add-feature-flag
description: Add and apply a Hoop feature flag without exposing new behavior by default.
license: Apache-2.0
---
# Add a feature flag

Use this procedure when a new feature, behavior change, or non-trivial code path may need staged rollout.

## Procedure

1. Ask whether the change should be feature-flagged. If the answer is no, stop this procedure and record that decision in the handoff.
2. Add one entry to `catalog` in `common/featureflag/featureflag.go` using `<stability>.<snake_case_name>`, a useful admin-facing description, `Default: false`, the selected stability, and every component that consumes the flag.
3. Gate every new path in each affected component: use `featureflag.IsEnabled` in the gateway, `featureflagstate.IsEnabled` in the agent, and `feature_flags` from `/serverinfo` in the webapp.
4. Preserve the existing behavior in each disabled branch. Do not leave experimental behavior reachable on `main` when the flag is off.
5. Add or update focused tests for both enabled and disabled behavior.
6. Run the verification recipes relevant to the touched components and report the commands and outcomes in the handoff.

If a component cannot receive or evaluate the flag safely, stop and resolve that support gap before merging the experimental behavior.

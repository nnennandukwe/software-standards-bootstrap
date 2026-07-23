# Security policy

## Supported versions

Security fixes are provided for the latest published minor release.

## Reporting a vulnerability

Please use GitHub private vulnerability reporting for `nnennandukwe/software-standards-bootstrap`. Do not open a public issue containing exploit details, repository secrets, or private target-repository evidence.

Include:

- affected version and operating system;
- the smallest safe reproduction;
- whether the issue crosses the read-only, path-containment, overwrite, or Git-mutation boundary; and
- any observed filesystem effects.

## Security boundaries

`ssb inspect` reads only committed Git objects selected from the pinned tree. It excludes secret-like paths but secret detection is defense in depth, not a guarantee that a repository contains no sensitive data.

The runtime makes no network requests and executes no target-repository code. `render` and `adr` are the only CLI write surfaces. They reject symlink escapes, malformed managed state, and collisions, but developers should still review all uncommitted changes before adoption.

Release consumers should verify both `checksums.txt` and the GitHub artifact attestation.

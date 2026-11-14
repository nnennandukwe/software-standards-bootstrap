# Continue interrupted standards generation

Run this command in the repository that holds the interrupted proposal:

```bash
ssb resume --repo . --format text
```

`resume` reads existing files and Git objects. It reports the current baseline,
reusable stages, exact raw-byte identities, work to repeat, and diagnostics.
It preserves canonical sources, manual edits, projections, Git state, and ADRs.
Use `--format json` for the same assessment as a schema 1 response. Rerun the
assessment after edits or another interruption. It's a point-in-time report;
validation and rendering still enforce their own checks before publication.

## Choose the next stage

| State | Reusable work | Next action |
|---|---|---|
| `needs_proposal` | Complete inventory verified against current HEAD | Review retained candidates and finish missing semantic work; write the final manifest last. |
| `needs_correction` | Verified inventory and unaffected proposal work | Correct reported sources or metadata, update affected digests deliberately, and validate again. |
| `ready_to_render` | Inventory and fully validated proposal | Preview and render through the existing commands. |
| `complete` | Validated proposal with current projection | Review the complete uncommitted result. |
| `blocked` / `existing_pack` | Only the stages explicitly reported reusable | Resolve the blocked state, or use existing-pack consumption or explicit maintenance. |

A missing manifest leaves semantic completion unproven. Partial files remain
review material: preserve them, compare candidates with pinned Git blobs, and
finish only missing work. Creating a new file must still stop if that target
exists. Retained files require explicit review before editing; a continuation
request doesn't authorize discarding manual changes. Unlisted files in the
pack's rules, verification, or automation directories block full proposal
validation until explicitly reconciled. `.agents/skills` is shared with
developer workflows and installed tooling. Assessment names unindexed skill
paths for explicit ownership and completion review before reuse. Manifest
validation covers indexed skills; other skills stay outside its reuse claim.

A tracked manifest or report returns `existing_pack`. This protects committed
packs, including adopted packs, from being mistaken for interrupted generation.
Git tracking alone can't establish the developer's adoption decision. Embedded
packs keep their existing maintenance workflow. `existing_pack` exits `1` to
stop generation continuation.

Projection drift returns `blocked`, sets `proposal_reusable` to false, and names
`projection_recovery` before validation and rendering in the repeat stages.
Preserve canonical sources and manual edits while reconciling the reported
derived files through explicit review. Rerun assessment afterward. A source
that's missing, nonregular, or mismatched with its manifest digest when captured
also stops the assessment. A later correction requires a fresh assessment.

## Correct a validation problem

1. Run `ssb resume --repo . --format json` and read its diagnostics. Exit `1`
   means correction or trust checks still block progression. Exit `0` can
   report `needs_proposal`; check the state and `proposal_reusable` before
   rendering.
2. Edit only affected canonical sources or manifest fields after reviewing the
   intended correction. Update each changed source's matching manifest
   `sha256:` value using its exact raw bytes, including line endings. For
   example, `shasum -a 256 .software-standards/rules/<id>.md` prints the hex
   portion; retain the `sha256:` prefix in the manifest. On PowerShell, use
   `(Get-FileHash -Algorithm SHA256 <path>).Hash.ToLowerInvariant()`.
3. Rerun assessment and validation, then preview the complete write set:

   ```bash
   ssb resume --repo . --format json
   ssb validate --repo . --format text
   ssb render --repo . --dry-run
   ```

4. After successful checks, run `ssb render --repo .`. Review every changed and
   untracked file with `git --no-optional-locks status --short --untracked-files=all`.

Inventory and unaffected semantic work remain reusable for source-only
corrections. Repeat semantic judgments where changes affect evidence, intent,
or an artifact's acceptance. The CLI never refreshes digests or repairs sources.
The existing inventory, manifest, native schemas, and byte contracts stay the
same. Generated `AGENTS.md` content and `.software-standards/routing/` remain
derived; edit canonical sources rather than their generated projection.

## Recover untrusted input

Incomplete, malformed, or inconsistent inventory can't support reuse. The CLI
checks complete schema 2 accounting and reconstructs the deterministic Git
inventory at the recorded limits. This verification uses Git blobs rather than
repeating the agent's semantic repository analysis. A normal completed-proposal
assessment reconstructs inventory once, then rechecks retained byte identities.
Separate validation and rendering commands check their inputs again. A changed
HEAD blocks reuse
of old inventory; a manifest with another baseline blocks proposal reuse.
Tracked baseline-source edits and renames also block continuation. Git's
`assume-unchanged` and `skip-worktree` index flags can hide edits, so a flagged
baseline source blocks assessment even when ordinary status looks clean.
Preserve its bytes and review the flags with `git ls-files -v --`. After
reviewing the file, the developer can restore visibility with
`git update-index --no-assume-unchanged -- <path>` or
`git update-index --no-skip-worktree -- <path>`, as applicable. Assessment never
clears these flags or restores files automatically. Resolve remaining visible
source edits before reassessment. Manual text surrounding
the managed `AGENTS.md` section in an existing regular root file is preserved by the existing renderer; edited
managed content or routing files trigger drift diagnostics. A missing or nonregular
tracked root file also blocks continuation. The manual-edit exception requires
a tracked index entry without merge conflicts at `AGENTS.md`; an untracked
replacement can't hide a staged root-file rename.

Preserve the entire interrupted pack and manual work. If the original baseline
and complete inspection response are available, restore them deliberately and
reassess. Otherwise, move the retained pack to a separate, reviewed backup
location before fresh inspection against a stable baseline. Keep partial
canonical files and repository skills for comparison; never overwrite them or
copy old hashes onto new evidence. A fresh generation must review every retained
candidate it intends to reuse. Moving or replacing retained work requires the
developer's deliberate action; `resume` performs neither operation.

Interruption during assessment exits `130` before output delivery. Rerun the
same command: assessment has no durable writes to recover. If rendering stops,
follow its existing recovery diagnostics and reassess before retrying. Every
new validation or rendering invocation rechecks its own input. Concurrent
changes detected during assessment reject the result and preserve those changes.
Captured inputs are checked again after the projection dry run, so changes
during that check also require a fresh assessment.
The final check includes unlisted native files, invalid proposal directories,
and an optional orientation file that appears without a manifest reference.
The manifest must bind the captured inventory and source bytes before full
validation. Validation must also use the captured manifest bytes; temporarily
replacing inputs and restoring their old bytes can't authorize reuse.

A successful projection remains a proposal until developer review and the
adoption decision through the developer-created pull request and merge. Stop
before an ADR unless the developer explicitly requests it. Recipe execution and
automation implementation keep their separate authorization requirements.

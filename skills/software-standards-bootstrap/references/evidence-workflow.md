# Evidence workflow

## Canonical bytes

Evidence is tied to the `baseline_commit` returned by `ssb inspect`. Hash the exact bytes from the inclusive one-based line range, preserving `LF` or `CRLF` endings. Do not hash copied prose, normalized whitespace, worktree-only edits, or model summaries.

`ssb validate` reads the same range from the pinned Git blob and reports the expected digest when a hash is wrong.

## Authority

Mark `authoritative: true` only when the repository itself treats the cited location as normative, such as a maintained architecture decision, contribution contract, security requirement, or configuration that owns the behavior. Repetition alone is not authority.

Without an authoritative source, require three consistent occurrences across at least two files. Similar-looking code with different state or risk boundaries is not consistent evidence.

## Existing proof

An existing command is deterministic evidence only for the bounded property it checks. Cite the repository location that defines the command. Do not run it and do not claim it passed.

Examples:

- a linter configuration can prove the encoded lint rule when the linter is run elsewhere;
- a test command can prove only its encoded assertions;
- a green build does not prove every integration or production interaction; and
- prose guidance without a checker remains guidance with a proof gap.

## Assessment-only candidates

Keep a candidate in `.software-standards/assessment.md` when:

- its score is below 25;
- evidence does not meet the authority/occurrence threshold;
- scope is unclear;
- classification depends on unverified behavior; or
- the recommendation is repository context rather than durable guidance.

Record why it was not emitted. Do not manufacture evidence to fill a rule count.

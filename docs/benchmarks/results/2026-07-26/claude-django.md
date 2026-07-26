# Django / Claude Code inventory-v2 proposal record

Generated on 2026-07-26. This record is at the mandatory developer-review gate;
it is not a completed benchmark judgment and contains no ADR.

## Runtime and immutable inputs

- Consumer: Claude Code 2.1.220
- Model: reported by the session environment as Opus 5 (1M context), model id
  `claude-opus-5[1m]`. **Not independently observable** from `claude --version`;
  recorded as self-reported rather than verified.
- Observable configuration: `learning` output style active. No reasoning-effort
  setting is exposed by the CLI, so none is claimed.
- Host: macOS 15.7.3 build 24G419 (`arm64`)
- Git: 2.39.5 (Apple Git-154)
- Repository: `github.com/django/django`
- Baseline commit: `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f`
- Evaluation branch: `ssb-claude-v2-evaluation`
- SSB source commit: `820c3a8cce538c0971713aa997992f05d8d3e0c2`
- Evaluator binary SHA-256:
  `c9d93a2aeef27249a1fc828fef025e77f6f0dd742d847bc4927cd8c538fd6fba`
- Raw inventory output SHA-256:
  `7dfbdc561cf24d8d2bef2807e4e1bf4b81266740deff444f36da9dee230d585c`

The Codex inventory-v2 Django pack was not read before this evaluation. This run
is independent.

## Inventory

- Contract: `ssb-inventory-v2`, schema 2
- Candidate: 7,001 files, 45,506,636 bytes
- Scanned: 7,001 files, 45,506,636 bytes
- Indexed: 5,619 files, 36,820,618 bytes
- Remaining: 0 files, 0 bytes
- Truncated: no
- Limits: 40,000 candidate files; 134,217,728 candidate bytes; 1,048,576 bytes
  per file
- Excluded: 1,382 binary, 72 vendor-or-generated tree, 4 symlink; all other
  categories 0

Under the previous inventory contract this repository truncated partway through
`tests/`. At this baseline the whole tree is indexed, so statements about what
the repository does *not* contain are reliable.

## Pending proposal

`ssb validate` passed with 9 rules, 1 related skill, and 100% evidence
resolution. Validation passed on the first attempt with no corrections.

| Rule | Primary topic | Classification | Score | Decision |
|---|---|---|---:|---|
| `preserve-deprecation-cycle` | compatibility | guidance | 86 | Pending |
| `keep-database-backend-modules-aligned` | architecture | guidance | 78 | Pending |
| `run-repository-linters` | maintainability | deterministic | 78 | Pending |
| `check-test-migrations` | correctness | deterministic | 74 | Pending |
| `keep-sync-async-api-parity` | compatibility | guidance | 74 | Pending |
| `file-trac-ticket-before-patch` | developer-experience | guidance | 74 | Pending |
| `disclose-ai-assistance` | compliance | guidance | 70 | Pending |
| `end-commit-messages-with-a-period` | maintainability | guidance | 70 | Pending |
| `do-not-request-automated-ai-review` | compliance | guidance | 68 | Pending |

Proposed skill: `add-async-api-variant` (`compatibility`) — **Pending review**.

## Structural review and candidate dispositions

All five dimensions were reviewed and recorded in
`.software-standards/assessment.md`.

| Dimension | Disposition |
|---|---|
| Package and dependency boundaries | Emitted `keep-database-backend-modules-aligned`. |
| Parallel implementation families | Emitted `keep-sync-async-api-parity` plus the proposed skill. |
| Platform and configuration seams | Assessment-only. The database seam is already governed by the backend rule and the migration check; a further rule would restate CI topology. |
| Public compatibility surfaces | Emitted `preserve-deprecation-cycle`, the strongest compatibility finding in the repository. |
| Source, test, and documentation symmetry | Emitted `check-test-migrations`. The release-notes obligation is folded into the deprecation rule and the skill. |

Verified rather than assumed:

- The backend module set is deliberately non-uniform. `postgresql` and `sqlite3`
  define no `validation.py`; `oracle` and `sqlite3` define no `compiler.py`. The
  rule tells the reader to match the modules the affected backend actually
  defines, because a rule demanding a uniform set would be wrong.
- The sync/async pairing recurs dozens of times across `django/` — `adelete` and
  `acreate` appear nine times each — far above the three-occurrence threshold.
  No document states the convention, which is recorded as the weakest authority
  in the retained set.

Two rules capture the repository's explicit position on AI-assisted
contribution: disclosure is mandatory, and requesting an automated AI review on
an upstream pull request is prohibited in two independent authoritative places
(the pull-request checklist and the repository's assistant instruction file).

## Rendered output

The derived `AGENTS.md` was created fresh; Django had no pre-existing
`AGENTS.md`. Digests:

- source digest:
  `sha256:7c6b8b015547acaab95b0a55741b49ad08a5e07639fc444c0a5168cde46bf928`
- content digest:
  `sha256:59184452465bcacf32b297fba96697505d339298f8ba04398794a8724e57c12e`

`ssb validate` was run again after rendering and passed.

## Proposed paths

```text
.agents/skills/add-async-api-variant/SKILL.md
.software-standards/assessment.md
.software-standards/rules/check-test-migrations.md
.software-standards/rules/disclose-ai-assistance.md
.software-standards/rules/do-not-request-automated-ai-review.md
.software-standards/rules/end-commit-messages-with-a-period.md
.software-standards/rules/file-trac-ticket-before-patch.md
.software-standards/rules/keep-database-backend-modules-aligned.md
.software-standards/rules/keep-sync-async-api-parity.md
.software-standards/rules/preserve-deprecation-cycle.md
.software-standards/rules/run-repository-linters.md
AGENTS.md
```

The untracked `.claude/skills/software-standards-bootstrap` symlink is the
evaluator's own skill attachment and is not part of the proposal.

## Safety and review boundary

- The clone was fresh and `HEAD` remained at
  `50c2b7c83661a61da48f78dd0130fc3cbf8ed39f` throughout.
- No Django code, test, hook, linter, formatter, script, or package manager was
  executed. `python scripts/check_migrations.py`,
  `isort --check --diff django tests scripts`, flake8, black, biome, zizmor, and
  the commit-message check jobs are cited only.
- No Django dependency was installed; no database was started.
- The Git index is clean: 0 staged paths and 0 modified tracked files. All
  proposal paths are untracked.
- No Git mutation beyond creating the initial attached evaluation branch.
- No ADR was previewed or created, and `ssb adr` was never run.

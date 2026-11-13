package cli

const helpText = `Software Standards Bootstrap

Usage:
  ssb inspect  [--repo PATH] [--format text|json] [resource limits]
  ssb validate [--repo PATH] [--format text|json]
  ssb render   [--repo PATH] [--review ID] [--dry-run]
  ssb adr      [--repo PATH] [--review ID] [--adr-dir PATH] [--dry-run]
  ssb prune    <inspect|validate|approve|apply|recover|status|verify> [options]
  ssb version
  ssb --version

Exit codes:
  0  success
  1  actionable-pack or prune-proposal validation failure
  2  usage or repository precondition failure
  3  unexpected internal failure
  4  inventory coverage incomplete
  130 interrupted before output delivery

Independent failures keep their exit code and recovery diagnostics.
`

const (
	inspectHelp = `Usage: ssb inspect [--repo PATH] [--format text|json] [--max-candidate-files N] [--max-candidate-bytes N] [--allow-partial]

  --repo PATH                 target Git repository (default ".")
  --format text|json          stable output format (default "text")
  --max-candidate-files N     candidate files that may be read (default 40000)
  --max-candidate-bytes N     candidate bytes that may be read (default 134217728)
  --allow-partial             return success for explicitly diagnostic partial coverage
`
	validateHelp = `Usage: ssb validate [--repo PATH] [--format text|json]

  --repo PATH              target Git repository (default ".")
  --format text|json       validation output format (default "text")
`
	renderHelp = `Usage: ssb render [--repo PATH] [--review ID] [--dry-run]

  --repo PATH              target Git repository (default ".")
  --review ID              record rerendering for an applied prune review
  --dry-run                preview the complete proposed root and routing write set
`
	adrHelp = `Usage: ssb adr [--repo PATH] [--review ID] [--adr-dir PATH] [--dry-run]

  --repo PATH              target Git repository (default ".")
  --review ID              record the ADR as a separate prune review state
  --adr-dir PATH           explicit repository-contained ADR directory
  --dry-run                print the complete proposed ADR without writing
`
	pruneHelp = `Usage: ssb prune <command> [options]

Commands:
  inspect   create an immutable, complete review context
  validate  validate a host-agent proposal without approving it
  approve   record one explicit decision for every action
  apply     show a dry run; pass --write to apply approved changes
  recover   restore an interrupted application
  status    report proposal, approval, no-change, application, render, ADR, and verification separately
  verify    validate external content-addressed check receipts
`
	pruneInspectHelp = `Usage: ssb prune inspect --review ID --capabilities PATH [--provenance PATH] [--repo PATH] [--format text|json] [resource limits]

  --review ID                  lower-case kebab-case review identifier
  --capabilities PATH          local point-in-time capability profile
  --provenance PATH            optional explicit artifact provenance manifest
  --repo PATH                  target Git repository (default ".")
  --format text|json           compact result format (default "text")
  --max-candidate-files N      candidate files that may be read (default 40000)
  --max-candidate-bytes N      candidate bytes that may be read (default 134217728)
`
	pruneValidateHelp = `Usage: ssb prune validate --review ID [--repo PATH] [--format text|json]

  --review ID              review bundle identifier
  --repo PATH              target Git repository (default ".")
  --format text|json       compact validation format (default "text")
`
	pruneApproveHelp = `Usage: ssb prune approve --review ID [--approve ID[,ID...]] [--reject ID[,ID...]] [--repo PATH]

  --review ID              validated review bundle identifier
  --approve ID[,ID...]     actions explicitly approved by a human
  --reject ID[,ID...]      actions explicitly rejected by a human
  --repo PATH              target Git repository (default ".")
`
	pruneApplyHelp = `Usage: ssb prune apply --review ID [--repo PATH] [--format text|json] [--write]

  Without --write, this command is a reviewable dry run.
`
	pruneRecoverHelp = `Usage: ssb prune recover --review ID [--repo PATH] [--clear-stale-lock]

  --review ID              interrupted review bundle identifier
  --repo PATH              target Git repository (default ".")
  --clear-stale-lock       clear review-owned crash-left locks after confirming no process is active
`
	pruneStatusHelp = `Usage: ssb prune status --review ID [--repo PATH] [--format text|json]

  --review ID              review bundle identifier
  --repo PATH              target Git repository (default ".")
  --format text|json       compact lifecycle-state format (default "text")
`
	pruneVerifyHelp = `Usage: ssb prune verify --review ID --receipts PATH [--repo PATH]

  --review ID              applied review bundle identifier
  --receipts PATH          directory of external content-addressed receipts
  --repo PATH              target Git repository (default ".")
`
)

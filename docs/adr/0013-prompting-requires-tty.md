# 13. Prompting requires a TTY

Date: 2026-11-13
Status: accepted

## Context

Without `--vars-file`, `jig render` prompted — including in CI, where
stdin is a pipe and the prompt hangs forever on a question nobody can
answer.

## Decision

`--vars-file` renders non-interactively; so does `--var` when stdin is not
a terminal (remaining variables take their resolved defaults).
Non-interactive stdin with no values at all is a hard error naming
`--vars-file`/`--var`. Interactive answers are validated as they are
given — a value failing its pattern or choices is rejected and re-prompted
immediately, not saved up for a failure at the end of the prompt run.

Terminal detection is a real terminal check (`term.IsTerminal`), not a
character-device check: `/dev/null` is a character device but not a
terminal, and CI runners, cron jobs and agent shells often present it as
stdin. It counts as non-interactive.

## Consequences

- Pipelines fail fast with an actionable message instead of hanging.
- `--var`-only is the natural CI form and works without a values file;
  fully default renders in scripts require one explicit flag
  (`--var` with any declared variable, or a vars-file) so "ran with all
  defaults" is always a choice, never an accident of a lost terminal.

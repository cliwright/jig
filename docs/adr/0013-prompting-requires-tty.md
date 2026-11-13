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
`--vars-file`/`--var`.

## Consequences

- Pipelines fail fast with an actionable message instead of hanging.
- `--var`-only is the natural CI form and works without a values file;
  fully default renders in scripts require one explicit flag
  (`--var` with any declared variable, or a vars-file) so "ran with all
  defaults" is always a choice, never an accident of a lost terminal.

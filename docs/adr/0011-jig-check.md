# 11. `jig check` dry-render command

Date: 2026-11-13
Status: accepted

## Context

A broken template shipped: an undeclared variable reference only surfaced
when a *user* rendered it. Template authors had no way to validate without
rendering for real, and unused-variable warnings only printed after files
hit disk.

## Decision

`jig check` dry-renders a template without writing anything: the manifest
is validated, the resolved defaults are validated against their own
patterns and choices, every file and directory name and every text file's
content is parsed and rendered against those defaults (missingkey=error,
internal collisions, containment — the rendered top-level project name
included — and symlink policy all apply), and references to undeclared
variables are rejected statically across every branch of every template,
not just the branches the defaults execute. Declared-but-unused variables
print as warnings. Exit non-zero on any error.

## Consequences

- Authors run it before shipping; CI can gate template repos on it.
- Checking and rendering share the walk and the content renderer, so check
  results predict render results — no second code path to drift.
- The limits, honestly: check validates against the resolved *defaults*, so
  a user-supplied value can still fail its own pattern or choices at render
  time; and it validates the template, not a future destination —
  writability and collisions with existing files are render-time concerns.

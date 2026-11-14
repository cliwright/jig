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
is validated, every file and directory name and every text file's content
is parsed and rendered against the resolved defaults (missingkey=error,
containment, and symlink policy all apply), and unused variables print as
warnings. Exit non-zero on any error.

## Consequences

- Authors run it before shipping; CI can gate template repos on it.
- Checking and rendering share the walk and the content renderer, so check
  results predict render results — no second code path to drift.
- The limits, honestly: check renders against the resolved *defaults*, so
  it cannot guarantee every user-supplied value combination renders
  cleanly; and it validates the template, not a future destination —
  writability and collisions are render-time concerns.

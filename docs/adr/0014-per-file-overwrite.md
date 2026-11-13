# 14. Overwrite refusal is per file, not per directory

Date: 2026-11-13
Status: accepted

## Context

The original rule was "refuse to render into a non-empty output directory;
`--force` overrides". In practice that forbids useful, safe renders —
adding a component into an existing project tree, or rendering a second
template next to the first — while the actual danger was always narrower:
clobbering an existing *file*.

## Decision

The unit of refusal is the file. The whole render is planned first; if any
planned file already exists, the render fails before writing anything
(unless `--force`). Rendering into a populated directory is fine as long as
no file collides.

## Consequences

- Multi-template project trees work without force flags.
- Safety is preserved by plan-then-render: a collision fails before any
  write, so a refused render never half-writes.
- Supersedes the directory-level rule from the original spec.

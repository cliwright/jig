# 16. Prepare everything before writing anything

Date: 2026-11-15
Status: accepted

## Context

Two gaps in the render pipeline undermined documented guarantees. First,
planning resolved names only; file contents rendered per-file *during* the
write phase, so a template error on the third file left the first two on
disk — a partial tree, despite "planned first" being a documented
property. Second, nothing compared planned destinations against each
other: two entries rendering to the same path (`{{.A}}.txt` and
`{{.B}}.txt` with equal values) silently overwrote one another, even
without `--force`.

## Decision

Rendering is two strict phases. **Prepare** (no destination filesystem
writes): render every name and every file's content into memory, vet
symlinks and special files, and reject internal collisions — two sources
rendering to one file, or a file and directory sharing a path. Distinct
directories may merge into one rendered directory (conflicting files
inside are still caught). **Write** runs only when prepare fully succeeds:
create directories, write the prepared bytes, recreate vetted symlinks.

## Consequences

- A template error — parse, execution, collision, containment — never
  leaves a partial tree. The "planned first" documentation is now true.
- Modes are exact: writes are followed by `Chmod` to the template's mode,
  so executables survive umask, and `--force` restores the template's mode
  instead of keeping the old file's.
- Accepted limits: disk failures mid-write and concurrent filesystem
  changes are outside the guarantee; projects are held in memory, which is
  irrelevant at scaffold sizes.
- `jig check` shares the prepare phase, so its results predict renders.

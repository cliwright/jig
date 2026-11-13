# 7. Rendered paths must stay inside the output directory

Date: 2026-11-13
Status: accepted

## Context

Rendered file and directory names come from variable values — user input.
The only thing stopping `--var PROJECT_NAME=../../evil` from writing
outside `--output` was the manifest's validation pattern, which the
*template author* controls. A malicious or sloppy registry template could
set a permissive pattern and turn rendering into an arbitrary file write —
exactly the "templates are vettable data" promise broken.

## Decision

jig enforces containment itself, independent of the manifest: every
rendered output path must stay inside the output directory. Empty names,
`..` segments, and absolute results are hard errors. Interior slashes are
allowed — a name may legitimately render to a nested path.

## Consequences

- Containment no longer depends on every manifest author getting their
  patterns right; the guarantee comes from the renderer.
- A legitimate-but-weird use (intentionally rendering outside `--output`)
  is impossible. Accepted: that use is indistinguishable from an attack.

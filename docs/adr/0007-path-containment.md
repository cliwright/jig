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

jig enforces containment itself, independent of the manifest, at two
levels:

1. **Lexical:** every rendered output path must stay inside the output
   directory. Empty names, `..` segments, and absolute results are hard
   errors. Interior slashes are allowed — a name may legitimately render
   to a nested path.
2. **Filesystem:** before writing, every existing component of every
   planned path is checked with `Lstat`; a symlink anywhere in the
   destination chain is a hard error, `--force` or not. Lexical
   containment alone is not enough — a symlinked directory already sitting
   in the output tree would otherwise redirect writes outside it, and a
   symlinked destination file would turn `--force` into an arbitrary-file
   clobber. `--force` means "overwrite regular files", never "follow
   someone's link".

## Consequences

- Containment no longer depends on every manifest author getting their
  patterns right; the guarantee comes from the renderer.
- A legitimate-but-weird use (intentionally rendering outside `--output`)
  is impossible. Accepted: that use is indistinguishable from an attack.
- The filesystem check closes symlink redirection at validation time; a
  TOCTOU race (an attacker swapping in a symlink between the check and the
  write) is accepted as out of scope for a local CLI rendering into the
  user's own directories.

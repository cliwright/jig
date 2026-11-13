# 6. Empty directories are mirrored, not special-cased

Date: 2026-11-08
Status: accepted

## Context

Scaffolds often want empty directories (`data/`, `output/`), and other
tools invent conventions (`.gitkeep` handling, placeholder creation) to
support them.

## Decision

Templates contain no empty directories, same as git. Placeholder files
(`.gitkeep`, `.keep`, `.empty`) are ordinary files: rendered with their
parent directory, no special handling.

## Consequences

- Zero special cases in the walker — a file is a file.
- Template authors use placeholder files, which are also self-documenting
  (a placeholder can contain a sentence about what the directory is for).

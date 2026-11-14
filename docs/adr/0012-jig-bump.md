# 12. `jig bump` version-copy command

Date: 2026-11-13
Status: accepted

## Context

Template layout mandates version directories (`v1/`, `v2/`, …), but
starting `v2` meant copy-pasting `v1` by hand — and `jig new` only ever
scaffolds `v1`.

## Decision

`jig bump <language>/<flavor>/<kind>` copies the latest version directory
to `v<N+1>` as the starting point for the next iteration. Old versions stay
put; renders pin theirs with `--version`. Bump refuses symlinks, matching
the default render policy.

## Consequences

- Version iteration is one command; the immutable-versions story (pin with
  `--version`, latest by default) stays intact.
- Bump is a pure file operation: it renders nothing and evaluates no
  manifest.
- Bump creates a starting point, not a compatibility promise: whether
  `v<N+1>` is a breaking change for existing values files is the author's
  call to make and document.

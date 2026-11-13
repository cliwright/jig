# 8. Symlinks rejected by default; `--trust` opts in

Date: 2026-11-13
Status: accepted

## Context

Templates shipped without a symlink policy: reading a symlink follows it,
so a template could silently copy the *target's content* — including a link
pointing anywhere on the host filesystem. Options: copy links as links,
follow them deliberately, or refuse them.

## Decision

Templates must not contain symlinks; rendering one fails by default.
`--trust` (library: `Options.Trust`) opts in: links are recreated as
symlinks in the output — but links resolving outside the template tree are
still refused. Trust is about intent, not containment.

## Consequences

- The default render of a stranger's template cannot be smuggled a pointer
  to the host filesystem.
- Template authors who legitimately want links get them with one flag, and
  still can't (accidentally) ship an escaping link.
- smith, as a library caller, renders with the default: reject.

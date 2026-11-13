# 15. No remote fetching, not even via git wrappers

Date: 2026-11-13
Status: rejected

## Context

A `jig fetch <git-url>` command was considered: clone a template repo into
`JIG_TEMPLATES_DIR`, then render locally as today. It looked like a clean
middle ground — fetching kept out of the render path, no go-git dependency
(shell out to the user's `git`).

## Decision

Rejected. Users clone a repo themselves and point `JIG_TEMPLATES_DIR` at
it. jig does not wrap git.

## Consequences

- Git already solves everything fetch would need — auth, updating
  (`git pull`), pinning (`git checkout <tag>`), private repos, mirroring —
  and users already know it. A wrapper is a narrower interface over the
  same failure modes.
- The trust story stays airtight ([ADR 4](0004-local-rendering-only.md)):
  jig never touches the network; the trust decision happens explicitly, at
  clone time, in the user's own git setup.
- `--template` accepts a direct filesystem path, so repos not following
  the `<language>/<flavor>/<kind>` layout still work.
- The cost is two lines of setup instead of one command — a documentation
  matter, not a feature gap.

# 4. Local rendering only

Date: 2026-11-08
Status: accepted

## Context

Should jig fetch templates from remotes (URLs, registries, git)? Fetching
would make one-command scaffolds possible, but pulls downloading, custody,
and trust into the rendering tool. smith already owns a registry and
custody chain for exactly this.

## Decision

jig renders what is on disk, period. Downloading and unpacking belong to
the caller: smith's registry/custody chain for smith users, plain `git
clone` for everyone else. (See [ADR 15](0015-no-remote-fetching.md) for the
git-wrapper variant of this question, also rejected.)

## Consequences

- jig has no network code, no auth, no caching, no ref semantics — and
  never inherits a remote's trust problems.
- A template may later ship inside a smith ingot: smith unpacks, jig
  renders.
- CLI users do one explicit step themselves (clone, then point
  `JIG_TEMPLATES_DIR` at it). Deliberate: the trust decision happens at
  clone time, in the user's own git setup.

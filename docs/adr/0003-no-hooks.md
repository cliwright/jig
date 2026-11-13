# 3. No hooks, no arbitrary code execution

Date: 2026-11-08
Status: accepted

## Context

cookiecutter's pre/post-gen hooks run arbitrary Python from inside a
template at scaffold time. That makes "render a stranger's template" a
code-execution decision, and is the main reason cookiecutter was replaced.
Any hook mechanism — scripts, plugin commands, template-side setup steps —
reopens that door.

## Decision

Never. Templates are inert, vettable data: rendering walks the tree and
writes files, and nothing in a template can execute. Post-render steps
(repo wiring, provenance stamps, next-step hints) are the caller's job,
never jig's and never the template's.

## Consequences

- A registry-distributed template is never "run this stranger's code" —
  this is the security model and a marketing line.
- Anything a hook could do, jig refuses to do. Setup steps belong in the
  rendered project's own tooling, documented in its README, run by a human
  who can see it.
- This decision constrains everything else: no engine slots
  ([ADR 1](0001-library-and-cli.md)), a vetted func map
  ([ADR 2](0002-text-template-only.md)), symlink suspicion
  ([ADR 8](0008-symlink-policy.md)).

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

Never. Templates are inert, vettable data. Precisely: no template-supplied
programs are executed. Rendering evaluates the standard template language
and invokes only jig's explicitly registered, reviewed functions — there is
no mechanism by which a template can run a script, a hook, or a shell
command. Post-render steps (repo wiring, provenance stamps, next-step
hints) are the caller's job, never jig's and never the template's.

Note the boundary of the guarantee: it covers rendering. What generated
files do when someone later executes them (a rendered `Makefile`, a setup
script) is outside jig's control and outside its claim.

## Consequences

- A registry-distributed template never runs code *at render time* — this
  is the security model and a marketing line.
- Anything a hook could do, jig refuses to do. Setup steps belong in the
  rendered project's own tooling, documented in its README, run by a human
  who can see it.
- This decision constrains everything else: no engine slots
  ([ADR 1](0001-library-and-cli.md)), a vetted func map
  ([ADR 2](0002-text-template-only.md)), symlink suspicion
  ([ADR 8](0008-symlink-policy.md)).

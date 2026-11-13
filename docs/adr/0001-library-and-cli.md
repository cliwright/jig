# 1. One module: library for smith, CLI for the world

Date: 2026-11-08
Status: accepted

## Context

jig replaces cookiecutter inside smith, but it also needs a public CLI for
everyone else and the OSS story. Options: smith shells out to the jig
binary; smith vendors a copy; smith imports jig as a Go library. A separate
question was whether the rendering engine should be a swappable slot in
smith (bring-your-own-template-engine).

## Decision

One Go module, `github.com/cliwright/jig`: the library at the root package,
the CLI at `cmd/jig`. smith imports the library in-process, pinned via
go.mod. The engine is NOT swappable inside smith: the rendering contract
(jig.json schema, func map, no-hooks) is part of smith's template contract
and trust model.

## Consequences

- No version skew, no PATH dependency, smith's single-binary story
  preserved.
- An engine slot would let cookiecutter-style hooks back in through the
  side door ([ADR 3](0003-no-hooks.md)) — the contract is the point, so the
  engine is fixed.
- Every library API change is a smith consideration, not just a CLI one.

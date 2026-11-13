# 2. `text/template` only, hand-rolled func map

Date: 2026-11-08
Status: accepted

## Context

Templates need conditionals, pipelines, and string munging. Go's
`text/template` covers the syntax; the question was the function library.
sprig is the de-facto standard (~100 functions) but is a third-party
dependency with a large surface, and most of its functions invite templates
to become programs.

## Decision

The standard library's `text/template`, plus a small hand-rolled func map
of string helpers (`lower`, `upper`, `replace`, `slugify`). No third-party
template libraries. Templates stay as simple as possible.

## Consequences

- Zero template-engine dependencies; the func map is small enough to vet
  line by line, which matters for smith's trust model.
- Every function added to the map is a forever-contract — the bar for new
  ones is high.
- Templates that need sprig's richer functions must restructure (usually:
  move the logic into a derived default or the caller's post-render step).

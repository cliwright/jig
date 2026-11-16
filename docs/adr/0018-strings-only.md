# 18. String values only (no lists, no typed variables)

Date: 2026-11-16
Status: rejected

## Context

Every variable in jig is a string: the manifest declares string defaults,
`--var` takes `key=value`, `--vars-file` takes a JSON object of strings,
and templates see a flat `map[string]string`. Multiple independent reviews
have flagged this as missing functionality — "no lists, no loops, you
cannot render one file per table" — so the decision is recorded here
rather than re-litigated per review.

The recurring use case is real: generate one file (or one config stanza)
per item in a list. But look at what adopting it costs. Lists mean typed
manifest values, a list syntax for `--var`, typed `--vars-file` handling,
per-item pattern validation, derivation over collections — and the hard
one, an interactive prompt UX for entering a list. "All values are
strings" is one sentence in the docs and zero code paths; "strings or
lists of strings" is a small type system, and per ADR 2, everything added
becomes a permanent contract.

The feedback so far is expectation mismatch, not blockage: reviewers
listed the absence as a con, then completed their actual scaffolds without
it. Booleans are already covered — `choices: ["yes", "no"]` plus
`{{if eq .X "yes"}}` — so the genuine gap is only iteration.

## Decision

Variables stay strings. No list type, no typed values, no `range`-able
collections. Iteration-shaped work belongs outside the renderer: render
once per item with different `--var` values, or generate the repetitive
files with a script (or an agent) in the target repo — the same place
jig sends everything a hook would have done (ADR 3).

## Consequences

- The manifest schema, value resolution, prompting and validation stay
  simple and stay strings; the docs promise stays one sentence.
- Templates remain close to WYSIWYG: no file appears N times because a
  value happened to be long.
- Revisit trigger: real users blocked on real templates, reported as an
  RFC — not review feedback alone. A reversal gets a new ADR superseding
  this one.

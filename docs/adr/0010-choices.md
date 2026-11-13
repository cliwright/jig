# 10. `choices` for enum variables

Date: 2026-11-13
Status: accepted

## Context

Enum-like variables (license, CI provider, database) were expressible only
as regex patterns — `^(MIT|Apache-2.0)$` — which prompts badly (the user
can't see the options) and reads worse than it should.

## Decision

Variables may declare `choices`: a fixed list of allowed values, mutually
exclusive with `pattern`. A static default must be one of the choices;
interactive prompts list the options; the final value is validated at
render time like any other.

## Consequences

- Enums are declarative and self-documenting in prompts.
- Two validation mechanisms now exist, kept disjoint (pattern XOR choices)
  so a variable's contract is never ambiguous.

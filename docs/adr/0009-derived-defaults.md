# 9. Derived defaults

Date: 2026-11-13
Status: accepted

## Context

The commonest real-template pattern is "package/module name defaults to a
munged project name". With static-only defaults this meant redundant
prompting or `{{ .PROJECT_NAME | slugify }}` repeated across every file —
the first hand-written template immediately wanted it.

## Decision

A variable's `default` may template over other variables
(`"default": "{{ .PROJECT_NAME | slugify }}"`). Defaults resolve in
dependency order at load; references to undeclared variables,
self-references, and cycles are load errors. Explicit overrides
(`--vars-file`, `--var`, interactive answers) win per-variable, and
dependents re-derive against the overridden value. A variable referenced by
another variable's default counts as used (no cruft warning).

## Consequences

- One prompt answers a whole family of derived names; overriding the root
  re-derives the family.
- Manifests carry a dependency graph — small (topological sort at load) and
  fully validated before any prompt is shown.
- Derived defaults compose with validation: the final value still faces
  its pattern or choices at render time.

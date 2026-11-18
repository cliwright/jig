# 19. Conditional files and directories: empty-rendered names are excluded

Date: 2026-11-18
Status: accepted

## Context

The most-requested scaffolder feature jig lacked: "include this file only
when a variable asks for it" — a Dockerfile for one choice and not the
other, a `cmd/` tree for CLIs but not libraries. Without it, every
combination of options needs its own template, which multiplies
maintenance exactly where a registry wants one canonical template.

Cookiecutter solves this with post-generation hooks that delete files;
Copier solves it with empty-rendering path segments. Hooks are rejected
here ([ADR 3](0003-no-hooks.md)) — the mechanism had to be inert data.

Until this ADR, a name that rendered to an empty string was a hard error
("rendered an empty name"), so the semantic slot was free: no working
template could produce one.

## Decision

A file or directory whose *name* renders to an empty string is excluded
from the output. Conditionals live in the name; content stays plain:

```
{{if eq .WITH_DOCKER `yes`}}Dockerfile{{end}}
{{if eq .KIND `cli`}}cmd{{end}}/main.go
```

(Backtick raw strings, not `"` — quotes are illegal in Windows file
names.)

Rules:

- Excluding a directory prunes its whole subtree.
- The top-level project directory (`{{.PROJECT_NAME}}`) may **not**
  render empty — that remains a hard error. An empty project name is a
  broken invocation, never a conditional.
- Excluded entries are still fully validated: names are contained,
  contents are parsed and executed against the current values, symlink
  and special-file policy applies. `jig check` covers every conditional
  branch, not only the ones the defaults select — the same totality
  ADR 11 demands of `{{define}}` bodies.
- Only emitted entries take part in collision detection: two siblings
  excluded under the same values cannot collide in the output.
- Variables referenced only in excluded branches count as *used* — no
  false unused-variable warnings for conditional-only variables.

A manifest-level alternative (`when:` expressions mapping paths to
conditions) was considered and rejected: it invents a second addressing
scheme for template content, splits the conditional away from the thing
it conditions, and buys nothing the name form cannot express.

## Consequences

- Optional content needs no hooks and no template multiplication — one
  template covers every combination of its choices.
- Template authors must know the empty-name rule: a variable typo that
  renders empty now *excludes* a file silently instead of failing. The
  mitigations are deliberate: `missingkey=error` catches undeclared
  variables, and an author who wants a mandatory file asserts it by not
  conditioning its name.
- Windows-safe conditional names require backtick raw strings (quotes
  are illegal in Windows file names); the docs say so.
- Verbatim entries are unaffected: their names are never rendered, so
  they can never be excluded.
- Render stays a pure function of template + values: exclusion is
  decided per render, so the same template rendered with different
  choices produces disjoint, correct trees.

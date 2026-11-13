# Jig project template rendering tool (like cookiecutter)

## Language - golang

## Goal
Create a stand alone cli application + reusable Go library that renders project templates.

## Decisions

Design decisions — including rejected ones — live in
[`docs/adr/`](../docs/adr/index.md) as numbered ADRs (accepted:
library/CLI split, text/template-only, no hooks, local rendering,
containment, symlink policy, derived defaults, choices, check, bump,
TTY prompting, per-file overwrite; rejected: .env support, remote
fetching). New decisions get a new ADR, not an entry here. This spec
is the forward-looking behavior contract only.

## Features

1. CLI entrypoint `jig`, built with cobra.
2. `jig.json` — the template manifest (like cookiecutter.json):
   variables, defaults (possibly derived from other variables), prompt
   text, choices, and validation patterns, used for interactive
   prompting. Validated against a closed JSON schema
   (schema file ships in the repo; structs hand-rolled — codegen is
   problematic, same discipline as smith's schemas).
3. Both a CLI and a reusable rendering library (decision 1).
4. The tool should be as simple and focused as possible.

## Template layout and versioning

Templates are addressed 3-part with a mandatory version directory:
`<language>/<flavor>/<kind>/v<N>/` — e.g. `python/astral/lib/v1/`.
Inside the version directory, the conventional project-name directory
comes first: `<template>/v1/{{.PROJECT_NAME}}/jig.json` — jig.json must
sit directly inside the top-level project-name directory.

- `jig render --template python/astral/lib` resolves
  `$JIG_TEMPLATES_DIR/python/astral/lib/v<N>/`.
- `--version N` selects the version; omitted = latest (highest N).
- `--template` also accepts a direct filesystem path to any template
  directory (bypasses JIG_TEMPLATES_DIR).
- JIG_TEMPLATES_DIR unset and a bare template name given → look in the
  cwd.

### PROJECT_NAME

`PROJECT_NAME` is a RESERVED variable: every jig.json must declare it
(with a validation pattern), because the rendered directory layout
depends on it. `jig.json` itself is the template's manifest and is
NEVER rendered into the output (like cookiecutter.json).

## CLI UX

```
jig new <language>/<flavor>/<kind>   # scaffolds a new TEMPLATE in registry
                                     # layout: <l>/<f>/<k>/v1/{{.PROJECT_NAME}}/jig.json
                                     # under --templates-dir (default $JIG_TEMPLATES_DIR,
                                     # else the cwd)
jig bump <language>/<flavor>/<kind>  # copies latest v<N> to v<N+1>
jig check --template python/astral/lib [--version 1]   # dry-render, no writes
jig render --template python/astral/lib [--version 1] \
    --output path/to/target [--vars-file jig-vars.json] [--var key=value ...]
    [--force] [--trust]
jig render --template path/to/foo --output path/to/target
```

- Interactive mode: prompt for each jig.json variable (default, prompt
  text, choices, validation pattern). Prompting requires a TTY:
  `--vars-file` ⇒ non-interactive; `--var` with non-TTY stdin ⇒
  non-interactive; neither ⇒ hard error.
- `--var key=value` repeatable. Precedence:
  defaults < vars-file < `--var` flags < interactive answers. An
  override re-derives any default built on its variable.
- Variables in jig.json never used by the template = warning (cruft
  check). Templates referencing undeclared variables = hard error
  (`missingkey=error`).

## Rendering rules

- Walk the template tree; render file/dir NAMES and text file CONTENTS
  with text/template + the func map.
- Binary files: copied verbatim (detected by content sniffing, not
  extension lists).
- Containment: every rendered path must stay inside `--output` —
  empty names, `..`, absolute results are hard errors (ADR 7).
- Symlinks: rejected by default; `--trust` recreates links that
  resolve inside the template tree (ADR 8).
- Overwrite behavior: the unit of refusal is the FILE. Rendering into
  a populated directory is fine as long as no planned file collides;
  `--force` overrides (ADR 14).
- Post-render steps (repo wiring, provenance stamps, next-step hints)
  are the CALLER's job, never jig's and never the template's — jig
  renders files and exits.

## smith integration (the boundary)

smith consumes jig as a library. `smith new` = resolve the template
from smith's registries (custody chain unchanged) → `jig.Render(dir,
vars)` in-process → smith-owned post-render steps (repo.yml
project_roots append, provenance stamp in the rendered smith.yml,
next-steps print). jig never learns about registries; smith never
learns about rendering. cookiecutter drops out of smith's BYOT list
(`smith doctor`, `smith init` seeds) once this lands.

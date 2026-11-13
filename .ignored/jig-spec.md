# Jig project template rendering tool (like cookiecutter)

## Language - golang

## Goal
Create a stand alone cli application + reusable Go library that renders project templates.

## Locked decisions (2026-11-08)

1. **Library for smith, CLI for the world.** smith imports jig as a Go
   library (in-process, pinned via go.mod — no version skew, no PATH
   dependency, smith's single-binary story preserved). The CLI is for
   everyone else and the OSS story. One module,
   `github.com/cliwright/jig`, with the lib at the root package and the
   CLI at `cmd/jig`. The engine is NOT swappable/BYOT inside smith: the
   rendering contract (jig.json schema, func map, no-hooks) is part of
   smith's template contract and trust model — an engine slot would let
   cookiecutter-style hooks back in through the side door.
2. **text/template only.** The built-in Go template engine, plus a small
   HAND-ROLLED func map of string helpers (`lower`, `replace`,
   `slugify`, …). No third-party template libs (no sprig). Keep
   templates as simple as possible.
3. **No hooks / no arbitrary code execution, ever.** Templates are
   inert, vettable data. This is the security model and a marketing
   line: a registry-distributed template must never be "run this
   stranger's code". (This is why cookiecutter is being replaced.)
4. **Local rendering only.** No remote fetching in jig. smith's
   registry/custody chain owns downloading and unpacking; jig renders
   what is on disk. (A template may later ship INSIDE a smith ingot —
   smith unpacks, jig renders.)
5. **No .env support.** Dropped as scope smell. `JIG_TEMPLATES_DIR`
   comes from the process environment only; smith passes resolved
   template dirs directly as a library caller and never uses the env
   var.
6. **Empty dirs: mirror what's there.** Templates contain no empty
   dirs (same as git). Placeholder files (`.gitkeep`, `.keep`,
   `.empty`) are ordinary files: rendered with their parent directory,
   no special handling.

## Locked decisions (2026-11-13)

7. **jig enforces path containment itself.** Rendered names (from user
   input via variable values) must stay inside the output directory:
   empty names, `..` segments, and absolute results are hard errors,
   regardless of what a manifest's validation patterns allow. Interior
   slashes are fine — a name may render to a nested path.
8. **No symlinks by default; `--trust` opts in.** Templates must not
   contain symlinks; a template holding one fails to render. `--trust`
   recreates them as symlinks in the output, but links resolving
   outside the template tree are still refused — trust is about
   intent, not containment.
9. **Derived defaults.** A variable's `default` may template over
   other variables (`{{ .PROJECT_NAME | slugify }}`). Defaults resolve
   in dependency order at load; undeclared references, self-references,
   and cycles are load errors. Explicit overrides win per-variable and
   re-derive their dependents. A variable referenced by another
   variable's default counts as used (no cruft warning).
10. **`choices` for enum variables.** Mutually exclusive with
    `pattern`; a static default must be one of the choices. Interactive
    prompts list the options.
11. **`jig check` dry-renders.** Validates the manifest, parses and
    renders every name and text file against the resolved defaults,
    reports unused variables; writes nothing; exits non-zero on error.
    The author-facing answer to "deployed a broken template".
12. **`jig bump` copies latest v<N> → v<N+1>.** Author-side version
    iteration; refuses symlinks like an untrusted render.
13. **Prompting requires a TTY.** `--vars-file` is non-interactive; so is
    `--var` when stdin is not a terminal (the rest take defaults).
    Non-interactive stdin with no values at all → hard error naming
    `--vars-file`/`--var`, instead of hanging on a prompt nobody can
    answer.
14. **Overwrite semantics (corrects "Rendering rules" below).** The
    unit of refusal is the FILE, not the output directory: rendering
    into a populated directory is allowed as long as no planned file
    collides; `--force` overrides per file.

## Features

1. CLI entrypoint `jig`, built with cobra.
2. `jig.json` — the template manifest (like cookiecutter.json):
   variables, defaults, prompt text, and validation patterns, used for
   interactive prompting. Validated against a closed JSON schema
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
jig render --template python/astral/lib [--version 1] \
    --output path/to/target [--vars-file jig-vars.json] [--var key=value ...]
jig render --template path/to/foo --output path/to/target
```

- Interactive mode: prompt for each jig.json variable (default, prompt
  text, validation pattern).
- `--vars-file` (JSON object of variable values) ⇒ NON-interactive.
  Missing required variables are a hard error.
- `--var key=value` repeatable. Precedence:
  defaults < vars-file < `--var` flags < interactive answers.
- Variables in jig.json never used by the template = warning (cruft
  check). Templates referencing undeclared variables = hard error
  (`missingkey=error`).

## Rendering rules

- Walk the template tree; render file/dir NAMES and text file CONTENTS
  with text/template + the func map.
- Binary files: copied verbatim (detected by content sniffing, not
  extension lists).
- Overwrite behavior (superseded by decision 14): refuse to render
  into a non-empty output directory; `--force` overrides.
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

# Implementation

jig templates are plain [Go `text/template`](https://pkg.go.dev/text/template)
— there is no custom template language to learn. Everything the standard
library syntax offers works in file contents **and** in file and directory
names:

```
{{.PROJECT_NAME}}                  a variable
{{.AUTHOR | upper}}                pipelines
{{if eq .WITH_TESTS "yes"}}…{{end}} conditionals (values are strings —
                                   compare explicitly)
{{- /* trimmed whitespace */ -}}   comments and trimming
```

Values are strings, so `range` has nothing to iterate — loops are not
supported today. The built-in functions (`printf`, `len`, `and`, …) are
available too.

Rendering is strict: referencing a variable the manifest does not declare
fails the render. The manifest is the whole contract — see
[the manifest reference](manifest.md).

## File names on Windows

Windows forbids `< > : " / \ | ? *` in file and directory names. That
matters for template **names**, not contents: a skeleton file named

```
src/{{.PROJECT_NAME | slugify}}.txt
```

renders fine on Linux and macOS but cannot even be checked out on Windows
— the `|` is illegal on disk before any rendering happens. In names,
prefer the function-call form, which avoids the pipe and renders to
exactly the same result:

```
src/{{slugify .PROJECT_NAME}}.txt
```

File and directory *contents* are unaffected — pipelines inside a file
work on every platform, since the content is rendered in memory and only
the result touches disk.

## Conditional files and directories

A file or directory whose *name* renders to an empty string is excluded
from the output — the conditional lives in the name, the content stays
plain:

```
{{if eq .WITH_DOCKER `yes`}}Dockerfile{{end}}
{{if eq .KIND `cli`}}cmd{{end}}/main.go
```

(Use backtick raw strings, not `"` — quotes are illegal in Windows file
names, see above.) Excluding a directory prunes its whole subtree.
Excluded entries are still fully validated — names, contents, and symlink
policy are checked as if they were rendered — so `jig check` covers every
conditional branch, not just the ones your defaults select. The top-level
project directory may not render empty; that stays a hard error. See
[ADR 19](adr/0019-conditional-files.md).

## String helpers

On top of the standard library, jig maps a deliberately tiny set of string
functions. No third-party function libraries — templates stay as simple as
possible.

| Function | Signature | Example | Result |
| --- | --- | --- | --- |
| `lower` | `lower s` | `{{lower .TITLE}}` | `My-Project` → `my-project` |
| `upper` | `upper s` | `{{upper .KIND}}` | `tool` → `TOOL` |
| `replace` | `replace old new s` | `{{.PROJECT_NAME \| replace "-" "_"}}` | `my-project` → `my_project` |
| `slugify` | `slugify s` | `{{slugify .TITLE}}` | `My Cool App!` → `my-cool-app` |
| `ldelim` | `ldelim` | `{{ldelim}}` | `{{` |
| `rdelim` | `rdelim` | `{{rdelim}}` | `}}` |
| `year` | `year` | `{{year}}` | → `2026` |
| `date` | `date` | `{{date}}` | → `2026-11-18` |

`year` and `date` are UTC — for copyright headers and generated-file
notices. A template using them is not bit-reproducible across time.

`replace` takes the string last so it composes in pipelines.

## Provenance: `.jig.json`

Every render writes a `.jig.json` record into the rendered project's
root — where the project came from and the values that produced it:

```json
{
  "version": 1,
  "template": "python/std/lib",
  "template_version": 1,
  "tool": "jig 0.4.8",
  "rendered_at": "2026-11-18T15:04:05Z",
  "values": { "PROJECT_NAME": "demo" }
}
```

The record is data for tooling — finding stale renders in a fleet,
re-rendering a project the same way — not an update mechanism (see
[ADR 20](adr/0020-provenance-record.md)). Consequences worth
knowing: **values are recorded verbatim**, so never prompt for secrets
in template variables; `.jig.json` is reserved — a template may not
ship its own; and the record joins the render like any other file, so
an existing one is never overwritten without `--force`.

## Templating files that contain `{{ }}`

CI workflows, Azure pipelines, and friends use `${{ }}` as their own
expression syntax — which collides with Go's delimiters when such a file is
rendered. Three ways out, in order of preference:

1. **Static files: use a `verbatim` directory.** Most workflow files need
   no jig variables at all; ship them unrendered.
2. **Hybrid files: use `ldelim`/`rdelim`.** When a file needs a few jig
   variables among many `${{ }}` expressions:

   ```yaml
   name: {{ .PROJECT_NAME }}
   on:
     push:
       branches: [${{ ldelim }} github.default_branch {{ rdelim }}]
   ```

   renders as:

   ```yaml
   name: my-tool
   on:
     push:
       branches: [${{ github.default_branch }}]
   ```

3. **A single occurrence:** the `text/template` escape,
   `${{"{{"}} … }}`, works too — but it doesn't stay readable beyond one
   or two uses.

`slugify` lowercases and collapses every run of characters outside
`[a-z0-9-]` into a single dash, trimming leading and trailing dashes —
handy for deriving URLs or directory names from free-text input.

A typical use is deriving consistent names from one answer:

```toml
[project]
name = "{{.PROJECT_NAME}}"

[tool.example]
package = "{{.PROJECT_NAME | replace \"-\" \"_\"}}"
module  = "github.com/example/{{slugify .PROJECT_NAME}}"
```

Derivations this common are often better declared once, in the manifest —
see [derived defaults](manifest.md#derived-defaults).

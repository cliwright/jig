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

`replace` takes the string last so it composes in pipelines.

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

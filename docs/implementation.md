# Implementation

jig templates are plain [Go `text/template`](https://pkg.go.dev/text/template)
— there is no custom template language to learn. Everything the standard
library syntax offers works in file contents **and** in file and directory
names:

```
{{.PROJECT_NAME}}                  a variable
{{.AUTHOR | upper}}                pipelines
{{if .WITH_TESTS}}…{{end}}         conditionals
{{range .ITEMS}}…{{end}}           loops
{{- /* trimmed whitespace */ -}}   comments and trimming
```

The built-in functions (`printf`, `len`, `and`, …) are available too.

Rendering is strict: referencing a variable the manifest does not declare
fails the render. The manifest is the whole contract — see
[the manifest reference](manifest.md).

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

`replace` takes the string last so it composes in pipelines.

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

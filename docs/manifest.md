# The `jig.json` manifest

Every template carries a manifest at the skeleton root:

```
<template>/v<N>/{{.PROJECT_NAME}}/jig.json
```

It declares what the template asks for and how it renders. The manifest
itself never renders into the output as it is the template's own metadata.

## Example

```json
{
  "version": 1,
  "variables": {
    "PROJECT_NAME": {
      "default": "my-project",
      "prompt": "Project name",
      "pattern": "^[a-z][a-z0-9-]*$"
    },
    "AUTHOR": {
      "default": "Ada Lovelace",
      "prompt": "The package author, first and last name"
    }
  },
  "verbatim": ["template"]
}
```

## Fields

| Field | Required | Description |
| --- | --- | --- |
| `version` | yes | Manifest schema version; currently `1` |
| `variables` | yes | Map of variable name to its declaration (below) |
| `verbatim` | no | Directory names whose subtrees copy as-is, unrendered |

### `variables.<NAME>`

| Field | Required | Description |
| --- | --- | --- |
| `default` | yes | Value used when the variable is not given otherwise; may template over other variables |
| `prompt` | no | Question shown in interactive mode |
| `pattern` | no | RE2 regular expression the final value must match (exclusive with `choices`) |
| `choices` | no | Fixed set of allowed values (exclusive with `pattern`) |

`PROJECT_NAME` is reserved and **must** be declared: the skeleton's
top-level directory is named `{{.PROJECT_NAME}}`, and the rendered output
directory takes its rendered value.

Patterns are validated at load time (they must compile) and every value is
checked against its pattern or choices at render time — including defaults,
so keep the two consistent. For `choices`, a static default must be one of
the listed values.

## Value types

There is exactly one type: **string**. Every variable's value — default,
`--vars-file` entry, `--var` flag, interactive answer — is a string, and
the manifest schema enforces it (`default` must be a string; a vars-file
must be a JSON object of string values). `pattern` and `choices` validate
strings.

This matters for conditionals. In Go templates a non-empty string is
truthy, so `{{if .WITH_TESTS}}` is true even when `WITH_TESTS` is the
string `"false"`. The idiomatic patterns:

- **Boolean-ish switches:** compare explicitly —
  `{{if eq .WITH_TESTS "yes"}}…{{end}}` — and declare the variable with
  `choices: ["yes", "no"]` so the prompt offers both.
- **Presence switches:** treat empty as off —
  `{{if .EXTRA}}…{{end}}` with `default: ""` — though note `default` is
  required and empty string is a valid one.
- **Numbers:** arrive as strings; compare with `eq` against string
  literals. There is no arithmetic in the func map by design.

Richer types (real booleans, numbers, lists) are a deliberate omission,
not an oversight; adding them would be a new ADR, not a quiet feature.

## Derived defaults

A default may reference other variables with the full template syntax:

```json
{
  "variables": {
    "PROJECT_NAME": { "default": "my-project", "pattern": "^[a-z][a-z0-9-]*$" },
    "PKG": { "default": "{{ .PROJECT_NAME | replace \"-\" \"_\" }}" }
  }
}
```

Defaults resolve in dependency order: references to undeclared variables,
self-references, and cycles fail at load time. An explicit value
(`--vars-file`, `--var`, or an interactive answer) wins for its variable,
and everything derived from it re-evaluates against the given value —
answering `PROJECT_NAME` differently also fixes `PKG`.

## Rendering rules

- **Names and contents** of text files render as Go `text/template` with
  the resolved values: `README.md`, `src/{{.PKG}}/main.go`, and so on.
- **Binary files** (images, archives, …) copy through unchanged.
- **Executable bits** are preserved.
- Using an **undeclared variable** anywhere in a name or body fails the
  render — the manifest is the whole contract.
- Declaring a variable the template never uses is not an error, but jig
  prints a warning so manifests don't collect cruft. (A variable referenced
  by another variable's default counts as used.)
- Rendered names must stay **inside the output directory** — values that
  render to `../…` or empty names fail the render no matter what the
  manifest allows.
- **Symlinks are rejected** by default; `jig render --trust` recreates them
  in the output, but only relative links that resolve inside the template
  tree. Symlinks already sitting in the destination are never followed,
  `--force` notwithstanding.
- The whole render is planned before anything is written: a collision or a
  bad template fails without leaving a half-rendered tree.
- `jig check` dry-renders all of this without writing — run it before
  shipping a template.

## Verbatim directories

`verbatim` lists directory **base names** that copy without rendering —
file names and file contents ship exactly as authored, wherever those
directories sit in the skeleton. This exists for templates that ship
templates: a scaffolding template can embed a nested template tree without
the outer render consuming the inner one's `{{…}}` placeholders.

```json
{ "verbatim": ["template"] }
```

Entries must be relative slash paths inside the skeleton (`template`,
`ci/shared`), never absolute or escaping.

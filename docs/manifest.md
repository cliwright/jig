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
| `default` | yes | Value used when the variable is not given otherwise |
| `prompt` | no | Question shown in interactive mode |
| `pattern` | no | RE2 regular expression the final value must match |

`PROJECT_NAME` is reserved and **must** be declared: the skeleton's
top-level directory is named `{{.PROJECT_NAME}}`, and the rendered output
directory takes its rendered value.

Patterns are validated at load time (they must compile) and every value is
checked against its pattern at render time — including defaults, so keep
the two consistent.

## Rendering rules

- **Names and contents** of text files render as Go `text/template` with
  the resolved values: `README.md`, `src/{{.PKG}}/main.go`, and so on.
- **Binary files** (images, archives, …) copy through unchanged.
- **Executable bits** are preserved.
- Using an **undeclared variable** anywhere in a name or body fails the
  render — the manifest is the whole contract.
- Declaring a variable the template never uses is not an error, but jig
  prints a warning so manifests don't collect cruft.
- The whole render is planned before anything is written: a collision or a
  bad template fails without leaving a half-rendered tree.

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

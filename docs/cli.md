# CLI reference

jig has two commands: `render` (use a template) and `new` (author one).
`jig --version` prints the version.

## jig render

```sh
jig render --template <name-or-path> --output <dir> [flags]
```

Renders a template into a new project directory under `--output`, prompting
interactively for each variable the template's manifest declares.

| Flag | Default | Description |
| --- | --- | --- |
| `--template` | *(required)* | Template name (resolved under `--templates-dir`) or a direct path to a template root or version directory |
| `--output` | *(required)* | Parent directory the rendered project lands in |
| `--version` | latest | Template version `v<N>`; the default picks the highest `N` |
| `--templates-dir` | `$JIG_TEMPLATES_DIR`, else `.` | Template registry root |
| `--vars-file` | — | JSON object of variable values; renders non-interactively |
| `--var key=value` | — | Set one variable; repeatable |
| `--force` | refuse | Overwrite existing files on collision |

Variable precedence: manifest defaults < `--vars-file` < `--var` <
interactive answers.

Rendering never overwrites an existing file unless `--force` is given.
Rendering into a populated directory is fine as long as no file collides —
the whole render is planned first, so a collision fails before anything is
written.

## jig new

```sh
jig new <language>/<flavor>/<kind> [--templates-dir <dir>]
```

Scaffolds a new template to author:
`<language>/<flavor>/<kind>/v1/{{.PROJECT_NAME}}/jig.json` plus a README to
start from. Segments are lowercase alphanumerics and hyphens, starting with
a letter. Fails if the template already exists — bump to `v2` by hand when
you need a breaking change.

## JIG_TEMPLATES_DIR

Both commands read the templates root from the environment, so set it once
and drop the flag:

```sh
export JIG_TEMPLATES_DIR="$HOME/templates"
jig render --template python/std/lib --output ./libs
```

Precedence: `--templates-dir` flag > `JIG_TEMPLATES_DIR` > the current
directory.

## How templates are resolved

`--template` accepts three forms:

1. **A name** like `python/std/lib` — looked up under the templates root,
   expecting version directories `v1`, `v2`, … underneath. The
   `<language>/<flavor>/<kind>` shape is the recommended convention, not a
   requirement: any relative path works, so `go-cli` or `internal/ci/bot`
   are equally valid names.
2. **A path to a template root** — a directory holding `v<N>` directories.
3. **A path to a version directory** — a directory holding the
   `{{.PROJECT_NAME}}/` skeleton directly; used as-is.

The skeleton directory is always named `{{.PROJECT_NAME}}` and always holds
the template's `jig.json`. See [the manifest reference](manifest.md).

# CLI reference

jig has six commands: `render` (use a template), `new` and `bump` (author
one), `check` (verify one), `agents` and `docs` (agent-facing help).
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
| `--trust` | reject | Allow symlinks in the template; recreated as links in the output, but only if they resolve inside the template tree |

Variable precedence: manifest defaults < `--vars-file` < `--var` <
interactive answers. An override re-derives any default built on it, so
`--var PROJECT_NAME=x` also updates a `MODULE_NAME` defaulting to
`{{ .PROJECT_NAME \| slugify }}`.

Prompting requires an interactive terminal. `--vars-file` renders
non-interactively; so does `--var` when stdin is not a terminal (remaining
variables take their resolved defaults). With piped or absent stdin and no
values at all, `render` fails fast instead of hanging on a prompt nobody
can answer.

Rendering never overwrites an existing file unless `--force` is given.
Rendering into a populated directory is fine as long as no file collides —
the whole render is planned first, so a collision fails before anything is
written. Rendered names must stay inside the output directory: a value that
renders to `../anything` or an empty name fails the render, regardless of
what the manifest's patterns allow.

## jig new

```sh
jig new <language>/<flavor>/<kind> [--templates-dir <dir>]
```

Scaffolds a new template to author:
`<language>/<flavor>/<kind>/v1/{{.PROJECT_NAME}}/jig.json` plus a README to
start from. Segments are lowercase alphanumerics and hyphens, starting with
a letter. Fails if the template already exists — use `jig bump` when you
need a new version.

## jig bump

```sh
jig bump <language>/<flavor>/<kind> [--templates-dir <dir>]
```

Copies the template's latest version directory to `v<N+1>` as the starting
point for its next iteration. Old versions stay put; renders pin theirs
with `--version`. Refuses symlinks, matching the default render policy.

## jig check

```sh
jig check --template <name-or-path> [--version N] [--templates-dir <dir>]
```

Dry-renders a template without writing anything: the manifest is validated,
every file and directory name and every text file's content is parsed and
rendered against the resolved defaults, and declared-but-unused variables
print as warnings. Exits non-zero on any error — run it before shipping a
template, and in CI.

## jig agents

```sh
jig agents
```

Prints the agent guide: a stable onboarding text for AI agents driving jig
— what it is, the trust model, how to run it non-interactively, and
pointers to `--help` and the documentation. The text is embedded from
[`agents.txt`](https://github.com/cliwright/jig/blob/main/agents.txt) and
deliberately documents no flags; discovery happens through `jig --help`.

## jig docs

```sh
jig docs            # list available pages
jig docs <page>     # print one as raw markdown
```

Prints the documentation embedded in the binary — the same operational
pages as jig.cliwright.dev (CLI, manifest, implementation, layouts), but
always exactly matching the version you're running. Raw markdown on
stdout, no pager: built for agents (and for `grep`). The ADRs are web-only
— design rationale for humans, on the docs site.

## JIG_TEMPLATES_DIR

Both commands read the templates root from the environment, so set it once
and drop the flag:

```sh
export JIG_TEMPLATES_DIR="$HOME/templates"
jig render --template python/std/lib --output ./libs
```

Precedence: `--templates-dir` flag > `JIG_TEMPLATES_DIR` > the current
directory.

## Getting templates from git

jig never fetches: clone a template repo yourself and point the templates
root at it. The trust decision stays with you, in your own git setup.

```sh
git clone https://github.com/example/jig-templates ~/jig-templates
export JIG_TEMPLATES_DIR="$HOME/jig-templates"
git -C "$JIG_TEMPLATES_DIR" checkout v1.2.0   # optional: pin a version
jig render --template python/std/lib --output ./libs
```

Update later with `git pull`. Repos that don't follow the
`<language>/<flavor>/<kind>` layout still work — pass a path straight to
`--template`.

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

<p align="center">
  <a href="https://www.fontspace.com/typo-draft-font-f41179">
    <img src="https://see.fontimg.com/api/rf5/BWWo5/YTZkZTMxNDlhNDEwNDZhZmFiZThhODFhNjA5N2U3NTgub3Rm/amln/typo-draft-demo.png?r=fs&h=140&w=2000&fg=5A3922&bg=FFFFFF&tb=1&s=70" alt="jig">
  </a>
</p>

## A Project Scaffolder

`jig` is a project scaffolding tool. A template is a directory of project
files plus a `jig.json` manifest declaring its variables. One Go binary, no
runtime, no plugins, no template-supplied code execution, simple
scaffolds, safe and repeatable.

## Why "jig"?

> A jig is a device or template that guides a tool relative to a workpiece
> to ensure accuracy, safety and repeatability.

This is the aim of `jig`: guide the render accurately, safely and repeatably.

## No hooks, by design

Most scaffolding tools run arbitrary code at scaffold time. Hooks, scripts and "setup"
baked into the template expose you to potential security risks.

`jig` does the opposite:

**no template-supplied program ever runs.** Rendering evaluates only the
standard Go template language and jig's own registered, reviewed string
functions — a template cannot execute a script, a hook, or a shell command.
A template is an ordinary directory tree with explicit variables and
predictable rendering rules.

This buys you:

- **Auditability.** You can read a template top to bottom and know exactly
  what lands on disk. Rendering a template you didn't write does not
  execute programs supplied by that template.

- **Diffability.** A template change is a plain text diff. Reviewing one is
  reviewing the scaffold itself.

- **Predictability.** The same template with the same values renders the
  same tree, every time, on every machine.

The trade-off is deliberate: anything a hook could do, jig refuses to do. If
a project needs a setup step, that step belongs in the project's own
tooling — documented in its README, run by a human who can see it. (The
guarantee covers rendering; what a rendered `Makefile` or setup script does
when *you* run it afterwards is yours to read first — see
[ADR 3](adr/0003-no-hooks.md).)

## Install

```sh
brew install cliwright/cliwright/jig
```

or with Go:

```sh
go install github.com/cliwright/jig/cmd/jig@latest
```

## Quickstart

Scaffold a template, fill it in, render it:

```sh
jig new python/std/lib --templates-dir ./templates
```

This creates `templates/python/std/lib/v1/{{.PROJECT_NAME}}/` with a starter
`jig.json` and README. Edit the skeleton. Any file or directory name may
contain `{{.PROJECT_NAME}}` or another declared variable (mind
[Windows-illegal characters in names](implementation.md#file-names-on-windows)).
Once you are finished authoring 
the template, validate and render:

```sh
jig check --template python/std/lib --templates-dir ./templates
jig render --template python/std/lib --templates-dir ./templates --output ./libs
```

jig prompts for each variable the manifest declares. For scripts and CI:

```sh
jig render --template python/std/lib --output ./libs \
  --vars-file values.json --var PROJECT_NAME=widget
```

When the template later changes in a way existing users shouldn't silently
inherit, `jig bump python/std/lib` copies `v1` to `v2` as the starting
point for the new version — old versions stay renderable with
`--version 1`.

## Ready-made templates

[cliwright/jig-templates](https://github.com/cliwright/jig-templates) is the
public template registry — clone it, point jig at it, render:

```sh
git clone https://github.com/cliwright/jig-templates ~/jig-templates
export JIG_TEMPLATES_DIR="$HOME/jig-templates"
jig render --template python/astral-pytest-zensical/tool --output .
```

## Agent support

jig is built to be driven by AI agents, not just humans:

- `jig agents` prints a stable onboarding text: what jig is, the trust
  model, how to render non-interactively, and where to look next. The root
  `--help` points agents at it.
- `jig docs` serves the documentation **embedded in the binary** — always
  exactly matching the version being run, available offline, as raw
  markdown on stdout. `jig docs` lists pages, `jig docs <page>` prints one.

Both are library functions too (`jig.AgentGuide`, `jig.Doc`), so tools
embedding jig — like smith — can surface the same text.

## Read next

- [CLI reference](cli.md) — commands, flags, and `JIG_TEMPLATES_DIR`.
- [The `jig.json` manifest](manifest.md) — variables, patterns, verbatim
  directories.
- [Implementation](implementation.md) — Go `text/template` syntax and the
  built-in string helpers.
- [Example layouts](example_layouts.md) — what templates look like in
  practice.
- [ADRs](adr/index.md) — why jig is the way it is, including rejected
  ideas.

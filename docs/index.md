<p align="center">
  <a href="https://www.fontspace.com/typo-draft-font-f41179">
    <img src="https://see.fontimg.com/api/rf5/BWWo5/YTZkZTMxNDlhNDEwNDZhZmFiZThhODFhNjA5N2U3NTgub3Rm/amln/typo-draft-demo.png?r=fs&h=140&w=2000&fg=5A3922&bg=FFFFFF&tb=1&s=70" alt="jig">
  </a>
</p>

## A Project Scaffolder

`jig` is a project scaffolding tool. A template is a directory of project
files plus a `jig.json` manifest declaring its variables. One Go binary, no
runtime, no plugins, no code execution at scaffold time — simple scaffolds,
safe and repeatable.

## Why "jig"?

> A jig is a device or template that guides a tool relative to a workpiece
> to ensure accuracy, safety and repeatability.

This is the aim of `jig`: guide the render accurately, safely and repeatably.

## No hooks, by design

Most scaffolding tools run arbitrary code at scaffold time. Hooks, scripts and "setup"
baked into the template expose you to potential security risks.

`jig` does the opposite:

**rendering never executes anything.** A template is WYSIWYG: file names and
file contents, with variables only where you need them.

This buys you:

- **Auditability.** You can read a template top to bottom and know exactly
  what lands on disk. Fetching and rendering a template you didn't write is
  safe, because there is nothing in it that *can* run.

- **Diffability.** A template change is a plain text diff. Reviewing one is
  reviewing the scaffold itself.

- **Predictability.** The same template with the same values renders the
  same tree, every time, on every machine.

The trade-off is deliberate: anything a hook could do, jig refuses to do. If
a project needs a setup step, that step belongs in the project's own
tooling — documented in its README, run by a human who can see it.

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
`jig.json` and README. Edit the skeleton — any file or directory name may
contain `{{.PROJECT_NAME}}` or another declared variable — then render:

```sh
jig render --template python/std/lib --templates-dir ./templates --output ./libs
```

jig prompts for each variable the manifest declares. For scripts and CI:

```sh
jig render --template python/std/lib --output ./libs \
  --vars-file values.json --var PROJECT_NAME=widget
```

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

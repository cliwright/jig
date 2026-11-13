<p align="center">
  <a href="https://www.fontspace.com/typo-draft-font-f41179">
    <img src="https://see.fontimg.com/api/rf5/BWWo5/YTZkZTMxNDlhNDEwNDZhZmFiZThhODFhNjA5N2U3NTgub3Rm/amln/typo-draft-demo.png?r=fs&h=140&w=2000&fg=5A3922&bg=FFFFFF&tb=1&s=70" alt="jig">
  </a>
</p>

Render project templates — no hooks, no code execution. Templates are data:
inert directories of Go `text/template` files plus a `jig.json` manifest
declaring the variables to prompt for. A single Go binary, nothing else to
install.

> **Source of truth:** jig is developed in a private monorepo and this
> repository is a published mirror, kept in sync automatically. Please open
> **issues** here freely. **Pull requests** are welcome too — a maintainer
> will import them into the monorepo (preserving your authorship) and close
> the PR with a link to the imported commit.

## Install

```sh
brew install cliwright/cliwright/jig
```

or with Go:

```sh
go install github.com/cliwright/jig/cmd/jig@latest
```

## Quickstart

```sh
# scaffold a new template to author your own
jig new python/std/lib --templates-dir ./templates

# render it into a new project (prompts for its variables)
jig render --template python/std/lib --templates-dir ./templates --output ./libs
```

Set `JIG_TEMPLATES_DIR` once and drop `--templates-dir` from every command.

## Documentation

Full docs live in [`docs/`](docs/index.md): the
[CLI reference](docs/cli.md), the [`jig.json` manifest](docs/manifest.md),
and [example layouts](docs/example_layouts.md).

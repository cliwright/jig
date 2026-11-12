<p align="center">
  <a href="https://www.fontspace.com/typo-draft-font-f41179">
    <img src="https://see.fontimg.com/api/rf5/BWWo5/YTZkZTMxNDlhNDEwNDZhZmFiZThhODFhNjA5N2U3NTgub3Rm/amln/typo-draft-demo.png?r=fs&h=140&w=2000&fg=5A3922&bg=FFFFFF&tb=1&s=70" alt="jig">
  </a>
</p>

Render project templates — no hooks, no code execution. Templates are data:
inert directories of Go `text/template` files plus a `jig.json` manifest
declaring the variables to prompt for.

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

## Usage

Render a template (prompting for its variables):

```sh
jig render python/std/lib --templates-dir ./templates --output ./libs
```

Scaffold a new template to author your own:

```sh
jig new python/std/lib --templates-dir ./templates
```

Templates are addressed `<language>/<flavor>/<kind>` and versioned (`v1`,
`v2`, …); the latest renders unless `--version` says otherwise. See
[`docs/`](docs/) for authoring and the manifest reference.

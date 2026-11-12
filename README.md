# jig

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

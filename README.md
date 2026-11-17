<p align="center">
  <a href="https://www.fontspace.com/typo-draft-font-f41179">
    <img src="https://see.fontimg.com/api/rf5/BWWo5/YTZkZTMxNDlhNDEwNDZhZmFiZThhODFhNjA5N2U3NTgub3Rm/amln/typo-draft-demo.png?r=fs&h=140&w=2000&fg=5A3922&bg=FFFFFF&tb=1&s=70" alt="jig">
  </a>
</p>

[![CI](https://github.com/cliwright/jig/actions/workflows/ci.yml/badge.svg)](https://github.com/cliwright/jig/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/cliwright/jig/graph/badge.svg)](https://codecov.io/gh/cliwright/jig)
[![Go Reference](https://pkg.go.dev/badge/github.com/cliwright/jig.svg)](https://pkg.go.dev/github.com/cliwright/jig)

Render project templates — no hooks, no template-supplied code execution,
templates are data.
A template is a directory of project files plus a `jig.json` manifest
declaring its variables. One Go binary, no runtime, no plugins, no code
execution at render time — simple scaffolds, safe and repeatable.

> A jig is a device or template that guides a tool relative to a workpiece
> to ensure accuracy, safety and repeatability. That is the aim of the jig
> project template renderer.

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

### Verifying a release

Release binaries ship with an SPDX SBOM per archive (`*.sbom*`) and a
keyless [cosign](https://docs.sigstore.dev/) signature over the checksums
(`checksums.txt.sigstore.json`). The signer identity is the release
workflow's GitHub OIDC token — no keys to distribute:

```sh
cosign verify-blob --bundle checksums.txt.sigstore.json \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp 'https://github.com/cliwright/' \
  checksums.txt
sha256sum --check --ignore-missing checksums.txt
```

## Quickstart

```sh
# scaffold a new template to author your own
jig new python/std/lib --templates-dir ./templates

# render it into a new project (prompts for its variables)
jig render --template python/std/lib --templates-dir ./templates --output ./libs
```

Set `JIG_TEMPLATES_DIR` once and drop `--templates-dir` from every command.
Want ready-made templates instead of authoring your own? Clone
[cliwright/jig-templates](https://github.com/cliwright/jig-templates) and
point `JIG_TEMPLATES_DIR` at it.

## For agents

jig ships first-class support for AI agents driving it:

```sh
jig agents    # onboarding: what jig is, trust model, non-interactive usage
jig docs      # list the documentation embedded in this binary
jig docs cli  # print one page — always matching the installed version
```

## Documentation

Full documentation lives at **[jig.cliwright.dev](https://jig.cliwright.dev)**:
the [CLI reference](https://jig.cliwright.dev/cli/), the [`jig.json`
manifest](https://jig.cliwright.dev/manifest/), the [template
implementation](https://jig.cliwright.dev/implementation/), and [example
layouts](https://jig.cliwright.dev/example_layouts/).

## License

[Apache 2.0](LICENSE). See the [changelog](CHANGELOG.md) for what changed
between versions, and [CONTRIBUTING.md](CONTRIBUTING.md) for how to
contribute (features start as an RFC, not a PR). Security reports:
[SECURITY.md](SECURITY.md).

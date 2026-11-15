# Contributing to jig

Thanks for your interest. jig is a deliberately small tool, and the
contribution process is designed to keep it that way.

## The short version

- **Bug fixes, docs, and tests:** open a pull request directly. No ceremony
  needed.
- **Features and behavior changes:** open an **RFC issue first**, no code.
  Pull requests implementing an un-discussed feature will be closed.

## RFC-first for features

jig's scope is a design decision, not an accident — see the
[ADRs](https://jig.cliwright.dev/adr/), including the rejected ones. Before
writing an RFC, please read the rejected ADRs in particular: hooks,
remote template fetching, `.env` support, and a larger template function
map have all been considered and turned down, with the reasoning recorded.

An RFC is a GitHub issue titled `RFC: <feature>` describing:

1. **What it should do** — the behavior, from a user's point of view.
2. **Why it's beneficial** — the problem you hit, and why it belongs in jig
   rather than in your own tooling around jig.

No implementation, no code sketches needed. The discussion happens on the
issue.

If an RFC is accepted, the repo owner writes it up as an ADR (the design
decision, its context, and its consequences) and implements it. That keeps
the ADR set as the single, coherent record of why jig works the way it
does.

## The mirror

jig is developed in a private monorepo; the GitHub repository is a
published mirror, kept in sync automatically. Pull requests can't be merged
directly — a maintainer imports them into the monorepo (preserving your
authorship) and closes the PR with a link to the imported commit.

## Ground rules for PRs

- Keep it small. jig's codebase is deliberately lean; a fix that needs a
  framework is probably the wrong fix.
- Match the surrounding style, and run `go test ./...`, `go vet ./...`, and
  `gofmt` before opening.
- Add or update tests for the behavior you're changing.
- Update the docs (and the changelog's Unreleased section) when behavior
  changes.

## License

By contributing, you agree that your contributions are licensed under the
[Apache License 2.0](LICENSE).

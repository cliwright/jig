# Changelog

All notable changes to jig are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and jig versions
with [Semantic Versioning](https://semver.org/): breaking library or CLI
changes bump the minor version until 1.0, then the major.

Releases before 0.4.1 predate this changelog.

## [Unreleased]

### Added

- ADR 18 records the strings-only decision: no list or typed variables,
  with the reasoning and the revisit trigger.

### Fixed

- `render` no longer mistakes `/dev/null` stdin for an interactive
  terminal — it is a character device but not a terminal. `--var`-only
  renders now work in CI runners, cron jobs and agent shells that present
  `/dev/null` as stdin, and value-less runs fail fast with an actionable
  error instead of dying on a prompt EOF.

### Changed

- Interactive answers are validated as they are given: a value failing its
  pattern or choices is rejected and re-prompted immediately, instead of
  failing the whole render after the last prompt.

## [0.4.2] - 2026-11-16

### Added

- The docs and README now link the public
  [cliwright/jig-templates](https://github.com/cliwright/jig-templates)
  registry, with clone-and-render examples.
- `jig agents` now points agents at the public registry.

## [0.4.1] - 2026-11-15

### Added

- jig is now licensed under the Apache License 2.0 (`LICENSE`).

### Changed

- `jig check` validates the resolved defaults against their own patterns
  and choices — a default violating its own validation now fails check.
- `jig check` applies path containment to the rendered top-level project
  name — a default like `../escape` now fails check.
- `jig check` rejects references to undeclared variables statically, across
  every branch of every template, not just the branches the resolved
  defaults execute.
- Interactive prompting runs in dependency order and re-derives live:
  answering a variable updates the defaults of everything derived from it
  before those dependents are prompted. Values supplied via
  `--var`/`--vars-file` are locked and never re-derived over.

### Fixed

- A template containing `{{if}}` without `{{else}}` crashed the renderer
  (typed-nil branch in the parse-tree walk).

### Breaking (library)

- The `Prompter` function type takes a third parameter,
  `locked map[string]bool`, naming the explicitly supplied variables a
  prompt must not re-derive over.

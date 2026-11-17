# Changelog

All notable changes to jig are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and jig versions
with [Semantic Versioning](https://semver.org/): breaking library or CLI
changes bump the minor version until 1.0, then the major.

Releases before 0.4.1 predate this changelog.

## [Unreleased]

### Added

- Releases now ship provenance: an SPDX SBOM per archive (syft) and a
  keyless cosign signature over `checksums.txt`
  (`checksums.txt.sigstore.json`), signed with the release workflow's
  GitHub OIDC identity — no keys to distribute. Verification instructions
  are in the README.
- New exported `LoadSkeletonManifest(versionDir, trust)`: loads a
  template's manifest after vetting it like any other template file. The
  CLI's prompt-time manifest load uses it too.

### Fixed

- `jig check` (and the renderer's unused-variable tracking) now traverses
  `{{define}}`d named templates, not just the root parse tree: an
  undeclared variable inside a named template fails check even when no
  executed branch invokes it. ADR 11's "every branch of every template"
  promise now holds.
- The manifest is vetted before it is read: a symlinked `jig.json` is
  refused under the default policy (`--trust` allows a relative link
  resolving inside the template tree, as for any other file), and a
  non-regular manifest is refused — a FIFO previously blocked the read
  forever.
- A manifest-only template now creates its output directory; Render
  previously reported an `OutputDir` that never existed.
- The Windows `--force` overwrite fallback no longer risks losing the
  original file: the existing destination is moved aside and restored if
  the retry rename fails, instead of being deleted first.
- A destination path with a regular file as an intermediate component is
  now refused explicitly and identically on every platform
  (`destination ... contains non-directory ...`). On Windows the write
  previously sailed past the destination-chain check — a path beneath a
  file stats as "not found" there — and failed later with a misleading
  mkdir error (`The system cannot find the path specified`).
- The test suite covers the previously untested error paths: manifest and
  value resolution failures, destination-chain and write-phase failures,
  unused-variable warnings, `--templates-dir` defaulting, every
  `collectUsed` branch kind, and `main`'s error exit. Coverage is now
  ~92% in the library and ~98% in the CLI.

## [0.4.5] - 2026-11-16

### Added

- ADR 18 records the strings-only decision: no list or typed variables,
  with the reasoning and the revisit trigger.

### Fixed

- `render` no longer mistakes `/dev/null` stdin for an interactive
  terminal — it is a character device but not a terminal. `--var`-only
  renders now work in CI runners, cron jobs and agent shells that present
  `/dev/null` as stdin, and value-less runs fail fast with an actionable
  error instead of dying on a prompt EOF.
- The test suite now runs green on Windows: one render test used a `|` in
  a templated file *name* — illegal on Windows filesystems, so the fixture
  could not even be created — and a force-render test asserted unix
  permission bits Windows does not have. No behavior change; the docs
  gained a "File names on Windows" section recommending the function-call
  form (`{{slugify .PROJECT_NAME}}`) over the piped form in skeleton file
  and directory names.

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

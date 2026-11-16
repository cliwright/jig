# Architecture Decision Records

Why jig is the way it is. ADRs record decisions — including rejected ones —
so the reasoning survives and settled questions stay settled. Numbers are
never reused; a reversed decision gets a new ADR that supersedes the old
one, never an edit.

## Accepted

- [1. One module: library for smith, CLI for the world](0001-library-and-cli.md)
- [2. `text/template` only, hand-rolled func map](0002-text-template-only.md)
- [3. No hooks, no arbitrary code execution](0003-no-hooks.md)
- [4. Local rendering only](0004-local-rendering-only.md)
- [6. Empty directories are mirrored, not special-cased](0006-empty-dirs-mirrored.md)
- [7. Rendered paths must stay inside the output directory](0007-path-containment.md)
- [8. Symlinks rejected by default; `--trust` opts in](0008-symlink-policy.md)
- [9. Derived defaults](0009-derived-defaults.md)
- [10. `choices` for enum variables](0010-choices.md)
- [11. `jig check` dry-render command](0011-jig-check.md)
- [12. `jig bump` version-copy command](0012-jig-bump.md)
- [13. Prompting requires a TTY](0013-prompting-requires-tty.md)
- [14. Overwrite refusal is per file, not per directory](0014-per-file-overwrite.md)
- [16. Prepare everything before writing anything](0016-prepare-then-write.md)
- [17. Escaping foreign template syntax with `ldelim`/`rdelim`](0017-escaping-foreign-syntax.md)

## Rejected

- [5. No `.env` support](0005-no-dotenv.md)
- [15. No remote fetching, not even via git wrappers](0015-no-remote-fetching.md)
- [18. String values only (no lists, no typed variables)](0018-strings-only.md)

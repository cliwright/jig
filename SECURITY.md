# Security Policy

## Reporting a vulnerability

Please report vulnerabilities **privately** to **jesse@cliwright.dev** —
not as a public GitHub issue. Include enough detail to reproduce the
problem (template layout, command, jig version, operating system).

You can expect an acknowledgement within a few days and a fix or a
documented decision before any public disclosure.

## Supported versions

Only the latest release receives security fixes. jig is pre-1.0; pin a
version and update deliberately.

## Scope

jig's security guarantee covers the **rendering** phase: path containment,
symlink and special-file handling, and the rule that no template-supplied
program executes during rendering. Reports about escaping those boundaries
are very much in scope.

Out of scope, by design: what generated files do when *you* execute them
afterwards, and the trust decision of which template repos you clone.

# 5. No `.env` support

Date: 2026-11-08
Status: rejected

## Context

An early draft had jig reading `.env` files for configuration
(`JIG_TEMPLATES_DIR`). It looked convenient for keeping a per-project
templates root.

## Decision

Rejected as scope smell. `JIG_TEMPLATES_DIR` comes from the process
environment only; smith passes resolved template dirs directly as a library
caller and never uses the variable.

## Consequences

- One configuration mechanism (flags > env var > cwd), no hidden
  file-based layer to debug.
- Users who want per-project env can use direnv or their shell — tools
  that already exist and do it better.

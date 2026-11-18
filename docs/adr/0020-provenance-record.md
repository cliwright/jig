# 20. Render provenance: the `.jig.json` record

Date: 2026-11-18
Status: accepted

## Context

Every review of jig against Copier and cruft lands on the same gap:
rendered projects are *orphaned* from their template. Nothing in the
output says where it came from or which values produced it, so no
tooling — jig's or anyone else's — can ever re-render, diff, or update
it. A platform team wanting to propagate a CI fix across fifty
scaffolded repos has nothing to key on.

A full update story (re-render plus merge) is a second product: Copier
does git three-way merges, and merge semantics are exactly the kind of
scope [ADR 3](0003-no-hooks.md) keeps out of jig. But the *record* an
update tool would need is cheap, inert data — and without it, no update
story can ever exist, jig's or external.

## Decision

When the caller supplies provenance, Render writes `.jig.json` into the
rendered project's root:

```json
{
  "version": 1,
  "template": "python/std/lib",
  "template_version": 1,
  "tool": "jig 0.4.8",
  "rendered_at": "2026-11-18T15:04:05Z",
  "values": { "PROJECT_NAME": "demo", "...": "..." }
}
```

- **The library is opt-in** (`Options.Provenance`): the renderer cannot
  know how its caller addressed the template, so the caller supplies the
  record. smith, embedding jig as a library, records its own provenance
  and passes nothing.
- **The CLI always writes it** — like cruft's `.cruft.json`, the record
  is most valuable when it is always there, not when the author
  remembered a flag.
- **Values are recorded verbatim** — enough to render the project again
  identically (modulo `date`/`year` stamps). Consequence: do not prompt
  for secrets in template variables; the docs say so.
- **`.jig.json` is reserved** when provenance is written: a template
  shipping its own fails the render rather than being silently replaced.

This is a record, not an update path. Re-rendering and merging stay
outside jig; the record makes them possible — for smith, for scripts,
for a future jig feature — without jig committing to merge semantics
now.

## Consequences

- Rendered projects are describable: template, version, values, tool,
  timestamp — everything a fleet tool needs to find stale renders.
- A hidden file appears in every CLI-rendered project; users who render
  into an existing repo notice one extra dotfile.
- A future update command, if one is ever justified, has its input
  contract already deployed in the wild — the hardest part of adding it
  later would have been retrofitting the record.
- `bump` is unaffected: it copies template versions, unrelated to
  rendered projects.

# 17. Escaping foreign template syntax with `ldelim`/`rdelim`

Date: 2026-11-15
Status: accepted

## Context

Real templates ship files whose own syntax collides with Go's delimiters:
GitHub Actions and Azure pipelines use `${{ }}` heavily, and such a file
fails to parse as a `text/template`. The built-in escape
(`${{"{{"}} … }}`) works but is unreadable beyond an occurrence or two —
and in workflow files, the foreign syntax is the *majority* and jig's
variables the minority, so escaping the majority is backwards ergonomics.

Options considered: per-template alternative delimiters (`.Delims()`) —
rejected for now, because every delimiter choice merely shifts the
collision to a different ecosystem (`[[ ]]` hits bash and C++ attributes,
`(( ))` hits shell arithmetic) and adds a contract dimension to the
manifest, docs, and `jig check`. Preprocessing `${{ }}` as opaque before
parsing — rejected as hidden magic that breaks the "it's just
`text/template`" contract (ADR 2).

## Decision

Two funcs, `ldelim` and `rdelim`, emit literal `{{` and `}}`:

```
${{ ldelim }} github.actor {{ rdelim }}   →   ${{ github.actor }}
```

Combined with the existing guidance: static files go in a `verbatim`
directory, hybrid files use `ldelim`/`rdelim`, a lone occurrence may use
the string-literal escape.

## Consequences

- Workflow and pipeline files are first-class template citizens with no
  new contract surface — the func map is the mechanism ADR 2 already
  blesses.
- The pattern is documented with examples so authors don't rediscover the
  string-literal trick the hard way.
- Delimiter switching is *deferred, not rejected*: if hybrid files prove
  numerous and painful once docker/tofu templates exist, a new ADR may
  revisit it with real usage data.

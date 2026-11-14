# Example layouts

A template is just a directory tree. These are sketches, not rules — the
only hard requirements are the `v<N>` version directories (when you use
versioned names at all) and a `{{.PROJECT_NAME}}/` skeleton holding a
`jig.json`.

## A Python CLI

```
templates/python/std/tool/
└── v1/
    └── {{.PROJECT_NAME}}/
        ├── jig.json
        ├── README.md
        ├── pyproject.toml
        ├── src/
        │   └── {{.PYTHON_PACKAGE}}/
        │       ├── __init__.py
        │       ├── __main__.py
        │       └── cli.py
        └── tests/
            ├── conftest.py
            └── test_{{.PYTHON_PACKAGE}}.py
```

Two variables at work: `PROJECT_NAME` names the rendered project directory,
`PYTHON_PACKAGE` shapes the importable package path. Note the constraint
difference — project names carry hyphens, package names can't, so they are
separate variables with separate patterns in the manifest.

## A Go tool

```
templates/go/std/tool/
├── v1/
│   └── {{.PROJECT_NAME}}/
│       ├── jig.json
│       ├── README.md
│       ├── go.mod
│       └── cmd/
│           └── {{.PROJECT_NAME}}/
│               └── main.go
└── v2/
    └── {{.PROJECT_NAME}}/
        ├── jig.json
        ├── README.md
        ├── go.mod
        ├── cmd/
        │   └── {{.PROJECT_NAME}}/
        │       └── main.go
        └── Taskfile.yml
```

Two versions side by side: `jig render` picks `v2` (the latest) by default,
`--version 1` renders the older shape. `jig bump go/std/tool` copies the
latest version to `v3` when the layout changes in a way existing users
shouldn't silently inherit — old versions stay renderable.

## No convention at all

The `<language>/<flavor>/<kind>` layout is a recommendation for
collections, not a rule. A one-off template can live anywhere and be
rendered by path:

```
~/templates/go-cli/v1/{{.PROJECT_NAME}}/…
```

```sh
jig render --template ~/templates/go-cli --output ./tools
```

## A template that ships a template

A template whose output *is itself a template* wraps the inner tree in a
verbatim directory so the outer render leaves its placeholders alone:

```
templates/meta/std/template/
└── v1/
    └── {{.PROJECT_NAME}}/
        ├── jig.json          # declares: "verbatim": ["template"]
        ├── README.md
        └── template/         # copied as-is, never rendered
            └── {{.PROJECT_NAME}}/
                └── jig.json
```

The inner `{{.PROJECT_NAME}}` survives the outer render untouched and comes
to life only when someone renders the scaffolded template later.

package jig

import (
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// funcMap is the hand-rolled set of string helpers available to templates.
// Deliberately tiny — no third-party template function libraries; templates
// stay as simple as possible. Unexported: the execution environment is part
// of jig's contract, so library consumers must not be able to extend it
// (see docs/adr/0002-text-template-only.md).
var funcMap = template.FuncMap{
	"lower":   strings.ToLower,
	"upper":   strings.ToUpper,
	"replace": replace,
	"slugify": slugify,
	"ldelim":  ldelim,
	"rdelim":  rdelim,
	"year":    year,
	"date":    date,
}

// replace implements {{ .Var | replace "old" "new" }} — all occurrences.
// The string comes last so the pipeline form works.
func replace(old, new, s string) string {
	return strings.ReplaceAll(s, old, new)
}

// ldelim and rdelim emit literal template delimiters, for files whose own
// syntax collides with Go's: CI workflows and the like use ${{ }} heavily,
// and {{ ldelim }} / {{ rdelim }} is how a template emits it — e.g.
// ${{ ldelim }} github.actor {{ rdelim }} renders as ${{ github.actor }}.
func ldelim() string { return "{{" }

func rdelim() string { return "}}" }

// year and date stamp renders — copyright headers, generated-file
// notices. Both UTC, so a render at least doesn't depend on the author's
// timezone; a template using them is of course not bit-reproducible
// across time.
func year() string { return strconv.Itoa(time.Now().UTC().Year()) }

func date() string { return time.Now().UTC().Format("2006-01-02") }

var slugSep = regexp.MustCompile(`[^a-z0-9-]+`)

// slugify lowercases s and collapses every run of characters outside
// [a-z0-9-] into a single dash, trimming leading and trailing dashes.
func slugify(s string) string {
	return strings.Trim(slugSep.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

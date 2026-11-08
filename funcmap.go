package jig

import (
	"regexp"
	"strings"
	"text/template"
)

// FuncMap is the hand-rolled set of string helpers available to templates.
// Deliberately tiny — no third-party template function libraries; templates
// stay as simple as possible.
var FuncMap = template.FuncMap{
	"lower":   strings.ToLower,
	"upper":   strings.ToUpper,
	"replace": replace,
	"slugify": slugify,
}

// replace implements {{ replace .Var "old" "new" }} — all occurrences.
func replace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

var slugSep = regexp.MustCompile(`[^a-z0-9-]+`)

// slugify lowercases s and collapses every run of characters outside
// [a-z0-9-] into a single dash, trimming leading and trailing dashes.
func slugify(s string) string {
	return strings.Trim(slugSep.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

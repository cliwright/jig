package jig

import (
	"testing"
	"text/template"
)

// collectUsed must traverse every branch kind the parse tree can hold, not
// just the if/else shapes real templates use — values are strings, but the
// walker must not assume it.
func TestCollectUsedTraversesAllBranchKinds(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []string
	}{
		{`{{with .a}}{{.b}}{{end}}`, []string{"a", "b"}},
		{`{{range .a}}{{.b}}{{end}}`, []string{"a", "b"}},
		{`{{template "t" .a}}`, []string{"a"}},
		{`{{template "t"}}`, nil}, // no argument: a nil pipe
		{`{{if .a}}{{end}}`, []string{"a"}},
	} {
		tmpl, err := template.New("test").Parse(tc.text)
		if err != nil {
			t.Fatalf("parsing %q: %v", tc.text, err)
		}
		used := map[string]bool{}
		collectUsed(tmpl.Tree.Root, used)
		for _, name := range tc.want {
			if !used[name] {
				t.Errorf("%q: variable %s not recorded (have %v)", tc.text, name, used)
			}
		}
	}
}

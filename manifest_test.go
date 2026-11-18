package jig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validManifest = `{
  "version": 1,
  "variables": {
    "PROJECT_NAME": {"default": "demo", "pattern": "^[a-z][a-z0-9-]*$"},
    "author": {"default": "cliwright", "prompt": "Author name"}
  }
}`

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "jig.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadManifest(t *testing.T) {
	m, err := LoadManifest(writeManifest(t, validManifest))
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	defaults, err := m.Defaults()
	if err != nil {
		t.Fatalf("Defaults: %v", err)
	}
	if got := defaults["PROJECT_NAME"]; got != "demo" {
		t.Errorf("PROJECT_NAME default = %q, want demo", got)
	}
	if m.Variables["author"].Prompt != "Author name" {
		t.Errorf("author prompt = %q", m.Variables["author"].Prompt)
	}
}

func TestLoadManifestRejects(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "missing PROJECT_NAME",
			content: `{"version": 1, "variables": {"x": {"default": "y"}}}`,
			want:    "PROJECT_NAME",
		},
		{
			name:    "unknown top-level key",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "d"}}, "bogus": 1}`,
			want:    "bogus",
		},
		{
			name:    "variable without default",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"prompt": "p"}}}`,
			want:    "default",
		},
		{
			name:    "bad validation pattern",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "d", "pattern": "["}}}`,
			want:    "invalid pattern",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadManifest(writeManifest(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestManifestValidate(t *testing.T) {
	m, err := LoadManifest(writeManifest(t, validManifest))
	if err != nil {
		t.Fatal(err)
	}

	if err := m.Validate(map[string]string{"PROJECT_NAME": "demo", "author": "x"}); err != nil {
		t.Errorf("valid values rejected: %v", err)
	}

	err = m.Validate(map[string]string{"PROJECT_NAME": "demo", "bogus": "x"})
	if err == nil || !strings.Contains(err.Error(), `undeclared variable "bogus"`) {
		t.Errorf("error = %v, want undeclared-variable error", err)
	}

	err = m.Validate(map[string]string{"PROJECT_NAME": "Not Slug", "author": "x"})
	if err == nil || !strings.Contains(err.Error(), "does not match pattern") {
		t.Errorf("error = %v, want pattern-mismatch error", err)
	}
}

func TestManifestUnused(t *testing.T) {
	m, err := LoadManifest(writeManifest(t, validManifest))
	if err != nil {
		t.Fatal(err)
	}
	unused := m.Unused(map[string]bool{"PROJECT_NAME": true})
	if len(unused) != 1 || unused[0] != "author" {
		t.Errorf("unused = %v, want [author]", unused)
	}
}

func TestDerivedDefaults(t *testing.T) {
	m, err := LoadManifest(writeManifest(t, `{
	  "version": 1,
	  "variables": {
	    "PROJECT_NAME": {"default": "My Project", "pattern": ".*"},
	    "module": {"default": "{{ .PROJECT_NAME | slugify }}"},
	    "pkg": {"default": "{{ .module | replace \"-\" \"_\" }}"}
	  }
	}`))
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	defaults, err := m.Defaults()
	if err != nil {
		t.Fatalf("Defaults: %v", err)
	}
	if defaults["module"] != "my-project" {
		t.Errorf("module = %q, want my-project", defaults["module"])
	}
	if defaults["pkg"] != "my_project" {
		t.Errorf("pkg = %q, want my_project", defaults["pkg"])
	}
	// an override re-derives its dependents
	resolved, err := m.resolveDefaults(map[string]string{"PROJECT_NAME": "Other Thing"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved["module"] != "other-thing" || resolved["pkg"] != "other_thing" {
		t.Errorf("override did not re-derive: %v", resolved)
	}
	// referenced-in-defaults variables are not "unused"; pkg, referenced
	// nowhere, is
	unused := m.Unused(map[string]bool{})
	if len(unused) != 1 || unused[0] != "pkg" {
		t.Errorf("unused = %v, want [pkg]", unused)
	}
}

func TestDerivedDefaultsRejected(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "undeclared reference",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "d"}, "x": {"default": "{{ .nope }}"}}}`,
			want:    "undeclared variable",
		},
		{
			name:    "self reference",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "d"}, "x": {"default": "{{ .x }}"}}}`,
			want:    "references itself",
		},
		{
			name: "cycle",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "d"},
				"a": {"default": "{{ .b }}"}, "b": {"default": "{{ .a }}"}}}`,
			want: "cyclic defaults",
		},
		{
			name:    "unparseable default",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "{{"}}}`,
			want:    "invalid default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadManifest(writeManifest(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestChoices(t *testing.T) {
	m, err := LoadManifest(writeManifest(t, `{
	  "version": 1,
	  "variables": {
	    "PROJECT_NAME": {"default": "demo"},
	    "license": {"default": "MIT", "choices": ["MIT", "Apache-2.0", "GPL-3.0"]}
	  }
	}`))
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if err := m.Validate(map[string]string{"PROJECT_NAME": "demo", "license": "Apache-2.0"}); err != nil {
		t.Errorf("valid choice rejected: %v", err)
	}
	err = m.Validate(map[string]string{"PROJECT_NAME": "demo", "license": "BSD"})
	if err == nil || !strings.Contains(err.Error(), "not one of its choices") {
		t.Errorf("error = %v, want choices rejection", err)
	}
}

func TestChoicesRejected(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "pattern and choices",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo", "pattern": ".*", "choices": ["demo"]}}}`,
			want:    "mutually exclusive",
		},
		{
			name:    "default not in choices",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo", "choices": ["a", "b"]}}}`,
			want:    "is not one of its choices",
		},
		{
			name:    "duplicate choices",
			content: `{"version": 1, "variables": {"PROJECT_NAME": {"default": "a", "choices": ["a", "a"]}}}`,
			want:    "are equal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadManifest(writeManifest(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestLoadManifestReadError(t *testing.T) {
	_, err := LoadManifest(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil || !strings.Contains(err.Error(), "reading") {
		t.Fatalf("error = %v, want a read error", err)
	}
}

func TestLoadManifestMalformedJSON(t *testing.T) {
	_, err := LoadManifest(writeManifest(t, "not json"))
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Fatalf("error = %v, want a parse error", err)
	}
}

func TestLoadManifestVerbatimEntries(t *testing.T) {
	manifest := `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}},
		"verbatim": ["template", ".github/workflows/ci.yml", "docs/helm"]}`
	if _, err := LoadManifest(writeManifest(t, manifest)); err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	for _, bad := range []string{".", "..", "../escape", "/abs", "a//b"} {
		m := `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}},
			"verbatim": ["` + bad + `"]}`
		if _, err := LoadManifest(writeManifest(t, m)); err == nil {
			t.Errorf("verbatim entry %q: want a load error", bad)
		}
	}
}

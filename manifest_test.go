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
	if got := m.Defaults()["PROJECT_NAME"]; got != "demo" {
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

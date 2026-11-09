package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cliwright/jig"
)

func TestRunNewScaffoldsIntoTemplatesDir(t *testing.T) {
	templatesDir := t.TempDir()
	work := t.TempDir()

	cmd := newNewCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Flags().Set("templates-dir", templatesDir); err != nil {
		t.Fatal(err)
	}
	if err := runNew(cmd, templatesDir, "python/std/tool"); err != nil {
		t.Fatalf("runNew: %v", err)
	}

	skel := filepath.Join(templatesDir, "python/std/tool/v1", jig.TemplateDirName)
	for _, f := range []string{"jig.json", "README.md"} {
		if _, err := os.Stat(filepath.Join(skel, f)); err != nil {
			t.Errorf("missing scaffolded file %s: %v", f, err)
		}
	}
	// The cwd must stay untouched — scaffolding lands in the templates dir.
	entries, err := os.ReadDir(work)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("cwd was polluted: %v", entries)
	}
	if !strings.Contains(out.String(), "jig render --template python/std/tool") {
		t.Errorf("hint = %q, want the render next step", out.String())
	}

	// The scaffolded template must load and render back.
	res, err := jig.Render(filepath.Join(templatesDir, "python/std/tool/v1"), t.TempDir(),
		map[string]string{"PROJECT_NAME": "roundtrip"}, jig.Options{})
	if err != nil {
		t.Fatalf("rendering the scaffolded template: %v", err)
	}
	if _, err := os.Stat(filepath.Join(res.OutputDir, "README.md")); err != nil {
		t.Errorf("round-trip render missing README.md: %v", err)
	}

	// Scaffolding over an existing template refuses.
	if err := runNew(cmd, templatesDir, "python/std/tool"); err == nil ||
		!strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want already-exists refusal", err)
	}
}

func TestRunNewRejectsBadNames(t *testing.T) {
	cmd := newNewCmd()
	cmd.SetOut(&bytes.Buffer{})
	for _, name := range []string{"python/tool", "Python/std/tool", "a/b/c/d"} {
		if err := runNew(cmd, t.TempDir(), name); err == nil {
			t.Errorf("runNew(%q): want an error", name)
		}
	}
}

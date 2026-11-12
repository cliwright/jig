package jig

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemplate builds a template version directory with the given files
// under {{.PROJECT_NAME}}/ and returns the version dir.
func writeTemplate(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	skel := filepath.Join(root, "v1", TemplateDirName)
	for name, content := range files {
		full := filepath.Join(skel, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(root, "v1")
}

const templateManifest = `{
  "version": 1,
  "variables": {
    "PROJECT_NAME": {"default": "demo", "pattern": "^[a-z][a-z0-9-]*$"},
    "author": {"default": "cliwright"}
  }
}`

func TestRender(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json":                  templateManifest,
		"README.md":                 "# {{.PROJECT_NAME}} by {{.author | upper}}\n",
		"src/{{.PROJECT_NAME}}.txt": "slug: {{.PROJECT_NAME | slugify}}\n",
	})
	out := t.TempDir()

	res, err := Render(dir, out, map[string]string{"PROJECT_NAME": "my-proj", "author": "jesse"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasSuffix(res.OutputDir, filepath.Join("my-proj")) {
		t.Errorf("OutputDir = %q, want .../my-proj", res.OutputDir)
	}
	if len(res.Unused) != 0 {
		t.Errorf("Unused = %v, want none", res.Unused)
	}

	read := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(out, "my-proj", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		return string(data)
	}
	if got := read("README.md"); got != "# my-proj by JESSE\n" {
		t.Errorf("README.md = %q", got)
	}
	if got := read("src/my-proj.txt"); got != "slug: my-proj\n" {
		t.Errorf("rendered name/content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(out, "my-proj", "jig.json")); !os.IsNotExist(err) {
		t.Error("jig.json must not render into the output")
	}
}

func TestRenderReportsUnusedVariables(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json": templateManifest,
		"f.txt":    "only the project name: {{.PROJECT_NAME}}\n",
	})
	res, err := Render(dir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(res.Unused) != 1 || res.Unused[0] != "author" {
		t.Errorf("Unused = %v, want [author]", res.Unused)
	}
}

func TestRenderCopiesBinariesVerbatim(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json": templateManifest,
	})
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x01}
	src := filepath.Join(dir, TemplateDirName, "logo.png")
	if err := os.WriteFile(src, png, 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()

	if _, err := Render(dir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(out, "demo", "logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, png) {
		t.Errorf("binary changed in render: %x", got)
	}
}

func TestRenderPreservesExecutableMode(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json": templateManifest,
	})
	script := filepath.Join(dir, TemplateDirName, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho {{.PROJECT_NAME}}\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()

	if _, err := Render(dir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	info, err := os.Stat(filepath.Join(out, "demo", "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %o, want 755", info.Mode().Perm())
	}
}

func TestRenderRejectsUndeclaredTemplateVar(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json": templateManifest,
		"oops.txt": "{{.nosuch}}\n",
	})
	_, err := Render(dir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "nosuch") {
		t.Fatalf("error = %v, want missingkey error naming the variable", err)
	}
}

func TestRenderRejectsPatternViolation(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json": templateManifest,
	})
	_, err := Render(dir, t.TempDir(), map[string]string{"PROJECT_NAME": "Bad Name", "author": "x"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "does not match pattern") {
		t.Fatalf("error = %v, want pattern error", err)
	}
}

func TestRenderRefusesToOverwriteFiles(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json": templateManifest,
		"f.txt":    "x",
	})
	out := t.TempDir()

	// A non-empty output directory is fine as long as no file collides —
	// rendering a second type version next to the first is the normal case.
	existing := filepath.Join(out, "demo")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existing, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	vars := map[string]string{"PROJECT_NAME": "demo", "author": "x"}
	if _, err := Render(dir, out, vars, Options{}); err != nil {
		t.Fatalf("Render alongside existing files: %v", err)
	}
	keep, err := os.ReadFile(filepath.Join(existing, "keep.txt"))
	if err != nil || string(keep) != "keep" {
		t.Fatalf("unrelated file was disturbed: %v %q", err, keep)
	}

	// A second identical render collides on f.txt and must refuse BEFORE
	// writing anything.
	_, err = Render(dir, out, vars, Options{})
	if err == nil || !strings.Contains(err.Error(), "would overwrite") {
		t.Fatalf("error = %v, want overwrite refusal", err)
	}

	// Force allows the overwrite.
	if _, err := Render(dir, out, vars, Options{Force: true}); err != nil {
		t.Fatalf("forced Render: %v", err)
	}
}

func TestResolveTemplate(t *testing.T) {
	base := t.TempDir()
	for _, v := range []string{"v1", "v2", "v10", "drafts"} {
		if err := os.MkdirAll(filepath.Join(base, "python/astral/lib", v), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	dir, err := ResolveTemplate(base, "python/astral/lib", 0)
	if err != nil || !strings.HasSuffix(dir, filepath.Join("python/astral/lib", "v10")) {
		t.Errorf("latest = %q, %v; want v10", dir, err)
	}
	dir, err = ResolveTemplate(base, "python/astral/lib", 2)
	if err != nil || !strings.HasSuffix(dir, "v2") {
		t.Errorf("exact = %q, %v; want v2", dir, err)
	}
	if _, err = ResolveTemplate(base, "python/astral/lib", 3); err == nil || !strings.Contains(err.Error(), "version 3 not found") {
		t.Errorf("error = %v, want missing-version error", err)
	}
	if _, err = ResolveTemplate(base, "nosuch/template", 0); err == nil {
		t.Error("want error for unknown template")
	}

	// A direct path to a template ROOT resolves versions inside it (a
	// registry-style name that happens to exist as a relative path must not
	// short-circuit version resolution).
	got, err := ResolveTemplate("", filepath.Join(base, "python/astral/lib"), 0)
	if err != nil || !strings.HasSuffix(got, "v10") {
		t.Errorf("path to root = %q, %v; want v10", got, err)
	}
	// A direct path to a version directory is used as-is.
	vdir := filepath.Join(base, "python/astral/lib", "v2", TemplateDirName)
	if err := os.MkdirAll(vdir, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = ResolveTemplate("", filepath.Join(base, "python/astral/lib", "v2"), 0)
	if err != nil || !strings.HasSuffix(got, "v2") {
		t.Errorf("path to version dir = %q, %v; want v2", got, err)
	}
}

func TestRenderVerbatimSubtree(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json":                            `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}}, "verbatim": ["template"]}`,
		"README.md":                           "# {{.PROJECT_NAME}}",
		"template/{{.PROJECT_NAME}}/jig.json": `{"version": 1, "variables": {"PROJECT_NAME": {"default": "inner"}}}`,
		"template/{{.PROJECT_NAME}}/notes.md": "literal {{.UNDECLARED}} stays",
	})
	out := t.TempDir()

	if _, err := Render(dir, out, map[string]string{"PROJECT_NAME": "demo"}, Options{}); err != nil {
		t.Fatalf("Render: %v", err)
	}

	// The outer skeleton renders; the verbatim subtree does not — names and
	// contents ship literally, including its own jig.json.
	readme, err := os.ReadFile(filepath.Join(out, "demo", "README.md"))
	if err != nil || string(readme) != "# demo" {
		t.Fatalf("README = %q, %v", readme, err)
	}
	inner := filepath.Join(out, "demo", "template", "{{.PROJECT_NAME}}")
	manifest, err := os.ReadFile(filepath.Join(inner, "jig.json"))
	if err != nil {
		t.Fatalf("verbatim jig.json missing: %v", err)
	}
	if !strings.Contains(string(manifest), `"inner"`) {
		t.Errorf("verbatim jig.json = %q, want unrendered", manifest)
	}
	notes, err := os.ReadFile(filepath.Join(inner, "notes.md"))
	if err != nil || string(notes) != "literal {{.UNDECLARED}} stays" {
		t.Errorf("notes.md = %q, %v — want literal copy", notes, err)
	}
}

func TestLoadManifestRejectsEscapingVerbatimPath(t *testing.T) {
	dir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}}, "verbatim": ["../outside"]}`,
	})
	_, err := LoadManifest(filepath.Join(dir, TemplateDirName, ManifestFileName))
	if err == nil || !strings.Contains(err.Error(), "verbatim path") {
		t.Fatalf("error = %v, want verbatim path rejection", err)
	}
}

package jig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemplate builds a template version directory: {{.PROJECT_NAME}}/
// holding the given files (jig.json included), and returns the version dir.
func writeTemplate(t *testing.T, files map[string]string) string {
	t.Helper()
	skel := filepath.Join(t.TempDir(), "v1", TemplateDirName)
	for rel, content := range files {
		path := filepath.Join(skel, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Dir(skel)
}

func TestRender(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":                            validManifest,
		"README.md":                           "# {{.PROJECT_NAME}} by {{.author}}\n",
		"src/{{.PROJECT_NAME | slugify}}.txt": "pkg {{.PROJECT_NAME}}\n",
	})
	out := filepath.Join(t.TempDir(), "out")
	res, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "my-demo", "author": "x"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if want := filepath.Join(out, "my-demo"); res.OutputDir != want {
		t.Errorf("OutputDir = %q, want %q", res.OutputDir, want)
	}
	data, err := os.ReadFile(filepath.Join(res.OutputDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "# my-demo by x\n" {
		t.Errorf("README = %q", got)
	}
	if _, err := os.Stat(filepath.Join(res.OutputDir, "src", "my-demo.txt")); err != nil {
		t.Errorf("rendered name missing: %v", err)
	}
}

func TestRenderRefusesEscape(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}}}`,
		"file.txt": "{{.PROJECT_NAME}}\n",
	})
	out := filepath.Join(t.TempDir(), "out")
	for _, name := range []string{"../evil", "a/../../b", "..", "."} {
		_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": name}, Options{})
		if err == nil || !strings.Contains(err.Error(), "escapes the output directory") {
			t.Errorf("PROJECT_NAME %q: error = %v, want escape refusal", name, err)
		}
	}
}

func TestRenderRefusesEscapingFileName(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":           validManifest,
		"{{.author}}.txt":    "x\n",
		"sub/{{.author}}.md": "x\n",
	})
	out := filepath.Join(t.TempDir(), "out")
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "../evil"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "escapes the output directory") {
		t.Fatalf("error = %v, want escape refusal", err)
	}
	// a nested-but-contained rendered name is fine
	_, err = Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "a/b"}, Options{})
	if err != nil {
		t.Fatalf("nested name refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "demo", "a", "b.txt")); err != nil {
		t.Errorf("nested file missing: %v", err)
	}
}

func TestRenderRefusesOverwrite(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"file.txt": "new\n",
	})
	out := t.TempDir()
	if err := os.MkdirAll(filepath.Join(out, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "demo", "file.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "would overwrite") {
		t.Fatalf("error = %v, want overwrite refusal", err)
	}
	data, _ := os.ReadFile(filepath.Join(out, "demo", "file.txt"))
	if string(data) != "old\n" {
		t.Error("existing file was clobbered")
	}
	if _, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Force: true}); err != nil {
		t.Fatalf("Force render: %v", err)
	}
}

// writeSymlinkTemplate builds a template holding one symlink with the given
// target, returning the version dir.
func writeSymlinkTemplate(t *testing.T, target string) string {
	t.Helper()
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"real.txt": "real\n",
	})
	link := filepath.Join(versionDir, TemplateDirName, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	return versionDir
}

func TestRenderRejectsSymlinks(t *testing.T) {
	versionDir := writeSymlinkTemplate(t, "real.txt")
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "must not contain symlinks") {
		t.Fatalf("error = %v, want symlink refusal", err)
	}
}

func TestRenderTrustRecreatesSymlinks(t *testing.T) {
	versionDir := writeSymlinkTemplate(t, "real.txt")
	out := t.TempDir()
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Trust: true})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	target, err := os.Readlink(filepath.Join(out, "demo", "link.txt"))
	if err != nil {
		t.Fatalf("expected a symlink: %v", err)
	}
	if target != "real.txt" {
		t.Errorf("link target = %q, want real.txt", target)
	}
}

func TestRenderTrustRefusesEscapingSymlink(t *testing.T) {
	versionDir := writeSymlinkTemplate(t, "../../etc/passwd")
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Trust: true})
	if err == nil || !strings.Contains(err.Error(), "outside the template tree") {
		t.Fatalf("error = %v, want escape refusal", err)
	}
}

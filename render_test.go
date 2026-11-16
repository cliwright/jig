package jig

import (
	"os"
	"path/filepath"
	"runtime"
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
	// "|" is illegal in Windows file names, so the piped form cannot exist
	// in a fixture there; the function-call form renders the same name.
	slugged := "src/{{.PROJECT_NAME | slugify}}.txt"
	if runtime.GOOS == "windows" {
		slugged = "src/{{slugify .PROJECT_NAME}}.txt"
	}
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":  validManifest,
		"README.md": "# {{.PROJECT_NAME}} by {{.author}}\n",
		slugged:     "pkg {{.PROJECT_NAME}}\n",
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

func TestRenderRefusesSymlinkedDestDir(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":             validManifest,
		"config/settings.yaml": "key: value\n",
	})
	out := t.TempDir()
	external := t.TempDir()
	if err := os.MkdirAll(filepath.Join(out, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(out, "demo", "config")); err != nil {
		t.Fatal(err)
	}
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "refusing to write through it") {
		t.Fatalf("error = %v, want symlinked-destination refusal", err)
	}
	entries, err := os.ReadDir(external)
	if err != nil || len(entries) != 0 {
		t.Fatalf("external dir should be untouched, got %v, %v", entries, err)
	}
}

func TestRenderRefusesSymlinkedDestFile(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"file.txt": "new\n",
	})
	out := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(target, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(out, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(out, "demo", "file.txt")); err != nil {
		t.Fatal(err)
	}
	// refused with and without Force: force overwrites regular files, it
	// never follows someone's link
	for _, opts := range []Options{{}, {Force: true}} {
		_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, opts)
		if err == nil || !strings.Contains(err.Error(), "refusing to write through it") {
			t.Fatalf("Force=%v: error = %v, want symlinked-destination refusal", opts.Force, err)
		}
	}
	data, _ := os.ReadFile(target)
	if string(data) != "original\n" {
		t.Fatalf("link target was clobbered: %q", data)
	}
}

func TestRenderTrustRejectsAbsoluteSymlink(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"real.txt": "real\n",
	})
	link := filepath.Join(versionDir, TemplateDirName, "link.txt")
	// absolute target that resolves INSIDE the template tree — still refused
	if err := os.Symlink(filepath.Join(versionDir, TemplateDirName, "real.txt"), link); err != nil {
		t.Fatal(err)
	}
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Trust: true})
	if err == nil || !strings.Contains(err.Error(), "absolute target") {
		t.Fatalf("error = %v, want absolute-symlink refusal", err)
	}
}

const verbatimManifest = `{
  "version": 1,
  "variables": {"PROJECT_NAME": {"default": "demo"}},
  "verbatim": ["template"]
}`

func TestRenderVerbatim(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":                            verbatimManifest,
		"{{.PROJECT_NAME}}/outer.txt":         "rendered {{.PROJECT_NAME}}\n",
		"template/{{.PROJECT_NAME}}/jig.json": `{"inner": "{{.PROJECT_NAME}}"}`,
	})
	out := t.TempDir()
	res, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// outside the verbatim dir: names and contents render
	data, err := os.ReadFile(filepath.Join(res.OutputDir, "demo", "outer.txt"))
	if err != nil || string(data) != "rendered demo\n" {
		t.Errorf("outer.txt = %q, %v", data, err)
	}
	// inside the verbatim dir: names and contents ship as authored
	inner := filepath.Join(res.OutputDir, "template", "{{.PROJECT_NAME}}", "jig.json")
	data, err = os.ReadFile(inner)
	if err != nil {
		t.Fatalf("inner file not at its unrendered path: %v", err)
	}
	if string(data) != `{"inner": "{{.PROJECT_NAME}}"}` {
		t.Errorf("inner content was rendered: %q", data)
	}
}

func TestRenderVerbatimDirStillVetted(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":          verbatimManifest,
		"template/real.txt": "x\n",
	})
	link := filepath.Join(versionDir, TemplateDirName, "template", "link.txt")
	if err := os.Symlink("real.txt", link); err != nil {
		t.Fatal(err)
	}
	// verbatim changes copying, never vetting
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "must not contain symlinks") {
		t.Fatalf("error = %v, want symlink refusal inside verbatim dir", err)
	}
	// with trust, a relative in-tree link inside a verbatim dir is fine
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo"}, Options{Trust: true})
	if err != nil {
		t.Fatalf("trusted render: %v", err)
	}
	target, err := os.Readlink(filepath.Join(res.OutputDir, "template", "link.txt"))
	if err != nil || target != "real.txt" {
		t.Errorf("link = %q, %v", target, err)
	}
}

func TestRenderRefusesSymlinkedProjectDir(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"file.txt": "x\n",
	})
	out := t.TempDir()
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(out, "demo")); err != nil {
		t.Fatal(err)
	}
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "refusing to write through it") {
		t.Fatalf("error = %v, want symlinked-project-dir refusal", err)
	}
}

func TestRenderRefusesEmptyRenderedName(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":    `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}, "suffix": {"default": ""}}}`,
		"{{.suffix}}": "x\n",
	})
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "suffix": ""}, Options{})
	if err == nil || !strings.Contains(err.Error(), "empty name") {
		t.Fatalf("error = %v, want empty-name refusal", err)
	}
}

func TestRenderCopiesBinariesVerbatim(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
	})
	// binary content (NUL byte) containing template-looking bytes: it must
	// copy through untouched, never parsed
	bin := append([]byte("{{.PROJECT_NAME}}\x00{{\xde\xad\xbe\xef"), []byte("garbage")...)
	src := filepath.Join(versionDir, TemplateDirName, "blob.bin")
	if err := os.WriteFile(src, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(res.OutputDir, "blob.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(bin) {
		t.Errorf("binary was modified: %q", got)
	}
}

func TestRenderRejectsInternalCollision(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {
			"PROJECT_NAME": {"default": "demo"},
			"A": {"default": "one"}, "B": {"default": "two"}}}`,
		"{{.A}}.txt": "from A\n",
		"{{.B}}.txt": "from B\n",
	})
	out := t.TempDir()
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "A": "same", "B": "same"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("error = %v, want collision refusal", err)
	}
	// nothing may be written for a refused plan
	entries, _ := os.ReadDir(out)
	if len(entries) != 0 {
		t.Errorf("output dir not empty after refused render: %v", entries)
	}
	// distinct names render fine
	if _, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "A": "one", "B": "two"}, Options{}); err != nil {
		t.Fatalf("distinct names refused: %v", err)
	}
}

func TestRenderRejectsFileDirCollision(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {
			"PROJECT_NAME": {"default": "demo"},
			"A": {"default": "x"}, "B": {"default": "y"}}}`,
		"{{.A}}":           "a file\n",
		"{{.B}}/inner.txt": "in a dir\n",
	})
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "A": "same", "B": "same"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("error = %v, want collision refusal", err)
	}
}

func TestRenderAllowsDirectoryMerge(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {
			"PROJECT_NAME": {"default": "demo"},
			"A": {"default": "x"}, "B": {"default": "y"}}}`,
		"{{.A}}/a.txt": "a\n",
		"{{.B}}/b.txt": "b\n",
	})
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "A": "shared", "B": "shared"}, Options{})
	if err != nil {
		t.Fatalf("directory merge refused: %v", err)
	}
	for _, f := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(res.OutputDir, "shared", f)); err != nil {
			t.Errorf("merged file %s missing: %v", f, err)
		}
	}
}

func TestCheckCatchesInternalCollision(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {
			"PROJECT_NAME": {"default": "demo"},
			"A": {"default": "same"}, "B": {"default": "same"}}}`,
		"{{.A}}.txt": "from A\n",
		"{{.B}}.txt": "from B\n",
	})
	if _, err := Check(versionDir); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("Check error = %v, want collision refusal", err)
	}
}

func TestRenderContentErrorLeavesNoPartialTree(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":    validManifest,
		"README.md":   "ok\n",
		"config.yaml": "ok\n",
		"zz_main.go":  "{{ .author | nosuchfunc }}\n",
	})
	out := t.TempDir()
	if _, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{}); err == nil {
		t.Fatal("want a render error")
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 0 {
		t.Errorf("partial tree left behind: %v", entries)
	}
}

func TestRenderForceDoesNotWriteThroughHardLink(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"file.txt": "new\n",
	})
	out := t.TempDir()
	external := filepath.Join(t.TempDir(), "external.txt")
	if err := os.WriteFile(external, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(out, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(out, "demo", "file.txt")
	if err := os.Link(external, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Force: true}); err != nil {
		t.Fatal(err)
	}
	// the render replaced the destination inode; the alias must be untouched
	data, _ := os.ReadFile(external)
	if string(data) != "original\n" {
		t.Errorf("hard-linked file outside the tree was modified: %q", data)
	}
	data, _ = os.ReadFile(dst)
	if string(data) != "new\n" {
		t.Errorf("destination = %q, want new", data)
	}
	ei, _ := os.Stat(external)
	di, _ := os.Stat(dst)
	if os.SameFile(ei, di) {
		t.Error("destination still shares an inode with the external file")
	}
}

func TestRenderRejectsCaseOnlyCollision(t *testing.T) {
	// source names differ; rendered names collide case-insensitively.
	// (Two literal files README.md/readme.md can't even coexist in the
	// fixture on a case-insensitive filesystem.)
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {
			"PROJECT_NAME": {"default": "demo"},
			"A": {"default": "README"}, "B": {"default": "readme"}}}`,
		"{{.A}}.md": "upper\n",
		"{{.B}}.md": "lower\n",
	})
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "A": "README", "B": "readme"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("error = %v, want case-collision refusal", err)
	}
}

func TestRenderWorkflowFile(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		".github/workflows/ci.yml": "name: {{ .PROJECT_NAME }}\n" +
			"on:\n  push:\n    branches: [${{ ldelim }} github.default_branch {{ rdelim }}]\n",
	})
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(res.OutputDir, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := "name: demo\non:\n  push:\n    branches: [${{ github.default_branch }}]\n"
	if string(data) != want {
		t.Errorf("ci.yml = %q, want %q", data, want)
	}
}

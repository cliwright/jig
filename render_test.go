package jig

import (
	"encoding/json"
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

func TestRenderSkipsEmptyRenderedName(t *testing.T) {
	// a name rendering to an empty string is the conditional mechanism:
	// the file is excluded, not an error (ADR 19)
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":    `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}, "suffix": {"default": ""}}}`,
		"{{.suffix}}": "x\n",
	})
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "suffix": ""}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	entries, err := os.ReadDir(res.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("output = %v, want the empty-named file excluded", entries)
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

func TestRenderManifestErrors(t *testing.T) {
	// a missing manifest fails before any planning
	versionDir := writeTemplate(t, map[string]string{
		"file.txt": "x\n",
	})
	if _, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{}); err == nil {
		t.Error("want a manifest read error")
	}
	// an unparseable manifest fails before any planning
	versionDir = writeTemplate(t, map[string]string{
		"jig.json": "not json",
	})
	if _, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{}); err == nil {
		t.Error("want a manifest parse error")
	}
	// values failing manifest validation fail before any planning
	versionDir = writeTemplate(t, map[string]string{
		"jig.json": validManifest,
	})
	if _, err := Render(versionDir, t.TempDir(), map[string]string{}, Options{}); err == nil {
		t.Error("want a validation error")
	}
}

func TestRenderRefusesWriteThroughFileComponent(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":     validManifest,
		"sub/file.txt": "x\n",
	})
	out := t.TempDir()
	if err := os.MkdirAll(filepath.Join(out, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	// "sub" exists as a regular file: writing beneath it must be refused
	// by name, identically on every platform
	if err := os.WriteFile(filepath.Join(out, "demo", "sub"), []byte("blocker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Force: true})
	if err == nil || !strings.Contains(err.Error(), "non-directory") {
		t.Fatalf("error = %v, want a non-directory refusal", err)
	}
}

func TestRenderBadNameSyntax(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"{{.txt":   "x\n",
	})
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("error = %v, want a name parse error", err)
	}
}

func TestRenderLongTextFile(t *testing.T) {
	// longer than the binary-sniff window: the content must still render
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":  validManifest,
		"long.txt":  strings.Repeat("# {{.PROJECT_NAME}}\n", 100),
		"short.txt": "{{.PROJECT_NAME}}\n",
	})
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(res.OutputDir, "long.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Repeat("# demo\n", 100)
	if string(data) != want {
		t.Errorf("long.txt is %d bytes, want %d fully rendered", len(data), len(want))
	}
}

func TestRenderForceSymlinkOverExistingFileFails(t *testing.T) {
	versionDir := writeSymlinkTemplate(t, "real.txt")
	out := t.TempDir()
	// the link's destination exists as a regular file: recreating the link
	// must fail rather than clobber it, even under --force
	if err := os.MkdirAll(filepath.Join(out, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "demo", "link.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Force: true, Trust: true})
	if err == nil || !strings.Contains(err.Error(), "linking") {
		t.Fatalf("error = %v, want a link failure", err)
	}
	data, _ := os.ReadFile(filepath.Join(out, "demo", "link.txt"))
	if string(data) != "keep\n" {
		t.Error("existing file was clobbered")
	}
}

func TestRenderRejectsSymlinkedManifest(t *testing.T) {
	// the manifest is the template's most security-relevant file: the
	// default no-symlink policy covers it too, for Render and Check
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"file.txt": "x\n",
	})
	outside := filepath.Join(t.TempDir(), "jig.json")
	if err := os.WriteFile(outside, []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(versionDir, TemplateDirName, "jig.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(versionDir, TemplateDirName, "jig.json")); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"PROJECT_NAME": "demo", "author": "x"}
	_, err := Render(versionDir, t.TempDir(), values, Options{})
	if err == nil || !strings.Contains(err.Error(), "must not contain symlinks") {
		t.Fatalf("Render error = %v, want a symlink refusal", err)
	}
	if _, err := Check(versionDir); err == nil || !strings.Contains(err.Error(), "must not contain symlinks") {
		t.Fatalf("Check error = %v, want a symlink refusal", err)
	}
}

func TestRenderTrustAllowsContainedSymlinkedManifest(t *testing.T) {
	// under --trust, a relative manifest link resolving inside the
	// template tree is as acceptable as any other contained link
	versionDir := writeTemplate(t, map[string]string{
		"file.txt": "x\n",
	})
	skel := filepath.Join(versionDir, TemplateDirName)
	if err := os.WriteFile(filepath.Join(skel, "real.jig.json"), []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.jig.json", filepath.Join(skel, "jig.json")); err != nil {
		t.Fatal(err)
	}
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Trust: true})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if _, err := os.Stat(filepath.Join(res.OutputDir, "file.txt")); err != nil {
		t.Errorf("rendered file missing: %v", err)
	}
}

func TestRenderTrustRejectsEscapingSymlinkedManifest(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"file.txt": "x\n",
	})
	outside := filepath.Join(t.TempDir(), "jig.json")
	if err := os.WriteFile(outside, []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(filepath.Join(versionDir, TemplateDirName), outside)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(rel, filepath.Join(versionDir, TemplateDirName, "jig.json")); err != nil {
		t.Fatal(err)
	}
	_, err = Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Trust: true})
	if err == nil || !strings.Contains(err.Error(), "outside the template tree") {
		t.Fatalf("error = %v, want an escape refusal", err)
	}
}

func TestRenderManifestOnlyTemplateCreatesOutputDir(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
	})
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	info, err := os.Stat(res.OutputDir)
	if err != nil || !info.IsDir() {
		t.Errorf("OutputDir = %s, stat: %v — want a created directory", res.OutputDir, err)
	}
}

func TestRenderConditionalFile(t *testing.T) {
	manifest := `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "demo"},
		"WITH_DOCKER": {"default": "no", "choices": ["yes", "no"]}}}`
	// backticks, not quotes: '"' is illegal in Windows file names
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":  manifest,
		"README.md": "# {{.PROJECT_NAME}}\n",
		"{{if eq .WITH_DOCKER `yes`}}Dockerfile{{end}}": "FROM scratch\n",
	})
	// excluded: nothing rendered, and the variable still counts as used
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "WITH_DOCKER": "no"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if _, err := os.Stat(filepath.Join(res.OutputDir, "Dockerfile")); !os.IsNotExist(err) {
		t.Error("Dockerfile rendered despite WITH_DOCKER=no")
	}
	if len(res.Unused) != 0 {
		t.Errorf("unused = %v, want none", res.Unused)
	}
	// included under the other choice
	res, err = Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "WITH_DOCKER": "yes"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if _, err := os.Stat(filepath.Join(res.OutputDir, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile missing with WITH_DOCKER=yes: %v", err)
	}
}

func TestRenderConditionalDirectoryPrunesSubtree(t *testing.T) {
	manifest := `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "demo"},
		"KIND": {"default": "lib", "choices": ["lib", "cli"]}}}`
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":  manifest,
		"README.md": "x\n",
		"{{if eq .KIND `cli`}}cmd{{end}}/main.go":    "package main\n",
		"{{if eq .KIND `cli`}}cmd{{end}}/extra/x.go": "package main\n",
	})
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "KIND": "lib"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	entries, err := os.ReadDir(res.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "README.md" {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("output = %v, want only README.md", names)
	}
	res, err = Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "KIND": "cli"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if _, err := os.Stat(filepath.Join(res.OutputDir, "cmd", "main.go")); err != nil {
		t.Errorf("cmd/main.go missing with KIND=cli: %v", err)
	}
	if _, err := os.Stat(filepath.Join(res.OutputDir, "cmd", "extra", "x.go")); err != nil {
		t.Errorf("cmd/extra/x.go missing with KIND=cli: %v", err)
	}
}

func TestRenderConditionalContentStillValidated(t *testing.T) {
	// a broken template hiding in an excluded branch still fails, for
	// Render and for Check
	manifest := `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "demo"},
		"X": {"default": "no", "choices": ["yes", "no"]}}}`
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":                       manifest,
		"{{if eq .X `yes`}}f.txt{{end}}": "{{.NOPE}}\n",
	})
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "X": "no"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "NOPE") {
		t.Fatalf("Render error = %v, want the excluded file's bad content to fail", err)
	}
	if _, err := Check(versionDir); err == nil || !strings.Contains(err.Error(), "NOPE") {
		t.Fatalf("Check error = %v, want the excluded file's bad content to fail", err)
	}
}

func TestRenderEmptyProjectNameStillRefused(t *testing.T) {
	// conditional exclusion is for files and directories INSIDE the
	// skeleton; the project directory itself may not render empty
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}}}`,
		"file.txt": "x\n",
	})
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": ""}, Options{})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %v, want an empty-name refusal", err)
	}
}

func TestRenderVerbatimSingleFilesAndPaths(t *testing.T) {
	// foreign-syntax content (${{ }} parses as Go template actions and
	// undeclared variables) must pass through byte-identical — including
	// at the skeleton root, which has no parent directory to list
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}},
			"verbatim": ["renovate.json", ".github/workflows/ci.yml", "docs/helm"]}`,
		"renovate.json":            "branch: ${{ x }}\n",
		".github/workflows/ci.yml": "on: [${{ github }}]\n",
		"docs/helm/chart.yaml":     "name: ${{ helm }}\n",
		"README.md":                "# {{.PROJECT_NAME}}\n",
	})
	res, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo"}, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for rel, want := range map[string]string{
		"renovate.json":            "branch: ${{ x }}\n",
		".github/workflows/ci.yml": "on: [${{ github }}]\n",
		"docs/helm/chart.yaml":     "name: ${{ helm }}\n",
	} {
		data, err := os.ReadFile(filepath.Join(res.OutputDir, filepath.FromSlash(rel)))
		if err != nil || string(data) != want {
			t.Errorf("%s = %q, %v — want verbatim %q", rel, data, err, want)
		}
	}
	data, err := os.ReadFile(filepath.Join(res.OutputDir, "README.md"))
	if err != nil || string(data) != "# demo\n" {
		t.Errorf("README.md = %q, %v — want rendered", data, err)
	}
}

func TestRenderWritesProvenance(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"file.txt": "x\n",
	})
	out := t.TempDir()
	values := map[string]string{"PROJECT_NAME": "demo", "author": "x"}
	res, err := Render(versionDir, out, values, Options{
		Provenance: &Provenance{Template: "python/std/lib", Version: 1, Tool: "jig test"},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(res.OutputDir, ProvenanceFileName))
	if err != nil {
		t.Fatalf("provenance record missing: %v", err)
	}
	var rec map[string]any
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatalf("record is not JSON: %v", err)
	}
	if rec["template"] != "python/std/lib" || rec["template_version"] != 1.0 || rec["tool"] != "jig test" {
		t.Errorf("record = %v", rec)
	}
	recorded, _ := rec["values"].(map[string]any)
	if recorded["PROJECT_NAME"] != "demo" || recorded["author"] != "x" {
		t.Errorf("recorded values = %v", recorded)
	}
	// and a render without provenance writes no record
	res, err = Render(versionDir, t.TempDir(), values, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(res.OutputDir, ProvenanceFileName)); !os.IsNotExist(err) {
		t.Error("record written without provenance requested")
	}
}

func TestRenderProvenanceConflictsWithTemplateFile(t *testing.T) {
	// a template shipping its own .jig.json fails rather than having it
	// silently replaced by the record
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":  validManifest,
		".jig.json": "mine\n",
	})
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{
		Provenance: &Provenance{Template: "x/y/z", Version: 1, Tool: "jig test"},
	})
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("error = %v, want a reserved-name refusal", err)
	}
}

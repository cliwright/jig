//go:build unix

package jig

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRenderRejectsNonRegularFiles(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
	})
	if err := syscall.Mkfifo(filepath.Join(versionDir, TemplateDirName, "fifo"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Render(versionDir, t.TempDir(), map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "non-regular file") {
		t.Fatalf("error = %v, want non-regular-file refusal", err)
	}
	// check catches it too, without hanging
	if _, err := Check(versionDir); err == nil || !strings.Contains(err.Error(), "non-regular file") {
		t.Fatalf("Check error = %v, want non-regular-file refusal", err)
	}
}

// Windows has no executable bit — Go reports every file as 0666 and Chmod
// only honors the read-only attribute — so mode restoration is unobservable
// there and this test is unix-only.
func TestRenderForceRestoresTemplateMode(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
	})
	script := filepath.Join(versionDir, TemplateDirName, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	values := map[string]string{"PROJECT_NAME": "demo", "author": "x"}
	res, err := Render(versionDir, out, values, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// destination loses the executable bit, then a force render must
	// restore the template's mode
	dst := filepath.Join(res.OutputDir, "run.sh")
	if err := os.Chmod(dst, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Render(versionDir, out, values, Options{Force: true}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %o, want 755 after force render", info.Mode().Perm())
	}
}

func TestBumpRefusesNonRegularFiles(t *testing.T) {
	templatesDir := t.TempDir()
	skel := filepath.Join(templatesDir, "x", "y", "z", "v1", TemplateDirName)
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(skel, "fifo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Bump(templatesDir, "x/y/z"); err == nil || !strings.Contains(err.Error(), "non-regular file") {
		t.Fatalf("error = %v, want non-regular-file refusal", err)
	}
}

func TestRenderForceRefusesSpecialDestFile(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": validManifest,
		"file.txt": "new\n",
	})
	out := t.TempDir()
	if err := os.MkdirAll(filepath.Join(out, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(out, "demo", "file.txt"), 0o644); err != nil {
		t.Fatal(err)
	}
	// must fail fast, not block opening the FIFO for writing
	_, err := Render(versionDir, out, map[string]string{"PROJECT_NAME": "demo", "author": "x"}, Options{Force: true})
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("error = %v, want not-a-regular-file refusal", err)
	}
}

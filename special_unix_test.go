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

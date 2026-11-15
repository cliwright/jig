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

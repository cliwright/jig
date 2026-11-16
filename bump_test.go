package jig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBump(t *testing.T) {
	templatesDir := t.TempDir()
	skel := filepath.Join(templatesDir, "python", "std", "lib", "v1", TemplateDirName)
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skel, "jig.json"), []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	next, err := Bump(templatesDir, "python/std/lib")
	if err != nil {
		t.Fatalf("Bump: %v", err)
	}
	if want := filepath.Join(templatesDir, "python", "std", "lib", "v2"); next != want {
		t.Fatalf("next = %q, want %q", next, want)
	}
	data, err := os.ReadFile(filepath.Join(next, TemplateDirName, "jig.json"))
	if err != nil || string(data) != validManifest {
		t.Fatalf("v2 manifest = %q, %v", data, err)
	}

	next, err = Bump(templatesDir, "python/std/lib")
	if err != nil || !strings.HasSuffix(next, "v3") {
		t.Fatalf("second bump = %q, %v, want v3", next, err)
	}

	if _, err := Bump(templatesDir, "python/std/missing"); err == nil {
		t.Fatal("want an error bumping a missing template")
	}
}

func TestBumpRefusesSymlinks(t *testing.T) {
	templatesDir := t.TempDir()
	skel := filepath.Join(templatesDir, "x", "y", "z", "v1", TemplateDirName)
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(skel, "real.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.txt", filepath.Join(skel, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := Bump(templatesDir, "x/y/z"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error = %v, want symlink refusal", err)
	}
}

func TestBumpRejectsNonVersionDir(t *testing.T) {
	// a directory holding {{.PROJECT_NAME}} resolves as a version dir
	// as-is; bumping one not named v<N> has no "next" to compute
	dir := t.TempDir()
	skel := filepath.Join(dir, "my-template", TemplateDirName)
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skel, "jig.json"), []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Bump(t.TempDir(), filepath.Join(dir, "my-template"))
	if err == nil || !strings.Contains(err.Error(), "not v<N>") {
		t.Fatalf("error = %v, want a not-v<N> refusal", err)
	}
}

func TestBumpRefusesExistingNextVersion(t *testing.T) {
	// bumping a version directory given directly (v1, while v2 exists)
	// would land on the existing v2
	templatesDir := t.TempDir()
	for _, v := range []string{"v1", "v2"} {
		skel := filepath.Join(templatesDir, "python", "std", "lib", v, TemplateDirName)
		if err := os.MkdirAll(skel, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skel, "jig.json"), []byte(validManifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Bump(templatesDir, filepath.Join(templatesDir, "python", "std", "lib", "v1"))
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want an already-exists refusal", err)
	}
}

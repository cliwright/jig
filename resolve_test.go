package jig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTemplateByName(t *testing.T) {
	templatesDir := t.TempDir()
	base := filepath.Join(templatesDir, "python", "std", "lib")
	// non-numeric and non-version dirs must be ignored
	for _, d := range []string{"v1", "v2", "v3", "vX", "v0", "notes"} {
		if err := os.MkdirAll(filepath.Join(base, d, TemplateDirName), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dir, err := ResolveTemplate(templatesDir, "python/std/lib", 0)
	if err != nil {
		t.Fatalf("ResolveTemplate: %v", err)
	}
	if want := filepath.Join(base, "v3"); dir != want {
		t.Errorf("latest = %q, want %q", dir, want)
	}
	dir, err = ResolveTemplate(templatesDir, "python/std/lib", 2)
	if err != nil {
		t.Fatalf("ResolveTemplate v2: %v", err)
	}
	if want := filepath.Join(base, "v2"); dir != want {
		t.Errorf("v2 = %q, want %q", dir, want)
	}
	// unavailable version names what exists
	_, err = ResolveTemplate(templatesDir, "python/std/lib", 4)
	if err == nil || !strings.Contains(err.Error(), "not found") || !strings.Contains(err.Error(), "[1 2 3]") {
		t.Errorf("error = %v, want not-found listing available versions", err)
	}
}

func TestResolveTemplateByPath(t *testing.T) {
	tmp := t.TempDir()
	// a path to a template root resolves its versions
	root := filepath.Join(tmp, "my-template")
	versionDir := filepath.Join(root, "v1")
	if err := os.MkdirAll(filepath.Join(versionDir, TemplateDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	dir, err := ResolveTemplate(".", root, 0)
	if err != nil || dir != versionDir {
		t.Errorf("root path = %q, %v, want %q", dir, err, versionDir)
	}
	// a path to a version directory is used as-is
	dir, err = ResolveTemplate(".", versionDir, 0)
	if err != nil || dir != versionDir {
		t.Errorf("version path = %q, %v, want %q", dir, err, versionDir)
	}
}

func TestResolveTemplateErrors(t *testing.T) {
	tmp := t.TempDir()
	if _, err := ResolveTemplate(tmp, "missing/template", 0); err == nil {
		t.Error("want an error for a missing template")
	}
	// a directory with no v<N> subdirectories
	empty := filepath.Join(tmp, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveTemplate(tmp, "empty", 0)
	if err == nil || !strings.Contains(err.Error(), "no version directories") {
		t.Errorf("error = %v, want no-version-directories", err)
	}
}

func TestResolveTemplateIgnoresCwdShadow(t *testing.T) {
	// a directory in the cwd sharing a template's name must not shadow
	// the registry: only explicit ./ or absolute paths are paths
	templatesDir := t.TempDir()
	versionDir := filepath.Join(templatesDir, "python", "std", "lib", "v1")
	if err := os.MkdirAll(filepath.Join(versionDir, TemplateDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	if err := os.MkdirAll(filepath.Join(work, "python", "std", "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	dir, err := ResolveTemplate(templatesDir, "python/std/lib", 0)
	if err != nil {
		t.Fatalf("ResolveTemplate: %v", err)
	}
	if dir != versionDir {
		t.Errorf("dir = %q, want %q — the cwd directory shadowed the registry", dir, versionDir)
	}
	// an explicit ./ prefix makes the path reading deliberate
	_, err = ResolveTemplate(templatesDir, "./python/std/lib", 0)
	if err == nil || !strings.Contains(err.Error(), "no version directories") {
		t.Errorf("error = %v, want the ./ path read (empty of versions)", err)
	}
}

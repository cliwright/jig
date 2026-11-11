package jig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testManifest(t *testing.T) *Manifest {
	t.Helper()
	dir := t.TempDir()
	skel := filepath.Join(dir, TemplateDirName)
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "my-project", "prompt": "Project name"},
		"author": {"default": "someone"}
	}}`
	if err := os.WriteFile(filepath.Join(skel, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(filepath.Join(skel, ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestResolveValuesDefaults(t *testing.T) {
	values, err := ResolveValues(testManifest(t), "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if values["PROJECT_NAME"] != "my-project" || values["author"] != "someone" {
		t.Fatalf("want defaults, got %v", values)
	}
}

func TestResolveValuesPrecedence(t *testing.T) {
	varsFile := filepath.Join(t.TempDir(), "vars.json")
	if err := os.WriteFile(varsFile, []byte(`{"author": "from-file", "PROJECT_NAME": "from-file"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var prompted map[string]string
	prompt := func(m *Manifest, values map[string]string) error {
		values["author"] = "from-prompt"
		prompted = values
		return nil
	}
	values, err := ResolveValues(testManifest(t), varsFile, []string{"PROJECT_NAME=from-flag"}, prompt)
	if err != nil {
		t.Fatal(err)
	}
	if values["PROJECT_NAME"] != "from-flag" {
		t.Fatalf("set should beat vars-file, got %q", values["PROJECT_NAME"])
	}
	if values["author"] != "from-prompt" {
		t.Fatalf("prompt should beat set, got %q", values["author"])
	}
	if prompted == nil {
		t.Fatal("prompt was never called")
	}
}

func TestResolveValuesRejectsMalformedSet(t *testing.T) {
	_, err := ResolveValues(testManifest(t), "", []string{"no-equals-sign"}, nil)
	if err == nil || !strings.Contains(err.Error(), "want key=value") {
		t.Fatalf("want key=value error, got %v", err)
	}
}

func TestStdinPrompterDefaultsOnEmpty(t *testing.T) {
	in := strings.NewReader("\ncustom-author\n")
	var out strings.Builder
	values, err := ResolveValues(testManifest(t), "", nil, StdinPrompter(in, &out))
	if err != nil {
		t.Fatal(err)
	}
	if values["PROJECT_NAME"] != "my-project" {
		t.Fatalf("empty answer should keep the default, got %q", values["PROJECT_NAME"])
	}
	if values["author"] != "custom-author" {
		t.Fatalf("answer should win over the default, got %q", values["author"])
	}
	if !strings.Contains(out.String(), "Project name [my-project]: ") {
		t.Fatalf("prompt should use the label and current value, got %q", out.String())
	}
}

func TestReadVarsFileRejectsNonObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vars.json")
	if err := os.WriteFile(path, []byte(`["not", "an", "object"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadVarsFile(path); err == nil {
		t.Fatal("want an error for a non-object vars file")
	}
}

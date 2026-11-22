package jig

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func manifestFromJSON(t *testing.T, manifest string) *Manifest {
	t.Helper()
	dir := t.TempDir()
	skel := filepath.Join(dir, TemplateDirName)
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skel, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(filepath.Join(skel, ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func testManifest(t *testing.T) *Manifest {
	t.Helper()
	return manifestFromJSON(t, `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "my-project", "prompt": "Project name"},
		"author": {"default": "someone"}
	}}`)
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
	prompt := func(m *Manifest, values map[string]string, locked map[string]bool) error {
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
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"array", `["not", "an", "object"]`, true},
		{"null", `null`, true},
		{"string value is a number", `{"author": 42}`, true},
		{"empty object", `{}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "vars.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			values, err := ReadVarsFile(path)
			if tt.wantErr && err == nil {
				t.Fatalf("want an error for %s, got values %v", tt.name, values)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("want no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestStdinPrompterShowsChoices(t *testing.T) {
	dir := t.TempDir()
	skel := filepath.Join(dir, TemplateDirName)
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "my-project"},
		"license": {"default": "MIT", "choices": ["MIT", "Apache-2.0"]}
	}}`
	if err := os.WriteFile(filepath.Join(skel, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(filepath.Join(skel, ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if _, err := ResolveValues(m, "", nil, StdinPrompter(strings.NewReader("\n\n"), &out)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "license (MIT | Apache-2.0) [MIT]: ") {
		t.Fatalf("prompt should list the choices, got %q", out.String())
	}
}

// derivedManifest declares a root variable and a dependent deriving from it.
const derivedManifest = `{"version": 1, "variables": {
	"PROJECT_NAME": {"default": "my-tool"},
	"PYTHON_PACKAGE": {"default": "{{ .PROJECT_NAME | replace \"-\" \"_\" }}"}
}}`

// Answering PROJECT_NAME must re-derive PYTHON_PACKAGE's default before it
// is prompted — both the shown default and the final value.
func TestStdinPrompterRederivesDependents(t *testing.T) {
	var out strings.Builder
	values, err := ResolveValues(manifestFromJSON(t, derivedManifest), "", nil,
		StdinPrompter(strings.NewReader("cool-cli\n\n"), &out))
	if err != nil {
		t.Fatal(err)
	}
	if values["PYTHON_PACKAGE"] != "cool_cli" {
		t.Fatalf("dependent should re-derive from the answer, got %q", values["PYTHON_PACKAGE"])
	}
	if !strings.Contains(out.String(), "PYTHON_PACKAGE [cool_cli]: ") {
		t.Fatalf("prompt should show the re-derived default, got %q", out.String())
	}
}

// An explicit --var value is locked: answering its ancestor interactively
// must not re-derive over it.
func TestStdinPrompterKeepsLockedOverride(t *testing.T) {
	var out strings.Builder
	values, err := ResolveValues(manifestFromJSON(t, derivedManifest), "",
		[]string{"PYTHON_PACKAGE=custom"}, StdinPrompter(strings.NewReader("cool-cli\n\n"), &out))
	if err != nil {
		t.Fatal(err)
	}
	if values["PYTHON_PACKAGE"] != "custom" {
		t.Fatalf("locked override should win, got %q", values["PYTHON_PACKAGE"])
	}
}

// An interactive answer for the dependent itself wins over re-derivation.
func TestStdinPrompterAnsweredDependentWins(t *testing.T) {
	values, err := ResolveValues(manifestFromJSON(t, derivedManifest), "", nil,
		StdinPrompter(strings.NewReader("cool-cli\nmy-own-pkg\n"), &strings.Builder{}))
	if err != nil {
		t.Fatal(err)
	}
	if values["PYTHON_PACKAGE"] != "my-own-pkg" {
		t.Fatalf("the explicit answer should win, got %q", values["PYTHON_PACKAGE"])
	}
}

// Prompting runs in dependency order — roots before their dependents, even
// when sorted order would ask the dependent first — and re-derivation is
// transitive through intermediate variables.
func TestStdinPrompterPromptsInDependencyOrder(t *testing.T) {
	m := manifestFromJSON(t, `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "demo"},
		"ZROOT": {"default": "z"},
		"AMID": {"default": "{{ .ZROOT }}-mid"},
		"BLEAF": {"default": "{{ .AMID }}-leaf"}
	}}`)
	var out strings.Builder
	values, err := ResolveValues(m, "", nil, StdinPrompter(strings.NewReader("\nx\n\n\n"), &out))
	if err != nil {
		t.Fatal(err)
	}
	if values["BLEAF"] != "x-mid-leaf" {
		t.Fatalf("re-derivation should be transitive, got %q", values["BLEAF"])
	}
	zroot := strings.Index(out.String(), "ZROOT [")
	amid := strings.Index(out.String(), "AMID [")
	if zroot < 0 || amid < 0 || amid < zroot {
		t.Fatalf("ZROOT should prompt before its dependent AMID, got %q", out.String())
	}
	if !strings.Contains(out.String(), "BLEAF [x-mid-leaf]: ") {
		t.Fatalf("prompt should show the transitively re-derived default, got %q", out.String())
	}
}

// An answer failing its choices is rejected and re-prompted immediately;
// the next valid answer proceeds.
func TestStdinPrompterRepromptsOnInvalidChoice(t *testing.T) {
	m := manifestFromJSON(t, `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "demo"},
		"license": {"default": "MIT", "choices": ["MIT", "Apache-2.0"]}
	}}`)
	var out strings.Builder
	values, err := ResolveValues(m, "", nil, StdinPrompter(strings.NewReader("\nGPL\nApache-2.0\n"), &out))
	if err != nil {
		t.Fatal(err)
	}
	if values["license"] != "Apache-2.0" {
		t.Fatalf("the valid retry should win, got %q", values["license"])
	}
	if !strings.Contains(out.String(), `value "GPL" is not one of its choices`) {
		t.Fatalf("the invalid answer should be rejected inline, got %q", out.String())
	}
	if strings.Count(out.String(), "license (MIT | Apache-2.0)") != 2 {
		t.Fatalf("the variable should be re-prompted after a bad answer, got %q", out.String())
	}
}

// Same for patterns: a mismatch is rejected inline, not at render time.
func TestStdinPrompterRepromptsOnPatternMismatch(t *testing.T) {
	m := manifestFromJSON(t, `{"version": 1, "variables": {
		"PROJECT_NAME": {"default": "demo", "pattern": "^[a-z][a-z0-9-]*$"}
	}}`)
	var out strings.Builder
	values, err := ResolveValues(m, "", nil, StdinPrompter(strings.NewReader("Bad_Name\ngood-name\n"), &out))
	if err != nil {
		t.Fatal(err)
	}
	if values["PROJECT_NAME"] != "good-name" {
		t.Fatalf("the valid retry should win, got %q", values["PROJECT_NAME"])
	}
	if !strings.Contains(out.String(), `value "Bad_Name" does not match pattern`) {
		t.Fatalf("the invalid answer should be rejected inline, got %q", out.String())
	}
}

func TestResolveValuesVarsFileError(t *testing.T) {
	_, err := ResolveValues(testManifest(t), filepath.Join(t.TempDir(), "missing.json"), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "reading") {
		t.Fatalf("error = %v, want a vars-file read error", err)
	}
}

func TestResolveValuesKeepsUndeclaredOverride(t *testing.T) {
	// an override for an undeclared variable is passed through for
	// Validate to report — it must reach the values map
	values, err := ResolveValues(testManifest(t), "", []string{"NOPE=x"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if values["NOPE"] != "x" {
		t.Errorf("undeclared override lost: %v", values)
	}
}

func TestResolveValuesPromptError(t *testing.T) {
	prompt := func(m *Manifest, values map[string]string, locked map[string]bool) error {
		return errors.New("prompt blew up")
	}
	_, err := ResolveValues(testManifest(t), "", nil, prompt)
	if err == nil || !strings.Contains(err.Error(), "prompt blew up") {
		t.Fatalf("error = %v, want the prompt error", err)
	}
}

func TestStdinPrompterEOFFails(t *testing.T) {
	// a closed stdin mid-prompt is an error, not a silent default
	prompt := StdinPrompter(strings.NewReader(""), &bytes.Buffer{})
	values := map[string]string{"PROJECT_NAME": "my-project", "author": "someone"}
	err := prompt(testManifest(t), values, nil)
	if err == nil || !strings.Contains(err.Error(), "reading answer") {
		t.Fatalf("error = %v, want a read error", err)
	}
}

func TestReadVarsFileMissing(t *testing.T) {
	if _, err := ReadVarsFile(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("want an error for a missing vars file")
	}
}

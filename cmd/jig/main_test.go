package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cliwright/jig"
)

// fixtureTemplate writes a minimal renderable template under a fresh
// templates dir and returns the templates dir.
func fixtureTemplate(t *testing.T) string {
	t.Helper()
	templatesDir := t.TempDir()
	skel := filepath.Join(templatesDir, "python", "std", "lib", "v1", jig.TemplateDirName)
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version": 1, "variables": {"PROJECT_NAME": {"default": "demo"}}}`
	if err := os.WriteFile(filepath.Join(skel, "jig.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skel, "file.txt"), []byte("{{.PROJECT_NAME}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return templatesDir
}

// run executes cmd with args, capturing output.
func run(t *testing.T, cmdArgs []string, stdin *os.File) (string, error) {
	t.Helper()
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	if stdin != nil {
		root.SetIn(stdin)
	}
	root.SetArgs(cmdArgs)
	err := root.Execute()
	return out.String(), err
}

// pipeStdin returns a non-TTY stdin (a pipe, not a character device), as
// an agent or CI pipeline would present.
func pipeStdin(t *testing.T) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

// devNullStdin returns os.DevNull as stdin — a character device but not a
// terminal, which is what some CI runners, cron jobs and agent shells
// present.
func devNullStdin(t *testing.T) *os.File {
	t.Helper()
	f, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

// /dev/null is a character device but not a terminal: it must count as
// non-interactive — failing fast without values, rendering with --var —
// instead of prompting into an EOF error.
func TestRenderCmdDevNullStdinIsNonInteractive(t *testing.T) {
	templatesDir := fixtureTemplate(t)
	_, err := run(t, []string{"render", "--template", "python/std/lib",
		"--templates-dir", templatesDir, "--output", t.TempDir()}, devNullStdin(t))
	if err == nil || !strings.Contains(err.Error(), "not interactive") {
		t.Fatalf("error = %v, want non-interactive refusal", err)
	}
	out := t.TempDir()
	_, err = run(t, []string{"render", "--template", "python/std/lib",
		"--templates-dir", templatesDir, "--output", out,
		"--var", "PROJECT_NAME=widget"}, devNullStdin(t))
	if err != nil {
		t.Fatalf("render with --var on /dev/null stdin: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "widget", "file.txt")); err != nil {
		t.Errorf("rendered file missing: %v", err)
	}
}

func TestRenderCmdRefusesNonInteractiveStdin(t *testing.T) {
	templatesDir := fixtureTemplate(t)
	_, err := run(t, []string{"render", "--template", "python/std/lib",
		"--templates-dir", templatesDir, "--output", t.TempDir()}, pipeStdin(t))
	if err == nil || !strings.Contains(err.Error(), "not interactive") {
		t.Fatalf("error = %v, want non-interactive refusal", err)
	}
}

func TestRenderCmdVarWorksNonInteractively(t *testing.T) {
	templatesDir := fixtureTemplate(t)
	out := t.TempDir()
	_, err := run(t, []string{"render", "--template", "python/std/lib",
		"--templates-dir", templatesDir, "--output", out,
		"--var", "PROJECT_NAME=widget"}, pipeStdin(t))
	if err != nil {
		t.Fatalf("render with --var: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(out, "widget", "file.txt"))
	if err != nil || string(data) != "widget\n" {
		t.Errorf("file.txt = %q, %v", data, err)
	}
}

func TestRenderCmdVarsFile(t *testing.T) {
	templatesDir := fixtureTemplate(t)
	out := t.TempDir()
	varsFile := filepath.Join(t.TempDir(), "vars.json")
	if err := os.WriteFile(varsFile, []byte(`{"PROJECT_NAME": "from-file"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, []string{"render", "--template", "python/std/lib",
		"--templates-dir", templatesDir, "--output", out,
		"--vars-file", varsFile}, pipeStdin(t))
	if err != nil {
		t.Fatalf("render with --vars-file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "from-file", "file.txt")); err != nil {
		t.Errorf("rendered file missing: %v", err)
	}
}

func TestRenderCmdPromptsWhenInteractive(t *testing.T) {
	templatesDir := fixtureTemplate(t)
	out := t.TempDir()
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	// a non-*os.File reader is treated as interactive and prompts
	root.SetIn(strings.NewReader("typed-name\n"))
	root.SetArgs([]string{"render", "--template", "python/std/lib",
		"--templates-dir", templatesDir, "--output", out})
	if err := root.Execute(); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "PROJECT_NAME [demo]:") {
		t.Errorf("expected a prompt, got %q", buf.String())
	}
	if _, err := os.Stat(filepath.Join(out, "typed-name", "file.txt")); err != nil {
		t.Errorf("prompt answer was not used: %v", err)
	}
}

func TestRenderCmdRequiresFlags(t *testing.T) {
	templatesDir := fixtureTemplate(t)
	if _, err := run(t, []string{"render", "--templates-dir", templatesDir, "--output", t.TempDir()}, nil); err == nil ||
		!strings.Contains(err.Error(), "--template is required") {
		t.Errorf("error = %v, want --template refusal", err)
	}
	if _, err := run(t, []string{"render", "--template", "python/std/lib", "--templates-dir", templatesDir}, nil); err == nil ||
		!strings.Contains(err.Error(), "--output is required") {
		t.Errorf("error = %v, want --output refusal", err)
	}
}

func TestCheckCmd(t *testing.T) {
	templatesDir := fixtureTemplate(t)
	out, err := run(t, []string{"check", "--template", "python/std/lib", "--templates-dir", templatesDir}, nil)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("output = %q, want ok", out)
	}
	// break the template: check must fail
	skel := filepath.Join(templatesDir, "python", "std", "lib", "v1", jig.TemplateDirName)
	if err := os.WriteFile(filepath.Join(skel, "bad.txt"), []byte("{{ .nope }}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, []string{"check", "--template", "python/std/lib", "--templates-dir", templatesDir}, nil); err == nil {
		t.Error("want check to fail on a broken template")
	}
}

func TestBumpCmd(t *testing.T) {
	templatesDir := fixtureTemplate(t)
	out, err := run(t, []string{"bump", "python/std/lib", "--templates-dir", templatesDir}, nil)
	if err != nil {
		t.Fatalf("bump: %v", err)
	}
	if !strings.Contains(out, "v2") {
		t.Errorf("output = %q, want the new version dir", out)
	}
	if _, err := os.Stat(filepath.Join(templatesDir, "python", "std", "lib", "v2", jig.TemplateDirName, "jig.json")); err != nil {
		t.Errorf("v2 manifest missing: %v", err)
	}
}

func TestDocsCmd(t *testing.T) {
	out, err := run(t, []string{"docs"}, nil)
	if err != nil {
		t.Fatalf("docs: %v", err)
	}
	if !strings.Contains(out, "cli") || !strings.Contains(out, "manifest") {
		t.Errorf("page list = %q", out)
	}
	out, err = run(t, []string{"docs", "manifest"}, nil)
	if err != nil || !strings.Contains(out, "jig.json") {
		t.Errorf("docs manifest = %v, %.40q", err, out)
	}
	if _, err := run(t, []string{"docs", "bogus"}, nil); err == nil {
		t.Error("want an error for a bogus page")
	}
}

func TestAgentsCmd(t *testing.T) {
	out, err := run(t, []string{"agents"}, nil)
	if err != nil {
		t.Fatalf("agents: %v", err)
	}
	if !strings.Contains(out, "jig.cliwright.dev") {
		t.Errorf("agents output should point at the docs, got %.40q", out)
	}
}

func TestRootHelpPointsAgentsAtAgentsCmd(t *testing.T) {
	out, err := run(t, []string{"--help"}, nil)
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	if !strings.Contains(out, "jig agents") {
		t.Errorf("root help should point agents at jig agents, got %q", out)
	}
}

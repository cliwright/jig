package jig

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// ManifestFileName is the template manifest's conventional file name,
// excluded from rendered output.
const ManifestFileName = "jig.json"

// TemplateDirName is the literal directory name every template uses for its
// top-level project directory: <template>/v<N>/{{.PROJECT_NAME}}/jig.json.
const TemplateDirName = "{{.PROJECT_NAME}}"

// Options controls one Render call.
type Options struct {
	// Force allows rendering into a non-empty output directory. The default
	// refuses, so an accidental render never clobbers existing work.
	Force bool
}

// Result reports what one Render did.
type Result struct {
	// OutputDir is the rendered project directory (outputDir/<rendered name>).
	OutputDir string
	// Unused names manifest variables no template file referenced — a cruft
	// warning for template authors, never an error.
	Unused []string
}

// ResolveTemplate locates the version directory of a template. name may be
// a registry-style name (<language>/<flavor>/<kind>, resolved under
// templatesDir), a path to a template root (holding v<N> directories), or a
// path to a version directory itself (holding {{.PROJECT_NAME}}/, used
// as-is). version <= 0 selects the latest (highest N).
func ResolveTemplate(templatesDir, name string, version int) (string, error) {
	base := name
	if info, err := os.Stat(name); err != nil || !info.IsDir() {
		base = filepath.Join(templatesDir, filepath.FromSlash(name))
	}
	if info, err := os.Stat(filepath.Join(base, TemplateDirName)); err == nil && info.IsDir() {
		return base, nil // already a version directory
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("template %q: %w", name, err)
	}
	var versions []int
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "v") {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(e.Name(), "v")); err == nil && n >= 1 {
			versions = append(versions, n)
		}
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("template %q: no version directories (v<N>) under %s", name, base)
	}
	sort.Ints(versions)
	if version <= 0 {
		version = versions[len(versions)-1]
	}
	dir := filepath.Join(base, "v"+strconv.Itoa(version))
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("template %q: version %d not found (available: %v)", name, version, versions)
	}
	return dir, nil
}

// Render renders the template in versionDir into outputDir. The version
// directory holds exactly one project skeleton under {{.PROJECT_NAME}}/
// whose jig.json declares the variables; the rendered skeleton lands at
// outputDir/<rendered name>. values must satisfy the manifest (see
// Manifest.Validate): rendering is hard-fail on undeclared variables
// (missingkey=error) and on pattern mismatches.
func Render(versionDir, outputDir string, values map[string]string, opts Options) (*Result, error) {
	skel := filepath.Join(versionDir, TemplateDirName)
	m, err := LoadManifest(filepath.Join(skel, ManifestFileName))
	if err != nil {
		return nil, err
	}
	if err := m.Validate(values); err != nil {
		return nil, err
	}

	used := map[string]bool{}
	topName, err := renderString(TemplateDirName, values, used)
	if err != nil {
		return nil, fmt.Errorf("rendering the project directory name: %w", err)
	}
	dest := filepath.Join(outputDir, topName)
	if err := checkWritable(dest, opts.Force); err != nil {
		return nil, err
	}

	err = filepath.WalkDir(skel, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(skel, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			out, err := renderString(rel, values, used)
			if err != nil {
				return fmt.Errorf("rendering directory name %s: %w", rel, err)
			}
			return os.MkdirAll(filepath.Join(dest, out), 0o755)
		}
		if rel == ManifestFileName {
			return nil // the manifest never renders into the output
		}
		out, err := renderString(rel, values, used)
		if err != nil {
			return fmt.Errorf("rendering file name %s: %w", rel, err)
		}
		return renderFile(path, filepath.Join(dest, out), values, used, d)
	})
	if err != nil {
		return nil, err
	}
	return &Result{OutputDir: dest, Unused: m.Unused(used)}, nil
}

// renderString renders one name or path fragment as a template. The
// template's option is missingkey=error: referencing an undeclared variable
// is a hard error, matching smith's type rendering discipline. Referenced
// variable names are recorded in used.
func renderString(s string, values map[string]string, used map[string]bool) (string, error) {
	t, err := template.New("name").Funcs(FuncMap).Option("missingkey=error").Parse(s)
	if err != nil {
		return "", err
	}
	collectUsed(t.Tree.Root, used)
	var buf bytes.Buffer
	if err := t.Execute(&buf, values); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderFile writes one rendered file: binaries copy verbatim (detected by
// content, not extension), text renders through text/template. The source
// file's mode bits are preserved so templates can ship executables.
func renderFile(src, dst string, values map[string]string, used map[string]bool, d fs.DirEntry) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if !isBinary(data) {
		t, err := template.New(filepath.Base(src)).Funcs(FuncMap).Option("missingkey=error").Parse(string(data))
		if err != nil {
			return fmt.Errorf("parsing %s: %w", src, err)
		}
		collectUsed(t.Tree.Root, used)
		var buf bytes.Buffer
		if err := t.Execute(&buf, values); err != nil {
			return fmt.Errorf("rendering %s: %w", src, err)
		}
		data = buf.Bytes()
	}
	info, err := d.Info()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

// isBinary reports whether the content looks binary: a NUL byte in the
// first 512 bytes (the heuristic git and net/http use).
func isBinary(data []byte) bool {
	const sniffLen = 512
	if len(data) > sniffLen {
		data = data[:sniffLen]
	}
	return bytes.IndexByte(data, 0) >= 0
}

// checkWritable refuses to render into an existing non-empty directory
// unless forced.
func checkWritable(dest string, force bool) error {
	entries, err := os.ReadDir(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("output directory %s is not empty (use --force to render anyway)", dest)
	}
	return nil
}

package jig

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
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
	// Force allows overwriting existing files. The default refuses, so an
	// accidental render never clobbers existing work.
	Force bool
	// Trust allows symlinks in the template: they are recreated as symlinks
	// in the output. The default rejects them — a registry-distributed
	// template is data, and links can point anywhere. Even under Trust,
	// links resolving outside the template tree are refused.
	Trust bool
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
//
// Rendering plans the whole tree first and refuses to overwrite any
// existing file (unless Force), so a failed render never half-writes and a
// render into a populated directory — e.g. another version of a type tree —
// is fine as long as no file collides.
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
	dest, err := containedPath(outputDir, topName)
	if err != nil {
		return nil, fmt.Errorf("rendering the project directory name: %w", err)
	}

	plan, err := planRender(skel, dest, values, used, m.Verbatim, opts.Trust)
	if err != nil {
		return nil, err
	}
	if !opts.Force {
		for _, f := range plan.files {
			if _, err := os.Stat(f.dst); err == nil {
				return nil, fmt.Errorf("rendering would overwrite %s (use --force to allow)", f.dst)
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("checking %s: %w", f.dst, err)
			}
		}
	}

	for _, d := range plan.dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("creating %s: %w", d, err)
		}
	}
	for _, f := range plan.files {
		if f.linkTarget != "" {
			if err := os.MkdirAll(filepath.Dir(f.dst), 0o755); err != nil {
				return nil, err
			}
			if err := os.Symlink(f.linkTarget, f.dst); err != nil {
				return nil, fmt.Errorf("linking %s: %w", f.dst, err)
			}
			continue
		}
		if f.verbatim {
			if err := copyFile(f.src, f.dst, f.entry); err != nil {
				return nil, err
			}
			continue
		}
		if err := renderFile(f.src, f.dst, values, used, f.entry); err != nil {
			return nil, err
		}
	}
	return &Result{OutputDir: dest, Unused: m.Unused(used)}, nil
}

// renderPlan is the walked, name-rendered tree: directories to create and
// files to render, in walk order.
type renderPlan struct {
	dirs  []string
	files []plannedFile
}

type plannedFile struct {
	src, dst   string
	entry      fs.DirEntry
	verbatim   bool
	linkTarget string // non-empty: recreate as a symlink with this target
}

// planRender walks skel and renders every relative name, returning the work
// without touching the filesystem. Entries beneath a manifest-declared
// verbatim directory keep their own names unrendered, while their ancestor
// directories render normally. WalkDir visits parents before children, so
// each entry's output path is its parent's rendered path plus its own name.
// Symlinks are rejected unless trust is set; trusted links must resolve
// inside the template tree and are recreated as links in the output.
func planRender(skel, dest string, values map[string]string, used map[string]bool, verbatim []string, trust bool) (*renderPlan, error) {
	plan := &renderPlan{}
	renderedDirs := map[string]string{"": dest, ".": dest}
	err := filepath.WalkDir(skel, func(path string, d fs.DirEntry, err error) error {
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
		if rel == ManifestFileName {
			return nil // the template's own manifest never renders into the output
		}
		relSlash := filepath.ToSlash(rel)
		parentOut := renderedDirs[pathpkg.Dir(relSlash)]
		name := filepath.Base(rel)
		kind := "file"
		if d.IsDir() {
			kind = "directory"
		} else if d.Type()&fs.ModeSymlink != 0 {
			kind = "symlink"
		}
		if !isVerbatim(relSlash, verbatim) {
			name, err = renderString(name, values, used)
			if err != nil {
				return fmt.Errorf("rendering %s name %s: %w", kind, rel, err)
			}
		}
		out, err := containedPath(parentOut, name)
		if err != nil {
			return fmt.Errorf("rendering %s name %s: %w", kind, rel, err)
		}
		if d.IsDir() {
			renderedDirs[relSlash] = out
			plan.dirs = append(plan.dirs, out)
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := checkSymlink(skel, path, trust)
			if err != nil {
				return err
			}
			plan.files = append(plan.files, plannedFile{src: path, dst: out, entry: d, linkTarget: linkTarget})
			return nil
		}
		plan.files = append(plan.files, plannedFile{src: path, dst: out, entry: d, verbatim: isVerbatim(relSlash, verbatim)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// checkSymlink vets a symlink found in the template. Links are rejected
// unless trust is set; a trusted link must resolve inside the template tree,
// so even a trusted template cannot smuggle in a pointer to the host
// filesystem. Returns the link target to recreate in the output.
func checkSymlink(skel, path string, trust bool) (string, error) {
	if !trust {
		return "", fmt.Errorf("template contains symlink %s: templates must not contain symlinks (use --trust to allow links inside the template tree)", path)
	}
	target, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("reading symlink %s: %w", path, err)
	}
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(path), resolved)
	}
	root, err := filepath.Abs(skel)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("symlink %s points outside the template tree (%s)", path, target)
	}
	return target, nil
}

// containedPath joins base and a rendered name, refusing results that escape
// base: rendered values come from user input, so jig enforces containment
// itself rather than trusting every manifest's validation pattern. Interior
// slashes are fine (a name may render to a nested path); empty names, ".."
// segments, and absolute results are not.
func containedPath(base, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("rendered an empty name")
	}
	out := filepath.Join(base, name)
	rel, err := filepath.Rel(base, out)
	if err != nil {
		return "", fmt.Errorf("rendered name %q: %w", name, err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("rendered name %q escapes the output directory", name)
	}
	return out, nil
}

// isVerbatim reports whether rel (slash form) lives beneath a directory
// whose base name is listed in verbatim. The matched directory's own path
// still renders — only its contents ship unrendered.
func isVerbatim(rel string, verbatim []string) bool {
	segments := strings.Split(rel, "/")
	for _, segment := range segments[:len(segments)-1] {
		if slices.Contains(verbatim, segment) {
			return true
		}
	}
	return false
}

// copyFile writes src to dst unchanged, preserving the source's mode bits.
func copyFile(src, dst string, d fs.DirEntry) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
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

// renderContent renders one file's content in memory: binaries pass through
// unchanged (detected by content, not extension), text renders through
// text/template.
func renderContent(src string, values map[string]string, used map[string]bool) ([]byte, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	if isBinary(data) {
		return data, nil
	}
	t, err := template.New(filepath.Base(src)).Funcs(FuncMap).Option("missingkey=error").Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", src, err)
	}
	collectUsed(t.Tree.Root, used)
	var buf bytes.Buffer
	if err := t.Execute(&buf, values); err != nil {
		return nil, fmt.Errorf("rendering %s: %w", src, err)
	}
	return buf.Bytes(), nil
}

// renderFile writes one rendered file. The source file's mode bits are
// preserved so templates can ship executables.
func renderFile(src, dst string, values map[string]string, used map[string]bool, d fs.DirEntry) error {
	data, err := renderContent(src, values, used)
	if err != nil {
		return err
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

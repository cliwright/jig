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
	// only relative links resolving inside the template tree are allowed.
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
// Rendering is fully prepared before anything is written: every name and
// every file's content is rendered in memory, destinations are checked for
// internal collisions, containment, symlinked components, and (unless
// Force) existing files. Only when all of that passes does the write phase
// create directories and files — a template error never leaves a partial
// tree. (Disk failures mid-write and concurrent filesystem changes remain
// outside the guarantee.)
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
	// Lexical containment is not filesystem containment: a symlink already
	// sitting in the destination tree would silently redirect writes outside
	// it. Refuse to write through any symlinked component, Force or not —
	// Force means "overwrite regular files", never "follow someone's link".
	for _, path := range append([]string{dest}, plan.dirs...) {
		if err := checkDestChain(outputDir, path); err != nil {
			return nil, err
		}
	}
	for _, f := range plan.files {
		if err := checkDestChain(outputDir, f.dst); err != nil {
			return nil, err
		}
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
		if err := writeFile(f.dst, f.data, f.mode); err != nil {
			return nil, err
		}
	}
	return &Result{OutputDir: dest, Unused: m.Unused(used)}, nil
}

// renderPlan is the fully prepared render: directories to create and files
// to write, contents already rendered in memory, in walk order.
type renderPlan struct {
	dirs  []string
	files []plannedFile
}

type plannedFile struct {
	dst        string
	data       []byte
	mode       fs.FileMode
	linkTarget string // non-empty: recreate as a symlink with this target
}

// planRender walks skel and prepares everything the write phase needs
// without touching the destination filesystem: names are rendered, file
// contents are rendered (or read, for verbatim and binary files), and
// planned destinations are checked against each other — two sources
// rendering to the same file, or a file and directory sharing a path, fail
// here rather than overwriting each other mid-write. Distinct directories
// may merge into one rendered directory; conflicting files inside it are
// still caught. Entries beneath a manifest-declared verbatim directory
// keep their own names and contents unrendered, while their ancestor
// directories render normally. WalkDir visits parents before children, so
// each entry's output path is its parent's rendered path plus its own
// name. Symlinks are rejected unless trust is set; trusted links must
// resolve inside the template tree and are recreated as links in the
// output.
func planRender(skel, dest string, values map[string]string, used map[string]bool, verbatim []string, trust bool) (*renderPlan, error) {
	plan := &renderPlan{}
	renderedDirs := map[string]string{"": dest, ".": dest}
	type dstKind struct {
		kind, rel string
	}
	planned := map[string]dstKind{}
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
			if prev, ok := planned[out]; ok && prev.kind != "directory" {
				return fmt.Errorf("collision: file %s and directory %s both render to %s", prev.rel, rel, out)
			}
			planned[out] = dstKind{kind: "directory", rel: rel}
			renderedDirs[relSlash] = out
			plan.dirs = append(plan.dirs, out)
			return nil
		}
		if prev, ok := planned[out]; ok {
			return fmt.Errorf("collision: %s %s and %s %s both render to %s", prev.kind, prev.rel, kind, rel, out)
		}
		planned[out] = dstKind{kind: kind, rel: rel}
		if d.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := checkSymlink(skel, path, trust)
			if err != nil {
				return err
			}
			plan.files = append(plan.files, plannedFile{dst: out, linkTarget: linkTarget})
			return nil
		}
		if !d.Type().IsRegular() {
			// FIFOs, devices, sockets: copying one would read from a special
			// file — a FIFO read blocks forever. Templates are data.
			return fmt.Errorf("template contains non-regular file %s: only directories and regular files are allowed", rel)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		var data []byte
		if isVerbatim(relSlash, verbatim) {
			if data, err = os.ReadFile(path); err != nil {
				return err
			}
		} else if data, err = renderContent(path, values, used); err != nil {
			return err
		}
		plan.files = append(plan.files, plannedFile{dst: out, data: data, mode: info.Mode().Perm()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// checkSymlink vets a symlink found in the template. Links are rejected
// unless trust is set. A trusted link must be relative and resolve inside
// the template tree: an absolute link would recreate as a pointer back to
// the author's machine, and an escaping link would smuggle in a pointer to
// the host filesystem. Returns the link target to recreate in the output.
func checkSymlink(skel, path string, trust bool) (string, error) {
	if !trust {
		return "", fmt.Errorf("template contains symlink %s: templates must not contain symlinks (use --trust to allow links inside the template tree)", path)
	}
	target, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("reading symlink %s: %w", path, err)
	}
	if filepath.IsAbs(target) {
		return "", fmt.Errorf("symlink %s has absolute target %s: only relative links inside the template tree are allowed, even with --trust", path, target)
	}
	resolved := filepath.Join(filepath.Dir(path), target)
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

// checkDestChain walks the existing components of dst (relative to root,
// the output directory) with Lstat and refuses if any is a symlink.
// containedPath keeps rendered paths lexically inside the output directory;
// this keeps the actual filesystem writes inside it. Checking stops at the
// first component that does not exist — everything below it will be created
// by the render itself. A symlink as root itself is the caller's explicit
// choice and allowed.
func checkDestChain(root, dst string) error {
	rel, err := filepath.Rel(root, dst)
	if err != nil {
		return err
	}
	cur := root
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		cur = filepath.Join(cur, seg)
		info, err := os.Lstat(cur)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("checking %s: %w", cur, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("destination %s contains symlink %s: refusing to write through it", dst, cur)
		}
	}
	return nil
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

// writeFile writes one prepared file with exactly the template's mode
// bits: WriteFile's perm only applies at creation and is masked by umask,
// so the Chmod makes the mode exact — templates can ship executables, and
// --force restores the template's mode rather than keeping the old file's.
func writeFile(dst string, data []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}

// renderString renders one name or path fragment as a template. The
// template's option is missingkey=error: referencing an undeclared variable
// is a hard error, matching smith's type rendering discipline. Referenced
// variable names are recorded in used.
func renderString(s string, values map[string]string, used map[string]bool) (string, error) {
	t, err := template.New("name").Funcs(funcMap).Option("missingkey=error").Parse(s)
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
	t, err := template.New(filepath.Base(src)).Funcs(funcMap).Option("missingkey=error").Parse(string(data))
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

// isBinary reports whether the content looks binary: a NUL byte in the
// first 512 bytes (the heuristic git and net/http use).
func isBinary(data []byte) bool {
	const sniffLen = 512
	if len(data) > sniffLen {
		data = data[:sniffLen]
	}
	return bytes.IndexByte(data, 0) >= 0
}

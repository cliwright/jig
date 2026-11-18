// Package jig renders project templates: inert directories of text/template
// files plus a jig.json manifest declaring the variables they prompt for.
// Templates never execute code — rendering walks the tree, renders file and
// directory names and text contents, and copies binaries verbatim.
package jig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/cliwright/jig/schemas"
)

// ReservedProjectName is the one variable every manifest must declare: the
// template's top-level directory (and the rendered output directory) is
// named by a template expression over it.
const ReservedProjectName = "PROJECT_NAME"

// Var is one declared template variable: its default (required), optional
// prompt text for interactive mode, and an optional validation the final
// value must pass — either a pattern or a fixed set of choices, never both.
// The default may template over other variables ("{{ .PROJECT_NAME |
// slugify }}"); see Manifest.parseDefaults.
type Var struct {
	Default string   `json:"default"`
	Prompt  string   `json:"prompt,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
	Choices []string `json:"choices,omitempty"`
}

// Manifest is the typed view of jig.json.
type Manifest struct {
	Version   int            `json:"version"`
	Variables map[string]Var `json:"variables"`
	// Verbatim lists directory base names whose subtrees copy without
	// rendering — names and contents ship as-is, wherever they sit in the
	// skeleton. For templates that ship templates (e.g. a type's embedded
	// template/ dir).
	Verbatim []string `json:"verbatim,omitempty"`

	patterns   map[string]*regexp.Regexp     // compiled from Var.Pattern at load
	order      []string                      // variables in dependency-resolved evaluation order
	defaults   map[string]*template.Template // parsed Var.Default per variable
	referenced map[string]bool               // variables another variable's default derives from
	children   map[string][]string           // variable → the variables whose defaults derive from it
}

// LoadManifest reads path, validates it against the embedded manifest
// schema, and enforces the semantic rules the schema cannot: PROJECT_NAME
// must be declared, and every validation pattern must compile (RE2).
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	schema, err := manifestSchema()
	if err != nil {
		return nil, fmt.Errorf("compiling embedded manifest schema: %w", err)
	}
	if err := schema.Validate(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	if _, ok := m.Variables[ReservedProjectName]; !ok {
		return nil, fmt.Errorf("%s: manifest must declare the reserved variable %s", path, ReservedProjectName)
	}
	m.patterns = make(map[string]*regexp.Regexp, len(m.Variables))
	for name, v := range m.Variables {
		if v.Pattern != "" && len(v.Choices) > 0 {
			return nil, fmt.Errorf("%s: variable %s: pattern and choices are mutually exclusive", path, name)
		}
		if len(v.Choices) > 0 && !slices.Contains(v.Choices, v.Default) && !strings.Contains(v.Default, "{{") {
			return nil, fmt.Errorf("%s: variable %s: default %q is not one of its choices %v", path, name, v.Default, v.Choices)
		}
		if v.Pattern == "" {
			continue
		}
		re, err := regexp.Compile(v.Pattern)
		if err != nil {
			return nil, fmt.Errorf("%s: variable %s: invalid pattern: %w", path, name, err)
		}
		m.patterns[name] = re
	}
	for _, v := range m.Verbatim {
		if !fs.ValidPath(v) || v == "." {
			return nil, fmt.Errorf("%s: verbatim entry %q must be a directory base name (e.g. \"template\") or a clean slash-separated relative path (e.g. \".github/workflows/ci.yml\")", path, v)
		}
	}
	if err := m.parseDefaults(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &m, nil
}

// parseDefaults compiles every variable's Default as a template and derives
// the evaluation order: a default may reference other variables
// ("{{ .PROJECT_NAME | slugify }}"), so defaults resolve in dependency
// order. References to undeclared variables, self-references, and cycles
// are load errors — a broken manifest never reaches a prompt.
func (m *Manifest) parseDefaults() error {
	m.defaults = make(map[string]*template.Template, len(m.Variables))
	m.referenced = map[string]bool{}
	m.children = map[string][]string{}
	indegree := map[string]int{}
	for _, name := range m.Names() {
		t, err := template.New("default:" + name).Funcs(funcMap).Option("missingkey=error").Parse(m.Variables[name].Default)
		if err != nil {
			return fmt.Errorf("variable %s: invalid default: %w", name, err)
		}
		m.defaults[name] = t
		deps := map[string]bool{}
		collectUsedAll(t, deps)
		for dep := range deps {
			if dep == name {
				return fmt.Errorf("variable %s: default references itself", name)
			}
			if _, ok := m.Variables[dep]; !ok {
				return fmt.Errorf("variable %s: default references undeclared variable %s", name, dep)
			}
			m.children[dep] = append(m.children[dep], name)
			indegree[name]++
			m.referenced[dep] = true
		}
	}
	var ready []string
	for _, name := range m.Names() {
		if indegree[name] == 0 {
			ready = append(ready, name)
		}
	}
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		m.order = append(m.order, name)
		for _, dep := range m.children[name] {
			if indegree[dep]--; indegree[dep] == 0 {
				ready = append(ready, dep)
			}
		}
		sort.Strings(ready)
	}
	if len(m.order) != len(m.Variables) {
		var cyclic []string
		for name, n := range indegree {
			if n > 0 {
				cyclic = append(cyclic, name)
			}
		}
		sort.Strings(cyclic)
		return fmt.Errorf("cyclic defaults involving %v", cyclic)
	}
	return nil
}

// Validate checks values against the manifest: every supplied variable must
// be declared, and every value must match its validation pattern or choices
// (when the variable declares one). Names are checked in sorted order so
// the first error is deterministic.
func (m *Manifest) Validate(values map[string]string) error {
	for _, name := range sortedKeys(values) {
		if _, ok := m.Variables[name]; !ok {
			return fmt.Errorf("undeclared variable %q (declared: %v)", name, m.Names())
		}
	}
	for _, name := range sortedKeys(m.Variables) {
		if err := m.validateVar(name, values[name]); err != nil {
			return err
		}
	}
	return nil
}

// validateVar checks one value against its variable's declared pattern or
// choices. Validate calls it for every variable; the interactive prompter
// calls it per answer, so a bad answer is rejected — and re-prompted —
// immediately instead of failing the whole render at the end.
func (m *Manifest) validateVar(name, value string) error {
	v := m.Variables[name]
	if re := m.patterns[name]; re != nil && !re.MatchString(value) {
		return fmt.Errorf("variable %s: value %q does not match pattern %q", name, value, v.Pattern)
	}
	if len(v.Choices) > 0 && !slices.Contains(v.Choices, value) {
		return fmt.Errorf("variable %s: value %q is not one of its choices %v", name, value, v.Choices)
	}
	return nil
}

// Names returns the declared variable names in sorted order.
func (m *Manifest) Names() []string {
	return sortedKeys(m.Variables)
}

// Unused reports declared variables that never appear in any rendered
// content — a cruft warning for template authors. used is the set of
// variable names collected during a dry render; variables referenced by
// another variable's default count as used.
func (m *Manifest) Unused(used map[string]bool) []string {
	var out []string
	for name := range m.Variables {
		if !used[name] && !m.referenced[name] {
			out = append(out, name)
		}
	}
	return out
}

// Defaults returns every variable's resolved default value.
func (m *Manifest) Defaults() (map[string]string, error) {
	return m.resolveDefaults(nil)
}

// resolveDefaults evaluates defaults in dependency order. A variable with an
// explicit override takes the override; its dependents then derive from the
// overridden value, so --var PROJECT_NAME=x re-derives everything built on
// PROJECT_NAME. Overrides for undeclared variables are ignored here —
// Validate reports them.
func (m *Manifest) resolveDefaults(overrides map[string]string) (map[string]string, error) {
	values := make(map[string]string, len(m.Variables))
	for _, name := range m.order {
		if v, ok := overrides[name]; ok {
			values[name] = v
			continue
		}
		var buf bytes.Buffer
		if err := m.defaults[name].Execute(&buf, values); err != nil {
			return nil, fmt.Errorf("variable %s: resolving default: %w", name, err)
		}
		values[name] = buf.String()
	}
	return values, nil
}

// rederive re-evaluates the defaults of everything transitively derived
// from changed, in dependency order, against the current values. Locked
// variables — supplied explicitly via --vars-file/--var, or already
// answered in the current prompt run — keep their values. Interactive
// prompting uses this to keep displayed derived defaults in step with the
// answers given so far.
func (m *Manifest) rederive(values map[string]string, changed string, locked map[string]bool) error {
	stale := map[string]bool{}
	queue := []string{changed}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		for _, child := range m.children[name] {
			if !stale[child] {
				stale[child] = true
				queue = append(queue, child)
			}
		}
	}
	for _, name := range m.order {
		if !stale[name] || locked[name] {
			continue
		}
		var buf bytes.Buffer
		if err := m.defaults[name].Execute(&buf, values); err != nil {
			return fmt.Errorf("variable %s: resolving default: %w", name, err)
		}
		values[name] = buf.String()
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var (
	schemaOnce sync.Once
	schemaComp *jsonschema.Schema
	schemaErr  error
)

// manifestSchema compiles the embedded manifest schema once, mirroring
// smith's validate package: every manifest is checked against the schema
// shipped in the binary.
func manifestSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		data, err := schemas.FS.ReadFile("jig-manifest.schema.json")
		if err != nil {
			schemaErr = err
			return
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			schemaErr = err
			return
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource("jig-manifest.schema.json", doc); err != nil {
			schemaErr = err
			return
		}
		schemaComp, schemaErr = c.Compile("jig-manifest.schema.json")
	})
	return schemaComp, schemaErr
}

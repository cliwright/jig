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
	"sort"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/cliwright/jig/schemas"
)

// ReservedProjectName is the one variable every manifest must declare: the
// template's top-level directory (and the rendered output directory) is
// named by a template expression over it.
const ReservedProjectName = "PROJECT_NAME"

// Var is one declared template variable: its default (required), optional
// prompt text for interactive mode, and an optional validation pattern the
// final value must match.
type Var struct {
	Default string `json:"default"`
	Prompt  string `json:"prompt,omitempty"`
	Pattern string `json:"pattern,omitempty"`
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

	patterns map[string]*regexp.Regexp // compiled from Var.Pattern at load
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
			return nil, fmt.Errorf("%s: verbatim path %q must be a relative slash path inside the template", path, v)
		}
	}
	return &m, nil
}

// Validate checks values against the manifest: every supplied variable must
// be declared, and every value must match its validation pattern (when the
// variable declares one). Names are checked in sorted order so the first
// error is deterministic.
func (m *Manifest) Validate(values map[string]string) error {
	for _, name := range sortedKeys(values) {
		if _, ok := m.Variables[name]; !ok {
			return fmt.Errorf("undeclared variable %q (declared: %v)", name, m.Names())
		}
	}
	for name, re := range m.patterns {
		if !re.MatchString(values[name]) {
			return fmt.Errorf("variable %s: value %q does not match pattern %q", name, values[name], m.Variables[name].Pattern)
		}
	}
	return nil
}

// Names returns the declared variable names in sorted order.
func (m *Manifest) Names() []string {
	return sortedKeys(m.Variables)
}

// Unused reports declared variables that never appear in any rendered
// content — a cruft warning for template authors. used is the set of
// variable names collected during a dry render.
func (m *Manifest) Unused(used map[string]bool) []string {
	var out []string
	for name := range m.Variables {
		if !used[name] {
			out = append(out, name)
		}
	}
	return out
}

// Defaults returns every variable's default value.
func (m *Manifest) Defaults() map[string]string {
	out := make(map[string]string, len(m.Variables))
	for name, v := range m.Variables {
		out[name] = v.Default
	}
	return out
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

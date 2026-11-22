package jig

import (
	"fmt"
	"path/filepath"
)

// CheckResult reports what a dry render found.
type CheckResult struct {
	// Unused names manifest variables no template file or default
	// referenced — a cruft warning for template authors, never an error.
	Unused []string
}

// Check dry-renders the template in versionDir without writing anything:
// the manifest is loaded and validated, the resolved defaults are validated
// against their own patterns and choices, and the whole render is prepared
// — every name and every text file's content is parsed and rendered, with
// missingkey=error, internal collisions, containment (the rendered top-level
// project name included), and symlink policy all applying — but the write
// phase never runs. References to undeclared variables are rejected
// statically, across every branch of every template: missingkey=error only
// fires for branches the defaults execute, so a typo'd variable in an
// untaken {{if}} branch would otherwise pass. Declared-but-unused variables
// are reported. The provenance file name (.jig.json) is reserved against
// the template's own files: the CLI's render always writes the record, so
// a template shipping its own fails here, not after a partial render.
// Template authors run it — and CI should — to catch a broken
// template before it ships.
func Check(versionDir string) (*CheckResult, error) {
	skel := filepath.Join(versionDir, TemplateDirName)
	m, err := LoadSkeletonManifest(versionDir, false)
	if err != nil {
		return nil, err
	}
	values, err := m.Defaults()
	if err != nil {
		return nil, err
	}
	if err := m.Validate(values); err != nil {
		return nil, fmt.Errorf("resolved defaults: %w", err)
	}
	used := map[string]bool{}
	topName, err := renderString(TemplateDirName, values, used)
	if err != nil {
		return nil, fmt.Errorf("rendering the project directory name: %w", err)
	}
	dest, err := containedPath("check", topName)
	if err != nil {
		return nil, fmt.Errorf("rendering the project directory name: %w", err)
	}
	// The CLI's render always records provenance, so check reserves the
	// name too — a template shipping its own .jig.json fails here, before
	// any render writes half a tree.
	if _, err := planRender(skel, dest, values, used, m.Verbatim, false, &Provenance{}); err != nil {
		return nil, err
	}
	for _, name := range sortedKeys(used) {
		if _, ok := m.Variables[name]; !ok {
			return nil, fmt.Errorf("template references undeclared variable %s", name)
		}
	}
	return &CheckResult{Unused: m.Unused(used)}, nil
}

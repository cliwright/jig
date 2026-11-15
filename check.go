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
// the manifest is loaded and validated, and the whole render is prepared
// against the resolved defaults — every name and every text file's content
// is parsed and rendered, with missingkey=error, internal collisions,
// containment, and symlink policy all applying — but the write phase never
// runs. Declared-but-unused variables are reported. Template authors run
// it — and CI should — to catch a broken template before it ships.
func Check(versionDir string) (*CheckResult, error) {
	skel := filepath.Join(versionDir, TemplateDirName)
	m, err := LoadManifest(filepath.Join(skel, ManifestFileName))
	if err != nil {
		return nil, err
	}
	values, err := m.Defaults()
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	if _, err := renderString(TemplateDirName, values, used); err != nil {
		return nil, fmt.Errorf("rendering the project directory name: %w", err)
	}
	if _, err := planRender(skel, "check", values, used, m.Verbatim, false); err != nil {
		return nil, err
	}
	return &CheckResult{Unused: m.Unused(used)}, nil
}

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
// the manifest is loaded and validated, every file and directory name and
// every text file's content is parsed and rendered against the resolved
// defaults (missingkey=error, containment, and symlink policy all apply),
// and declared-but-unused variables are reported. Template authors run it —
// and CI should — to catch a broken template before it ships.
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
	plan, err := planRender(skel, "check", values, used, m.Verbatim, false)
	if err != nil {
		return nil, err
	}
	for _, f := range plan.files {
		if f.verbatim || f.linkTarget != "" {
			continue
		}
		if _, err := renderContent(f.src, values, used); err != nil {
			return nil, err
		}
	}
	return &CheckResult{Unused: m.Unused(used)}, nil
}

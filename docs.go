package jig

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strings"
)

// docsFS holds the project's markdown documentation, embedded so the docs
// an agent reads always match the binary it's driving — no version skew
// with the published site, no network access.
//
//go:embed docs
var docsFS embed.FS

// DocPages lists the available documentation page names (docs/ paths minus
// the .md suffix, e.g. "cli", "manifest"). The ADRs are excluded: they're
// design rationale for humans, published on the docs site.
func DocPages() []string {
	var pages []string
	// Walking an embedded FS cannot fail at runtime; the tree is fixed at
	// build time.
	_ = fs.WalkDir(docsFS, "docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "stylesheets" || d.Name() == "adr" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			pages = append(pages, strings.TrimSuffix(strings.TrimPrefix(path, "docs/"), ".md"))
		}
		return nil
	})
	sort.Strings(pages)
	return pages
}

// Doc returns one documentation page as raw markdown. page is a name from
// DocPages (a .md suffix is tolerated).
func Doc(page string) (string, error) {
	page = strings.TrimSuffix(page, ".md")
	if !slices.Contains(DocPages(), page) {
		return "", fmt.Errorf("unknown docs page %q (available: %s)", page, strings.Join(DocPages(), ", "))
	}
	data, err := docsFS.ReadFile("docs/" + page + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

package jig

import (
	"strings"
	"testing"
)

func TestDocPages(t *testing.T) {
	pages := DocPages()
	for _, want := range []string{"cli", "manifest", "implementation", "index"} {
		found := false
		for _, p := range pages {
			if p == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DocPages missing %q (have %v)", want, pages)
		}
	}
	for _, p := range pages {
		if strings.HasPrefix(p, "adr") {
			t.Errorf("ADRs should not be embedded, found %q", p)
		}
	}
}

func TestDoc(t *testing.T) {
	doc, err := Doc("cli")
	if err != nil {
		t.Fatalf("Doc(cli): %v", err)
	}
	if !strings.Contains(doc, "jig render") {
		t.Error("cli doc should mention jig render")
	}
	// a .md suffix is tolerated
	if _, err := Doc("manifest.md"); err != nil {
		t.Errorf("Doc(manifest.md): %v", err)
	}
	_, err = Doc("bogus")
	if err == nil || !strings.Contains(err.Error(), "available:") {
		t.Errorf("Doc(bogus) = %v, want an error listing pages", err)
	}
}

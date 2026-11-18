package jig

import (
	"regexp"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct{ in, want string }{
		{"My Project", "my-project"},
		{"already-slug", "already-slug"},
		{"lots  of__seps", "lots-of-seps"},
		{"--padded--", "padded"},
		{"UPPER Case 123", "upper-case-123"},
	}
	for _, tt := range tests {
		if got := slugify(tt.in); got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestReplace(t *testing.T) {
	if got := replace("-", "_", "a-b-c"); got != "a_b_c" {
		t.Errorf("replace = %q", got)
	}
	got, err := renderString(`{{ .name | replace "-" "_" }}`, map[string]string{"name": "a-b"}, map[string]bool{})
	if err != nil || got != "a_b" {
		t.Errorf("pipeline replace = %q, %v", got, err)
	}
}

func TestRenderStringUsesFuncMapAndMissingKey(t *testing.T) {
	got, err := renderString("{{.name | slugify}}", map[string]string{"name": "My Proj"}, map[string]bool{})
	if err != nil || got != "my-proj" {
		t.Errorf("renderString = %q, %v", got, err)
	}
	if _, err := renderString("{{.missing}}", map[string]string{}, map[string]bool{}); err == nil {
		t.Error("want missingkey=error")
	}
}

func TestDelims(t *testing.T) {
	got, err := renderString(
		`run: echo "${{ ldelim }} github.ref {{ rdelim }}" # {{ .PROJECT_NAME }}`,
		map[string]string{"PROJECT_NAME": "demo"}, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	want := `run: echo "${{ github.ref }}" # demo`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestYearAndDate(t *testing.T) {
	got, err := renderString("{{year}}", nil, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^20[0-9]{2}$`).MatchString(got) {
		t.Errorf("year = %q, want a four-digit year", got)
	}
	got, err = renderString("{{date}}", nil, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^20[0-9]{2}-[01][0-9]-[0-3][0-9]$`).MatchString(got) {
		t.Errorf("date = %q, want YYYY-MM-DD", got)
	}
}

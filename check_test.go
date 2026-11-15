package jig

import (
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":  validManifest,
		"README.md": "# {{.PROJECT_NAME}} by {{.author}}\n",
	})
	res, err := Check(versionDir)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Unused) != 0 {
		t.Errorf("unused = %v, want none", res.Unused)
	}
}

func TestCheckReportsUnused(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json":  validManifest,
		"README.md": "# {{.PROJECT_NAME}}\n",
	})
	res, err := Check(versionDir)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Unused) != 1 || res.Unused[0] != "author" {
		t.Errorf("unused = %v, want [author]", res.Unused)
	}
}

func TestCheckCatchesBrokenTemplates(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "undeclared variable in content",
			files: map[string]string{
				"jig.json":  validManifest,
				"README.md": "{{ .nope }}\n",
			},
			want: "nope",
		},
		{
			name: "unparseable content",
			files: map[string]string{
				"jig.json":  validManifest,
				"README.md": "{{\n",
			},
			want: "parsing",
		},
		{
			name: "undeclared variable in file name",
			files: map[string]string{
				"jig.json":       validManifest,
				"{{ .nope }}.md": "x\n",
			},
			want: "nope",
		},
		{
			name: "bad manifest",
			files: map[string]string{
				"jig.json": `{"version": 1, "variables": {"x": {"default": "y"}}}`,
			},
			want: "PROJECT_NAME",
		},
		{
			name: "default violating its own pattern",
			files: map[string]string{
				"jig.json": `{"version": 1, "variables": {
					"PROJECT_NAME": {"default": "demo"},
					"author": {"default": "Not Lower!", "pattern": "^[a-z]+$"}
				}}`,
				"README.md": "{{ .author }}\n",
			},
			want: "does not match pattern",
		},
		{
			name: "escaping project name",
			files: map[string]string{
				"jig.json": `{"version": 1, "variables": {
					"PROJECT_NAME": {"default": "../escape"}
				}}`,
				"README.md": "x\n",
			},
			want: "escapes the output directory",
		},
		{
			name: "undeclared variable in an untaken branch",
			files: map[string]string{
				"jig.json": `{"version": 1, "variables": {
					"PROJECT_NAME": {"default": "demo"},
					"WITH_TESTS": {"default": "no", "choices": ["yes", "no"]}
				}}`,
				"README.md": "{{if eq .WITH_TESTS \"yes\"}}{{ .NOPE }}{{end}}\n",
			},
			want: "undeclared variable NOPE",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Check(writeTemplate(t, tt.files))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

// A declared variable in a branch the defaults don't execute must still
// pass: the static undeclared-variable check rejects only names the
// manifest does not declare.
func TestCheckAllowsDeclaredVariablesInUntakenBranches(t *testing.T) {
	versionDir := writeTemplate(t, map[string]string{
		"jig.json": `{"version": 1, "variables": {
			"PROJECT_NAME": {"default": "demo"},
			"WITH_TESTS": {"default": "no", "choices": ["yes", "no"]}
		}}`,
		"README.md": "{{if eq .WITH_TESTS \"yes\"}}{{ .PROJECT_NAME }}{{end}}\n",
	})
	if _, err := Check(versionDir); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cliwright/jig"
)

var segment = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

const starterManifest = `{
  "version": 1,
  "variables": {
    "PROJECT_NAME": {
      "default": "my-project",
      "prompt": "Project name",
      "pattern": "^[a-z][a-z0-9-]*$"
    }
  }
}
`

func newNewCmd() *cobra.Command {
	var templatesDir string
	cmd := &cobra.Command{
		Use:   "new <language>/<flavor>/<kind>",
		Short: "Scaffold a new template in registry layout",
		Long: `Scaffold a new template in registry layout under the templates
directory: <language>/<flavor>/<kind>/v1/{{.PROJECT_NAME}}/jig.json plus a
README to start from. Flavor is mandatory — "std" when the language has a
standard toolchain, otherwise a name unique within the language namespace.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew(cmd, templatesDir, args[0])
		},
	}
	cmd.Flags().StringVar(&templatesDir, "templates-dir", os.Getenv("JIG_TEMPLATES_DIR"),
		"template registry root (default: $JIG_TEMPLATES_DIR or the current directory)")
	return cmd
}

func runNew(cmd *cobra.Command, templatesDir, name string) error {
	if templatesDir == "" {
		templatesDir = "."
	}
	segs := strings.Split(name, "/")
	if len(segs) != 3 {
		return fmt.Errorf("template name must be <language>/<flavor>/<kind>, got %q", name)
	}
	for _, s := range segs {
		if !segment.MatchString(s) {
			return fmt.Errorf("segment %q must be lowercase alphanumerics and hyphens, starting with a letter", s)
		}
	}
	dir := filepath.Join(templatesDir, filepath.FromSlash(name), "v1", jig.TemplateDirName)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%s already exists", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"jig.json":  starterManifest,
		"README.md": "# {{.PROJECT_NAME}}\n",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "scaffolded %s\nnext: jig render --template %s --templates-dir %s\n",
		dir, name, templatesDir)
	return nil
}

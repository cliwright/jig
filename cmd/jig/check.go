package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cliwright/jig"
)

func newCheckCmd() *cobra.Command {
	var (
		templateArg  string
		version      int
		templatesDir string
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Dry-render a template, reporting errors and unused variables",
		Long: `Dry-render a template without writing anything: the manifest is
validated, every name and text file is parsed and rendered against the
resolved defaults, and declared-but-unused variables are reported as
warnings. Exits non-zero on any error — run it before shipping a template.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if templateArg == "" {
				return fmt.Errorf("--template is required")
			}
			return runCheck(cmd, templateArg, version, templatesDir)
		},
	}
	f := cmd.Flags()
	f.StringVar(&templateArg, "template", "", "template name (<lang>/<flavor>/<kind>) or path to a version directory")
	f.IntVar(&version, "version", 0, "template version (default: latest)")
	f.StringVar(&templatesDir, "templates-dir", os.Getenv("JIG_TEMPLATES_DIR"), "template registry root (default: $JIG_TEMPLATES_DIR or the current directory)")
	return cmd
}

func runCheck(cmd *cobra.Command, templateArg string, version int, templatesDir string) error {
	if templatesDir == "" {
		templatesDir = "."
	}
	dir, err := jig.ResolveTemplate(templatesDir, templateArg, version)
	if err != nil {
		return err
	}
	res, err := jig.Check(dir)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	for _, name := range res.Unused {
		fmt.Fprintf(out, "warning: variable %s is declared but never used by the template\n", name)
	}
	fmt.Fprintf(out, "ok %s\n", dir)
	return nil
}

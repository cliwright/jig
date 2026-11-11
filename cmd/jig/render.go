package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cliwright/jig"
)

func newRenderCmd() *cobra.Command {
	var (
		templateArg  string
		output       string
		version      int
		templatesDir string
		varsFile     string
		varFlags     []string
		force        bool
	)
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a template into a new project directory",
		Long: `Render a template into a new project directory.

--template names a template (<language>/<flavor>/<kind>, resolved under
--templates-dir with the latest version unless --version is given) or a
direct path to a template version directory. Interactive prompting runs
unless --vars-file is given; --var key=value overrides individual
variables. Precedence: defaults < vars-file < --var < interactive answers.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if templateArg == "" {
				return fmt.Errorf("--template is required")
			}
			if output == "" {
				return fmt.Errorf("--output is required")
			}
			return runRender(cmd, templateArg, output, version, templatesDir, varsFile, varFlags, force)
		},
	}
	f := cmd.Flags()
	f.StringVar(&templateArg, "template", "", "template name (<lang>/<flavor>/<kind>) or path to a version directory")
	f.StringVar(&output, "output", "", "parent directory for the rendered project")
	f.IntVar(&version, "version", 0, "template version (default: latest)")
	f.StringVar(&templatesDir, "templates-dir", os.Getenv("JIG_TEMPLATES_DIR"), "template registry root (default: $JIG_TEMPLATES_DIR or the current directory)")
	f.StringVar(&varsFile, "vars-file", "", "JSON file of variable values (non-interactive)")
	f.StringArrayVar(&varFlags, "var", nil, "set a variable (key=value, repeatable)")
	f.BoolVar(&force, "force", false, "render even if the output directory is not empty")
	return cmd
}

func runRender(cmd *cobra.Command, templateArg, output string, version int, templatesDir, varsFile string, varFlags []string, force bool) error {
	if templatesDir == "" {
		templatesDir = "."
	}
	dir, err := jig.ResolveTemplate(templatesDir, templateArg, version)
	if err != nil {
		return err
	}
	m, err := jig.LoadManifest(filepath.Join(dir, jig.TemplateDirName, jig.ManifestFileName))
	if err != nil {
		return err
	}

	var prompt jig.Prompter
	if varsFile == "" {
		prompt = jig.StdinPrompter(cmd.InOrStdin(), cmd.OutOrStdout())
	}
	values, err := jig.ResolveValues(m, varsFile, varFlags, prompt)
	if err != nil {
		return err
	}

	res, err := jig.Render(dir, output, values, jig.Options{Force: force})
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "rendered %s\n", res.OutputDir)
	for _, name := range res.Unused {
		fmt.Fprintf(out, "warning: variable %s is declared but never used by the template\n", name)
	}
	return nil
}

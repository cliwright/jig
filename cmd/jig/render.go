package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

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
		trust        bool
	)
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a template into a new project directory",
		Long: `Render a template into a new project directory.

--template names a template (<language>/<flavor>/<kind>, resolved under
--templates-dir with the latest version unless --version is given) or a
direct path to a template version directory. Interactive prompting runs
unless --vars-file is given; --var key=value overrides individual
variables. Precedence: defaults < vars-file < --var < interactive answers.

Rendering never runs template-supplied programs and never overwrites
existing files unless --force is given; rendering into a populated
directory is fine as long as no file collides.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if templateArg == "" {
				return fmt.Errorf("--template is required")
			}
			if output == "" {
				return fmt.Errorf("--output is required")
			}
			return runRender(cmd, templateArg, output, version, templatesDir, varsFile, varFlags, force, trust)
		},
	}
	f := cmd.Flags()
	f.StringVar(&templateArg, "template", "", "template name (<lang>/<flavor>/<kind>) or path to a version directory (prefix relative paths with ./)")
	f.StringVar(&output, "output", "", "parent directory for the rendered project")
	f.IntVar(&version, "version", 0, "template version (default: latest)")
	f.StringVar(&templatesDir, "templates-dir", os.Getenv("JIG_TEMPLATES_DIR"), "template registry root (default: $JIG_TEMPLATES_DIR or the current directory)")
	f.StringVar(&varsFile, "vars-file", "", "JSON file of variable values (non-interactive)")
	f.StringArrayVar(&varFlags, "var", nil, "set a variable (key=value, repeatable)")
	f.BoolVar(&force, "force", false, "overwrite existing files on collision (default: refuse)")
	f.BoolVar(&trust, "trust", false, "allow symlinks inside the template tree (default: reject)")
	return cmd
}

func runRender(cmd *cobra.Command, templateArg, output string, version int, templatesDir, varsFile string, varFlags []string, force, trust bool) error {
	if templatesDir == "" {
		templatesDir = "."
	}
	dir, err := jig.ResolveTemplate(templatesDir, templateArg, version)
	if err != nil {
		return err
	}
	m, err := jig.LoadSkeletonManifest(dir, trust)
	if err != nil {
		return err
	}

	var prompt jig.Prompter
	if varsFile == "" {
		// A real terminal check, not a character-device check: /dev/null is
		// a character device but not a terminal, and CI runners, cron and
		// agent shells often present it as stdin. Non-*os.File readers (test
		// doubles) count as interactive.
		interactive := true
		if f, ok := cmd.InOrStdin().(*os.File); ok {
			interactive = term.IsTerminal(int(f.Fd()))
		}
		if !interactive && len(varFlags) == 0 {
			return fmt.Errorf("stdin is not interactive: provide values with --vars-file or --var")
		}
		if interactive {
			prompt = jig.StdinPrompter(cmd.InOrStdin(), cmd.OutOrStdout())
		}
	}
	values, err := jig.ResolveValues(m, varsFile, varFlags, prompt)
	if err != nil {
		return err
	}

	res, err := jig.Render(dir, output, values, jig.Options{Force: force, Trust: trust})
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

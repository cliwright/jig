package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cliwright/jig"
)

func newBumpCmd() *cobra.Command {
	var templatesDir string
	cmd := &cobra.Command{
		Use:   "bump <language>/<flavor>/<kind>",
		Short: "Copy a template's latest version to v<N+1>",
		Long: `Copy a template's latest version directory to v<N+1> as the
starting point for its next iteration. Old versions stay put — renders pin
theirs with --version.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if templatesDir == "" {
				templatesDir = "."
			}
			next, err := jig.Bump(templatesDir, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", next)
			return nil
		},
	}
	cmd.Flags().StringVar(&templatesDir, "templates-dir", os.Getenv("JIG_TEMPLATES_DIR"),
		"template registry root (default: $JIG_TEMPLATES_DIR or the current directory)")
	return cmd
}

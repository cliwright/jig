package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cliwright/jig"
)

func newAgentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "Print the agent guide (what jig is, trust model, usage pointers)",
		Long: `Print the agent guide: a stable, self-contained onboarding text
for AI agents driving jig — what it is, its trust model, how to run it
non-interactively, and where the real documentation lives.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), jig.AgentGuide())
			return err
		},
	}
}

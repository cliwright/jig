package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cliwright/jig"
)

func newDocsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "docs [page]",
		Short: "Print the embedded documentation (always matches this binary's version)",
		Long: `Print the documentation embedded in this binary as raw markdown.
With no page, lists what's available. These docs always describe the
binary you're running — unlike the website, they can't be newer or older
than your jig.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if len(args) == 0 {
				fmt.Fprintln(out, "available pages:")
				for _, page := range jig.DocPages() {
					fmt.Fprintf(out, "  %s\n", page)
				}
				return nil
			}
			doc, err := jig.Doc(args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(out, doc)
			return err
		},
	}
}

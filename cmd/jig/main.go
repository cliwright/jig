// jig renders project templates: inert directories of text/template files
// driven by a jig.json manifest. No hooks, no code execution — templates
// are data.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is stamped at release time (goreleaser -X main.version=...);
// builds from source report "dev".
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "jig",
		Short: "Render project templates (no hooks, no code execution — templates are data)",
		Long: `jig renders project templates: directories of Go text/template files
plus a jig.json manifest declaring the variables they prompt for.

Templates are addressed <language>/<flavor>/<kind> with a mandatory version
directory (v<N>) and rendered strictly: no hooks, no scripts, no arbitrary
code execution.

Agent using jig? Start with: jig agents`,
		Version:      version,
		SilenceUsage: true,
		// main prints the error itself
		SilenceErrors: true,
	}
	root.AddCommand(newRenderCmd(), newNewCmd(), newCheckCmd(), newBumpCmd(), newAgentsCmd(), newDocsCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

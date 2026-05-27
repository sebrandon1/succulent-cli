package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	listOutputFormat string
	listNoDetail     bool
	listNoCache      bool
	listConcurrency  int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available environments",
	Long:  `Fetch the main page and list all available environments with status details.`,
	Example: `  succulent-cli list
  succulent-cli list --no-detail
  succulent-cli list --no-cache
  succulent-cli list --output json`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if listNoDetail {
			return listBasic()
		}

		return listDetailed()
	},
}

func listBasic() error {
	envs, err := sharedClient.ListEnvironments()
	if err != nil {
		return fmt.Errorf("listing environments: %w", err)
	}

	if listOutputFormat == "json" {
		return printJSON(envs)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tGROUP")

	for _, env := range envs {
		group := env.Group
		if group == "" {
			group = "-"
		}

		fmt.Fprintf(w, "%s\t%s\n", env.Name, group)
	}

	return w.Flush()
}

func listDetailed() error {
	var cache *lib.Cache
	if !listNoCache && sharedCache != nil {
		cache = sharedCache
	}

	fmt.Fprintf(os.Stderr, "Fetching info for all environments (concurrency: %d)...\n", listConcurrency)

	details, err := sharedClient.ListEnvironmentsWithInfo(listConcurrency, cache)
	if err != nil {
		return fmt.Errorf("listing environments: %w", err)
	}

	if listOutputFormat == "json" {
		return printJSON(details)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tGROUP\tSTATUS\tNODES\tOWNER\tINSTALLER IP")

	for _, d := range details {
		group := d.Group
		if group == "" {
			group = "-"
		}

		owner := d.Owner
		if owner == "" {
			owner = "-"
		}

		ip := d.InstallerIP
		if ip == "" {
			ip = "-"
		}

		nodes := fmt.Sprintf("%d/%d", d.NodesUp, d.NodeCount)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", d.Name, group, d.Status, nodes, owner, ip)
	}

	return w.Flush()
}

func init() {
	listCmd.Flags().StringVarP(&listOutputFormat, "output", "o", "table", "Output format (json or table)")
	listCmd.Flags().BoolVar(&listNoDetail, "no-detail", false, "Skip fetching per-environment info (fast mode)")
	listCmd.Flags().BoolVar(&listNoCache, "no-cache", false, "Bypass the info cache")
	listCmd.Flags().IntVar(&listConcurrency, "concurrency", 10, "Number of parallel info fetches")

	rootCmd.AddCommand(listCmd)
}

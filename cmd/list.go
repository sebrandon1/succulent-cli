package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listOutputFormat string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available environments",
	Long:  `Fetch the main page and list all available environments with their host groups.`,
	Example: `  succulent-cli list
  succulent-cli list --output json`,
	RunE: func(_ *cobra.Command, _ []string) error {
		envs, err := sharedClient.ListEnvironments()
		if err != nil {
			return fmt.Errorf("listing environments: %w", err)
		}

		if listOutputFormat == "table" {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tGROUP")

			for _, env := range envs {
				group := env.Group
				if group == "" {
					group = "-"
				}

				fmt.Fprintf(w, "%s\t%s\n", env.Name, group)
			}

			w.Flush()

			return nil
		}

		return printJSON(envs)
	},
}

func init() {
	listCmd.Flags().StringVarP(&listOutputFormat, "output", "o", "table", "Output format (json or table)")

	rootCmd.AddCommand(listCmd)
}

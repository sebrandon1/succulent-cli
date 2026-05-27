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
	Run: func(_ *cobra.Command, _ []string) {
		envs, err := sharedClient.ListEnvironments()
		if err != nil {
			fmt.Printf("Error listing environments: %v\n", err)
			os.Exit(1)
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
		} else {
			printJSON(envs)
		}
	},
}

func init() {
	listCmd.Flags().StringVarP(&listOutputFormat, "output", "o", "table", "Output format (json or table)")

	rootCmd.AddCommand(listCmd)
}

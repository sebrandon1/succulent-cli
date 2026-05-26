package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get cluster node information for an environment",
	Long:  `Fetch the infoplan page for the specified environment, parse the HTML table, and output structured node information as JSON.`,
	Run: func(_ *cobra.Command, _ []string) {
		info, err := sharedClient.GetInfoPlan(envName)
		if err != nil {
			fmt.Printf("Error fetching info: %v\n", err)
			os.Exit(1)
		}

		printJSON(info)
	},
}

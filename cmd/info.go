package cmd

import (
	"fmt"
	"os"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get cluster node information for an environment",
	Long:  `Fetch the infoplan page for the specified environment, parse the HTML table, and output structured node information as JSON.`,
	Run: func(_ *cobra.Command, _ []string) {
		if envName == "" {
			fmt.Println("Error: --env is required")
			os.Exit(1)
		}

		client := lib.NewClient(succulentURL, !verifySSL)

		info, err := client.GetInfoPlan(envName)
		if err != nil {
			fmt.Printf("Error fetching info: %v\n", err)
			os.Exit(1)
		}

		printJSON(info)
	},
}

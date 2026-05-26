package cmd

import (
	"fmt"
	"os"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var confirmDelete bool

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an environment",
	Long:  `Delete the specified environment from the succulent service. Requires --confirm flag for safety.`,
	Run: func(_ *cobra.Command, _ []string) {
		if envName == "" {
			fmt.Println("Error: --env is required")
			os.Exit(1)
		}

		if !confirmDelete {
			fmt.Println("Error: --confirm is required to delete an environment")
			os.Exit(1)
		}

		client := lib.NewClient(succulentURL, !verifySSL)

		if err := client.DeleteEnvironment(envName); err != nil {
			fmt.Printf("Error deleting environment: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Environment %s deleted successfully\n", envName)
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion (required)")

	rootCmd.AddCommand(deleteCmd)
}

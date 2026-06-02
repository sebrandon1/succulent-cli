package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var confirmDelete bool

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete an environment",
	Long:    `Delete the specified environment from the succulent service. Requires --confirm flag for safety.`,
	Example: `  succulent-cli delete --env myenv --confirm`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if !confirmDelete {
			return fmt.Errorf("--confirm is required to delete an environment")
		}

		if err := sharedClient.DeleteEnvironment(envName); err != nil {
			return fmt.Errorf("deleting environment: %w", err)
		}

		return printResult(CommandResult{
			Status:      "deleted",
			Environment: envName,
			Message:     fmt.Sprintf("Environment %s deleted successfully", envName),
		}, outputFormat)
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion (required)")

	rootCmd.AddCommand(deleteCmd)
}

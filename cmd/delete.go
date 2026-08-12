package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	confirmDelete bool
	dryRunDelete  bool
)

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete an environment",
	Long:    `Delete the specified environment from the succulent service. Requires --confirm flag for safety.`,
	Example: `  succulent-cli delete --env myenv --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if dryRunDelete {
			printDryRun("delete", envName, nil)
			return nil
		}

		if !confirmDelete {
			return fmt.Errorf("--confirm is required to delete an environment (use --dry-run to preview)")
		}

		if err := sharedClient.DeleteEnvironment(cmd.Context(), envName); err != nil {
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
	deleteCmd.Flags().BoolVar(&dryRunDelete, "dry-run", false, "Show what would be done without executing")

	rootCmd.AddCommand(deleteCmd)
}

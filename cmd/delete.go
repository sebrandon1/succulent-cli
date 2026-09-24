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
		if !dryRunDelete && !confirmDelete {
			return fmt.Errorf("--confirm is required to delete an environment (use --dry-run to preview)")
		}

		return runForEnvironmentTargets(func(target string) (string, error) {
			if dryRunDelete {
				printDryRun("delete", target, nil)
				return fmt.Sprintf("[dry-run] Would delete %s", target), nil
			}
			if err := runAuditedOperationForEnvironment(target, "delete", nil, func() error {
				return sharedClient.DeleteEnvironment(cmd.Context(), target)
			}); err != nil {
				return "", fmt.Errorf("deleting environment: %w", err)
			}
			return fmt.Sprintf("Environment %s deleted successfully", target), nil
		}, func(target, message string) error {
			return printResult(CommandResult{Status: "deleted", Environment: target, Message: message}, outputFormat)
		}, dryRunDelete)
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion (required)")
	deleteCmd.Flags().BoolVar(&dryRunDelete, "dry-run", false, "Show what would be done without executing")

	rootCmd.AddCommand(deleteCmd)
}

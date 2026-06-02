package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch cluster provisioning until all nodes are up",
	Long:  `Monitor provisioning progress until all nodes are up with assigned IPs, then print the installer IP.`,
	Example: `  succulent-cli watch --env myenv
  succulent-cli watch --env myenv --max-wait 90 --poll-interval 15`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if pollIntervalSecs < minPollIntervalSecs {
			return fmt.Errorf("--poll-interval must be at least %d seconds", minPollIntervalSecs)
		}

		ip, err := sharedClient.WaitForClusterReady(envName, maxWaitMinutes, pollIntervalSecs, os.Stderr)
		if err != nil {
			return fmt.Errorf("watching cluster: %w", err)
		}

		return printResult(CommandResult{
			Status:      "ready",
			Environment: envName,
			Message:     fmt.Sprintf("\nInstaller IP: %s", ip),
		}, outputFormat)
	},
}

func init() {
	watchCmd.Flags().IntVar(&maxWaitMinutes, "max-wait", defaultMaxWaitMinutes, "Maximum minutes to wait")
	watchCmd.Flags().IntVar(&pollIntervalSecs, "poll-interval", defaultPollIntervalSecs, "Seconds between status checks")

	rootCmd.AddCommand(watchCmd)
}

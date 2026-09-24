package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var controlPlaneOnly bool

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch cluster provisioning until nodes are up",
	Long: `Monitor provisioning progress until nodes are up with assigned IPs, then print the installer IP.
Use --control-plane-only to report ready once the installer and masters are up, without waiting for workers.`,
	Example: `  succulent-cli watch --env myenv
  succulent-cli watch --env myenv --control-plane-only
  succulent-cli watch --env myenv --max-wait 90 --poll-interval 15`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if pollIntervalSecs < minPollIntervalSecs {
			return fmt.Errorf("--poll-interval must be at least %d seconds", minPollIntervalSecs)
		}

		ip, err := sharedClient.WaitForClusterReady(cmd.Context(), envName, maxWaitMinutes, pollIntervalSecs, os.Stderr, controlPlaneOnly)
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
	watchCmd.Flags().BoolVar(&controlPlaneOnly, "control-plane-only", false, "Report ready when installer and masters are up (don't wait for workers)")

	rootCmd.AddCommand(watchCmd)
}

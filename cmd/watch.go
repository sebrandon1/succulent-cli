package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var controlPlaneOnly bool

var (
	watchAfterProvision       bool
	provisionWaitMinutes      int
	provisionPollIntervalSecs int
	provisionControlPlaneOnly bool
)

func addProvisionWatchFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&watchAfterProvision, "watch", false, "Wait for the cluster to become ready after submission")
	cmd.Flags().IntVar(&provisionWaitMinutes, "max-wait", defaultMaxWaitMinutes, "Maximum minutes to wait with --watch")
	cmd.Flags().IntVar(&provisionPollIntervalSecs, "poll-interval", defaultPollIntervalSecs, "Seconds between checks with --watch")
	cmd.Flags().BoolVar(&provisionControlPlaneOnly, "control-plane-only", false, "With --watch, consider ready when installer and masters are up")
}

func validateProvisionWatchFlags(dryRun bool) error {
	if !watchAfterProvision {
		return nil
	}
	if dryRun {
		return fmt.Errorf("--watch cannot be used with --dry-run")
	}
	if provisionWaitMinutes <= 0 {
		return fmt.Errorf("--max-wait must be greater than zero")
	}
	if provisionPollIntervalSecs < minPollIntervalSecs {
		return fmt.Errorf("--poll-interval must be at least %d seconds", minPollIntervalSecs)
	}
	return nil
}

func waitForProvisioning(cmd *cobra.Command, environment, operation, message string) (string, error) {
	if !watchAfterProvision {
		return message, nil
	}
	fmt.Fprintf(os.Stderr, "%s; waiting for cluster readiness\n", message)
	installerIP, err := sharedClient.WaitForClusterReady(cmd.Context(), environment, provisionWaitMinutes, provisionPollIntervalSecs, os.Stderr, provisionControlPlaneOnly)
	if err != nil {
		return "", fmt.Errorf("%s request submitted, but waiting for the cluster failed: %w", operation, err)
	}
	return fmt.Sprintf("%s; cluster ready (installer IP: %s)", message, installerIP), nil
}

func provisionResultStatus() string {
	if watchAfterProvision {
		return "ready"
	}
	return "submitted"
}

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

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
	Run: func(_ *cobra.Command, _ []string) {
		if pollIntervalSecs < minPollIntervalSecs {
			fmt.Printf("Error: --poll-interval must be at least %d seconds\n", minPollIntervalSecs)
			os.Exit(1)
		}

		ip, err := sharedClient.WaitForClusterReady(envName, maxWaitMinutes, pollIntervalSecs, os.Stdout)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nInstaller IP: %s\n", ip)
	},
}

func init() {
	watchCmd.Flags().IntVar(&maxWaitMinutes, "max-wait", defaultMaxWaitMinutes, "Maximum minutes to wait")
	watchCmd.Flags().IntVar(&pollIntervalSecs, "poll-interval", defaultPollIntervalSecs, "Seconds between status checks")

	rootCmd.AddCommand(watchCmd)
}

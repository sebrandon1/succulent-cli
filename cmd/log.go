package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Get Ansible playbook log output for an environment",
	Long:  `Fetch the ZTP provisioning log (raw Ansible output) for the specified environment and stream it to stdout.`,
	Run: func(_ *cobra.Command, _ []string) {
		if err := sharedClient.StreamLog(envName, os.Stdout); err != nil {
			fmt.Printf("Error fetching log: %v\n", err)
			os.Exit(1)
		}
	},
}

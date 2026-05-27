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
	Example: `  succulent-cli get log --env myenv
  succulent-cli get log --env myenv | tail -50`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := sharedClient.StreamLog(envName, os.Stdout); err != nil {
			return fmt.Errorf("fetching log: %w", err)
		}

		return nil
	},
}

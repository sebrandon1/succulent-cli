package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var followLog bool

const logFollowPollInterval = 2 * time.Second

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Get Ansible playbook log output for an environment",
	Long:  `Fetch the ZTP provisioning log (raw Ansible output) for the specified environment and stream it to stdout. Use --follow to print new output until the playbook finishes or the command is canceled.`,
	Example: `  succulent-cli get log --env myenv
	  succulent-cli get log --env myenv --follow
  succulent-cli get log --env myenv | tail -50`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		var err error
		if followLog {
			err = sharedClient.FollowLog(cmd.Context(), envName, os.Stdout, logFollowPollInterval)
		} else {
			err = sharedClient.StreamLog(cmd.Context(), envName, os.Stdout)
		}
		if err != nil {
			return fmt.Errorf("fetching log: %w", err)
		}

		return nil
	},
}

func init() {
	logCmd.Flags().BoolVar(&followLog, "follow", false, "Poll for new output until the Ansible playbook finishes")
}

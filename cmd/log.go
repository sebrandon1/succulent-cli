package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var followLog bool

const logFollowPollInterval = 2 * time.Second

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Get Ansible playbook log output for an environment",
	Long:  `Fetch the ZTP provisioning log (raw Ansible output) for the specified environment and print it to stdout. While a playbook runs, plain get log prints the output received before --timeout and exits; use --follow to stream until the playbook finishes or the command is canceled.`,
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
		if errors.Is(err, lib.ErrLogStillStreaming) {
			fmt.Fprintln(os.Stderr, "\nNote: the playbook is still running; showing the output so far (use --follow to keep streaming)")
			return nil
		}
		if err != nil {
			return fmt.Errorf("fetching log: %w", err)
		}

		return nil
	},
}

func init() {
	logCmd.Flags().BoolVar(&followLog, "follow", false, "Stream new output until the Ansible playbook finishes")
}

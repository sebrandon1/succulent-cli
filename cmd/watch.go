package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

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
	cmd.Flags().IntVar(&provisionPollIntervalSecs, "poll-interval", defaultPollIntervalSecs, "Seconds between checks with --watch (0 uses adaptive polling; minimum override is 5)")
	cmd.Flags().BoolVar(&provisionControlPlaneOnly, "control-plane-only", false, "With --watch, consider ready when installer and masters are up")
}

func validatePollInterval(pollIntervalSecs int) error {
	if pollIntervalSecs < 0 || (pollIntervalSecs > 0 && pollIntervalSecs < minPollIntervalSecs) {
		return fmt.Errorf("--poll-interval must be 0 (adaptive) or at least %d seconds", minPollIntervalSecs)
	}
	return nil
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
	if err := validatePollInterval(provisionPollIntervalSecs); err != nil {
		return err
	}
	return nil
}

func waitForProvisioning(cmd *cobra.Command, environment, operation, message string) (string, error) {
	if !watchAfterProvision {
		return message, nil
	}
	fmt.Fprintf(os.Stderr, "%s; waiting for cluster readiness\n", message)
	return waitForClusterReadiness(cmd, environment, operation, message, provisionWaitMinutes)
}

func waitForClusterReadiness(cmd *cobra.Command, environment, operation, message string, maxWait int) (string, error) {
	installerIP, err := sharedClient.WaitForClusterReady(cmd.Context(), environment, maxWait, provisionPollIntervalSecs, os.Stderr, provisionControlPlaneOnly)
	if err != nil {
		return "", fmt.Errorf("%s request submitted, but waiting for the cluster failed: %w", operation, err)
	}
	return fmt.Sprintf("%s; cluster ready (installer IP: %s)", message, installerIP), nil
}

// waitForZTPProvisioning watches a ZTP run through its Ansible log. Node
// status alone can't show ZTP progress: reprovisioning reuses nodes that are
// already up, and a failed playbook leaves them up. It waits for a run newer
// than baseline (see lib.Client.LogPrefix), follows it to the play recap,
// fails if any host failed, then confirms the nodes are ready.
func waitForZTPProvisioning(cmd *cobra.Command, environment, message string, baseline []byte) (string, error) {
	if !watchAfterProvision {
		return message, nil
	}
	fmt.Fprintf(os.Stderr, "%s; waiting for the ZTP playbook to finish\n", message)

	deadline := time.Now().Add(time.Duration(provisionWaitMinutes) * time.Minute)
	ctx, cancel := context.WithDeadline(cmd.Context(), deadline)
	defer cancel()

	progress := &taskProgressWriter{w: os.Stderr}
	if err := sharedClient.WaitForPlaybook(ctx, environment, baseline, playbookPollInterval(), progress); err != nil {
		return "", fmt.Errorf("ZTP provision request submitted, but the playbook did not succeed: %w", err)
	}

	remaining := int(math.Ceil(time.Until(deadline).Minutes()))
	return waitForClusterReadiness(cmd, environment, "ZTP provision", message+"; playbook finished", max(remaining, 1))
}

const defaultPlaybookPollInterval = 15 * time.Second

func playbookPollInterval() time.Duration {
	if provisionPollIntervalSecs > 0 {
		return time.Duration(provisionPollIntervalSecs) * time.Second
	}
	return defaultPlaybookPollInterval
}

// taskProgressWriter turns a streamed Ansible log into short progress output:
// only the PLAY, TASK and PLAY RECAP headers.
type taskProgressWriter struct {
	w       io.Writer
	partial []byte
}

func (t *taskProgressWriter) Write(p []byte) (int, error) {
	t.partial = append(t.partial, p...)
	for {
		end := bytes.IndexByte(t.partial, '\n')
		if end < 0 {
			break
		}
		line := strings.TrimRight(string(t.partial[:end]), "\r")
		t.partial = t.partial[end+1:]

		if strings.HasPrefix(line, "PLAY ") || strings.HasPrefix(line, "TASK [") {
			fmt.Fprintf(t.w, "  %s\n", strings.TrimRight(line, " *"))
		}
	}
	return len(p), nil
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
		if err := validatePollInterval(pollIntervalSecs); err != nil {
			return err
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
	watchCmd.Flags().IntVar(&pollIntervalSecs, "poll-interval", defaultPollIntervalSecs, "Seconds between status checks (0 uses adaptive polling; minimum override is 5)")
	watchCmd.Flags().BoolVar(&controlPlaneOnly, "control-plane-only", false, "Report ready when installer and masters are up (don't wait for workers)")

	rootCmd.AddCommand(watchCmd)
}

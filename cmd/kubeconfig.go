package cmd

import (
	"fmt"
	"os"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	remoteUser     string
	remotePassword string
	remotePath     string
	destPath       string
	waitForReady   bool
	strictSSH      bool
)

var fetchKubeconfigCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch kubeconfig from a cluster's installer node",
	Long: `Scrape the installer node IP from the infoplan page and SCP the
kubeconfig to a local path.

SSH host key checking is disabled by default (StrictHostKeyChecking=no,
UserKnownHostsFile=/dev/null). That is insecure, but typical for this lab
where installer VMs are frequently reprovisioned. Use --strict-ssh to
verify the host against ~/.ssh/known_hosts instead.`,
	Example: `  succulent-cli kubeconfig fetch --env myenv
  succulent-cli kubeconfig fetch --env myenv --wait
  succulent-cli kubeconfig fetch --env myenv --strict-ssh
  succulent-cli kubeconfig fetch --env myenv --dest ./kubeconfig --user kni`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		user := viper.GetString("remote_user")
		password := viper.GetString("remote_password")
		path := viper.GetString("remote_path")
		targets := currentEnvironmentTargets()
		if err := validateBatchDestination(destPath, targets); err != nil {
			return err
		}
		return runForEnvironmentTargets(func(target string) (string, error) {
			var installerIP string
			if waitForReady {
				ip, err := sharedClient.WaitForClusterReady(cmd.Context(), target, maxWaitMinutes, pollIntervalSecs, os.Stdout, controlPlaneOnly)
				if err != nil {
					return "", fmt.Errorf("waiting for cluster: %w", err)
				}
				installerIP = ip
			} else {
				info, err := sharedClient.GetInfoPlan(cmd.Context(), target)
				if err != nil {
					return "", fmt.Errorf("fetching cluster info: %w", err)
				}
				installerIP = info.InstallerIP
			}
			if installerIP == "" {
				return "", fmt.Errorf("could not determine installer IP for %s; try: succulent-cli get info --env %s", target, target)
			}

			dest, err := environmentDestination(destPath, target, targets)
			if err != nil {
				return "", err
			}
			if dest == "" {
				dest, err = defaultDestPath(target, cmdNameKubeconfig)
				if err != nil {
					return "", err
				}
			}

			strict := viper.GetBool("strict_ssh")
			if !strict {
				if err := lib.RemoveSSHHostKeyContext(cmd.Context(), installerIP); err != nil {
					fmt.Printf("Warning: could not remove SSH host key: %v\n", err)
				}
			}
			if err := lib.FetchKubeconfigContext(cmd.Context(), installerIP, user, password, path, dest, strict); err != nil {
				return "", fmt.Errorf("fetching kubeconfig: %w", err)
			}
			return fmt.Sprintf("Kubeconfig saved to: %s", dest), nil
		}, func(_ string, message string) error {
			fmt.Println(message)
			return nil
		}, false)
	},
}

func init() {
	fetchKubeconfigCmd.Flags().StringVar(&remoteUser, "user", defaultRemoteUser, "Remote SSH user")
	fetchKubeconfigCmd.Flags().StringVar(&remotePassword, "password", "", "SSH password (requires sshpass)")
	fetchKubeconfigCmd.Flags().StringVar(&remotePath, "path", defaultRemotePath, "Remote kubeconfig path")
	fetchKubeconfigCmd.Flags().StringVar(&destPath, "dest", "", "Local destination path (default: ~/Downloads/succulent/{env}/kubeconfig)")
	fetchKubeconfigCmd.Flags().BoolVar(&waitForReady, "wait", false, "Wait for cluster nodes to be up before fetching")
	fetchKubeconfigCmd.Flags().BoolVar(&strictSSH, "strict-ssh", false, "Enable SSH host key checking (insecure checking is the lab default)")
	fetchKubeconfigCmd.Flags().BoolVar(&controlPlaneOnly, "control-plane-only", false, "With --wait, report ready when installer and masters are up")
	fetchKubeconfigCmd.Flags().IntVar(&maxWaitMinutes, "max-wait", defaultMaxWaitMinutes, "Maximum minutes to wait for cluster ready")
	fetchKubeconfigCmd.Flags().IntVar(&pollIntervalSecs, "poll-interval", defaultPollIntervalSecs, "Seconds between status checks when waiting")

	_ = viper.BindPFlag("remote_user", fetchKubeconfigCmd.Flags().Lookup("user"))
	_ = viper.BindPFlag("remote_password", fetchKubeconfigCmd.Flags().Lookup("password"))
	_ = viper.BindPFlag("remote_path", fetchKubeconfigCmd.Flags().Lookup("path"))
	_ = viper.BindPFlag("strict_ssh", fetchKubeconfigCmd.Flags().Lookup("strict-ssh"))
}

package cmd

import (
	"fmt"
	"os"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	remoteUser   string
	remotePath   string
	destPath     string
	waitForReady bool
)

var fetchKubeconfigCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch kubeconfig from a cluster's installer node",
	Long: `Scrape the installer node IP from the infoplan page, remove stale SSH
host keys, and SCP the kubeconfig to a local path.`,
	Example: `  succulent-cli kubeconfig fetch --env myenv
  succulent-cli kubeconfig fetch --env myenv --wait
  succulent-cli kubeconfig fetch --env myenv --dest ./kubeconfig --user kni`,
	Run: func(_ *cobra.Command, _ []string) {
		var installerIP string

		if waitForReady {
			ip, err := sharedClient.WaitForClusterReady(envName, maxWaitMinutes, pollIntervalSecs, os.Stdout)
			if err != nil {
				fmt.Printf("Error waiting for cluster: %v\n", err)
				os.Exit(1)
			}

			installerIP = ip
		} else {
			info, err := sharedClient.GetInfoPlan(envName)
			if err != nil {
				fmt.Printf("Error fetching cluster info: %v\n", err)
				os.Exit(1)
			}

			installerIP = info.InstallerIP
		}

		if installerIP == "" {
			fmt.Println("Error: could not determine installer IP")
			os.Exit(1)
		}

		dest := destPath
		if dest == "" {
			var err error

			dest, err = defaultDestPath(envName, cmdNameKubeconfig)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}

		if err := lib.RemoveSSHHostKey(installerIP); err != nil {
			fmt.Printf("Warning: could not remove SSH host key: %v\n", err)
		}

		if err := lib.FetchKubeconfig(installerIP, remoteUser, remotePath, dest); err != nil {
			fmt.Printf("Error fetching kubeconfig: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Kubeconfig saved to: %s\n", dest)
	},
}

func init() {
	fetchKubeconfigCmd.Flags().StringVar(&remoteUser, "user", defaultRemoteUser, "Remote SSH user")
	fetchKubeconfigCmd.Flags().StringVar(&remotePath, "path", defaultRemotePath, "Remote kubeconfig path")
	fetchKubeconfigCmd.Flags().StringVar(&destPath, "dest", "", "Local destination path (default: ~/Downloads/{env}-kubeconfig)")
	fetchKubeconfigCmd.Flags().BoolVar(&waitForReady, "wait", false, "Wait for all cluster nodes to be up before fetching")
	fetchKubeconfigCmd.Flags().IntVar(&maxWaitMinutes, "max-wait", defaultMaxWaitMinutes, "Maximum minutes to wait for cluster ready")
	fetchKubeconfigCmd.Flags().IntVar(&pollIntervalSecs, "poll-interval", defaultPollIntervalSecs, "Seconds between status checks when waiting")
}

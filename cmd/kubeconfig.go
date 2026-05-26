package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	remoteUser     string
	remotePath     string
	destPath       string
	waitForReady   bool
	maxWaitMinutes int
	pollInterval   int
)

var fetchKubeconfigCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch kubeconfig from a cluster's installer node",
	Long: `Scrape the installer node IP from the succulent infoplan page,
remove stale SSH host keys, and SCP the kubeconfig to a local path.

Optionally wait for all cluster nodes to be up before fetching.`,
	Run: func(_ *cobra.Command, _ []string) {
		if envName == "" {
			fmt.Println("Error: --env is required")
			os.Exit(1)
		}

		client := lib.NewClient(succulentURL, !verifySSL)

		var installerIP string

		if waitForReady {
			ip, err := client.WaitForClusterReady(envName, maxWaitMinutes, pollInterval)
			if err != nil {
				fmt.Printf("Error waiting for cluster: %v\n", err)
				os.Exit(1)
			}

			installerIP = ip
		} else {
			info, err := client.GetInfoPlan(envName)
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
			home, _ := os.UserHomeDir()
			dest = filepath.Join(home, defaultDestDir, envName+"-kubeconfig")
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
	fetchKubeconfigCmd.Flags().IntVar(&maxWaitMinutes, "max-wait", 60, "Maximum minutes to wait for cluster ready")
	fetchKubeconfigCmd.Flags().IntVar(&pollInterval, "poll-interval", 30, "Seconds between status checks when waiting")
}

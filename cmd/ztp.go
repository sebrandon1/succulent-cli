package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	ztpOwner        string
	ztpEmail        string
	ztpSNOTag       string
	ztpSNORelease   string
	ztpSNOFullTag   string
	ztpSpokeTag     string
	ztpSpokeRelease string
	ztpSpokeFullTag string
	ztpType         string
	ztpStopBefore   bool
	ztpVMMasters    string
	ztpBMMasters    string
	ztpBMWorkers    string
	ztpVMWorkers    string
	ztpKCChoice     string
	ztpKCDest       string
)

var ztpCmd = &cobra.Command{
	Use:   "ztp",
	Short: "ZTP cluster management commands",
}

var ztpProvisionCmd = &cobra.Command{
	Use:   cmdNameProvision,
	Short: "Provision a ZTP hub and spoke cluster",
	Long:  `Submit a ZTP provisioning request for the specified environment.`,
	Example: `  succulent-cli ztp provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --spoke-tag 4.17
  succulent-cli ztp provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --spoke-tag 4.17 --type mno --vm-masters 3`,
	Run: func(_ *cobra.Command, _ []string) {
		owner := ztpOwner
		if owner == "" {
			owner = viper.GetString("default_owner")
		}

		email := ztpEmail
		if email == "" {
			email = viper.GetString("default_email")
		}

		if owner == "" {
			fmt.Println("Error: --owner is required (or set default_owner in config)")
			os.Exit(1)
		}

		if email == "" {
			fmt.Println("Error: --email is required (or set default_email in config)")
			os.Exit(1)
		}

		req := lib.ZTPRequest{
			Owner:                owner,
			Email:                email,
			SNOTag:               ztpSNOTag,
			SNORelease:           ztpSNORelease,
			SNOFullTag:           ztpSNOFullTag,
			ZTPTag:               ztpSpokeTag,
			ZTPRelease:           ztpSpokeRelease,
			ZTPFullTag:           ztpSpokeFullTag,
			ZTPType:              ztpType,
			StopBeforeDeployment: ztpStopBefore,
			VMMastersCount:       ztpVMMasters,
			BMMastersHosts:       ztpBMMasters,
			BMWorkersHosts:       ztpBMWorkers,
			VMWorkersCount:       ztpVMWorkers,
		}

		if err := sharedClient.ProvisionZTP(envName, &req); err != nil {
			fmt.Printf("Error submitting ZTP provision request: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("ZTP provision request submitted for %s (type: %s)\n", envName, ztpType)
	},
}

var ztpKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the ZTP kubeconfig",
	Example: `  succulent-cli ztp kubeconfig --env myenv --choice management
  succulent-cli ztp kubeconfig --env myenv --choice spoke`,
	Run: func(_ *cobra.Command, _ []string) {
		data, err := sharedClient.GetZTPKubeconfig(envName, ztpKCChoice)
		if err != nil {
			fmt.Printf("Error fetching ZTP kubeconfig: %v\n", err)
			os.Exit(1)
		}

		dest := ztpKCDest
		if dest == "" {
			var destErr error

			dest, destErr = defaultDestPath(envName, "ztp-"+ztpKCChoice+"-kubeconfig")
			if destErr != nil {
				fmt.Printf("Error: %v\n", destErr)
				os.Exit(1)
			}
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			fmt.Printf("Error creating destination directory: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(dest, data, 0o600); err != nil {
			fmt.Printf("Error writing kubeconfig: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("ZTP %s kubeconfig saved to: %s\n", ztpKCChoice, dest)
	},
}

func init() {
	ztpProvisionCmd.Flags().StringVar(&ztpOwner, "owner", "", "Username (owner)")
	ztpProvisionCmd.Flags().StringVar(&ztpEmail, "email", "", "Email address for notifications")
	ztpProvisionCmd.Flags().StringVar(&ztpSNOTag, "sno-tag", "", "Hub (SNO) cluster OCP tag (e.g., 4.17)")
	ztpProvisionCmd.Flags().StringVar(&ztpSNORelease, "sno-release", "nightly", "Hub cluster release type")
	ztpProvisionCmd.Flags().StringVar(&ztpSNOFullTag, "sno-full-tag", "", "Hub cluster full OCP tag")
	ztpProvisionCmd.Flags().StringVar(&ztpSpokeTag, "spoke-tag", "", "Spoke cluster OCP tag (e.g., 4.17)")
	ztpProvisionCmd.Flags().StringVar(&ztpSpokeRelease, "spoke-release", "nightly", "Spoke cluster release type")
	ztpProvisionCmd.Flags().StringVar(&ztpSpokeFullTag, "spoke-full-tag", "", "Spoke cluster full OCP tag")
	ztpProvisionCmd.Flags().StringVar(&ztpType, "type", "sno", "ZTP type: sno or mno")
	ztpProvisionCmd.Flags().BoolVar(&ztpStopBefore, "stop-before-deployment", false, "Stop before actual spoke deployment for manual GitOps changes")
	ztpProvisionCmd.Flags().StringVar(&ztpVMMasters, "vm-masters", "3", "Number of VM masters (MNO only)")
	ztpProvisionCmd.Flags().StringVar(&ztpBMMasters, "bm-masters", "", "Comma-separated baremetal master hosts (MNO only)")
	ztpProvisionCmd.Flags().StringVar(&ztpBMWorkers, "bm-workers", "", "Comma-separated baremetal worker hosts")
	ztpProvisionCmd.Flags().StringVar(&ztpVMWorkers, "vm-workers", "1", "Number of VM workers")

	ztpKubeconfigCmd.Flags().StringVar(&ztpKCChoice, "choice", "", "Kubeconfig type: management or spoke")
	ztpKubeconfigCmd.Flags().StringVar(&ztpKCDest, "dest", "", "Local destination path")
	markFlagRequired(ztpKubeconfigCmd.MarkFlagRequired("choice"))

	ztpCmd.AddCommand(ztpProvisionCmd)
	ztpCmd.AddCommand(ztpKubeconfigCmd)

	rootCmd.AddCommand(ztpCmd)
}

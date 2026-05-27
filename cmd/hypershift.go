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
	hsOwner         string
	hsEmail         string
	hsSNOTag        string
	hsSNORelease    string
	hsSNOFullTag    string
	hsHCPTag        string
	hsHCPRelease    string
	hsHCPFullTag    string
	hsVMWorkers     string
	hsImageOverride string
	hsKCChoice      string
	hsKCDest        string
)

var hypershiftCmd = &cobra.Command{
	Use:   "hypershift",
	Short: "Hypershift cluster management commands",
}

var hsProvisionCmd = &cobra.Command{
	Use:   cmdNameProvision,
	Short: "Provision a Hypershift hosted cluster",
	Long:  `Submit a Hypershift provisioning request for the specified environment.`,
	Example: `  succulent-cli hypershift provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --hcp-tag 4.17
  succulent-cli hypershift provision --env myenv --owner myuser --email user@example.com --sno-tag 4.16 --hcp-full-tag 4.15.0-rc.1 --vm-workers 2`,
	Run: func(_ *cobra.Command, _ []string) {
		owner := hsOwner
		if owner == "" {
			owner = viper.GetString("default_owner")
		}

		email := hsEmail
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

		req := lib.HypershiftRequest{
			Owner:          owner,
			Email:          email,
			SNOTag:         hsSNOTag,
			SNORelease:     hsSNORelease,
			SNOFullTag:     hsSNOFullTag,
			HCPTag:         hsHCPTag,
			HCPRelease:     hsHCPRelease,
			HCPFullTag:     hsHCPFullTag,
			VMWorkersCount: hsVMWorkers,
			ImageOverride:  hsImageOverride,
		}

		if err := sharedClient.ProvisionHypershift(envName, &req); err != nil {
			fmt.Printf("Error submitting Hypershift provision request: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Hypershift provision request submitted for %s\n", envName)
	},
}

var hsKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the Hypershift kubeconfig",
	Example: `  succulent-cli hypershift kubeconfig --env myenv --choice management
  succulent-cli hypershift kubeconfig --env myenv --choice hosted`,
	Run: func(_ *cobra.Command, _ []string) {
		data, err := sharedClient.GetHypershiftKubeconfig(envName, hsKCChoice)
		if err != nil {
			fmt.Printf("Error fetching Hypershift kubeconfig: %v\n", err)
			os.Exit(1)
		}

		dest := hsKCDest
		if dest == "" {
			var destErr error

			dest, destErr = defaultDestPath(envName, "hypershift-"+hsKCChoice+"-kubeconfig")
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

		fmt.Printf("Hypershift %s kubeconfig saved to: %s\n", hsKCChoice, dest)
	},
}

func init() {
	hsProvisionCmd.Flags().StringVar(&hsOwner, "owner", "", "Username (owner)")
	hsProvisionCmd.Flags().StringVar(&hsEmail, "email", "", "Email address for notifications")
	hsProvisionCmd.Flags().StringVar(&hsSNOTag, "sno-tag", "", "Management cluster OCP tag (e.g., 4.17)")
	hsProvisionCmd.Flags().StringVar(&hsSNORelease, "sno-release", "nightly", "Management cluster release type")
	hsProvisionCmd.Flags().StringVar(&hsSNOFullTag, "sno-full-tag", "", "Management cluster full OCP tag")
	hsProvisionCmd.Flags().StringVar(&hsHCPTag, "hcp-tag", "", "Hosted cluster OCP tag (e.g., 4.17)")
	hsProvisionCmd.Flags().StringVar(&hsHCPRelease, "hcp-release", "nightly", "Hosted cluster release type")
	hsProvisionCmd.Flags().StringVar(&hsHCPFullTag, "hcp-full-tag", "", "Hosted cluster full OCP tag")
	hsProvisionCmd.Flags().StringVar(&hsVMWorkers, "vm-workers", "0", "Number of VM workers for hosted cluster")
	hsProvisionCmd.Flags().StringVar(&hsImageOverride, "image-override", "", "Hypershift operator image override")

	hsKubeconfigCmd.Flags().StringVar(&hsKCChoice, "choice", "", "Kubeconfig type: management or hosted")
	hsKubeconfigCmd.Flags().StringVar(&hsKCDest, "dest", "", "Local destination path")
	markFlagRequired(hsKubeconfigCmd.MarkFlagRequired("choice"))

	hypershiftCmd.AddCommand(hsProvisionCmd)
	hypershiftCmd.AddCommand(hsKubeconfigCmd)

	rootCmd.AddCommand(hypershiftCmd)
}

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
	snoOwner     string
	snoEmail     string
	snoOCPTag    string
	snoRelease   string
	snoFullTag   string
	snoFullImage string
	snoKCDest    string
)

var snoProvisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision an SNO cluster",
	Long:  `Submit an SNO provisioning request to the succulent service for the specified environment.`,
	Example: `  succulent-cli sno provision --env myenv --owner myuser --email user@example.com --ocp-tag 4.17
  succulent-cli sno provision --env myenv --owner myuser --email user@example.com --full-ocp-tag 4.17.0-0.nightly-2026-05-20-123456`,
	Run: func(_ *cobra.Command, _ []string) {
		owner := snoOwner
		if owner == "" {
			owner = viper.GetString("default_owner")
		}

		email := snoEmail
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

		req := lib.SNOProvisionRequest{
			Owner:         owner,
			Email:         email,
			OCPTag:        snoOCPTag,
			ReleaseType:   snoRelease,
			FullOCPTag:    snoFullTag,
			FullImageName: snoFullImage,
		}

		if err := sharedClient.ProvisionSNO(envName, &req); err != nil {
			fmt.Printf("Error submitting SNO provision request: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("SNO provision request submitted for %s\n", envName)
	},
}

var snoKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the SNO kubeconfig for an environment",
	Example: `  succulent-cli sno kubeconfig --env myenv
  succulent-cli sno kubeconfig --env myenv --dest ./kubeconfig`,
	Run: func(_ *cobra.Command, _ []string) {
		data, err := sharedClient.GetSNOKubeconfig(envName)
		if err != nil {
			fmt.Printf("Error fetching SNO kubeconfig: %v\n", err)
			os.Exit(1)
		}

		dest := snoKCDest
		if dest == "" {
			var destErr error

			dest, destErr = defaultDestPath(envName, "sno-kubeconfig")
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

		fmt.Printf("SNO kubeconfig saved to: %s\n", dest)
	},
}

func init() {
	snoProvisionCmd.Flags().StringVar(&snoOwner, "owner", "", "Username (owner)")
	snoProvisionCmd.Flags().StringVar(&snoEmail, "email", "", "Email address for notifications")
	snoProvisionCmd.Flags().StringVar(&snoOCPTag, "ocp-tag", "", "OCP tag (e.g., 4.17)")
	snoProvisionCmd.Flags().StringVar(&snoRelease, "release-type", "nightly", "OCP release type (e.g., nightly, ci)")
	snoProvisionCmd.Flags().StringVar(&snoFullTag, "full-ocp-tag", "", "Full OCP tag (e.g., 4.14.0-0.nightly-2023-12-14-072431)")
	snoProvisionCmd.Flags().StringVar(&snoFullImage, "full-image", "", "Full image name to use for installation")

	snoKubeconfigCmd.Flags().StringVar(&snoKCDest, "dest", "", "Local destination path (default: ~/Downloads/{env}-sno-kubeconfig)")
}

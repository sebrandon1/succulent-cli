package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
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
	Use:   cmdNameProvision,
	Short: "Provision an SNO cluster",
	Long:  `Submit an SNO provisioning request to the succulent service for the specified environment.`,
	Example: `  succulent-cli sno provision --env myenv --owner myuser --email user@example.com --ocp-tag 4.17
  succulent-cli sno provision --env myenv --owner myuser --email user@example.com --full-ocp-tag 4.17.0-0.nightly-2026-05-20-123456`,
	RunE: func(_ *cobra.Command, _ []string) error {
		owner, email, err := resolveOwnerEmail(snoOwner, snoEmail)
		if err != nil {
			return err
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
			return fmt.Errorf("submitting SNO provision request: %w", err)
		}

		fmt.Printf("SNO provision request submitted for %s\n", envName)

		return nil
	},
}

var snoKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the SNO kubeconfig for an environment",
	Example: `  succulent-cli sno kubeconfig --env myenv
  succulent-cli sno kubeconfig --env myenv --dest ./kubeconfig`,
	RunE: func(_ *cobra.Command, _ []string) error {
		data, err := sharedClient.GetSNOKubeconfig(envName)
		if err != nil {
			return fmt.Errorf("fetching SNO kubeconfig: %w", err)
		}

		dest := snoKCDest
		if dest == "" {
			dest, err = defaultDestPath(envName, "sno-kubeconfig")
			if err != nil {
				return err
			}
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			return fmt.Errorf("creating destination directory: %w", err)
		}

		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return fmt.Errorf("writing kubeconfig: %w", err)
		}

		fmt.Printf("SNO kubeconfig saved to: %s\n", dest)

		return nil
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

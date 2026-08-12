package cmd

import (
	"fmt"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	confirmSNO   bool
	dryRunSNO    bool
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
	Long: `Submit an SNO provisioning request to the succulent service for the specified environment.

Version flags (use one approach):
  --ocp-tag + --release-type    Short tag with release type (e.g., --ocp-tag 4.17 --release-type nightly)
  --full-ocp-tag                Exact build tag (e.g., 4.17.0-0.nightly-2026-05-20-123456)
  --full-image                  Full container image reference

If --full-ocp-tag or --full-image is set, --ocp-tag and --release-type are ignored.`,
	Example: `  succulent-cli sno provision --env myenv --owner myuser --email user@example.com --ocp-tag 4.17 --confirm
  succulent-cli sno provision --env myenv --owner myuser --email user@example.com --full-ocp-tag 4.17.0-0.nightly-2026-05-20-123456 --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !confirmSNO {
			return fmt.Errorf("--confirm is required to provision an SNO cluster")
		}

		if err := validateOCPTag(snoOCPTag, "--ocp-tag"); err != nil {
			return err
		}

		if err := validateOCPTag(snoFullTag, "--full-ocp-tag"); err != nil {
			return err
		}

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

		if dryRunSNO {
			printDryRun("provision SNO on", envName, req.FormValues())
			return nil
		}

		if err := sharedClient.ProvisionSNO(cmd.Context(), envName, &req); err != nil {
			return fmt.Errorf("submitting SNO provision request: %w", err)
		}

		return printResult(CommandResult{
			Status:      "submitted",
			Environment: envName,
			Message:     fmt.Sprintf("SNO provision request submitted for %s", envName),
		}, outputFormat)
	},
}

var snoKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the SNO kubeconfig for an environment",
	Example: `  succulent-cli sno kubeconfig --env myenv
  succulent-cli sno kubeconfig --env myenv --dest ./kubeconfig`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		data, err := sharedClient.GetSNOKubeconfig(cmd.Context(), envName)
		if err != nil {
			return fmt.Errorf("fetching SNO kubeconfig: %w", err)
		}

		dest, err := saveKubeconfig(data, snoKCDest, envName, "sno-kubeconfig")
		if err != nil {
			return err
		}

		fmt.Printf("SNO kubeconfig saved to: %s\n", dest)

		return nil
	},
}

func init() {
	snoProvisionCmd.Flags().StringVar(&snoOwner, "owner", "", "Username (owner)")
	snoProvisionCmd.Flags().StringVar(&snoEmail, "email", "", "Email address for notifications")
	snoProvisionCmd.Flags().StringVar(&snoOCPTag, "ocp-tag", "", "OCP version tag (e.g., 4.17); used with --release-type")
	snoProvisionCmd.Flags().StringVar(&snoRelease, "release-type", "nightly", "Release type: nightly (default) or ci; used with --ocp-tag")
	snoProvisionCmd.Flags().StringVar(&snoFullTag, "full-ocp-tag", "", "Full OCP tag; overrides --ocp-tag and --release-type")
	snoProvisionCmd.Flags().StringVar(&snoFullImage, "full-image", "", "Full container image reference; overrides all other tag flags")
	snoProvisionCmd.Flags().BoolVar(&confirmSNO, "confirm", false, "Confirm provisioning (required)")
	snoProvisionCmd.Flags().BoolVar(&dryRunSNO, "dry-run", false, "Show what would be sent without executing")

	snoKubeconfigCmd.Flags().StringVar(&snoKCDest, "dest", "", "Local destination path (default: ~/Downloads/{env}-sno-kubeconfig)")
}

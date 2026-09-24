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

If --full-ocp-tag or --full-image is set, --ocp-tag and --release-type are ignored.

When stdin is a TTY, missing --owner, --email, and --ocp-tag are prompted instead of failing immediately.`,
	Example: `  succulent-cli sno provision --env myenv --owner myuser --email user@example.com --ocp-tag 4.17 --confirm
  succulent-cli sno provision --env myenv --owner myuser --email user@example.com --full-ocp-tag 4.17.0-0.nightly-2026-05-20-123456 --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !confirmSNO {
			return fmt.Errorf("--confirm is required to provision an SNO cluster (use --dry-run to preview)")
		}

		ocpTag := snoOCPTag
		if ocpTag == "" && snoFullTag == "" && snoFullImage == "" {
			var err error
			ocpTag, err = promptOptional(ocpTag)
			if err != nil {
				return err
			}
		}

		if err := validateOCPTag(ocpTag, "--ocp-tag"); err != nil {
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
			OCPTag:        ocpTag,
			ReleaseType:   snoRelease,
			FullOCPTag:    snoFullTag,
			FullImageName: snoFullImage,
		}

		return runForEnvironmentTargets(func(target string) (string, error) {
			if dryRunSNO {
				printDryRun("provision SNO on", target, req.FormValues())
				return fmt.Sprintf("[dry-run] Would provision SNO on %s", target), nil
			}
			if err := sharedClient.ProvisionSNO(cmd.Context(), target, &req); err != nil {
				return "", fmt.Errorf("submitting SNO provision request: %w; verify env exists with: succulent-cli list", err)
			}
			return fmt.Sprintf("SNO provision request submitted for %s", target), nil
		}, func(target, message string) error {
			return printResult(CommandResult{Status: "submitted", Environment: target, Message: message}, outputFormat)
		}, dryRunSNO)
	},
}

var snoKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the SNO kubeconfig for an environment",
	Example: `  succulent-cli sno kubeconfig --env myenv
  succulent-cli sno kubeconfig --env myenv --dest ./kubeconfig`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		targets := currentEnvironmentTargets()
		if err := validateBatchDestination(snoKCDest, targets); err != nil {
			return err
		}
		return runForEnvironmentTargets(func(target string) (string, error) {
			data, err := sharedClient.GetSNOKubeconfig(cmd.Context(), target)
			if err != nil {
				return "", fmt.Errorf("fetching SNO kubeconfig: %w", err)
			}
			destPath, err := environmentDestination(snoKCDest, target, targets)
			if err != nil {
				return "", err
			}
			dest, err := saveKubeconfig(data, destPath, target, "sno-kubeconfig")
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("SNO kubeconfig saved to: %s", dest), nil
		}, func(_ string, message string) error {
			fmt.Println(message)
			return nil
		}, false)
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

	snoKubeconfigCmd.Flags().StringVar(&snoKCDest, "dest", "", "Local destination path (default: ~/Downloads/succulent/{env}/sno-kubeconfig)")
}

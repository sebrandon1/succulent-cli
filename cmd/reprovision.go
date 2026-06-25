package cmd

import (
	"fmt"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	confirmReprovision   bool
	reprovEmail          string
	reprovOwner          string
	reprovTag            string
	reprovVersion        string
	reprovOpenshiftImage string
	reprovDiskSize       string
	reprovVirtualWorkers string
	reprovAddlWorkers    string
	reprovEndDate        string
	reprovKcliParams     string
)

var reprovisionCmd = &cobra.Command{
	Use:   "reprovision",
	Short: "Reprovision an MNO cluster",
	Long:  `Submit a reprovisioning request to the succulent service for the specified environment.`,
	Example: `  succulent-cli reprovision --env myenv --email user@example.com --owner myuser --ocp-tag 4.17 --confirm
  succulent-cli reprovision --env myenv --email user@example.com --owner myuser --ocp-tag 4.18 --release-type ci --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !confirmReprovision {
			return fmt.Errorf("--confirm is required to reprovision an environment")
		}

		owner, email, err := resolveOwnerEmail(reprovOwner, reprovEmail)
		if err != nil {
			return err
		}

		req := lib.ReprovisionRequest{
			Email:             email,
			Owner:             owner,
			Tag:               reprovTag,
			Version:           reprovVersion,
			OpenshiftImage:    reprovOpenshiftImage,
			DiskSize:          reprovDiskSize,
			VirtualWorkers:    reprovVirtualWorkers,
			AdditionalWorkers: reprovAddlWorkers,
			EndDate:           reprovEndDate,
			KcliParams:        reprovKcliParams,
		}

		if err := sharedClient.Reprovision(cmd.Context(), envName, &req); err != nil {
			return fmt.Errorf("submitting reprovision request: %w", err)
		}

		return printResult(CommandResult{
			Status:      "submitted",
			Environment: envName,
			Message:     fmt.Sprintf("Reprovision request submitted for %s (OCP %s %s)", envName, reprovTag, reprovVersion),
		}, outputFormat)
	},
}

func init() {
	reprovisionCmd.Flags().StringVar(&reprovEmail, "email", "", "Email address for notifications")
	reprovisionCmd.Flags().StringVar(&reprovOwner, "owner", "", "Username (owner)")
	reprovisionCmd.Flags().StringVar(&reprovTag, "ocp-tag", "", "OCP version tag (e.g., 4.17, 4.18)")
	reprovisionCmd.Flags().StringVar(&reprovTag, "tag", "", "OCP version tag (deprecated: use --ocp-tag)")
	reprovisionCmd.Flags().StringVar(&reprovVersion, "release-type", "nightly", "Release type (e.g., nightly, ci)")
	reprovisionCmd.Flags().StringVar(&reprovVersion, "version", "nightly", "Release type (deprecated: use --release-type)")

	_ = reprovisionCmd.Flags().MarkDeprecated("tag", "use --ocp-tag instead")
	_ = reprovisionCmd.Flags().MarkDeprecated("version", "use --release-type instead")
	reprovisionCmd.Flags().StringVar(&reprovOpenshiftImage, "openshift-image", "", "Full OpenShift image URL")
	reprovisionCmd.Flags().StringVar(&reprovDiskSize, "disk-size", "50", "Disk size in GB")
	reprovisionCmd.Flags().StringVar(&reprovVirtualWorkers, "virtual-workers", "true", "Enable virtual workers")
	reprovisionCmd.Flags().StringVar(&reprovAddlWorkers, "additional-workers", "false", "Additional baremetal workers (false to disable, or comma-separated hostnames)")
	reprovisionCmd.Flags().StringVar(&reprovEndDate, "end-date", "", "End date for the environment")
	reprovisionCmd.Flags().StringVar(&reprovKcliParams, "kcli-params", "", "Additional kcli parameters (key:value format)")

	reprovisionCmd.Flags().BoolVar(&confirmReprovision, "confirm", false, "Confirm reprovisioning (required)")

	cobra.CheckErr(reprovisionCmd.MarkFlagRequired("ocp-tag"))

	rootCmd.AddCommand(reprovisionCmd)
}

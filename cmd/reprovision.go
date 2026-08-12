package cmd

import (
	"fmt"
	"time"

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

	dryRunReprovision bool

	// Deprecated flag variables
	reprovTagDeprecated     string
	reprovVersionDeprecated string
)

var reprovisionCmd = &cobra.Command{
	Use:   "reprovision",
	Short: "Reprovision an MNO cluster",
	Long: `Submit a reprovisioning request to the succulent service for the specified environment.

Use --ocp-tag with --release-type (default: nightly) to specify the OCP version.`,
	Example: `  succulent-cli reprovision --env myenv --email user@example.com --owner myuser --ocp-tag 4.17 --confirm
  succulent-cli reprovision --env myenv --email user@example.com --owner myuser --ocp-tag 4.18 --release-type ci --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !confirmReprovision {
			return fmt.Errorf("--confirm is required to reprovision an environment (use --dry-run to preview)")
		}

		// Prefer new flags, fall back to deprecated if new flags not set
		if reprovTag == "" && reprovTagDeprecated != "" {
			reprovTag = reprovTagDeprecated
		}
		if reprovVersion == "nightly" && reprovVersionDeprecated != "nightly" {
			reprovVersion = reprovVersionDeprecated
		}

		if err := validateOCPTag(reprovTag, "--ocp-tag"); err != nil {
			return err
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

		if dryRunReprovision {
			printDryRun("reprovision", envName, req.FormValues())
			return nil
		}

		// Reprovision can take several minutes, use 5-minute timeout
		client := sharedClient.WithTimeout(5 * time.Minute)
		if err := client.Reprovision(cmd.Context(), envName, &req); err != nil {
			return fmt.Errorf("submitting reprovision request: %w; verify env exists with: succulent-cli list", err)
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
	reprovisionCmd.Flags().StringVar(&reprovTag, "ocp-tag", "", "OCP version tag (e.g., 4.17, 5.0); required")
	reprovisionCmd.Flags().StringVar(&reprovTagDeprecated, "tag", "", "OCP version tag (deprecated: use --ocp-tag)")
	reprovisionCmd.Flags().StringVar(&reprovVersion, "release-type", "nightly", "Release type: nightly (default) or ci; used with --ocp-tag")
	reprovisionCmd.Flags().StringVar(&reprovVersionDeprecated, "version", "nightly", "Release type (deprecated: use --release-type)")

	_ = reprovisionCmd.Flags().MarkDeprecated("tag", "use --ocp-tag instead")
	_ = reprovisionCmd.Flags().MarkDeprecated("version", "use --release-type instead")
	reprovisionCmd.Flags().StringVar(&reprovOpenshiftImage, "openshift-image", "", "Full OpenShift image URL")
	reprovisionCmd.Flags().StringVar(&reprovDiskSize, "disk-size", "50", "Disk size in GB")
	reprovisionCmd.Flags().StringVar(&reprovVirtualWorkers, "virtual-workers", "true", "Enable virtual workers")
	reprovisionCmd.Flags().StringVar(&reprovAddlWorkers, "additional-workers", "false", "Additional baremetal workers (false to disable, or comma-separated hostnames)")
	reprovisionCmd.Flags().StringVar(&reprovEndDate, "end-date", "", "End date for the environment")
	reprovisionCmd.Flags().StringVar(&reprovKcliParams, "kcli-params", "", "Additional kcli parameters (key:value format)")

	reprovisionCmd.Flags().BoolVar(&confirmReprovision, "confirm", false, "Confirm reprovisioning (required)")
	reprovisionCmd.Flags().BoolVar(&dryRunReprovision, "dry-run", false, "Show what would be sent without executing")

	cobra.CheckErr(reprovisionCmd.MarkFlagRequired("ocp-tag"))

	rootCmd.AddCommand(reprovisionCmd)
}

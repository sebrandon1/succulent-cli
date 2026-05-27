package cmd

import (
	"fmt"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
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
	Example: `  succulent-cli reprovision --env myenv --email user@example.com --owner myuser --tag 4.17
  succulent-cli reprovision --env myenv --email user@example.com --owner myuser --tag 4.18 --version ci`,
	RunE: func(_ *cobra.Command, _ []string) error {
		email := reprovEmail
		if email == "" {
			email = viper.GetString("default_email")
		}

		owner := reprovOwner
		if owner == "" {
			owner = viper.GetString("default_owner")
		}

		if email == "" {
			return fmt.Errorf("--email is required (or set default_email in config)")
		}

		if owner == "" {
			return fmt.Errorf("--owner is required (or set default_owner in config)")
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

		if err := sharedClient.Reprovision(envName, &req); err != nil {
			return fmt.Errorf("submitting reprovision request: %w", err)
		}

		fmt.Printf("Reprovision request submitted for %s (OCP %s %s)\n", envName, reprovTag, reprovVersion)

		return nil
	},
}

func init() {
	reprovisionCmd.Flags().StringVar(&reprovEmail, "email", "", "Email address for notifications")
	reprovisionCmd.Flags().StringVar(&reprovOwner, "owner", "", "Username (owner)")
	reprovisionCmd.Flags().StringVar(&reprovTag, "tag", "", "OCP version tag (e.g., 4.17, 4.18)")
	reprovisionCmd.Flags().StringVar(&reprovVersion, "version", "nightly", "Release version (e.g., nightly, ci)")
	reprovisionCmd.Flags().StringVar(&reprovOpenshiftImage, "openshift-image", "", "Full OpenShift image URL")
	reprovisionCmd.Flags().StringVar(&reprovDiskSize, "disk-size", "50", "Disk size in GB")
	reprovisionCmd.Flags().StringVar(&reprovVirtualWorkers, "virtual-workers", "true", "Enable virtual workers")
	reprovisionCmd.Flags().StringVar(&reprovAddlWorkers, "additional-workers", "", "Comma-separated extra baremetal worker names")
	reprovisionCmd.Flags().StringVar(&reprovEndDate, "end-date", "", "End date for the environment")
	reprovisionCmd.Flags().StringVar(&reprovKcliParams, "kcli-params", "", "Additional kcli parameters (key:value format)")

	cobra.CheckErr(reprovisionCmd.MarkFlagRequired("tag"))

	rootCmd.AddCommand(reprovisionCmd)
}

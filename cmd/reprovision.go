package cmd

import (
	"fmt"
	"os"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
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
	Long: `Submit a reprovisioning request to the succulent service for the specified
environment. This POSTs form data to the /exposeform/{env} endpoint.`,
	Run: func(_ *cobra.Command, _ []string) {
		if envName == "" {
			fmt.Println("Error: --env is required")
			os.Exit(1)
		}

		client := lib.NewClient(succulentURL, !verifySSL)

		req := lib.ReprovisionRequest{
			Email:             reprovEmail,
			Owner:             reprovOwner,
			Tag:               reprovTag,
			Version:           reprovVersion,
			OpenshiftImage:    reprovOpenshiftImage,
			DiskSize:          reprovDiskSize,
			VirtualWorkers:    reprovVirtualWorkers,
			AdditionalWorkers: reprovAddlWorkers,
			EndDate:           reprovEndDate,
			KcliParams:        reprovKcliParams,
		}

		if err := client.Reprovision(envName, &req); err != nil {
			fmt.Printf("Error submitting reprovision request: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Reprovision request submitted for %s (OCP %s %s)\n", envName, reprovTag, reprovVersion)
	},
}

func init() {
	reprovisionCmd.Flags().StringVar(&reprovEmail, "email", "", "Email address for notifications")
	reprovisionCmd.Flags().StringVar(&reprovOwner, "owner", "", "Kerberos username (owner)")
	reprovisionCmd.Flags().StringVar(&reprovTag, "tag", "", "OCP version tag (e.g., 4.17, 4.18)")
	reprovisionCmd.Flags().StringVar(&reprovVersion, "version", "nightly", "Release version (e.g., nightly, ci)")
	reprovisionCmd.Flags().StringVar(&reprovOpenshiftImage, "openshift-image", "", "Full OpenShift image URL")
	reprovisionCmd.Flags().StringVar(&reprovDiskSize, "disk-size", "50", "Disk size in GB")
	reprovisionCmd.Flags().StringVar(&reprovVirtualWorkers, "virtual-workers", "true", "Enable virtual workers")
	reprovisionCmd.Flags().StringVar(&reprovAddlWorkers, "additional-workers", "", "Comma-separated extra baremetal worker names")
	reprovisionCmd.Flags().StringVar(&reprovEndDate, "end-date", "", "End date for the environment")
	reprovisionCmd.Flags().StringVar(&reprovKcliParams, "kcli-params", "", "Additional kcli parameters (key:value format)")

	markFlagRequired(reprovisionCmd.MarkFlagRequired("email"))
	markFlagRequired(reprovisionCmd.MarkFlagRequired("owner"))
	markFlagRequired(reprovisionCmd.MarkFlagRequired("tag"))

	rootCmd.AddCommand(reprovisionCmd)
}

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
	Use:   "provision",
	Short: "Provision an SNO cluster",
	Long: `Submit an SNO provisioning request to the succulent service for
the specified environment.`,
	Run: func(_ *cobra.Command, _ []string) {
		if envName == "" {
			fmt.Println("Error: --env is required")
			os.Exit(1)
		}

		client := lib.NewClient(succulentURL, !verifySSL)

		req := lib.SNOProvisionRequest{
			Owner:         snoOwner,
			Email:         snoEmail,
			OCPTag:        snoOCPTag,
			ReleaseType:   snoRelease,
			FullOCPTag:    snoFullTag,
			FullImageName: snoFullImage,
		}

		if err := client.ProvisionSNO(envName, &req); err != nil {
			fmt.Printf("Error submitting SNO provision request: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("SNO provision request submitted for %s\n", envName)
	},
}

var snoKubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Download the SNO kubeconfig for an environment",
	Run: func(_ *cobra.Command, _ []string) {
		if envName == "" {
			fmt.Println("Error: --env is required")
			os.Exit(1)
		}

		client := lib.NewClient(succulentURL, !verifySSL)

		data, err := client.GetSNOKubeconfig(envName)
		if err != nil {
			fmt.Printf("Error fetching SNO kubeconfig: %v\n", err)
			os.Exit(1)
		}

		dest := snoKCDest
		if dest == "" {
			home, _ := os.UserHomeDir()
			dest = filepath.Join(home, defaultDestDir, envName+"-sno-kubeconfig")
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
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
	snoProvisionCmd.Flags().StringVar(&snoOwner, "owner", "", "Kerberos username (owner)")
	snoProvisionCmd.Flags().StringVar(&snoEmail, "email", "", "Email address for notifications")
	snoProvisionCmd.Flags().StringVar(&snoOCPTag, "ocp-tag", "", "OCP tag (e.g., 4.17)")
	snoProvisionCmd.Flags().StringVar(&snoRelease, "release-type", "nightly", "OCP release type (e.g., nightly, ci)")
	snoProvisionCmd.Flags().StringVar(&snoFullTag, "full-ocp-tag", "", "Full OCP tag (e.g., 4.14.0-0.nightly-2023-12-14-072431)")
	snoProvisionCmd.Flags().StringVar(&snoFullImage, "full-image", "", "Full image name to use for installation")

	markFlagRequired(snoProvisionCmd.MarkFlagRequired("owner"))
	markFlagRequired(snoProvisionCmd.MarkFlagRequired("email"))

	snoKubeconfigCmd.Flags().StringVar(&snoKCDest, "dest", "", "Local destination path (default: ~/Downloads/{env}-sno-kubeconfig)")
}

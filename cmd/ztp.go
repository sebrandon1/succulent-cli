package cmd

import (
	"fmt"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	confirmZTP      bool
	dryRunZTP       bool
	ztpOwner        string
	ztpEmail        string
	ztpSNOTag       string
	ztpSNORelease   string
	ztpSNOFullTag   string
	ztpSpokeTag     string
	ztpSpokeRelease string
	ztpSpokeFullTag string
	ztpType         string
	ztpStopBefore   bool
	ztpVMMasters    string
	ztpBMMasters    string
	ztpBMWorkers    string
	ztpVMWorkers    string
	ztpKCChoice     string
	ztpKCDest       string
)

var ztpCmd = &cobra.Command{
	Use:   "ztp",
	Short: "ZTP cluster management commands",
}

var ztpProvisionCmd = &cobra.Command{
	Use:   cmdNameProvision,
	Short: "Provision a ZTP hub and spoke cluster",
	Long: `Submit a ZTP provisioning request for the specified environment.

Version flags for hub and spoke (use one approach per cluster):
  --sno-tag + --sno-release       Short tag with release type for the hub cluster
  --sno-full-tag                  Exact build tag for the hub cluster
  --spoke-tag + --spoke-release   Short tag with release type for spoke clusters
  --spoke-full-tag                Exact build tag for spoke clusters

Full tags override their corresponding short tag + release type flags.`,
	Example: `  succulent-cli ztp provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --spoke-tag 4.17 --confirm
  succulent-cli ztp provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --spoke-tag 4.17 --type mno --vm-masters 3 --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !confirmZTP {
			return fmt.Errorf("--confirm is required to provision a ZTP cluster (use --dry-run to preview)")
		}

		for _, v := range []struct{ tag, flag string }{
			{ztpSNOTag, "--sno-tag"}, {ztpSNOFullTag, "--sno-full-tag"},
			{ztpSpokeTag, "--spoke-tag"}, {ztpSpokeFullTag, "--spoke-full-tag"},
		} {
			if err := validateOCPTag(v.tag, v.flag); err != nil {
				return err
			}
		}

		owner, email, err := resolveOwnerEmail(ztpOwner, ztpEmail)
		if err != nil {
			return err
		}

		req := lib.ZTPRequest{
			Owner:                owner,
			Email:                email,
			SNOTag:               ztpSNOTag,
			SNORelease:           ztpSNORelease,
			SNOFullTag:           ztpSNOFullTag,
			ZTPTag:               ztpSpokeTag,
			ZTPRelease:           ztpSpokeRelease,
			ZTPFullTag:           ztpSpokeFullTag,
			ZTPType:              ztpType,
			StopBeforeDeployment: ztpStopBefore,
			VMMastersCount:       ztpVMMasters,
			BMMastersHosts:       ztpBMMasters,
			BMWorkersHosts:       ztpBMWorkers,
			VMWorkersCount:       ztpVMWorkers,
		}

		if dryRunZTP {
			printDryRun("provision ZTP on", envName, req.FormValues())
			return nil
		}

		if err := sharedClient.ProvisionZTP(cmd.Context(), envName, &req); err != nil {
			return fmt.Errorf("submitting ZTP provision request: %w; verify env exists with: succulent-cli list", err)
		}

		return printResult(CommandResult{
			Status:      "submitted",
			Environment: envName,
			Message:     fmt.Sprintf("ZTP provision request submitted for %s (type: %s)", envName, ztpType),
		}, outputFormat)
	},
}

var ztpKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the ZTP kubeconfig",
	Example: `  succulent-cli ztp kubeconfig --env myenv --choice management
  succulent-cli ztp kubeconfig --env myenv --choice spoke`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		data, err := sharedClient.GetZTPKubeconfig(cmd.Context(), envName, ztpKCChoice)
		if err != nil {
			return fmt.Errorf("fetching ZTP kubeconfig: %w", err)
		}

		dest, err := saveKubeconfig(data, ztpKCDest, envName, "ztp-"+ztpKCChoice+"-kubeconfig")
		if err != nil {
			return err
		}

		fmt.Printf("ZTP %s kubeconfig saved to: %s\n", ztpKCChoice, dest)

		return nil
	},
}

func init() {
	ztpProvisionCmd.Flags().StringVar(&ztpOwner, "owner", "", "Username (owner)")
	ztpProvisionCmd.Flags().StringVar(&ztpEmail, "email", "", "Email address for notifications")
	ztpProvisionCmd.Flags().StringVar(&ztpSNOTag, "sno-tag", "", "Hub cluster OCP tag (e.g., 4.17); used with --sno-release")
	ztpProvisionCmd.Flags().StringVar(&ztpSNORelease, "sno-release", "nightly", "Hub release type: nightly (default) or ci; used with --sno-tag")
	ztpProvisionCmd.Flags().StringVar(&ztpSNOFullTag, "sno-full-tag", "", "Hub cluster full OCP tag; overrides --sno-tag and --sno-release")
	ztpProvisionCmd.Flags().StringVar(&ztpSpokeTag, "spoke-tag", "", "Spoke cluster OCP tag (e.g., 4.17); used with --spoke-release")
	ztpProvisionCmd.Flags().StringVar(&ztpSpokeRelease, "spoke-release", "nightly", "Spoke release type: nightly (default) or ci; used with --spoke-tag")
	ztpProvisionCmd.Flags().StringVar(&ztpSpokeFullTag, "spoke-full-tag", "", "Spoke cluster full OCP tag; overrides --spoke-tag and --spoke-release")
	ztpProvisionCmd.Flags().StringVar(&ztpType, "type", "sno", "ZTP type: sno or mno")
	ztpProvisionCmd.Flags().BoolVar(&ztpStopBefore, "stop-before-deployment", false, "Stop before actual spoke deployment for manual GitOps changes")
	ztpProvisionCmd.Flags().StringVar(&ztpVMMasters, "vm-masters", "3", "Number of VM masters (MNO only)")
	ztpProvisionCmd.Flags().StringVar(&ztpBMMasters, "bm-masters", "", "Comma-separated baremetal master hosts (MNO only)")
	ztpProvisionCmd.Flags().StringVar(&ztpBMWorkers, "bm-workers", "", "Comma-separated baremetal worker hosts")
	ztpProvisionCmd.Flags().StringVar(&ztpVMWorkers, "vm-workers", "1", "Number of VM workers")
	ztpProvisionCmd.Flags().BoolVar(&confirmZTP, "confirm", false, "Confirm provisioning (required)")
	ztpProvisionCmd.Flags().BoolVar(&dryRunZTP, "dry-run", false, "Show what would be sent without executing")

	ztpKubeconfigCmd.Flags().StringVar(&ztpKCChoice, "choice", "", "Kubeconfig type: management or spoke")
	ztpKubeconfigCmd.Flags().StringVar(&ztpKCDest, "dest", "", "Local destination path")
	cobra.CheckErr(ztpKubeconfigCmd.MarkFlagRequired("choice"))

	ztpCmd.AddCommand(ztpProvisionCmd)
	ztpCmd.AddCommand(ztpKubeconfigCmd)

	rootCmd.AddCommand(ztpCmd)
}

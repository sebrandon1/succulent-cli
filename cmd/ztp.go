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

Full tags override their corresponding short tag + release type flags.

When stdin is a TTY, missing --owner and --email are prompted instead of failing immediately.`,
	Example: `  succulent-cli ztp provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --spoke-tag 4.17 --confirm
  succulent-cli ztp provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --spoke-tag 4.17 --type mno --vm-masters 3 --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !confirmZTP {
			return fmt.Errorf("--confirm is required to provision a ZTP cluster (use --dry-run to preview)")
		}
		if err := validateProvisionWatchFlags(dryRunZTP); err != nil {
			return err
		}

		for _, v := range []struct{ tag, flag string }{
			{ztpSNOTag, "--sno-tag"}, {ztpSNOFullTag, "--sno-full-tag"},
			{ztpSpokeTag, "--spoke-tag"}, {ztpSpokeFullTag, "--spoke-full-tag"},
		} {
			if err := validateOCPTag(v.tag, v.flag); err != nil {
				return err
			}
		}

		if err := validateNumericFlag(ztpVMMasters, "vm-masters"); err != nil {
			return err
		}

		if err := validateNumericFlag(ztpVMWorkers, "vm-workers"); err != nil {
			return err
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

		return runForEnvironmentTargets(func(target string) (string, error) {
			if dryRunZTP {
				printDryRun("provision ZTP on", target, req.FormValues())
				return fmt.Sprintf("[dry-run] Would provision ZTP on %s", target), nil
			}
			if err := runAuditedOperationForEnvironment(target, "ztp provision", auditParameters(
				"owner", owner, "type", ztpType, "sno_tag", ztpSNOTag, "sno_full_tag", ztpSNOFullTag,
				"spoke_tag", ztpSpokeTag, "spoke_full_tag", ztpSpokeFullTag,
			), func() error {
				return sharedClient.ProvisionZTP(cmd.Context(), target, &req)
			}); err != nil {
				return "", fmt.Errorf("submitting ZTP provision request: %w; verify env exists with: succulent-cli list", err)
			}
			message := fmt.Sprintf("ZTP provision request submitted for %s (type: %s)", target, ztpType)
			return waitForProvisioning(cmd, target, "ZTP provision", message)
		}, func(target, message string) error {
			return printResult(CommandResult{Status: provisionResultStatus(), Environment: target, Message: message}, outputFormat)
		}, dryRunZTP)
	},
}

var ztpKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the ZTP kubeconfig",
	Example: `  succulent-cli ztp kubeconfig --env myenv --choice management
  succulent-cli ztp kubeconfig --env myenv --choice spoke`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		targets := currentEnvironmentTargets()
		if err := validateBatchDestination(ztpKCDest, targets); err != nil {
			return err
		}
		return runForEnvironmentTargets(func(target string) (string, error) {
			destPath, err := environmentDestination(ztpKCDest, target, targets)
			if err != nil {
				return "", err
			}
			var dest string
			err = runAuditedOperationForEnvironment(target, "ztp kubeconfig", auditParameters(
				"choice", ztpKCChoice, "destination", destPath,
			), func() error {
				data, err := sharedClient.GetZTPKubeconfig(cmd.Context(), target, ztpKCChoice)
				if err != nil {
					return fmt.Errorf("fetching ZTP kubeconfig: %w", err)
				}
				dest, err = saveKubeconfig(data, destPath, target, "ztp-"+ztpKCChoice+"-kubeconfig")
				return err
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("ZTP %s kubeconfig saved to: %s", ztpKCChoice, dest), nil
		}, func(_ string, message string) error {
			fmt.Println(message)
			return nil
		}, false)
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
	addProvisionWatchFlags(ztpProvisionCmd)

	ztpKubeconfigCmd.Flags().StringVar(&ztpKCChoice, "choice", "", "Kubeconfig type: management or spoke")
	ztpKubeconfigCmd.Flags().StringVar(&ztpKCDest, "dest", "", "Local destination path (default: ~/Downloads/succulent/{env}/ztp-{choice}-kubeconfig)")
	cobra.CheckErr(ztpKubeconfigCmd.MarkFlagRequired("choice"))

	ztpCmd.AddCommand(ztpProvisionCmd)
	ztpCmd.AddCommand(ztpKubeconfigCmd)

	rootCmd.AddCommand(ztpCmd)
}

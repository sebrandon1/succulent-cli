package cmd

import (
	"fmt"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	confirmHS       bool
	dryRunHS        bool
	hsOwner         string
	hsEmail         string
	hsSNOTag        string
	hsSNORelease    string
	hsSNOFullTag    string
	hsHCPTag        string
	hsHCPRelease    string
	hsHCPFullTag    string
	hsVMWorkers     string
	hsImageOverride string
	hsKCChoice      string
	hsKCDest        string
)

var hypershiftCmd = &cobra.Command{
	Use:   "hypershift",
	Short: "Hypershift cluster management commands",
}

var hsProvisionCmd = &cobra.Command{
	Use:   cmdNameProvision,
	Short: "Provision a Hypershift hosted cluster",
	Long: `Submit a Hypershift provisioning request for the specified environment.

Version flags for management and hosted clusters (use one approach per cluster):
  --sno-tag + --sno-release     Short tag with release type for the management (SNO) cluster
  --sno-full-tag                Exact build tag for the management cluster
  --hcp-tag + --hcp-release     Short tag with release type for the hosted cluster
  --hcp-full-tag                Exact build tag for the hosted cluster

Full tags override their corresponding short tag + release type flags.`,
	Example: `  succulent-cli hypershift provision --env myenv --owner myuser --email user@example.com --sno-tag 4.17 --hcp-tag 4.17 --confirm
  succulent-cli hypershift provision --env myenv --owner myuser --email user@example.com --sno-tag 4.16 --hcp-full-tag 4.15.0-rc.1 --vm-workers 2 --confirm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !confirmHS {
			return fmt.Errorf("--confirm is required to provision a Hypershift cluster")
		}

		for _, v := range []struct{ tag, flag string }{
			{hsSNOTag, "--sno-tag"}, {hsSNOFullTag, "--sno-full-tag"},
			{hsHCPTag, "--hcp-tag"}, {hsHCPFullTag, "--hcp-full-tag"},
		} {
			if err := validateOCPTag(v.tag, v.flag); err != nil {
				return err
			}
		}

		owner, email, err := resolveOwnerEmail(hsOwner, hsEmail)
		if err != nil {
			return err
		}

		req := lib.HypershiftRequest{
			Owner:          owner,
			Email:          email,
			SNOTag:         hsSNOTag,
			SNORelease:     hsSNORelease,
			SNOFullTag:     hsSNOFullTag,
			HCPTag:         hsHCPTag,
			HCPRelease:     hsHCPRelease,
			HCPFullTag:     hsHCPFullTag,
			VMWorkersCount: hsVMWorkers,
			ImageOverride:  hsImageOverride,
		}

		if dryRunHS {
			printDryRun("provision Hypershift on", envName, req.FormValues())
			return nil
		}

		if err := sharedClient.ProvisionHypershift(cmd.Context(), envName, &req); err != nil {
			return fmt.Errorf("submitting Hypershift provision request: %w", err)
		}

		return printResult(CommandResult{
			Status:      "submitted",
			Environment: envName,
			Message:     fmt.Sprintf("Hypershift provision request submitted for %s", envName),
		}, outputFormat)
	},
}

var hsKubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Download the Hypershift kubeconfig",
	Example: `  succulent-cli hypershift kubeconfig --env myenv --choice management
  succulent-cli hypershift kubeconfig --env myenv --choice hosted`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		data, err := sharedClient.GetHypershiftKubeconfig(cmd.Context(), envName, hsKCChoice)
		if err != nil {
			return fmt.Errorf("fetching Hypershift kubeconfig: %w", err)
		}

		dest, err := saveKubeconfig(data, hsKCDest, envName, "hypershift-"+hsKCChoice+"-kubeconfig")
		if err != nil {
			return err
		}

		fmt.Printf("Hypershift %s kubeconfig saved to: %s\n", hsKCChoice, dest)

		return nil
	},
}

func init() {
	hsProvisionCmd.Flags().StringVar(&hsOwner, "owner", "", "Username (owner)")
	hsProvisionCmd.Flags().StringVar(&hsEmail, "email", "", "Email address for notifications")
	hsProvisionCmd.Flags().StringVar(&hsSNOTag, "sno-tag", "", "Management cluster OCP tag (e.g., 4.17); used with --sno-release")
	hsProvisionCmd.Flags().StringVar(&hsSNORelease, "sno-release", "nightly", "Management release type: nightly (default) or ci; used with --sno-tag")
	hsProvisionCmd.Flags().StringVar(&hsSNOFullTag, "sno-full-tag", "", "Management cluster full OCP tag; overrides --sno-tag and --sno-release")
	hsProvisionCmd.Flags().StringVar(&hsHCPTag, "hcp-tag", "", "Hosted cluster OCP tag (e.g., 4.17); used with --hcp-release")
	hsProvisionCmd.Flags().StringVar(&hsHCPRelease, "hcp-release", "nightly", "Hosted release type: nightly (default) or ci; used with --hcp-tag")
	hsProvisionCmd.Flags().StringVar(&hsHCPFullTag, "hcp-full-tag", "", "Hosted cluster full OCP tag; overrides --hcp-tag and --hcp-release")
	hsProvisionCmd.Flags().StringVar(&hsVMWorkers, "vm-workers", "0", "Number of VM workers for hosted cluster")
	hsProvisionCmd.Flags().StringVar(&hsImageOverride, "image-override", "", "Hypershift operator image override")
	hsProvisionCmd.Flags().BoolVar(&confirmHS, "confirm", false, "Confirm provisioning (required)")
	hsProvisionCmd.Flags().BoolVar(&dryRunHS, "dry-run", false, "Show what would be sent without executing")

	hsKubeconfigCmd.Flags().StringVar(&hsKCChoice, "choice", "", "Kubeconfig type: management or hosted")
	hsKubeconfigCmd.Flags().StringVar(&hsKCDest, "dest", "", "Local destination path")
	cobra.CheckErr(hsKubeconfigCmd.MarkFlagRequired("choice"))

	hypershiftCmd.AddCommand(hsProvisionCmd)
	hypershiftCmd.AddCommand(hsKubeconfigCmd)

	rootCmd.AddCommand(hypershiftCmd)
}

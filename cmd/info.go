package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var infoNoCache bool

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get cluster node information for an environment",
	Long:  `Fetch the infoplan page for the specified environment and output structured node information.`,
	Example: `  succulent-cli get info --env myenv
  succulent-cli get info --env myenv --output table
  succulent-cli get info --env myenv --no-cache`,
	RunE: func(_ *cobra.Command, _ []string) error {
		var info *lib.ClusterInfo

		if !infoNoCache && sharedCache != nil {
			if cached, ok := sharedCache.GetInfo(envName); ok {
				fmt.Fprintln(os.Stderr, "(cached)")
				info = cached
			}
		}

		if info == nil {
			var err error

			info, err = sharedClient.GetInfoPlan(envName)
			if err != nil {
				return fmt.Errorf("fetching info: %w", err)
			}

			if sharedCache != nil {
				sharedCache.SetInfo(envName, info)
			}
		}

		if outputFormat == "table" {
			printInfoTable(info)

			return nil
		}

		return printJSON(info)
	},
}

func printInfoTable(info *lib.ClusterInfo) {
	if info.PlanName != "" {
		fmt.Printf("Plan: %s", info.PlanName)

		if info.Client != "" {
			fmt.Printf("  Client: %s", info.Client)
		}

		if info.CreationDate != "" {
			fmt.Printf("  Created: %s", info.CreationDate)
		}

		fmt.Println()
	}

	if info.InstallerIP != "" {
		fmt.Printf("Installer IP: %s\n", info.InstallerIP)
	}

	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tIP")

	for _, node := range info.Nodes {
		ip := node.IP
		if ip == "" {
			ip = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", node.Name, node.Status, ip)
	}

	w.Flush()
}

func init() {
	infoCmd.Flags().BoolVar(&infoNoCache, "no-cache", false, "Bypass the info cache")
}

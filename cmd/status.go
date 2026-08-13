package cmd

import (
	"fmt"
	"strings"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

const (
	iconStatusUp         = "✅"
	iconStatusDown       = "❌"
	iconStatusPartial    = "⚠️"
	headerSeparatorWidth = 60
	statusLabelReady     = "Ready"
	statusLabelDown      = "Down"
	statusLabelPartial   = "Partial"
)

var nodeTypeOrder = []string{lib.NodeTypeInstaller, lib.NodeTypeBootstrap, lib.NodeTypeMaster, lib.NodeTypeWorker}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show environment status in a user-friendly format",
	Long: `Display environment status with a more narrative presentation than 'get info'.
Shows cluster information, node states, and overall health in an easy-to-read format.`,
	Example: `  succulent-cli status --env myenv
  succulent-cli status --env myenv --output json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		info, err := sharedClient.GetInfoPlan(cmd.Context(), envName)
		if err != nil {
			return fmt.Errorf("fetching environment status: %w", err)
		}

		if outputFormat == "json" {
			return printJSON(info)
		}

		printStatus(info)

		return nil
	},
}

func printStatus(info *lib.ClusterInfo) {
	printStatusHeader(info)
	printClusterInfo(info)

	if len(info.Nodes) == 0 {
		fmt.Println("Nodes: None provisioned")
		fmt.Println()
		return
	}

	nodesUp, nodesByType := categorizeNodes(info.Nodes)
	printNodeSummary(len(info.Nodes), nodesUp)
	printNodeDetails(nodesByType)
}

func printStatusHeader(info *lib.ClusterInfo) {
	fmt.Printf("Environment: %s\n", info.Environment)
	fmt.Println(strings.Repeat("=", headerSeparatorWidth))
	fmt.Println()
}

func printClusterInfo(info *lib.ClusterInfo) {
	fmt.Println("Cluster Information:")
	if info.PlanName != "" {
		fmt.Printf("  Plan Name:      %s\n", info.PlanName)
	}
	if info.Client != "" {
		fmt.Printf("  Client:         %s\n", info.Client)
	}
	if info.CreationDate != "" {
		fmt.Printf("  Created:        %s\n", info.CreationDate)
	}
	if info.InstallerIP != "" {
		fmt.Printf("  Installer IP:   %s\n", info.InstallerIP)
	}
	fmt.Println()
}

func categorizeNodes(nodes []lib.NodeInfo) (int, map[string][]lib.NodeInfo) {
	nodesUp := 0
	nodesByType := make(map[string][]lib.NodeInfo)

	for _, node := range nodes {
		if node.Status == lib.StatusUp {
			nodesUp++
		}
		nodesByType[node.NodeType] = append(nodesByType[node.NodeType], node)
	}

	return nodesUp, nodesByType
}

func printNodeSummary(total, nodesUp int) {
	nodesDown := total - nodesUp
	fmt.Printf("Nodes: %d total (%d up, %d down)\n", total, nodesUp, nodesDown)
	fmt.Println()

	var status, statusIcon string
	switch nodesUp {
	case total:
		status = statusLabelReady
		statusIcon = iconStatusUp
	case 0:
		status = statusLabelDown
		statusIcon = iconStatusDown
	default:
		status = statusLabelPartial
		statusIcon = iconStatusPartial
	}

	fmt.Printf("Overall Status: %s %s\n", statusIcon, status)
	fmt.Println()
}

func printNodeDetails(nodesByType map[string][]lib.NodeInfo) {
	fmt.Println("Node Details:")

	for _, nodeType := range nodeTypeOrder {
		nodes := nodesByType[nodeType]
		if len(nodes) == 0 {
			continue
		}

		displayType := nodeType
		if len(nodeType) > 0 {
			displayType = strings.ToUpper(nodeType[:1]) + nodeType[1:]
		}
		fmt.Printf("  %s:\n", displayType)

		for _, node := range nodes {
			statusIcon := iconStatusUp
			if node.Status != lib.StatusUp {
				statusIcon = iconStatusDown
			}

			ip := node.IP
			if ip == "" {
				ip = "-"
			}

			fmt.Printf("    %s %s - %s [%s]\n", statusIcon, node.Name, ip, node.Status)
		}

		fmt.Println()
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

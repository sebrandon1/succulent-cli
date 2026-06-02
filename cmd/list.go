package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	listNoDetail    bool
	listNoCache     bool
	listConcurrency int
	listSortBy      string
	listFilter      string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available environments",
	Long:  `Fetch the main page and list all available environments with status details.`,
	Example: `  succulent-cli list
  succulent-cli list --no-detail
  succulent-cli list --no-cache
  succulent-cli list --output json`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if listNoDetail {
			return listBasic()
		}

		return listDetailed()
	},
}

func listBasic() error {
	envs, err := sharedClient.ListEnvironments()
	if err != nil {
		return fmt.Errorf("listing environments: %w", err)
	}

	if outputFormat == "json" {
		return printJSON(envs)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tGROUP")

	for _, env := range envs {
		group := env.Group
		if group == "" {
			group = "-"
		}

		fmt.Fprintf(w, "%s\t%s\n", env.Name, group)
	}

	return w.Flush()
}

func listDetailed() error {
	var cache *lib.Cache
	if !listNoCache && sharedCache != nil {
		cache = sharedCache
	}

	fmt.Fprintf(os.Stderr, "Fetching info for all environments (concurrency: %d)...\n", listConcurrency)

	details, err := sharedClient.ListEnvironmentsWithInfo(listConcurrency, cache)
	if err != nil {
		return fmt.Errorf("listing environments: %w", err)
	}

	if listFilter != "" {
		details, err = filterDetails(details, listFilter)
		if err != nil {
			return err
		}
	}

	sortDetails(details, listSortBy)

	if outputFormat == "json" {
		return printJSON(details)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tGROUP\tSTATUS\tNODES\tOWNER\tINSTALLER IP")

	for _, d := range details {
		group := d.Group
		if group == "" {
			group = "-"
		}

		owner := d.Owner
		if owner == "" {
			owner = "-"
		}

		ip := d.InstallerIP
		if ip == "" {
			ip = "-"
		}

		nodes := fmt.Sprintf("%d/%d", d.NodesUp, d.NodeCount)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", d.Name, group, d.Status, nodes, owner, ip)
	}

	return w.Flush()
}

func filterDetails(details []lib.EnvironmentDetail, filter string) ([]lib.EnvironmentDetail, error) {
	parts := strings.SplitN(filter, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid filter format %q; expected key=value (e.g., status=active)", filter)
	}

	key := strings.ToLower(parts[0])
	val := strings.ToLower(parts[1])

	var filtered []lib.EnvironmentDetail

	for _, d := range details {
		var fieldVal string

		switch key {
		case "name":
			fieldVal = d.Name
		case "status":
			fieldVal = d.Status
		case "group":
			fieldVal = d.Group
		case "owner":
			fieldVal = d.Owner
		default:
			return nil, fmt.Errorf("unknown filter key %q; valid keys: name, status, group, owner", key)
		}

		if strings.EqualFold(fieldVal, val) {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

func sortDetails(details []lib.EnvironmentDetail, sortBy string) {
	sort.Slice(details, func(i, j int) bool {
		switch sortBy {
		case "status":
			return details[i].Status < details[j].Status
		case "group":
			return details[i].Group < details[j].Group
		case "nodes-up":
			return details[i].NodesUp > details[j].NodesUp
		default:
			return details[i].Name < details[j].Name
		}
	})
}

func init() {
	listCmd.Flags().BoolVar(&listNoDetail, "no-detail", false, "Skip fetching per-environment info (fast mode)")
	listCmd.Flags().BoolVar(&listNoCache, "no-cache", false, "Bypass the info cache")
	listCmd.Flags().IntVar(&listConcurrency, "concurrency", 10, "Number of parallel info fetches")
	listCmd.Flags().StringVar(&listSortBy, "sort", "name", "Sort by field: name, status, group, nodes-up")
	listCmd.Flags().StringVar(&listFilter, "filter", "", "Filter environments: key=value (e.g., status=active, group=Lab1)")

	rootCmd.AddCommand(listCmd)
}

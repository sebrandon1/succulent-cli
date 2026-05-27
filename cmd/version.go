package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of succulent-cli",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Printf("succulent-cli %s\n", rootCmd.Version)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

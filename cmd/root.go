package cmd

import (
	"fmt"
	"os"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	succulentURL     string
	envName          string
	verifySSL        bool
	sharedClient     *lib.Client
	maxWaitMinutes   int
	pollIntervalSecs int
)

var rootCmd = &cobra.Command{
	Use:   "succulent-cli",
	Short: "CLI for the succulent ZTP lab cluster management service",
	Long: `A CLI tool for interacting with the succulent ZTP lab cluster
management service.

Supports cluster info, provisioning (MNO and SNO), log streaming,
kubeconfig retrieval, and environment deletion.`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if skipEnvValidation(cmd) {
			return nil
		}

		if envName == "" {
			return fmt.Errorf("--env is required")
		}

		sharedClient = lib.NewClient(succulentURL, !verifySSL)

		return nil
	},
}

func SetVersion(v string) {
	rootCmd.Version = v
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get information from succulent",
}

var kubeconfigCmd = &cobra.Command{
	Use:   cmdNameKubeconfig,
	Short: "Kubeconfig management commands",
}

var snoCmd = &cobra.Command{
	Use:   "sno",
	Short: "SNO cluster management commands",
}

func skipEnvValidation(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		name := c.Name()
		if name == "completion" || name == "help" || name == "__complete" || name == "version" {
			return true
		}
	}

	return false
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func init() {
	rootCmd.PersistentFlags().StringVar(&succulentURL, "url",
		envOrDefault("SUCCULENT_URL", lib.DefaultSucculentURL),
		"Succulent base URL (env: SUCCULENT_URL)")
	rootCmd.PersistentFlags().StringVar(&envName, "env",
		envOrDefault("SUCCULENT_ENV", ""),
		"Environment name (e.g., env1, env2)")
	rootCmd.PersistentFlags().BoolVar(&verifySSL, "verify-ssl", false,
		"Enable SSL certificate verification (default: disabled)")

	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(kubeconfigCmd)
	rootCmd.AddCommand(snoCmd)

	getCmd.AddCommand(logCmd)
	getCmd.AddCommand(infoCmd)

	kubeconfigCmd.AddCommand(fetchKubeconfigCmd)

	snoCmd.AddCommand(snoProvisionCmd)
	snoCmd.AddCommand(snoKubeconfigCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

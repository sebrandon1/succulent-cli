package cmd

import (
	"os"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
)

var (
	succulentURL string
	envName      string
	verifySSL    bool
)

var rootCmd = &cobra.Command{
	Use:   "succulent-cli",
	Short: "CLI for the succulent ZTP lab cluster management service",
	Long: `A CLI tool for interacting with the succulent ZTP lab cluster
management service at succulent.eng.redhat.com.

Supports cluster info, provisioning (MNO and SNO), log streaming,
kubeconfig retrieval, and environment deletion.`,
}

func SetVersion(v string) {
	rootCmd.Version = v
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get information from succulent",
}

var kubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Kubeconfig management commands",
}

var snoCmd = &cobra.Command{
	Use:   "sno",
	Short: "SNO cluster management commands",
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
		"Environment name (e.g., cnfdc3, cnfdc7)")
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

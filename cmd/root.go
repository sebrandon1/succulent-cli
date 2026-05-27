package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

		succulentURL = viper.GetString("url")
		verifySSL = viper.GetBool("verify_ssl")
		sharedClient = lib.NewClient(succulentURL, !verifySSL)

		if skipEnvRequirement(cmd) {
			return nil
		}

		envName = viper.GetString("env")

		if envName == "" {
			return fmt.Errorf("--env is required (or set in config file / SUCCULENT_ENV)")
		}

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

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".config", "succulent-cli")
}

func skipEnvRequirement(cmd *cobra.Command) bool {
	return cmd.Name() == "list"
}

func skipEnvValidation(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		name := c.Name()
		if name == "completion" || name == "help" || name == "__complete" || name == "version" || name == cmdNameConfig {
			return true
		}
	}

	return false
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir())

	viper.SetEnvPrefix("SUCCULENT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("url", lib.DefaultSucculentURL)
	viper.SetDefault("env", "")
	viper.SetDefault("verify_ssl", false)
	viper.SetDefault("remote_user", defaultRemoteUser)
	viper.SetDefault("remote_path", defaultRemotePath)
	viper.SetDefault("default_email", "")
	viper.SetDefault("default_owner", "")

	_ = viper.ReadInConfig()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&succulentURL, "url", lib.DefaultSucculentURL,
		"Succulent base URL (env: SUCCULENT_URL)")
	rootCmd.PersistentFlags().StringVar(&envName, "env", "",
		"Environment name (e.g., env1, env2)")
	rootCmd.PersistentFlags().BoolVar(&verifySSL, "verify-ssl", false,
		"Enable SSL certificate verification")

	_ = viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	_ = viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env"))
	_ = viper.BindPFlag("verify_ssl", rootCmd.PersistentFlags().Lookup("verify-ssl"))

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

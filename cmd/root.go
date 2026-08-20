package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validEnvName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

var (
	succulentURL     string
	envName          string
	verifySSL        bool
	caCertPath       string
	outputFormat     string
	httpTimeout      int
	sharedClient     *lib.Client
	sharedCache      *lib.Cache
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
		if err := setupLogger(); err != nil {
			return err
		}

		if skipEnvValidation(cmd) {
			return nil
		}

		succulentURL = viper.GetString("url")
		verifySSL = viper.GetBool("verify_ssl")
		caCertPath = viper.GetString("ca_cert")

		var err error

		sharedClient, err = lib.NewClientWithTimeout(succulentURL, !verifySSL, caCertPath, time.Duration(httpTimeout)*time.Second)
		if err != nil {
			return err
		}

		sharedClient.Logger = slog.Default()

		sharedCache = lib.NewCache(configDir(), 60*time.Second)

		if skipEnvRequirement(cmd) {
			return nil
		}

		envName = viper.GetString("env")

		if envName == "" {
			return fmt.Errorf("--env is required (set via flag, SUCCULENT_ENV env var, or config file at %s/config.yaml)", configDir())
		}

		if !validEnvName.MatchString(envName) {
			return fmt.Errorf("invalid environment name %q: must be alphanumeric with hyphens or underscores (e.g., cnfdt16, my-env-01)", envName)
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
		if name == "completion" || name == "help" || name == "__complete" || name == "version" || name == cmdNameConfig || name == "health" {
			return true
		}
	}

	return false
}

func setupLogger() error {
	verbose := viper.GetBool("verbose")
	quiet := viper.GetBool("quiet")

	if verbose && quiet {
		return fmt.Errorf("--verbose and --quiet cannot both be set")
	}

	level := slog.LevelWarn
	if quiet {
		level = slog.LevelError
	}

	if verbose {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	return nil
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
	viper.SetDefault("strict_ssh", false)
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
	rootCmd.PersistentFlags().StringVar(&caCertPath, "ca-cert", "",
		"Path to CA certificate bundle (PEM format)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table",
		"Output format (json or table)")
	rootCmd.PersistentFlags().IntVar(&httpTimeout, "timeout", 60,
		"HTTP request timeout in seconds")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false,
		"Enable debug logging to stderr")
	rootCmd.PersistentFlags().Bool("quiet", false,
		"Log errors only")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false,
		"Disable ANSI color in table output (also honors NO_COLOR)")

	_ = viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	_ = viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env"))
	_ = viper.BindPFlag("verify_ssl", rootCmd.PersistentFlags().Lookup("verify-ssl"))
	_ = viper.BindPFlag("ca_cert", rootCmd.PersistentFlags().Lookup("ca-cert"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))

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

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sebrandon1/go-toolbox/cobrautil"
	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validEnvName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

var (
	cliVersion       = "dev"
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
	skipVersionCheck bool
)

var rootCmd = &cobra.Command{
	Use:   "succulent-cli",
	Short: "CLI for the succulent ZTP lab cluster management service",
	Long: `A CLI tool for interacting with the succulent ZTP lab cluster
management service.

Supports cluster info, provisioning (MNO and SNO), log streaming,
kubeconfig retrieval, and environment deletion.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := persistentPreRun(cmd, args); err != nil {
			return err
		}

		// Flags and --env are valid, so any later error is a runtime failure
		// (API, network, server): print the error without the usage text.
		cmd.SilenceUsage = true

		return nil
	},
}

func persistentPreRun(cmd *cobra.Command, args []string) error {
	if err := prepareRun(); err != nil {
		return err
	}

	var err error
	sharedAuditLog, err = lib.NewAuditLog(auditLogPath())
	if err != nil {
		return fmt.Errorf("initializing audit log: %w", err)
	}

	if skipEnvValidation(cmd) {
		return nil
	}

	if err := initSharedClient(cmd.Context()); err != nil {
		return err
	}

	selectedEnvironments = nil

	if supportsEnvironmentGroups(cmd) {
		targets, err := resolveEnvironmentTargets(viper.GetString("env"), args)
		if err != nil {
			return err
		}
		selectedEnvironments = targets
		if len(targets) > 0 {
			envName = targets[0]
			return nil
		}
	}

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
}

func SetVersion(v string) {
	cobrautil.SetVersion(rootCmd, v)
	cliVersion = rootCmd.Version
}

func prepareRun() error {
	if err := setupLogger(); err != nil {
		return err
	}

	return validateOutputFormat()
}

func validateOutputFormat() error {
	outputFormat = strings.ToLower(strings.TrimSpace(outputFormat))
	if outputFormat != "table" && outputFormat != "json" {
		return fmt.Errorf("invalid --output %q: must be table or json", outputFormat)
	}

	return nil
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

// initSharedClient builds the shared HTTP client and info cache from the
// resolved flag, env, and config values.
func initSharedClient(ctx context.Context) error {
	succulentURL = viper.GetString("url")
	verifySSL = viper.GetBool("verify_ssl")
	caCertPath = viper.GetString("ca_cert")

	client, err := lib.NewClientWithTimeout(succulentURL, !verifySSL, caCertPath, time.Duration(httpTimeout)*time.Second)
	if err != nil {
		return err
	}

	client.SetUserAgentVersion(cliVersion)
	client.Logger = slog.Default()
	sharedClient = client

	if !viper.GetBool("skip_version_check") && cliVersion != "devel" && cliVersion != "dev" && cliVersion != "" {
		warnOnVersionMismatch(ctx, sharedClient, cliVersion, os.Stderr)
	}

	cacheTTL, err := resolveCacheTTL()
	if err != nil {
		return err
	}

	sharedCache = lib.NewCache(configDir(), cacheTTL)

	return nil
}

// durationSetting reads a duration-valued setting from viper as a string and
// parses it with time.ParseDuration. Viper's own GetDuration accepts bare
// numbers and turns them into nanoseconds, so "30" would silently mean 30ns;
// ParseDuration rejects it instead.
func durationSetting(key string, def time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(viper.GetString(key))
	if raw == "" {
		return def, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: use a duration with units, for example 30s or 2m", key, raw)
	}

	return d, nil
}

// resolveCacheTTL returns the effective info-cache TTL. Zero disables caching;
// negative values are rejected.
func resolveCacheTTL() (time.Duration, error) {
	ttl, err := durationSetting("cache_ttl", defaultCacheTTL)
	if err != nil {
		return 0, err
	}

	if ttl < 0 {
		return 0, fmt.Errorf("cache TTL must not be negative (got %s); use 0 to disable caching", ttl)
	}

	return ttl, nil
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
	// Durations are stored as strings: a time.Duration default would be
	// written to config.yaml as integer nanoseconds by `config set`.
	viper.SetDefault("cache_ttl", defaultCacheTTL.String())

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
	rootCmd.PersistentFlags().Duration("cache-ttl", defaultCacheTTL,
		"Environment info cache TTL (e.g. 60s); 0 disables caching (env: SUCCULENT_CACHE_TTL)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false,
		"Enable debug logging to stderr")
	rootCmd.PersistentFlags().Bool("quiet", false,
		"Log errors only")
	rootCmd.PersistentFlags().BoolVar(&skipVersionCheck, "no-version-check", false,
		"Skip checking server API version compatibility")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false,
		"Disable ANSI color in table output (also honors NO_COLOR)")

	_ = viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	_ = viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env"))
	_ = viper.BindPFlag("verify_ssl", rootCmd.PersistentFlags().Lookup("verify-ssl"))
	_ = viper.BindPFlag("ca_cert", rootCmd.PersistentFlags().Lookup("ca-cert"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	_ = viper.BindPFlag("skip_version_check", rootCmd.PersistentFlags().Lookup("no-version-check"))
	_ = viper.BindPFlag("cache_ttl", rootCmd.PersistentFlags().Lookup("cache-ttl"))

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
	return cobrautil.Execute(rootCmd)
}

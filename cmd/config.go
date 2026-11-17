package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	cacheFileName   = "cache.json"
	defaultCacheTTL = 60 * time.Second
)

var validConfigKeys = []string{
	"url", "env", "verify_ssl", "strict_ssh",
	"remote_user", "remote_path",
	"default_email", "default_owner",
	"timeout", "dial_timeout", "tls_handshake_timeout",
	"cache_ttl",
}

var boolConfigKeys = map[string]bool{
	"verify_ssl": true,
	"strict_ssh": true,
}

// intConfigKeys are whole numbers written as strings and read back with
// viper.GetInt.
var intConfigKeys = map[string]bool{
	"timeout": true,
}

// durationConfigKeys are stored as strings such as "10s" and must parse with
// time.ParseDuration. All of them must be positive.
var durationConfigKeys = map[string]bool{
	"dial_timeout":          true,
	"tls_handshake_timeout": true,
}

// nonNegativeDurationConfigKeys may be zero; cache_ttl uses 0 to disable
// caching.
var nonNegativeDurationConfigKeys = map[string]bool{
	"cache_ttl": true,
}

// validateConfigDurationValue checks a duration-valued config key before it is
// written, so a bad value is rejected at `config set` time instead of breaking
// the next command.
func validateConfigDurationValue(key, value string) error {
	d, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("%s must be a duration with units, for example 60s or 2m", key)
	}

	if nonNegativeDurationConfigKeys[key] {
		if d < 0 {
			return fmt.Errorf("%s must not be negative; use 0 to disable caching", key)
		}

		return nil
	}

	if d <= 0 {
		return fmt.Errorf("%s must be greater than 0", key)
	}

	return nil
}

var configCmd = &cobra.Command{
	Use:   cmdNameConfig,
	Short: "Manage succulent-cli configuration",
}

var configShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show the resolved configuration",
	Example: `  succulent-cli config show`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if f := viper.ConfigFileUsed(); f != "" {
			fmt.Printf("Config file: %s\n\n", f)
		} else {
			fmt.Printf("Config file: (none)\n\n")
		}

		for _, key := range validConfigKeys {
			fmt.Printf("%-22s %v\n", key+":", viper.Get(key))
		}

		fmt.Printf("\n%-16s %s\n", "cache_file:", filepath.Join(configDir(), cacheFileName))

		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:     "path",
	Short:   "Print the config file path",
	Example: `  succulent-cli config path`,
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Println(filepath.Join(configDir(), "config.yaml"))

		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:     "init",
	Short:   "Create a default config file",
	Example: `  succulent-cli config init`,
	RunE: func(_ *cobra.Command, _ []string) error {
		dir := configDir()
		path := filepath.Join(dir, "config.yaml")

		if _, err := os.Stat(path); err == nil {
			fmt.Printf("Config file already exists: %s\n", path)

			return nil
		}

		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		defaultConfig := `# succulent-cli configuration
# Values here are overridden by environment variables (SUCCULENT_ prefix) and CLI flags.

# url: "https://succulent.example.com"
# env: "myenv"
# verify_ssl: false
# strict_ssh: false
# remote_user: "root"
# remote_path: "/root/ocp/auth/kubeconfig"
# default_email: "user@example.com"
# default_owner: "myuser"
# timeout: 60                      # whole HTTP request timeout, in seconds
# dial_timeout: "10s"              # TCP connection-establishment timeout
# tls_handshake_timeout: "10s"     # TLS handshake timeout
# cache_ttl: "60s"                 # environment info cache TTL; "0s" disables caching
`

		if err := os.WriteFile(path, []byte(defaultConfig), 0o600); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}

		fmt.Printf("Config file created: %s\n", path)

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Example: `  succulent-cli config set url https://succulent.example.com
  succulent-cli config set default_email user@example.com
  succulent-cli config set verify_ssl true
  succulent-cli config set strict_ssh true`,
	Args: cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		// Validate key
		validKey := false
		for _, vk := range validConfigKeys {
			if key == vk {
				validKey = true
				break
			}
		}
		if !validKey {
			return fmt.Errorf("invalid config key '%s'. Valid keys: %s",
				key, strings.Join(validConfigKeys, ", "))
		}

		if intConfigKeys[key] {
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("%s must be a whole number of seconds (got %q)", key, value)
			}
		}

		if durationConfigKeys[key] || nonNegativeDurationConfigKeys[key] {
			if err := validateConfigDurationValue(key, value); err != nil {
				return err
			}
		}

		// Type coercion for boolean
		if boolConfigKeys[key] {
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("%s must be a boolean (true/false/yes/no/1/0)", key)
			}
			viper.Set(key, boolVal)
		} else {
			viper.Set(key, value)
		}

		configPath := filepath.Join(configDir(), "config.yaml")

		if err := os.MkdirAll(configDir(), 0o750); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		if err := viper.WriteConfigAs(configPath); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}

		fmt.Printf("Set %s in %s\n", key, configPath)

		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit config file in $EDITOR",
	Example: `  succulent-cli config edit
  EDITOR=vim succulent-cli config edit`,
	RunE: func(_ *cobra.Command, _ []string) error {
		configPath := filepath.Join(configDir(), "config.yaml")

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("config file does not exist. Run 'succulent-cli config init' first")
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		cmd := exec.Command(editor, configPath) // #nosec G204 -- editor is $EDITOR or vi; path is the local config file
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		fmt.Printf("Config file saved: %s\n", configPath)

		return nil
	},
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage environment info cache",
}

var cacheClearCmd = &cobra.Command{
	Use:     "clear",
	Short:   "Clear the environment info cache",
	Example: `  succulent-cli config cache clear`,
	RunE: func(_ *cobra.Command, _ []string) error {
		cachePath := filepath.Join(configDir(), cacheFileName)

		if err := os.Remove(cachePath); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Cache is already empty")
				return nil
			}
			return fmt.Errorf("clearing cache: %w", err)
		}

		fmt.Printf("Cache cleared: %s\n", cachePath)

		return nil
	},
}

var cacheShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show cached environment info",
	Example: `  succulent-cli config cache show
  succulent-cli config cache show --output json`,
	RunE: func(_ *cobra.Command, _ []string) error {
		cachePath := filepath.Join(configDir(), cacheFileName)

		data, err := os.ReadFile(cachePath) // #nosec G304 -- path is under the CLI config directory
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No cache file found")
				return nil
			}

			return fmt.Errorf("reading cache file: %w", err)
		}

		var cache struct {
			Environments map[string]struct {
				Info struct {
					Environment  string `json:"environment"`
					InstallerIP  string `json:"installer_ip"`
					CreationDate string `json:"creation_date"`
					NodeCount    int    `json:"node_count"`
				} `json:"info"`
				FetchedAt string `json:"fetched_at"`
			} `json:"environments"`
		}

		if err := json.Unmarshal(data, &cache); err != nil {
			return fmt.Errorf("parsing cache file: %w", err)
		}

		if outputFormat == "json" {
			fmt.Println(string(data))
			return nil
		}

		if len(cache.Environments) == 0 {
			fmt.Println("Cache is empty")
			return nil
		}

		fmt.Printf("%-20s %-20s %-20s %s\n", "Environment", "Installer IP", "Nodes", "Cached At")
		fmt.Println(strings.Repeat("-", 80))

		for env, entry := range cache.Environments {
			fmt.Printf("%-20s %-20s %-20d %s\n",
				env,
				entry.Info.InstallerIP,
				entry.Info.NodeCount,
				entry.FetchedAt)
		}

		return nil
	},
}

var cacheStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show cache statistics",
	Example: `  succulent-cli config cache status`,
	RunE: func(_ *cobra.Command, _ []string) error {
		cachePath := filepath.Join(configDir(), cacheFileName)

		fmt.Printf("Cache location: %s\n", cachePath)

		data, err := os.ReadFile(cachePath) // #nosec G304 -- path is under the CLI config directory
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Status: Cache file does not exist")
				return nil
			}

			return fmt.Errorf("reading cache file: %w", err)
		}

		var cache struct {
			Environments map[string]struct {
				FetchedAt string `json:"fetched_at"`
			} `json:"environments"`
		}

		if err := json.Unmarshal(data, &cache); err != nil {
			return fmt.Errorf("parsing cache file: %w", err)
		}

		fmt.Printf("Size: %d bytes\n", len(data))
		fmt.Printf("Entry count: %d\n", len(cache.Environments))

		if len(cache.Environments) > 0 {
			var oldest, newest string
			for _, entry := range cache.Environments {
				if oldest == "" || entry.FetchedAt < oldest {
					oldest = entry.FetchedAt
				}
				if newest == "" || entry.FetchedAt > newest {
					newest = entry.FetchedAt
				}
			}

			fmt.Printf("Oldest entry: %s\n", oldest)
			fmt.Printf("Newest entry: %s\n", newest)
		}

		ttl, err := resolveCacheTTL()
		if err != nil {
			return err
		}

		if ttl == 0 {
			fmt.Printf("TTL: %s (caching disabled)\n", ttl)
		} else {
			fmt.Printf("TTL: %s\n", ttl)
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(configGroupsCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configHistoryCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configEditCmd)

	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheShowCmd)
	cacheCmd.AddCommand(cacheStatusCmd)
	configCmd.AddCommand(cacheCmd)

	rootCmd.AddCommand(configCmd)
}

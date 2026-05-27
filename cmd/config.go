package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   cmdNameConfig,
	Short: "Manage succulent-cli configuration",
}

var configShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show the resolved configuration",
	Example: `  succulent-cli config show`,
	RunE: func(_ *cobra.Command, _ []string) error {
		keys := []string{
			"url", "env", "verify_ssl",
			"remote_user", "remote_path",
			"default_email", "default_owner",
		}

		if f := viper.ConfigFileUsed(); f != "" {
			fmt.Printf("Config file: %s\n\n", f)
		} else {
			fmt.Printf("Config file: (none)\n\n")
		}

		for _, key := range keys {
			fmt.Printf("%-16s %v\n", key+":", viper.Get(key))
		}

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
# remote_user: "root"
# remote_path: "/root/ocp/auth/kubeconfig"
# default_email: "user@example.com"
# default_owner: "myuser"
`

		if err := os.WriteFile(path, []byte(defaultConfig), 0o600); err != nil {
			return fmt.Errorf("writing config file: %w", err)
		}

		fmt.Printf("Config file created: %s\n", path)

		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)

	rootCmd.AddCommand(configCmd)
}

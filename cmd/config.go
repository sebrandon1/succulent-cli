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
	Run: func(_ *cobra.Command, _ []string) {
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
	},
}

var configPathCmd = &cobra.Command{
	Use:     "path",
	Short:   "Print the config file path",
	Example: `  succulent-cli config path`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(filepath.Join(configDir(), "config.yaml"))
	},
}

var configInitCmd = &cobra.Command{
	Use:     "init",
	Short:   "Create a default config file",
	Example: `  succulent-cli config init`,
	Run: func(_ *cobra.Command, _ []string) {
		dir := configDir()
		path := filepath.Join(dir, "config.yaml")

		if _, err := os.Stat(path); err == nil {
			fmt.Printf("Config file already exists: %s\n", path)
			return
		}

		if err := os.MkdirAll(dir, 0o750); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			os.Exit(1)
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
			fmt.Printf("Error writing config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Config file created: %s\n", path)
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)

	rootCmd.AddCommand(configCmd)
}

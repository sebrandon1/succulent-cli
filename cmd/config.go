package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validConfigKeys = []string{
	"url", "env", "verify_ssl",
	"remote_user", "remote_path",
	"default_email", "default_owner",
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

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Example: `  succulent-cli config set url https://succulent.example.com
  succulent-cli config set default_email user@example.com
  succulent-cli config set verify_ssl true`,
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

		// Type coercion for boolean
		if key == "verify_ssl" {
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("verify_ssl must be a boolean (true/false/yes/no/1/0)")
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

		cmd := exec.Command(editor, configPath)
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

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configEditCmd)

	rootCmd.AddCommand(configCmd)
}

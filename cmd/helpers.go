package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func printJSON(data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Println(string(output))

	return nil
}

func resolveOwnerEmail(owner, email string) (string, string, error) {
	if owner == "" {
		owner = viper.GetString("default_owner")
	}

	if email == "" {
		email = viper.GetString("default_email")
	}

	if owner == "" {
		return "", "", fmt.Errorf("--owner is required (or set default_owner in config)")
	}

	if email == "" {
		return "", "", fmt.Errorf("--email is required (or set default_email in config)")
	}

	return owner, email, nil
}

func defaultDestPath(env, suffix string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	return filepath.Join(home, defaultDestDir, env+"-"+suffix), nil
}

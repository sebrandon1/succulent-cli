package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/viper"
)

type CommandResult struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
	Message     string `json:"message"`
}

func printJSON(data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Println(string(output))

	return nil
}

func printResult(result CommandResult, format string) error {
	if format == "json" {
		return printJSON(result)
	}

	fmt.Println(result.Message)

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
		return "", "", fmt.Errorf("--owner is required (or set default_owner in config); try: succulent-cli config init")
	}

	if email == "" {
		return "", "", fmt.Errorf("--email is required (or set default_email in config); try: succulent-cli config init")
	}

	return owner, email, nil
}

func saveKubeconfig(data []byte, dest, env, suffix string) (string, error) {
	if dest == "" {
		d, err := defaultDestPath(env, suffix)
		if err != nil {
			return "", err
		}

		dest = d
	}

	if err := lib.ValidateKubeconfig(data); err != nil {
		return "", fmt.Errorf("invalid kubeconfig data: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return "", fmt.Errorf("creating destination directory: %w", err)
	}

	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return "", fmt.Errorf("writing kubeconfig: %w", err)
	}

	return dest, nil
}

func defaultDestPath(env, suffix string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	return filepath.Join(home, defaultDestDir, "succulent", env, suffix), nil
}

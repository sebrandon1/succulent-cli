package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func printJSON(data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Println(string(output))

	return nil
}

func defaultDestPath(env, suffix string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	return filepath.Join(home, defaultDestDir, env+"-"+suffix), nil
}

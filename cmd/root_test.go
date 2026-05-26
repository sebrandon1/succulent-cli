package cmd

import (
	"testing"

	"github.com/sebrandon1/succulent-cli/lib"
)

func TestEnvFlagExists(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("env")
	if flag == nil {
		t.Fatal("Expected --env flag on rootCmd, not found")
	}
}

func TestURLFlagExists(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("url")
	if flag == nil {
		t.Fatal("Expected --url flag on rootCmd, not found")
	}

	if flag.DefValue != lib.DefaultSucculentURL {
		if envOrDefault("SUCCULENT_URL", "") == "" {
			t.Errorf("Expected --url default %s, got %s", lib.DefaultSucculentURL, flag.DefValue)
		}
	}
}

func TestVerifySSLFlagExists(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("verify-ssl")
	if flag == nil {
		t.Fatal("Expected --verify-ssl flag on rootCmd, not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected --verify-ssl default 'false', got '%s'", flag.DefValue)
	}
}

func TestSetVersion(t *testing.T) {
	SetVersion("v1.0.0")
	if rootCmd.Version != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got '%s'", rootCmd.Version)
	}
}

func TestGetSubcommands(t *testing.T) {
	subcommands := getCmd.Commands()

	found := map[string]bool{"log": false, "info": false}

	for _, cmd := range subcommands {
		if _, ok := found[cmd.Name()]; ok {
			found[cmd.Name()] = true
		}
	}

	for name, exists := range found {
		if !exists {
			t.Errorf("Expected subcommand %q under 'get', not found", name)
		}
	}
}

func TestSNOSubcommands(t *testing.T) {
	subcommands := snoCmd.Commands()

	found := map[string]bool{"provision": false, "kubeconfig": false}

	for _, cmd := range subcommands {
		if _, ok := found[cmd.Name()]; ok {
			found[cmd.Name()] = true
		}
	}

	for name, exists := range found {
		if !exists {
			t.Errorf("Expected subcommand %q under 'sno', not found", name)
		}
	}
}

package cmd

import (
	"testing"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
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
		t.Errorf("Expected --url default %s, got %s", lib.DefaultSucculentURL, flag.DefValue)
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

func TestRootSubcommands(t *testing.T) {
	subcommands := rootCmd.Commands()

	expected := map[string]bool{
		"get":         false,
		"delete":      false,
		"reprovision": false,
		"kubeconfig":  false,
		"sno":         false,
		"watch":       false,
		"version":     false,
		"config":      false,
	}

	for _, cmd := range subcommands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, exists := range expected {
		if !exists {
			t.Errorf("Expected subcommand %q under root, not found", name)
		}
	}
}

func TestKubeconfigSubcommands(t *testing.T) {
	subcommands := kubeconfigCmd.Commands()

	found := false

	for _, cmd := range subcommands {
		if cmd.Name() == "fetch" {
			found = true
		}
	}

	if !found {
		t.Error("Expected subcommand 'fetch' under 'kubeconfig', not found")
	}
}

func TestConfigSubcommands(t *testing.T) {
	subcommands := configCmd.Commands()

	expected := map[string]bool{"show": false, "path": false, "init": false}

	for _, cmd := range subcommands {
		if _, ok := expected[cmd.Name()]; ok {
			expected[cmd.Name()] = true
		}
	}

	for name, exists := range expected {
		if !exists {
			t.Errorf("Expected subcommand %q under 'config', not found", name)
		}
	}
}

func TestSkipEnvValidationForCompletion(t *testing.T) {
	fakeCompletion := &cobra.Command{Use: "bash"}
	fakeParent := &cobra.Command{Use: "completion"}
	fakeParent.AddCommand(fakeCompletion)

	if !skipEnvValidation(fakeCompletion) {
		t.Error("Expected skipEnvValidation to return true for completion subcommand")
	}
}

func TestSkipEnvValidationForHelp(t *testing.T) {
	fakeHelp := &cobra.Command{Use: "help"}

	if !skipEnvValidation(fakeHelp) {
		t.Error("Expected skipEnvValidation to return true for help command")
	}
}

func TestSkipEnvValidationForConfig(t *testing.T) {
	fakeConfig := &cobra.Command{Use: "show"}
	fakeParent := &cobra.Command{Use: "config"}
	fakeParent.AddCommand(fakeConfig)

	if !skipEnvValidation(fakeConfig) {
		t.Error("Expected skipEnvValidation to return true for config subcommand")
	}
}

func TestSkipEnvValidationForRegularCommand(t *testing.T) {
	if skipEnvValidation(infoCmd) {
		t.Error("Expected skipEnvValidation to return false for info command")
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

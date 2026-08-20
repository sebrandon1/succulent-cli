package cmd

import (
	"strings"
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
		"list":        false,
		"hypershift":  false,
		"ztp":         false,
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

func TestValidEnvName(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		valid bool
	}{
		{"simple", "cnfdt16", true},
		{"with hyphens", "my-env-01", true},
		{"with underscores", "env_test_1", true},
		{"uppercase", "ENV01", true},
		{"mixed case", "CnfDt16", true},
		{"empty", "", false},
		{"starts with hyphen", "-env", false},
		{"starts with underscore", "_env", false},
		{"spaces", "my env", false},
		{"special chars", "env@1", false},
		{"path traversal", "../etc", false},
		{"slash", "env/name", false},
		{"semicolon injection", "env;rm", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validEnvName.MatchString(tt.env)
			if got != tt.valid {
				t.Errorf("validEnvName.MatchString(%q) = %v, want %v", tt.env, got, tt.valid)
			}
		})
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

func TestStrictSSHFlagExists(t *testing.T) {
	flag := fetchKubeconfigCmd.Flags().Lookup("strict-ssh")
	if flag == nil {
		t.Fatal("Expected --strict-ssh flag on kubeconfig fetch, not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected --strict-ssh default 'false', got %q", flag.DefValue)
	}
}

func TestVerboseFlagExists(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Fatal("Expected --verbose flag on rootCmd, not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected --verbose default 'false', got %q", flag.DefValue)
	}

	if flag.Shorthand != "v" {
		t.Errorf("Expected --verbose shorthand 'v', got %q", flag.Shorthand)
	}
}

func TestQuietFlagExists(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("quiet")
	if flag == nil {
		t.Fatal("Expected --quiet flag on rootCmd, not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected --quiet default 'false', got %q", flag.DefValue)
	}
}

func TestNoColorFlagExists(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("no-color")
	if flag == nil {
		t.Fatal("Expected --no-color flag on rootCmd, not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected --no-color default 'false', got %q", flag.DefValue)
	}
}

func TestFlagParsing_VerboseQuietConflict(t *testing.T) {
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		_ = rootCmd.PersistentFlags().Set("verbose", "false")
		_ = rootCmd.PersistentFlags().Set("quiet", "false")
	})

	rootCmd.SetArgs([]string{"version", "--verbose", "--quiet"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --verbose and --quiet are both set")
	}

	if !strings.Contains(err.Error(), "cannot both be set") {
		t.Errorf("unexpected error: %v", err)
	}
}

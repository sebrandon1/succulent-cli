package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	shellBash = "bash"
	shellZsh  = "zsh"
	shellFish = "fish"
)

var (
	completionDryRun bool
	completionShell  string
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for succulent-cli.

Supports bash, zsh, and fish shells.`,
}

var completionInstallCmd = &cobra.Command{
	Use:   "install [bash|zsh|fish]",
	Short: "Install shell completion for the current or specified shell",
	Long: `Install shell completion by generating the completion script and updating
the appropriate shell configuration file.

If no shell is specified, auto-detects from the SHELL environment variable.

The completion script is written to ~/.config/succulent-cli/completions/<shell>/
and a source line is added to the shell's rc file (~/.bashrc, ~/.zshrc, or
~/.config/fish/config.fish).

Use --dry-run to see what would be done without making any changes.`,
	Example: `  # Auto-detect shell and install
  succulent-cli completion install

  # Install for a specific shell
  succulent-cli completion install bash

  # Preview what would be done
  succulent-cli completion install --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := completionShell
		if len(args) > 0 {
			shell = args[0]
		}

		if shell == "" {
			detected, err := detectShell()
			if err != nil {
				return fmt.Errorf("detecting shell: %w (specify shell explicitly with: completion install [bash|zsh|fish])", err)
			}
			shell = detected
		}

		if !isValidShell(shell) {
			return fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish)", shell)
		}

		if err := installCompletion(cmd.Root(), shell, completionDryRun); err != nil {
			return fmt.Errorf("installing completion: %w", err)
		}

		return nil
	},
}

func detectShell() (string, error) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return "", fmt.Errorf("SHELL environment variable not set")
	}

	shellName := filepath.Base(shellPath)

	switch shellName {
	case shellBash:
		return shellBash, nil
	case shellZsh:
		return shellZsh, nil
	case shellFish:
		return shellFish, nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shellName)
	}
}

func isValidShell(shell string) bool {
	return shell == shellBash || shell == shellZsh || shell == shellFish
}

func installCompletion(rootCmd *cobra.Command, shell string, dryRun bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	completionDir := filepath.Join(home, ".config", "succulent-cli", "completions", shell)
	completionFile := filepath.Join(completionDir, "succulent-cli")

	var rcFile string

	switch shell {
	case shellBash:
		rcFile = filepath.Join(home, ".bashrc")
	case shellZsh:
		rcFile = filepath.Join(home, ".zshrc")
	case shellFish:
		rcFile = filepath.Join(home, ".config", "fish", "config.fish")
	}

	sourceLine := fmt.Sprintf("source %s", completionFile)

	fmt.Printf("Installing %s completion...\n", shell)
	fmt.Printf("  Completion script: %s\n", completionFile)
	fmt.Printf("  RC file:           %s\n", rcFile)
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run mode - no changes will be made")
		fmt.Println()
		fmt.Println("Would perform:")
		fmt.Printf("  1. Create directory: %s\n", completionDir)
		fmt.Printf("  2. Generate completion script: %s\n", completionFile)
		fmt.Printf("  3. Add source line to %s:\n", rcFile)
		fmt.Printf("     %s\n", sourceLine)
		return nil
	}

	if err := os.MkdirAll(completionDir, 0o750); err != nil {
		return fmt.Errorf("creating completion directory: %w", err)
	}

	completionScript, err := generateCompletionScript(rootCmd, shell)
	if err != nil {
		return fmt.Errorf("generating completion script: %w", err)
	}

	if err := os.WriteFile(completionFile, []byte(completionScript), 0o600); err != nil {
		return fmt.Errorf("writing completion file: %w", err)
	}

	if err := addSourceLine(rcFile, sourceLine); err != nil {
		return fmt.Errorf("updating rc file: %w", err)
	}

	fmt.Println("✓ Completion installed successfully!")
	fmt.Println()
	fmt.Println("To activate, run:")
	fmt.Printf("  source %s\n", rcFile)
	fmt.Println()
	fmt.Println("Or restart your shell.")

	return nil
}

func generateCompletionScript(rootCmd *cobra.Command, shell string) (string, error) {
	var sb strings.Builder

	var err error
	switch shell {
	case shellBash:
		err = rootCmd.GenBashCompletionV2(&sb, true)
	case shellZsh:
		err = rootCmd.GenZshCompletion(&sb)
	case shellFish:
		err = rootCmd.GenFishCompletion(&sb, true)
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}

	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func addSourceLine(rcFile, sourceLine string) error {
	rcDir := filepath.Dir(rcFile)
	if err := os.MkdirAll(rcDir, 0o750); err != nil {
		return fmt.Errorf("creating rc directory: %w", err)
	}

	content, err := os.ReadFile(rcFile) // #nosec G304 -- rc path is derived from the home directory
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(rcFile, []byte(sourceLine+"\n"), 0o600)
		}
		return fmt.Errorf("reading rc file: %w", err)
	}

	if strings.Contains(string(content), sourceLine) {
		fmt.Println("Note: Source line already exists in rc file")
		return nil
	}

	updatedContent := string(content)
	if !strings.HasSuffix(updatedContent, "\n") {
		updatedContent += "\n"
	}
	updatedContent += "\n# succulent-cli completion\n"
	updatedContent += sourceLine + "\n"

	// #nosec G703 -- rc path is derived from the home directory and a fixed filename
	if err := os.WriteFile(rcFile, []byte(updatedContent), 0o600); err != nil {
		return fmt.Errorf("writing rc file: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(completionInstallCmd)

	completionInstallCmd.Flags().BoolVar(&completionDryRun, "dry-run", false, "Show what would be done without making changes")
	completionInstallCmd.Flags().StringVar(&completionShell, "shell", "", "Shell to install completion for (bash, zsh, fish)")
}

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name        string
		shellEnv    string
		wantShell   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "bash",
			shellEnv:  "/bin/bash",
			wantShell: "bash",
			wantErr:   false,
		},
		{
			name:      "zsh",
			shellEnv:  "/bin/zsh",
			wantShell: "zsh",
			wantErr:   false,
		},
		{
			name:      "fish",
			shellEnv:  "/usr/local/bin/fish",
			wantShell: "fish",
			wantErr:   false,
		},
		{
			name:        "unsupported shell",
			shellEnv:    "/bin/sh",
			wantErr:     true,
			errContains: "unsupported shell",
		},
		{
			name:        "empty SHELL",
			shellEnv:    "",
			wantErr:     true,
			errContains: "SHELL environment variable not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.shellEnv)

			got, err := detectShell()
			if tt.wantErr {
				if err == nil {
					t.Errorf("detectShell() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("detectShell() error = %v, want to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("detectShell() unexpected error: %v", err)
				return
			}

			if got != tt.wantShell {
				t.Errorf("detectShell() = %v, want %v", got, tt.wantShell)
			}
		})
	}
}

func TestIsValidShell(t *testing.T) {
	tests := []struct {
		shell string
		want  bool
	}{
		{"bash", true},
		{"zsh", true},
		{"fish", true},
		{"sh", false},
		{"powershell", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			if got := isValidShell(tt.shell); got != tt.want {
				t.Errorf("isValidShell(%q) = %v, want %v", tt.shell, got, tt.want)
			}
		})
	}
}

func TestGenerateCompletionScript(t *testing.T) {
	tests := []struct {
		name      string
		shell     string
		wantErr   bool
		wantEmpty bool
	}{
		{
			name:      "bash",
			shell:     "bash",
			wantErr:   false,
			wantEmpty: false,
		},
		{
			name:      "zsh",
			shell:     "zsh",
			wantErr:   false,
			wantEmpty: false,
		},
		{
			name:      "fish",
			shell:     "fish",
			wantErr:   false,
			wantEmpty: false,
		},
		{
			name:    "unsupported",
			shell:   "powershell",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := generateCompletionScript(rootCmd, tt.shell)
			if tt.wantErr {
				if err == nil {
					t.Errorf("generateCompletionScript() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("generateCompletionScript() unexpected error: %v", err)
				return
			}

			if tt.wantEmpty && script != "" {
				t.Errorf("generateCompletionScript() expected empty, got %d bytes", len(script))
			}

			if !tt.wantEmpty && script == "" {
				t.Error("generateCompletionScript() expected non-empty script, got empty")
			}
		})
	}
}

func TestAddSourceLine(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		sourceLine      string
		wantContains    string
		wantErr         bool
	}{
		{
			name:            "new file",
			existingContent: "",
			sourceLine:      "source /path/to/completion",
			wantContains:    "source /path/to/completion",
			wantErr:         false,
		},
		{
			name:            "existing file without source line",
			existingContent: "export PATH=$PATH:/usr/local/bin\n",
			sourceLine:      "source /path/to/completion",
			wantContains:    "source /path/to/completion",
			wantErr:         false,
		},
		{
			name:            "existing file with source line",
			existingContent: "source /path/to/completion\n",
			sourceLine:      "source /path/to/completion",
			wantContains:    "source /path/to/completion",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			rcFile := filepath.Join(tmpDir, "testrc")

			if tt.existingContent != "" {
				if err := os.WriteFile(rcFile, []byte(tt.existingContent), 0o644); err != nil {
					t.Fatalf("Failed to create test rc file: %v", err)
				}
			}

			err := addSourceLine(rcFile, tt.sourceLine)
			if tt.wantErr {
				if err == nil {
					t.Error("addSourceLine() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("addSourceLine() unexpected error: %v", err)
				return
			}

			content, err := os.ReadFile(rcFile)
			if err != nil {
				t.Fatalf("Failed to read rc file: %v", err)
			}

			if !strings.Contains(string(content), tt.wantContains) {
				t.Errorf("addSourceLine() file content does not contain %q\nGot: %s", tt.wantContains, content)
			}

			lines := strings.Split(string(content), "\n")
			count := 0
			for _, line := range lines {
				if strings.TrimSpace(line) == tt.sourceLine {
					count++
				}
			}

			if count > 1 {
				t.Errorf("addSourceLine() source line appears %d times, expected 1", count)
			}
		})
	}
}

func TestInstallCompletionDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := installCompletion(rootCmd, "bash", true)
	if err != nil {
		t.Errorf("installCompletion() with dry-run failed: %v", err)
	}

	completionFile := filepath.Join(tmpDir, ".config", "succulent-cli", "completions", "bash", "succulent-cli")
	if _, err := os.Stat(completionFile); !os.IsNotExist(err) {
		t.Error("installCompletion() with dry-run created files, expected no changes")
	}

	rcFile := filepath.Join(tmpDir, ".bashrc")
	if _, err := os.Stat(rcFile); !os.IsNotExist(err) {
		t.Error("installCompletion() with dry-run modified rc file, expected no changes")
	}
}

func TestInstallCompletionBash(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := installCompletion(rootCmd, "bash", false)
	if err != nil {
		t.Fatalf("installCompletion() failed: %v", err)
	}

	completionFile := filepath.Join(tmpDir, ".config", "succulent-cli", "completions", "bash", "succulent-cli")
	if _, err := os.Stat(completionFile); os.IsNotExist(err) {
		t.Error("installCompletion() did not create completion file")
	}

	content, err := os.ReadFile(completionFile)
	if err != nil {
		t.Fatalf("Failed to read completion file: %v", err)
	}

	if len(content) == 0 {
		t.Error("installCompletion() created empty completion file")
	}

	rcFile := filepath.Join(tmpDir, ".bashrc")
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read rc file: %v", err)
	}

	expectedSource := "source " + completionFile
	if !strings.Contains(string(rcContent), expectedSource) {
		t.Errorf("rc file does not contain expected source line %q\nGot: %s", expectedSource, rcContent)
	}
}

func TestInstallCompletionZsh(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := installCompletion(rootCmd, "zsh", false)
	if err != nil {
		t.Fatalf("installCompletion() failed: %v", err)
	}

	completionFile := filepath.Join(tmpDir, ".config", "succulent-cli", "completions", "zsh", "succulent-cli")
	if _, err := os.Stat(completionFile); os.IsNotExist(err) {
		t.Error("installCompletion() did not create completion file")
	}

	rcFile := filepath.Join(tmpDir, ".zshrc")
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read rc file: %v", err)
	}

	expectedSource := "source " + completionFile
	if !strings.Contains(string(rcContent), expectedSource) {
		t.Errorf("rc file does not contain expected source line %q", expectedSource)
	}
}

func TestInstallCompletionFish(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := installCompletion(rootCmd, "fish", false)
	if err != nil {
		t.Fatalf("installCompletion() failed: %v", err)
	}

	completionFile := filepath.Join(tmpDir, ".config", "succulent-cli", "completions", "fish", "succulent-cli")
	if _, err := os.Stat(completionFile); os.IsNotExist(err) {
		t.Error("installCompletion() did not create completion file")
	}

	rcFile := filepath.Join(tmpDir, ".config", "fish", "config.fish")
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read rc file: %v", err)
	}

	expectedSource := "source " + completionFile
	if !strings.Contains(string(rcContent), expectedSource) {
		t.Errorf("rc file does not contain expected source line %q", expectedSource)
	}
}

func TestInstallCompletionIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	if err := installCompletion(rootCmd, "bash", false); err != nil {
		t.Fatalf("First installCompletion() failed: %v", err)
	}

	if err := installCompletion(rootCmd, "bash", false); err != nil {
		t.Fatalf("Second installCompletion() failed: %v", err)
	}

	rcFile := filepath.Join(tmpDir, ".bashrc")
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read rc file: %v", err)
	}

	completionFile := filepath.Join(tmpDir, ".config", "succulent-cli", "completions", "bash", "succulent-cli")
	expectedSource := "source " + completionFile

	count := strings.Count(string(rcContent), expectedSource)
	if count != 1 {
		t.Errorf("Source line appears %d times after two installs, expected 1", count)
	}
}

func TestCompletionInstallCommand(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	if err := completionInstallCmd.RunE(completionInstallCmd, []string{"bash"}); err != nil {
		t.Errorf("RunE with positional arg failed: %v", err)
	}

	rcFile := filepath.Join(tmpDir, ".bashrc")
	if _, err := os.Stat(rcFile); os.IsNotExist(err) {
		t.Error("RunE did not create rc file")
	}
}

func TestCompletionInstallCommandInvalidShell(t *testing.T) {
	err := completionInstallCmd.RunE(completionInstallCmd, []string{"powershell"})
	if err == nil {
		t.Error("RunE expected error for invalid shell, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("RunE error = %v, want 'unsupported shell'", err)
	}
}

func TestCompletionInstallCommandAutoDetect(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SHELL", "/bin/zsh")

	if err := completionInstallCmd.RunE(completionInstallCmd, []string{}); err != nil {
		t.Errorf("RunE with auto-detect failed: %v", err)
	}

	rcFile := filepath.Join(tmpDir, ".zshrc")
	if _, err := os.Stat(rcFile); os.IsNotExist(err) {
		t.Error("RunE did not create .zshrc (auto-detection may have failed)")
	}
}

func TestCompletionInstallCommandDetectionFails(t *testing.T) {
	t.Setenv("SHELL", "")

	err := completionInstallCmd.RunE(completionInstallCmd, []string{})
	if err == nil {
		t.Error("RunE expected error when SHELL not set, got nil")
	}
	if !strings.Contains(err.Error(), "detecting shell") {
		t.Errorf("RunE error = %v, want 'detecting shell'", err)
	}
}

package cmd

import (
	"strings"
	"testing"
)

func TestVersionCommandPrintsConfiguredVersion(t *testing.T) {
	previous := rootCmd.Version
	t.Cleanup(func() { SetVersion(previous) })

	SetVersion("v9.9.9")

	stdout := captureCommandStdout(t)

	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := stdout(); !strings.Contains(got, "succulent-cli v9.9.9") {
		t.Fatalf("version output = %q, want it to contain %q", got, "succulent-cli v9.9.9")
	}
}

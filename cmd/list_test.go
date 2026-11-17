package cmd

import (
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestListProgressWriter(t *testing.T) {
	origTTY := stderrIsTTY
	t.Cleanup(func() {
		stderrIsTTY = origTTY
		_ = rootCmd.PersistentFlags().Set("quiet", "false")
	})

	tests := []struct {
		name  string
		tty   bool
		quiet bool
		want  io.Writer
	}{
		{name: "terminal shows progress", tty: true, want: os.Stderr},
		{name: "piped output hides progress", tty: false, want: io.Discard},
		{name: "quiet hides progress on a terminal", tty: true, quiet: true, want: io.Discard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderrIsTTY = func() bool { return tt.tty }
			// Set the bound flag, not viper directly: viper.Set would override
			// the flag binding for every later test.
			if err := rootCmd.PersistentFlags().Set("quiet", strconv.FormatBool(tt.quiet)); err != nil {
				t.Fatal(err)
			}

			if got := listProgressWriter(); got != tt.want {
				t.Errorf("listProgressWriter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListCommandRejectsNonPositiveConcurrency(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	listNoDetail = false
	_ = listCmd.Flags().Set("no-detail", "false")

	t.Cleanup(func() {
		listConcurrency = 10
		listNoDetail = false
		_ = listCmd.Flags().Set("concurrency", "10")
		_ = listCmd.Flags().Set("no-detail", "false")
	})

	for _, value := range []string{"0", "-1"} {
		t.Run(value, func(t *testing.T) {
			rootCmd.SetArgs([]string{"list", "--concurrency", value})

			done := make(chan error, 1)
			go func() { done <- rootCmd.Execute() }()

			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), "--concurrency must be at least 1") {
					t.Fatalf("Execute() error = %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("list hung instead of rejecting the concurrency value")
			}
		})
	}
}

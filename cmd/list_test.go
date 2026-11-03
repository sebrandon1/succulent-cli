package cmd

import (
	"io"
	"os"
	"strconv"
	"testing"
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

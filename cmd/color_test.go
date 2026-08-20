package cmd

import (
	"strings"
	"testing"

	"github.com/sebrandon1/succulent-cli/lib"
)

func withColorEnabled(t *testing.T) {
	t.Helper()

	origTTY := stdoutIsTTY
	origNoColor := noColor
	origFormat := outputFormat

	stdoutIsTTY = func() bool { return true }
	noColor = false
	outputFormat = "table"
	t.Setenv("NO_COLOR", "")

	t.Cleanup(func() {
		stdoutIsTTY = origTTY
		noColor = origNoColor
		outputFormat = origFormat
	})
}

func TestColorStatus(t *testing.T) {
	withColorEnabled(t)

	tests := []struct {
		status string
		code   string
	}{
		{"active", ansiGreen},
		{"ACTIVE", ansiGreen},
		{"Ready", ansiGreen},
		{lib.StatusUp, ansiGreen},
		{"partial", ansiYellow},
		{"Partial", ansiYellow},
		{"empty", ansiGray},
		{"down", ansiRed},
		{"Down", ansiRed},
		{lib.StatusDown, ansiRed},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := colorStatus(tt.status)
			if !strings.Contains(got, tt.code) {
				t.Errorf("colorStatus(%q) = %q, want ANSI code %q", tt.status, got, tt.code)
			}

			if !strings.Contains(got, tt.status) {
				t.Errorf("colorStatus(%q) lost original text, got %q", tt.status, got)
			}

			if !strings.HasSuffix(got, ansiReset) {
				t.Errorf("colorStatus(%q) missing reset, got %q", tt.status, got)
			}
		})
	}

	if got := colorStatus("unknown"); got != "unknown" {
		t.Errorf("colorStatus(unknown) = %q, want unchanged", got)
	}
}

func TestColorEnabled(t *testing.T) {
	withColorEnabled(t)

	if !colorEnabled() {
		t.Fatal("expected color enabled for TTY table output")
	}

	t.Run("no-color flag", func(t *testing.T) {
		noColor = true
		t.Cleanup(func() { noColor = false })

		if colorEnabled() {
			t.Fatal("expected --no-color to disable color")
		}

		if got := colorStatus("active"); got != "active" {
			t.Errorf("expected uncolored status, got %q", got)
		}
	})

	t.Run("NO_COLOR env", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")

		if colorEnabled() {
			t.Fatal("expected NO_COLOR to disable color")
		}

		if got := colorStatus("partial"); got != "partial" {
			t.Errorf("expected uncolored status, got %q", got)
		}
	})

	t.Run("json output", func(t *testing.T) {
		outputFormat = "json"
		t.Cleanup(func() { outputFormat = "table" })

		if colorEnabled() {
			t.Fatal("expected JSON output to disable color")
		}
	})

	t.Run("non-tty", func(t *testing.T) {
		stdoutIsTTY = func() bool { return false }
		t.Cleanup(func() { stdoutIsTTY = func() bool { return true } })

		if colorEnabled() {
			t.Fatal("expected non-TTY stdout to disable color")
		}
	})
}

func TestStdoutIsTerminal(t *testing.T) {
	_ = stdoutIsTerminal()
}

func TestColorizeEmptyCode(t *testing.T) {
	withColorEnabled(t)

	if got := colorize("text", ""); got != "text" {
		t.Errorf("colorize with empty code = %q, want text", got)
	}
}

func TestANSICodeLengths(t *testing.T) {
	// Equal-length color codes keep tabwriter columns aligned.
	for _, code := range []string{ansiGreen, ansiYellow, ansiRed, ansiGray} {
		if len(code) != 5 {
			t.Errorf("ANSI color code %q has length %d, want 5", code, len(code))
		}
	}

	if len(ansiReset) != 4 {
		t.Errorf("ANSI reset %q has length %d, want 4", ansiReset, len(ansiReset))
	}
}

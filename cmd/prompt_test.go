package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestMain(m *testing.M) {
	isInteractive = func() bool { return false }
	os.Exit(m.Run())
}

func withPromptIO(t *testing.T, input string) *bytes.Buffer {
	t.Helper()

	origInteractive := isInteractive
	origReader := promptReader
	origWriter := promptWriter
	origLineReader := promptLineReader

	out := &bytes.Buffer{}
	isInteractive = func() bool { return true }
	promptReader = strings.NewReader(input)
	promptWriter = out
	promptLineReader = nil

	t.Cleanup(func() {
		isInteractive = origInteractive
		promptReader = origReader
		promptWriter = origWriter
		promptLineReader = origLineReader
	})

	return out
}

func TestResolveOwnerEmailInteractive(t *testing.T) {
	viper.Set("default_owner", "")
	viper.Set("default_email", "")
	t.Cleanup(func() {
		viper.Set("default_owner", "")
		viper.Set("default_email", "")
	})

	out := withPromptIO(t, "prompteduser\nuser@example.com\n")

	owner, email, err := resolveOwnerEmail("", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if owner != "prompteduser" {
		t.Errorf("Expected owner %q, got %q", "prompteduser", owner)
	}

	if email != "user@example.com" {
		t.Errorf("Expected email %q, got %q", "user@example.com", email)
	}

	got := out.String()
	if !strings.Contains(got, promptOwner) || !strings.Contains(got, promptEmail) {
		t.Errorf("Expected owner and email prompts, got %q", got)
	}
}

func TestResolveOwnerEmailInteractivePartial(t *testing.T) {
	viper.Set("default_owner", "")
	viper.Set("default_email", "")

	withPromptIO(t, "user@example.com\n")

	owner, email, err := resolveOwnerEmail("flagowner", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if owner != "flagowner" {
		t.Errorf("Expected owner %q, got %q", "flagowner", owner)
	}

	if email != "user@example.com" {
		t.Errorf("Expected email %q, got %q", "user@example.com", email)
	}
}

func TestResolveOwnerEmailInteractiveEmptyRejected(t *testing.T) {
	viper.Set("default_owner", "")
	viper.Set("default_email", "")

	withPromptIO(t, "\n")

	_, _, err := resolveOwnerEmail("", "user@example.com")
	if err == nil {
		t.Fatal("Expected error for empty prompted owner, got nil")
	}

	if !strings.Contains(err.Error(), "--owner is required") {
		t.Errorf("Expected owner required error, got %v", err)
	}
}

func TestResolveOwnerEmailInteractiveInvalidEmail(t *testing.T) {
	viper.Set("default_owner", "")
	viper.Set("default_email", "")

	withPromptIO(t, "not-an-email\n")

	_, _, err := resolveOwnerEmail("myuser", "")
	if err == nil {
		t.Fatal("Expected error for prompted email without @, got nil")
	}

	if !strings.Contains(err.Error(), "must contain @") {
		t.Errorf("Expected 'must contain @' in error, got: %v", err)
	}
}

func TestResolveOwnerEmailInteractiveEOF(t *testing.T) {
	viper.Set("default_owner", "")
	viper.Set("default_email", "")

	withPromptIO(t, "")

	_, _, err := resolveOwnerEmail("", "user@example.com")
	if err == nil {
		t.Fatal("Expected EOF error, got nil")
	}

	if !strings.Contains(err.Error(), "EOF") {
		t.Errorf("Expected EOF error, got %v", err)
	}
}

func TestResolveOwnerEmailUsesConfigDefaults(t *testing.T) {
	viper.Set("default_owner", "cfguser")
	viper.Set("default_email", "cfg@example.com")
	t.Cleanup(func() {
		viper.Set("default_owner", "")
		viper.Set("default_email", "")
	})

	withPromptIO(t, "")

	owner, email, err := resolveOwnerEmail("", "")
	if err != nil {
		t.Fatalf("Expected config defaults to skip prompt, got %v", err)
	}

	if owner != "cfguser" {
		t.Errorf("Expected owner %q, got %q", "cfguser", owner)
	}

	if email != "cfg@example.com" {
		t.Errorf("Expected email %q, got %q", "cfg@example.com", email)
	}
}

func TestResolveOwnerEmailInteractiveEmptyEmailRejected(t *testing.T) {
	viper.Set("default_owner", "")
	viper.Set("default_email", "")

	withPromptIO(t, "\n")

	_, _, err := resolveOwnerEmail("myuser", "")
	if err == nil {
		t.Fatal("Expected error for empty prompted email, got nil")
	}

	if !strings.Contains(err.Error(), "--email is required") {
		t.Errorf("Expected email required error, got %v", err)
	}
}

func TestResolveOCPTag(t *testing.T) {
	t.Run("provided", func(t *testing.T) {
		got, err := resolveOCPTag("4.17")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if got != "4.17" {
			t.Errorf("Expected 4.17, got %q", got)
		}
	})

	t.Run("missing non-interactive", func(t *testing.T) {
		_, err := resolveOCPTag("")
		if err == nil {
			t.Fatal("Expected error for missing tag, got nil")
		}

		if !strings.Contains(err.Error(), "--ocp-tag is required") {
			t.Errorf("Expected required error, got %v", err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := resolveOCPTag("latest")
		if err == nil {
			t.Fatal("Expected validation error, got nil")
		}
	})

	t.Run("prompted", func(t *testing.T) {
		out := withPromptIO(t, "4.18\n")

		got, err := resolveOCPTag("")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if got != "4.18" {
			t.Errorf("Expected 4.18, got %q", got)
		}

		if !strings.Contains(out.String(), promptOCPTag) {
			t.Errorf("Expected OCP tag prompt, got %q", out.String())
		}
	})

	t.Run("prompted invalid", func(t *testing.T) {
		withPromptIO(t, "latest\n")

		_, err := resolveOCPTag("")
		if err == nil {
			t.Fatal("Expected validation error for prompted tag, got nil")
		}
	})
}

func TestPromptOptional(t *testing.T) {
	t.Run("keeps existing value", func(t *testing.T) {
		got, err := promptOptional("4.17")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if got != "4.17" {
			t.Errorf("Expected 4.17, got %q", got)
		}
	})

	t.Run("non-interactive empty stays empty", func(t *testing.T) {
		got, err := promptOptional("")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if got != "" {
			t.Errorf("Expected empty, got %q", got)
		}
	})

	t.Run("interactive empty accepted", func(t *testing.T) {
		withPromptIO(t, "\n")

		got, err := promptOptional("")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if got != "" {
			t.Errorf("Expected empty skip, got %q", got)
		}
	})

	t.Run("interactive value", func(t *testing.T) {
		withPromptIO(t, "4.19\n")

		got, err := promptOptional("")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if got != "4.19" {
			t.Errorf("Expected 4.19, got %q", got)
		}
	})
}

func TestPromptLineEOFWithoutNewline(t *testing.T) {
	withPromptIO(t, "solo")

	got, err := promptLine(promptOwner)
	if err != nil {
		t.Fatalf("Expected no error for EOF with data, got %v", err)
	}

	if got != "solo" {
		t.Errorf("Expected %q, got %q", "solo", got)
	}
}

func TestPromptLineWriteError(t *testing.T) {
	origWriter := promptWriter
	promptWriter = errWriter{}
	t.Cleanup(func() { promptWriter = origWriter })

	_, err := promptLine(promptOwner)
	if err == nil {
		t.Fatal("Expected write error, got nil")
	}

	if !strings.Contains(err.Error(), "writing prompt") {
		t.Errorf("Expected writing prompt error, got %v", err)
	}
}

func TestStdinIsTerminal(t *testing.T) {
	// go test may or may not attach a TTY; just ensure it does not panic.
	_ = stdinIsTerminal()
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write fail")
}

var _ io.Writer = errWriter{}

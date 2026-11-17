package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTimeoutFlagsExist(t *testing.T) {
	tests := []struct {
		name string
		def  string
	}{
		{"timeout", "60"},
		{"dial-timeout", "10s"},
		{"tls-handshake-timeout", "10s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := rootCmd.PersistentFlags().Lookup(tt.name)
			if flag == nil {
				t.Fatalf("Expected --%s persistent flag", tt.name)
			}

			if flag.DefValue != tt.def {
				t.Errorf("Expected default %q, got %q", tt.def, flag.DefValue)
			}
		})
	}
}

func TestResolveClientOptionsDefaults(t *testing.T) {
	cleanConfig(t)

	opts, err := resolveClientOptions()
	if err != nil {
		t.Fatalf("resolveClientOptions failed: %v", err)
	}

	want := [3]time.Duration{60 * time.Second, 10 * time.Second, 10 * time.Second}
	got := [3]time.Duration{opts.Timeout, opts.DialTimeout, opts.TLSHandshakeTimeout}

	if got != want {
		t.Errorf("Expected %v, got %v", want, got)
	}
}

func TestResolveClientOptionsPrecedence(t *testing.T) {
	for _, name := range []string{"timeout", "dial-timeout", "tls-handshake-timeout"} {
		resetFlag(t, name)
	}

	cleanConfig(t)

	configYAML := "timeout: 7\ndial_timeout: \"2s\"\ntls_handshake_timeout: \"3s\"\n"
	if err := os.WriteFile(filepath.Join(configDir(), "config.yaml"), []byte(configYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	initConfig()

	assertOptions := func(t *testing.T, want [3]time.Duration) {
		t.Helper()

		opts, err := resolveClientOptions()
		if err != nil {
			t.Fatalf("resolveClientOptions failed: %v", err)
		}

		got := [3]time.Duration{opts.Timeout, opts.DialTimeout, opts.TLSHandshakeTimeout}
		if got != want {
			t.Errorf("Expected %v, got %v", want, got)
		}
	}

	t.Run("config file", func(t *testing.T) {
		assertOptions(t, [3]time.Duration{7 * time.Second, 2 * time.Second, 3 * time.Second})
	})

	t.Run("environment beats config", func(t *testing.T) {
		t.Setenv("SUCCULENT_TIMEOUT", "5")
		t.Setenv("SUCCULENT_DIAL_TIMEOUT", "4s")
		t.Setenv("SUCCULENT_TLS_HANDSHAKE_TIMEOUT", "6s")

		assertOptions(t, [3]time.Duration{5 * time.Second, 4 * time.Second, 6 * time.Second})
	})

	t.Run("flag beats environment", func(t *testing.T) {
		t.Setenv("SUCCULENT_TIMEOUT", "5")
		t.Setenv("SUCCULENT_DIAL_TIMEOUT", "4s")
		t.Setenv("SUCCULENT_TLS_HANDSHAKE_TIMEOUT", "6s")

		for name, value := range map[string]string{
			"timeout":               "9",
			"dial-timeout":          "1s",
			"tls-handshake-timeout": "2s",
		} {
			if err := rootCmd.PersistentFlags().Set(name, value); err != nil {
				t.Fatal(err)
			}
		}

		assertOptions(t, [3]time.Duration{9 * time.Second, time.Second, 2 * time.Second})
	})

	cleanConfig(t)
}

func TestResolveClientOptionsRejectsNonPositive(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
	}{
		{"zero timeout", "SUCCULENT_TIMEOUT", "0"},
		{"negative timeout", "SUCCULENT_TIMEOUT", "-1"},
		{"zero dial timeout", "SUCCULENT_DIAL_TIMEOUT", "0s"},
		{"negative dial timeout", "SUCCULENT_DIAL_TIMEOUT", "-2s"},
		{"zero handshake timeout", "SUCCULENT_TLS_HANDSHAKE_TIMEOUT", "0s"},
		{"unitless timeout", "SUCCULENT_TIMEOUT", "30s"},
	}

	for _, name := range []string{"timeout", "dial-timeout", "tls-handshake-timeout"} {
		resetFlag(t, name)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanConfig(t)
			t.Setenv(tt.env, tt.value)

			if _, err := resolveClientOptions(); err == nil {
				t.Fatalf("Expected an error for %s=%s", tt.env, tt.value)
			}
		})
	}
}

func TestValidateConfigDurationValueTimeouts(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "10s", false},
		{"zero", "0s", true},
		{"negative", "-1s", true},
		{"no units", "10", true},
		{"garbage", "soon", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigDurationValue("dial_timeout", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfigDurationValue(%q) error = %v, wantErr %v",
					tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestConfigInitTemplateIncludesTimeouts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := configInitCmd.RunE(configInitCmd, nil); err != nil {
		t.Fatalf("configInitCmd failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(configDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"timeout", "dial_timeout", "tls_handshake_timeout"} {
		if !strings.Contains(string(data), key) {
			t.Errorf("Expected the config template to mention %s", key)
		}
	}
}

func TestPersistentPreRunUsesConfiguredTimeouts(t *testing.T) {
	for _, name := range []string{"timeout", "dial-timeout", "tls-handshake-timeout"} {
		resetFlag(t, name)
	}

	cleanConfig(t)

	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testInfoHTML))
	})
	defer cleanup()

	rootCmd.SetArgs([]string{
		"get", "info", "--env", "testenv", "--no-cache",
		"--timeout", "5",
		"--dial-timeout", "2s",
		"--tls-handshake-timeout", "3s",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("get info failed: %v", err)
	}

	if sharedClient == nil {
		t.Fatal("Expected a shared client")
	}

	if sharedClient.Timeout != 5*time.Second {
		t.Errorf("Expected 5s overall timeout, got %s", sharedClient.Timeout)
	}

	if sharedClient.DialTimeout != 2*time.Second {
		t.Errorf("Expected 2s dial timeout, got %s", sharedClient.DialTimeout)
	}

	if sharedClient.TLSHandshakeTimeout != 3*time.Second {
		t.Errorf("Expected 3s TLS handshake timeout, got %s", sharedClient.TLSHandshakeTimeout)
	}
}

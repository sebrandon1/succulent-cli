package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// resetFlag restores a persistent flag to its default and clears the Changed
// bit. A flag left changed by one test would otherwise outrank env and config
// in every later test in the package.
func resetFlag(t *testing.T, name string) {
	t.Helper()

	flag := rootCmd.PersistentFlags().Lookup(name)
	if flag == nil {
		t.Fatalf("persistent flag %q not found", name)
	}

	originalValue := flag.Value.String()
	originalChanged := flag.Changed

	// Clear the Changed bit for the duration of this test: a flag left changed
	// by an earlier test otherwise outranks env and config in viper's lookup.
	if err := flag.Value.Set(flag.DefValue); err != nil {
		t.Fatalf("resetting flag %q: %v", name, err)
	}

	flag.Changed = false

	t.Cleanup(func() {
		if err := flag.Value.Set(originalValue); err != nil {
			t.Errorf("restoring flag %q: %v", name, err)
		}

		flag.Changed = originalChanged
	})
}

// cleanConfig points configDir() at an empty config file in a fresh HOME and
// reloads viper, so a config value read by an earlier test cannot linger.
func cleanConfig(t *testing.T) {
	t.Helper()

	t.Setenv("HOME", t.TempDir())

	if err := os.MkdirAll(configDir(), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(configDir(), "config.yaml"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	initConfig()
}

func TestCacheTTLFlagExists(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("cache-ttl")
	if flag == nil {
		t.Fatal("Expected --cache-ttl persistent flag")
	}

	if flag.DefValue != defaultCacheTTL.String() {
		t.Errorf("Expected default %q, got %q", defaultCacheTTL.String(), flag.DefValue)
	}
}

func TestResolveCacheTTLPrecedence(t *testing.T) {
	resetFlag(t, "cache-ttl")
	cleanConfig(t)

	// Default: no config file value, no env var, flag unchanged.
	if got, err := resolveCacheTTL(); err != nil || got != defaultCacheTTL {
		t.Fatalf("Expected default %s, got %s (err=%v)", defaultCacheTTL, got, err)
	}

	// Config file.
	configPath := filepath.Join(configDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("cache_ttl: \"30s\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	initConfig()

	if got, err := resolveCacheTTL(); err != nil || got != 30*time.Second {
		t.Fatalf("Expected 30s from config, got %s (err=%v)", got, err)
	}

	// Environment beats config.
	t.Setenv("SUCCULENT_CACHE_TTL", "45s")

	if got, err := resolveCacheTTL(); err != nil || got != 45*time.Second {
		t.Fatalf("Expected 45s from env, got %s (err=%v)", got, err)
	}

	// Flag beats env.
	if err := rootCmd.PersistentFlags().Set("cache-ttl", "2s"); err != nil {
		t.Fatal(err)
	}

	if got, err := resolveCacheTTL(); err != nil || got != 2*time.Second {
		t.Fatalf("Expected 2s from flag, got %s (err=%v)", got, err)
	}

	// Leave viper with an empty config so the value read here does not leak
	// into tests that run after this one.
	cleanConfig(t)
}

func TestResolveCacheTTLAllowsZero(t *testing.T) {
	cleanConfig(t)
	t.Setenv("SUCCULENT_CACHE_TTL", "0")

	got, err := resolveCacheTTL()
	if err != nil {
		t.Fatalf("Expected 0 to be accepted, got error: %v", err)
	}

	if got != 0 {
		t.Errorf("Expected 0, got %s", got)
	}
}

func TestResolveCacheTTLRejectsNegative(t *testing.T) {
	cleanConfig(t)
	t.Setenv("SUCCULENT_CACHE_TTL", "-1s")

	if _, err := resolveCacheTTL(); err == nil {
		t.Fatal("Expected an error for a negative TTL")
	} else if !strings.Contains(err.Error(), "negative") {
		t.Errorf("Expected a 'negative' error, got %v", err)
	}
}

func TestResolveCacheTTLRejectsUnitlessValue(t *testing.T) {
	cleanConfig(t)
	t.Setenv("SUCCULENT_CACHE_TTL", "60")

	if _, err := resolveCacheTTL(); err == nil {
		t.Fatal("Expected unitless values to be rejected, not read as nanoseconds")
	}
}

func TestValidateConfigDurationValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid duration", "90s", false},
		{"zero disables caching", "0", false},
		{"negative", "-1s", true},
		{"no units", "60", true},
		{"garbage", "soon", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigDurationValue("cache_ttl", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfigDurationValue(%q) error = %v, wantErr %v",
					tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidConfigKeysIncludesCacheTTL(t *testing.T) {
	if !slices.Contains(validConfigKeys, "cache_ttl") {
		t.Error("Expected cache_ttl in validConfigKeys")
	}
}

func TestConfigInitTemplateIncludesCacheTTL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := configInitCmd.RunE(configInitCmd, nil); err != nil {
		t.Fatalf("configInitCmd failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(configDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "cache_ttl") {
		t.Error("Expected the config template to mention cache_ttl")
	}
}

func TestCacheStatusReportsDisabledTTL(t *testing.T) {
	cleanConfig(t)
	t.Setenv("SUCCULENT_CACHE_TTL", "0")

	cacheData := `{"environments":{"env1":{"info":{"environment":"env1"},"fetched_at":"2026-01-01T00:00:00Z"}}}`

	if err := os.WriteFile(filepath.Join(configDir(), cacheFileName), []byte(cacheData), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cacheStatusCmd.RunE(cacheStatusCmd, nil); err != nil {
		t.Fatalf("cache status failed with caching disabled: %v", err)
	}
}

func TestCacheTTLZeroSkipsCacheForInfoCommand(t *testing.T) {
	resetFlag(t, "cache-ttl")
	cleanConfig(t)

	var requests atomic.Int32

	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testInfoHTML))
	})
	defer cleanup()

	for range 2 {
		rootCmd.SetArgs([]string{"get", "info", "--env", "testenv", "--cache-ttl", "0"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("get info failed: %v", err)
		}
	}

	if got := requests.Load(); got != 2 {
		t.Errorf("Expected 2 requests with caching disabled, got %d", got)
	}

	if _, err := os.Stat(filepath.Join(configDir(), cacheFileName)); !os.IsNotExist(err) {
		t.Errorf("Expected no cache file with --cache-ttl 0, got err=%v", err)
	}
}

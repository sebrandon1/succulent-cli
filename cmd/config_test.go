package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func setupTestConfigDir(t *testing.T) func() {
	t.Helper()

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".config", "succulent-cli")

	if err := os.MkdirAll(configFile, 0o750); err != nil {
		t.Fatal(err)
	}

	origHome := os.Getenv("HOME")

	if err := os.Setenv("HOME", dir); err != nil {
		t.Fatal(err)
	}

	return func() {
		os.Setenv("HOME", origHome)
	}
}

func TestCacheClearCommand(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	cachePath := filepath.Join(configDir(), "cache.json")

	cacheData := map[string]interface{}{
		"environments": map[string]interface{}{
			"env1": map[string]interface{}{
				"info":       map[string]interface{}{"environment": "env1"},
				"fetched_at": time.Now().Format(time.RFC3339),
			},
		},
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cacheClearCmd.RunE(nil, nil); err != nil {
		t.Fatalf("cacheClearCmd failed: %v", err)
	}

	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("Expected cache file to be deleted")
	}
}

func TestCacheClearCommandEmpty(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	if err := cacheClearCmd.RunE(nil, nil); err != nil {
		t.Fatalf("cacheClearCmd failed on empty cache: %v", err)
	}
}

func TestCacheShowCommand(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	cachePath := filepath.Join(configDir(), "cache.json")

	cacheData := map[string]interface{}{
		"environments": map[string]interface{}{
			"env1": map[string]interface{}{
				"info": map[string]interface{}{
					"environment":   "env1",
					"installer_ip":  "192.168.1.100",
					"creation_date": "2026-08-12",
					"node_count":    3,
				},
				"fetched_at": "2026-08-12T10:00:00Z",
			},
		},
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cacheShowCmd.RunE(nil, nil); err != nil {
		t.Fatalf("cacheShowCmd failed: %v", err)
	}
}

func TestCacheShowCommandEmpty(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	if err := cacheShowCmd.RunE(nil, nil); err != nil {
		t.Fatalf("cacheShowCmd failed on empty cache: %v", err)
	}
}

func TestCacheStatusCommand(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	cachePath := filepath.Join(configDir(), "cache.json")

	cacheData := map[string]interface{}{
		"environments": map[string]interface{}{
			"env1": map[string]interface{}{
				"info":       map[string]interface{}{"environment": "env1"},
				"fetched_at": "2026-08-12T10:00:00Z",
			},
			"env2": map[string]interface{}{
				"info":       map[string]interface{}{"environment": "env2"},
				"fetched_at": "2026-08-12T11:00:00Z",
			},
		},
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cacheStatusCmd.RunE(nil, nil); err != nil {
		t.Fatalf("cacheStatusCmd failed: %v", err)
	}
}

func TestCacheStatusCommandNoFile(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	if err := cacheStatusCmd.RunE(nil, nil); err != nil {
		t.Fatalf("cacheStatusCmd failed on missing file: %v", err)
	}
}

func TestCacheShowCorruptJSON(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	cachePath := filepath.Join(configDir(), "cache.json")

	if err := os.WriteFile(cachePath, []byte("{not valid json}"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := cacheShowCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("Expected error for corrupt JSON, got nil")
	}

	if !strings.Contains(err.Error(), "parsing cache file") {
		t.Errorf("Expected 'parsing cache file' error, got: %v", err)
	}
}

func TestCacheStatusCorruptJSON(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	cachePath := filepath.Join(configDir(), "cache.json")

	if err := os.WriteFile(cachePath, []byte("{not valid json}"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := cacheStatusCmd.RunE(nil, nil)
	if err == nil {
		t.Fatal("Expected error for corrupt JSON, got nil")
	}

	if !strings.Contains(err.Error(), "parsing cache file") {
		t.Errorf("Expected 'parsing cache file' error, got: %v", err)
	}
}

func TestCacheShowCommandJSON(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	cachePath := filepath.Join(configDir(), "cache.json")

	cacheData := map[string]interface{}{
		"environments": map[string]interface{}{
			"env1": map[string]interface{}{
				"info": map[string]interface{}{
					"environment":  "env1",
					"installer_ip": "192.168.1.100",
				},
				"fetched_at": "2026-08-12T10:00:00Z",
			},
		},
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	origOutput := viper.GetString("output")
	viper.Set("output", "json")
	defer viper.Set("output", origOutput)

	if err := cacheShowCmd.RunE(nil, nil); err != nil {
		t.Fatalf("cacheShowCmd with JSON output failed: %v", err)
	}
}

func TestCacheShowCommandEmptyEnvironments(t *testing.T) {
	cleanup := setupTestConfigDir(t)
	defer cleanup()

	cachePath := filepath.Join(configDir(), "cache.json")

	cacheData := map[string]interface{}{
		"environments": map[string]interface{}{},
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := cacheShowCmd.RunE(nil, nil); err != nil {
		t.Fatalf("cacheShowCmd failed on empty environments: %v", err)
	}
}

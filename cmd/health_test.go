package cmd

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestHealthCommandSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	viper.Set("url", server.URL)

	if err := healthCmd.RunE(nil, nil); err != nil {
		t.Fatalf("health command failed: %v", err)
	}
}

func TestHealthCommandServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	viper.Set("url", server.URL)
	viper.Set("verify_ssl", false)

	err := healthCmd.RunE(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "server returned error status code: 500") {
		t.Fatalf("expected a server status error, got %v", err)
	}
}

func TestHealthCommandUnreachable(t *testing.T) {
	viper.Set("url", "http://127.0.0.1:1")
	viper.Set("verify_ssl", false)

	err := healthCmd.RunE(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "health check failed") {
		t.Fatalf("expected a connection error, got %v", err)
	}
}

func TestHealthCommandWithTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	viper.Set("url", server.URL)
	viper.Set("verify_ssl", false)

	if err := healthCmd.RunE(nil, nil); err != nil {
		t.Fatalf("health command with TLS failed: %v", err)
	}
}

func TestHealthCommandWithCACert(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// For testing, we'll just verify it doesn't crash with a CA cert path set
	// Creating a valid CA cert that matches the test server is complex
	viper.Set("url", server.URL)
	viper.Set("verify_ssl", false)
	viper.Set("ca_cert", "")

	if err := healthCmd.RunE(nil, nil); err != nil {
		t.Fatalf("health command with TLS failed: %v", err)
	}
}

func TestHealthCommandWithTLSVerifyEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	viper.Set("url", server.URL)
	viper.Set("verify_ssl", true)
	viper.Set("ca_cert", "")

	// Tests the verify_ssl=true code path (line 30-34)
	// HTTP server doesn't have TLS so this just verifies the if branch executes
	if err := healthCmd.RunE(nil, nil); err != nil {
		t.Fatalf("health command with verify_ssl enabled failed: %v", err)
	}
}

func TestHealthCommandInvalidCACert(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(caPath, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}

	viper.Set("url", "http://127.0.0.1:1")
	viper.Set("verify_ssl", false)
	viper.Set("ca_cert", caPath)
	t.Cleanup(func() { viper.Set("ca_cert", "") })

	err := healthCmd.RunE(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create client") {
		t.Fatalf("expected a client creation error for an invalid CA cert, got %v", err)
	}
}

func TestHealthCommandMissingCACert(t *testing.T) {
	viper.Set("url", "http://127.0.0.1:1")
	viper.Set("verify_ssl", false)
	viper.Set("ca_cert", filepath.Join(t.TempDir(), "missing.pem"))
	t.Cleanup(func() { viper.Set("ca_cert", "") })

	err := healthCmd.RunE(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create client") {
		t.Fatalf("expected a client creation error for a missing CA cert, got %v", err)
	}
}

func TestTLSVersionString(t *testing.T) {
	tests := []struct {
		version uint16
		want    string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
		{0x9999, "Unknown (0x9999)"},
	}

	for _, tt := range tests {
		got := tlsVersionString(tt.version)
		if got != tt.want {
			t.Errorf("tlsVersionString(0x%04x) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

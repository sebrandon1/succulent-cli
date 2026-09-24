package lib

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetServerVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			t.Errorf("Expected /version path, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"apiVersion":"1.2","version":"1.2.3","gitCommit":"abc123"}`)
	}))
	defer server.Close()

	version, err := newTestClient(server.URL).GetServerVersion(context.Background())
	if err != nil {
		t.Fatalf("GetServerVersion failed: %v", err)
	}
	if version.APIVersion != "1.2" || version.Version != "1.2.3" || version.GitCommit != "abc123" {
		t.Errorf("Unexpected server version: %+v", version)
	}
}

func TestGetServerVersionTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := newTestClient(server.URL).GetServerVersion(ctx)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
}

func TestGetServerVersionRejectsMissingEndpoint(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	_, err := newTestClient(server.URL).GetServerVersion(context.Background())
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("Expected HTTP 404 error, got %v", err)
	}
}

func TestCheckVersionCompatibility(t *testing.T) {
	tests := []struct {
		name       string
		cliVersion string
		server     ServerVersion
		compatible bool
		wantText   string
	}{
		{
			name:       "matching major and minor",
			cliVersion: "v1.2.0",
			server:     ServerVersion{APIVersion: "1.2", Version: "1.2.7"},
			compatible: true,
		},
		{
			name:       "CLI older than server API",
			cliVersion: "v1.2.0",
			server:     ServerVersion{APIVersion: "1.3", Version: "1.3.4"},
			compatible: false,
			wantText:   "server API v1.3 is newer than CLI v1.2.0",
		},
		{
			name:       "CLI newer than server API",
			cliVersion: "v1.4.0",
			server:     ServerVersion{APIVersion: "1.3", Version: "1.3.4"},
			compatible: false,
			wantText:   "CLI v1.4.0 is newer than server API v1.3",
		},
		{
			name:       "unparseable development versions are ignored",
			cliVersion: "development-build",
			server:     ServerVersion{Version: "1.3.4"},
			compatible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compatible, warning := CheckVersionCompatibility(tt.cliVersion, tt.server)
			if compatible != tt.compatible {
				t.Errorf("compatible = %t, want %t", compatible, tt.compatible)
			}
			if tt.wantText == "" && warning != "" {
				t.Errorf("Expected no warning, got %q", warning)
			}
			if tt.wantText != "" && !strings.Contains(warning, tt.wantText) {
				t.Errorf("Expected warning to contain %q, got %q", tt.wantText, warning)
			}
		})
	}
}

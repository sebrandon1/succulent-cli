package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sebrandon1/succulent-cli/lib"
)

func TestWarnOnVersionMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			t.Errorf("Expected /version path, got %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"apiVersion":"1.3","version":"1.3.0"}`))
	}))
	defer server.Close()

	client, err := lib.NewClient(server.URL, true, "")
	if err != nil {
		t.Fatal(err)
	}
	var warnings bytes.Buffer
	warnOnVersionMismatch(context.Background(), client, "1.2.4", &warnings)
	if !strings.Contains(warnings.String(), "version mismatch") {
		t.Errorf("Expected mismatch warning, got %q", warnings.String())
	}
}

func TestWarnOnVersionMismatchSkipsDevelopmentVersion(t *testing.T) {
	client, err := lib.NewClient("http://127.0.0.1:1", true, "")
	if err != nil {
		t.Fatal(err)
	}
	var warnings bytes.Buffer
	warnOnVersionMismatch(context.Background(), client, "devel", &warnings)
	if warnings.Len() != 0 {
		t.Errorf("Expected no warning for development version, got %q", warnings.String())
	}
}

func TestWarnOnVersionMismatchIgnoresUnavailableEndpoint(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client, err := lib.NewClient(server.URL, true, "")
	if err != nil {
		t.Fatal(err)
	}
	var warnings bytes.Buffer
	warnOnVersionMismatch(context.Background(), client, "1.2.0", &warnings)
	if warnings.Len() != 0 {
		t.Errorf("Expected unavailable endpoint to be ignored, got %q", warnings.String())
	}
}

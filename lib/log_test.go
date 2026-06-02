package lib

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamLog(t *testing.T) {
	mockLog := "PLAY [all] ***\nTASK [setup] ***\nok: [testenv1-installer]\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ztp_log/testenv1" {
			t.Errorf("Expected path /ztp_log/testenv1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockLog))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	var buf bytes.Buffer
	if err := client.StreamLog(testEnv, &buf); err != nil {
		t.Fatalf("StreamLog failed: %v", err)
	}

	if buf.String() != mockLog {
		t.Errorf("Expected log output %q, got %q", mockLog, buf.String())
	}
}

func TestStreamLogError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	var buf bytes.Buffer
	if err := client.StreamLog(testEnv, &buf); err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}
}

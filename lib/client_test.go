package lib

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient("https://example.com", true, "")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.BaseURL != "https://example.com" {
		t.Errorf("Expected BaseURL https://example.com, got %s", client.BaseURL)
	}

	if client.HTTPClient == nil {
		t.Fatal("Expected HTTPClient to be non-nil")
	}

	if client.HTTPClient.Timeout.Seconds() != 60 {
		t.Errorf("Expected 60s timeout, got %v", client.HTTPClient.Timeout)
	}
}

func TestNewClientWithVerifySSL(t *testing.T) {
	client, err := NewClient("https://example.com", false, "")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.BaseURL != "https://example.com" {
		t.Errorf("Expected BaseURL https://example.com, got %s", client.BaseURL)
	}
}

func TestNewClientWithInvalidCACert(t *testing.T) {
	_, err := NewClient("https://example.com", false, "/nonexistent/ca.pem")
	if err == nil {
		t.Fatal("Expected error for nonexistent CA cert, got nil")
	}
}

func TestNewClientWithBadCACert(t *testing.T) {
	badCert := filepath.Join(t.TempDir(), "bad-ca.pem")
	if err := os.WriteFile(badCert, []byte("not a real certificate"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := NewClient("https://example.com", false, badCert)
	if err == nil {
		t.Fatal("Expected error for invalid PEM content, got nil")
	}
}

func TestGetRawErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.MaxRetries = 1

	resp, err := client.getRaw(context.Background(), server.URL+"/test")
	if err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}

	if resp != nil {
		t.Fatal("Expected nil response on error")
	}
}

func TestGetRawSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	resp, err := client.getRaw(context.Background(), server.URL+"/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestPostFormErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.MaxRetries = 1

	err := client.postForm(context.Background(), server.URL+"/test", nil)
	if err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

func TestPostFormSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	err := client.postForm(context.Background(), server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestPostFormRedirectStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	err := client.postForm(context.Background(), server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Expected no error for 201 status, got %v", err)
	}
}

func TestGetRawInvalidURL(t *testing.T) {
	client := newTestClient("http://invalid.localhost.test:1")
	client.MaxRetries = 1

	_, err := client.getRaw(context.Background(), "http://invalid.localhost.test:1/test")
	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}
}

func TestRetryOn500ThenSuccess(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	resp, err := client.getRaw(context.Background(), server.URL+"/test")
	if err != nil {
		t.Fatalf("Expected success after retries, got %v", err)
	}
	defer resp.Body.Close()

	if got := attempts.Load(); got != 3 {
		t.Errorf("Expected 3 attempts, got %d", got)
	}
}

func TestNoRetryOn400(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.getRaw(context.Background(), server.URL+"/test")
	if err == nil {
		t.Fatal("Expected error for 400 response, got nil")
	}

	if got := attempts.Load(); got != 1 {
		t.Errorf("Expected 1 attempt (no retry on 4xx), got %d", got)
	}
}

func TestRetryExhausted(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.getRaw(context.Background(), server.URL+"/test")
	if err == nil {
		t.Fatal("Expected error after exhausted retries, got nil")
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("Expected 3 attempts, got %d", got)
	}
}

func TestReadLimited(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		limit     int64
		wantErr   bool
		errSubstr string
	}{
		{
			name:  "under limit",
			data:  []byte("hello"),
			limit: 100,
		},
		{
			name:  "exactly at limit",
			data:  bytes.Repeat([]byte("a"), 100),
			limit: 100,
		},
		{
			name:      "over limit",
			data:      bytes.Repeat([]byte("a"), 101),
			limit:     100,
			wantErr:   true,
			errSubstr: "truncated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := readLimited(bytes.NewReader(tt.data), tt.limit)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readLimited() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("Expected error containing %q, got: %v", tt.errSubstr, err)
			}

			if !tt.wantErr && !bytes.Equal(result, tt.data) {
				t.Errorf("Expected data to match input")
			}
		})
	}
}

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{500, "500 bytes"},
		{1024, "1KB"},
		{4096, "4KB"},
		{1024 * 1024, "1MB"},
		{10 * 1024 * 1024, "10MB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := humanizeBytes(tt.input)
			if got != tt.want {
				t.Errorf("humanizeBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	client, err := NewClient("https://example.com", true, "")
	if err != nil {
		t.Fatal(err)
	}

	newClient := client.WithTimeout(120 * time.Second)

	if newClient.Timeout != 120*time.Second {
		t.Errorf("Expected 120s timeout, got %v", newClient.Timeout)
	}

	if newClient.HTTPClient.Timeout != 120*time.Second {
		t.Errorf("Expected HTTPClient timeout 120s, got %v", newClient.HTTPClient.Timeout)
	}

	if newClient.BaseURL != client.BaseURL {
		t.Errorf("Expected same BaseURL, got %s", newClient.BaseURL)
	}

	if newClient.MaxRetries != client.MaxRetries {
		t.Errorf("Expected same MaxRetries, got %d", newClient.MaxRetries)
	}

	if client.Timeout != 60*time.Second {
		t.Error("Original client timeout should be unchanged")
	}
}

func TestPostFormRawNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("some error body"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.postFormRaw(context.Background(), server.URL+"/test", nil)
	if err == nil {
		t.Fatal("Expected error for 400 response, got nil")
	}

	if !strings.Contains(err.Error(), "bad request") {
		t.Errorf("Expected friendly error message, got: %v", err)
	}
}

func TestFriendlyHTTPError(t *testing.T) {
	tests := []struct {
		code     int
		contains string
	}{
		{400, "bad request"},
		{401, "access denied"},
		{403, "access denied"},
		{404, "not found"},
		{405, "out of date"},
		{409, "conflict"},
		{422, "invalid parameters"},
		{500, "internal server error"},
		{502, "unavailable"},
		{503, "unavailable"},
		{504, "timeout"},
		{418, "HTTP 418"},
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.contains, " ", "-"), func(t *testing.T) {
			got := friendlyHTTPError(tt.code)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("friendlyHTTPError(%d) = %q, want substring %q", tt.code, got, tt.contains)
			}
		})
	}
}

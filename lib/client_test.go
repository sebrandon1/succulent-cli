package lib

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
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

func TestPostFormNoRetryOn500(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.MaxRetries = 3

	err := client.postForm(context.Background(), server.URL+"/test", nil)
	if err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}

	if got := attempts.Load(); got != 1 {
		t.Errorf("Expected 1 attempt (no retry on POST), got %d", got)
	}
}

func TestPostFormRawNoRetryOn500(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.MaxRetries = 3

	_, err := client.postFormRaw(context.Background(), server.URL+"/test", nil)
	if err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}

	if got := attempts.Load(); got != 1 {
		t.Errorf("Expected 1 attempt (no retry on POST), got %d", got)
	}
}

func TestPostFormNoRetryOnTransportError(t *testing.T) {
	var attempts atomic.Int32
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts.Add(1)

		return nil, errors.New("connection refused")
	})

	client := newTestClient("http://example.test")
	client.MaxRetries = 3
	client.HTTPClient.Transport = transport

	err := client.postForm(context.Background(), "http://example.test/test", nil)
	if err == nil {
		t.Fatal("Expected transport error, got nil")
	}

	if got := attempts.Load(); got != 1 {
		t.Errorf("Expected 1 attempt (no retry on POST), got %d", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRetryBackoffHonorsCancel(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.RetryBaseDelay = 500 * time.Millisecond
	client.MaxRetries = 3

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		_, err := client.getRaw(ctx, server.URL+"/test")
		errCh <- err
	}()

	deadline := time.Now().Add(2 * time.Second)
	for attempts.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if attempts.Load() < 1 {
		t.Fatal("first request never completed")
	}

	cancel()
	started := time.Now()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Expected context.Canceled, got %v", err)
		}

		if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
			t.Fatalf("cancel waited %v; backoff should abort immediately", elapsed)
		}
	case <-time.After(time.Second):
		t.Fatal("getRaw did not return after cancel")
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

func TestCopyLimitedDoesNotWriteOverflowByte(t *testing.T) {
	var buf bytes.Buffer
	data := bytes.Repeat([]byte("a"), 101)

	err := copyLimited(&buf, bytes.NewReader(data), 100)
	assertTruncated(t, err)

	if buf.Len() != 100 {
		t.Errorf("copyLimited wrote %d bytes, want 100 (overflow byte must not be copied)", buf.Len())
	}
}

func TestCopyLimitedReadError(t *testing.T) {
	err := copyLimited(io.Discard, errReader{}, 100)
	if err == nil {
		t.Fatal("Expected read error from copyLimited, got nil")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestPostFormRawBodyTooLarge(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"success status", http.StatusOK},
		{"error status", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.Copy(w, oversizedBody())
			}))
			defer server.Close()

			client := newTestClient(server.URL)

			_, err := client.postFormRaw(context.Background(), server.URL+"/test", nil)
			assertTruncated(t, err)
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

func TestEndpointURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		pathFmt string
		args    []any
		want    string
	}{
		{
			name:    "no-args path",
			baseURL: "https://example.com",
			pathFmt: endpointDelete,
			want:    "https://example.com/exposedelete",
		},
		{
			name:    "env interpolation",
			baseURL: "https://example.com",
			pathFmt: endpointInfoPlan,
			args:    []any{"myenv"},
			want:    "https://example.com/infoplan/myenv",
		},
		{
			name:    "trailing slash on BaseURL",
			baseURL: "https://example.com/",
			pathFmt: endpointSNOProvision,
			args:    []any{"env1"},
			want:    "https://example.com/sno/env1",
		},
		{
			name:    "root path",
			baseURL: "https://example.com",
			pathFmt: endpointRoot,
			want:    "https://example.com/",
		},
		{
			name:    "JoinPath fallback on invalid BaseURL",
			baseURL: "://bad",
			pathFmt: endpointDelete,
			want:    "://bad/exposedelete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{BaseURL: tt.baseURL}

			got := c.endpointURL(tt.pathFmt, tt.args...)
			if got != tt.want {
				t.Errorf("endpointURL(%q, %v) = %q, want %q", tt.pathFmt, tt.args, got, tt.want)
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

	logger := slog.New(slog.DiscardHandler)
	client.Logger = logger

	copied := client.WithTimeout(30 * time.Second)
	if copied.Logger != logger {
		t.Error("Expected Logger to be copied by WithTimeout")
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

func TestClient_WithLogger_Verbose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := newTestClient(server.URL)
	client.Logger = logger

	resp, err := client.getRaw(context.Background(), server.URL+"/test")
	if err != nil {
		t.Fatalf("getRaw: %v", err)
	}
	defer resp.Body.Close()

	got := buf.String()
	if !strings.Contains(got, "GET") {
		t.Errorf("expected method GET in log, got %q", got)
	}

	if !strings.Contains(got, "/test") {
		t.Errorf("expected URL path in log, got %q", got)
	}

	if !strings.Contains(got, "status=200") {
		t.Errorf("expected status=200 in log, got %q", got)
	}
}

func TestClient_NilLoggerNoPanic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.Logger = nil

	resp, err := client.getRaw(context.Background(), server.URL+"/test")
	if err != nil {
		t.Fatalf("nil logger should not panic, got %v", err)
	}
	defer resp.Body.Close()
}

package lib

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://example.com", true)

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
	client := NewClient("https://example.com", false)

	if client.BaseURL != "https://example.com" {
		t.Errorf("Expected BaseURL https://example.com, got %s", client.BaseURL)
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

	resp, err := client.getRaw(server.URL + "/test")
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

	resp, err := client.getRaw(server.URL + "/test")
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

	err := client.postForm(server.URL+"/test", nil)
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

	err := client.postForm(server.URL+"/test", nil)
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

	err := client.postForm(server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("Expected no error for 201 status, got %v", err)
	}
}

func TestGetRawInvalidURL(t *testing.T) {
	client := newTestClient("http://invalid.localhost.test:1")
	client.MaxRetries = 1

	_, err := client.getRaw("http://invalid.localhost.test:1/test")
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

	resp, err := client.getRaw(server.URL + "/test")
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

	_, err := client.getRaw(server.URL + "/test")
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

	_, err := client.getRaw(server.URL + "/test")
	if err == nil {
		t.Fatal("Expected error after exhausted retries, got nil")
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("Expected 3 attempts, got %d", got)
	}
}

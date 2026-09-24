package lib

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	if err := client.StreamLog(context.Background(), testEnv, &buf); err != nil {
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
	if err := client.StreamLog(context.Background(), testEnv, &buf); err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}
}

func TestStreamLogBodyTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, oversizedBody())
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	var buf bytes.Buffer
	err := client.StreamLog(context.Background(), testEnv, &buf)
	assertTruncated(t, err)
}

func TestFollowLogWritesOnlyNewOutputUntilPlayRecap(t *testing.T) {
	responses := []string{
		"PLAY [all] ***\nTASK [setup] ***\n",
		"PLAY [all] ***\nTASK [setup] ***\nok: [testenv1-installer]\n",
		"PLAY [all] ***\nTASK [setup] ***\nok: [testenv1-installer]\n\nPLAY RECAP ***\ntestenv1-installer : ok=1 changed=0 failed=0\n",
	}
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ztp_log/testenv1" {
			t.Errorf("Expected path /ztp_log/testenv1, got %s", r.URL.Path)
		}
		if requestCount >= len(responses) {
			t.Errorf("unexpected extra log request")
			return
		}
		_, _ = w.Write([]byte(responses[requestCount]))
		requestCount++
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	var buf bytes.Buffer
	if err := client.FollowLog(context.Background(), testEnv, &buf, time.Millisecond); err != nil {
		t.Fatalf("FollowLog failed: %v", err)
	}

	if got := buf.String(); got != responses[len(responses)-1] {
		t.Errorf("Expected complete log without repeated output %q, got %q", responses[len(responses)-1], got)
	}
	if requestCount != len(responses) {
		t.Errorf("Expected %d requests, got %d", len(responses), requestCount)
	}
}

func TestFollowLogStopsWhenContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("PLAY [all] ***\n"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := cancelingLogWriter{cancel: cancel}
	err := client.FollowLog(ctx, testEnv, &w, time.Millisecond)
	if err != context.Canceled {
		t.Fatalf("Expected context cancellation error, got %v", err)
	}
	if got := w.buf.String(); got != "PLAY [all] ***\n" {
		t.Errorf("Expected initial log output before cancellation, got %q", got)
	}
}

func TestFollowLogHandlesLogLargerThanResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", MaxResponseSize+1))
		_, _ = io.WriteString(w, "\nPLAY RECAP ***\n")
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	if err := client.FollowLog(context.Background(), testEnv, io.Discard, time.Millisecond); err != nil {
		t.Fatalf("FollowLog failed for large log: %v", err)
	}
}

func TestFollowLogRejectsNonPositivePollInterval(t *testing.T) {
	client := newTestClient("http://127.0.0.1")
	err := client.FollowLog(context.Background(), testEnv, io.Discard, 0)
	if err == nil {
		t.Fatal("Expected an error for a non-positive poll interval, got nil")
	}
}

func TestFollowLogReturnsErrorWhenLogShrinks(t *testing.T) {
	responses := []string{"PLAY [all] ***\nTASK [one] ***\n", "short"}
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, responses[requestCount])
		requestCount++
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.FollowLog(context.Background(), testEnv, io.Discard, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "shrank while following") {
		t.Fatalf("Expected a shrinking log error, got %v", err)
	}
}

func TestFollowLogReturnsWriterError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "PLAY [all] ***\n")
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	wantErr := errors.New("writer failed")
	err := client.FollowLog(context.Background(), testEnv, failingLogWriter{err: wantErr}, time.Millisecond)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected writer error %v, got %v", wantErr, err)
	}
}

type cancelingLogWriter struct {
	cancel context.CancelFunc
	buf    bytes.Buffer
}

func (w *cancelingLogWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	w.cancel()
	return n, err
}

type failingLogWriter struct {
	err error
}

func (w failingLogWriter) Write([]byte) (int, error) {
	return 0, w.err
}

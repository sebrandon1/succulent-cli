package lib

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

// writeAndFlush sends a chunk of a streamed log response immediately.
func writeAndFlush(t *testing.T, w http.ResponseWriter, chunk string) {
	t.Helper()
	_, _ = io.WriteString(w, chunk)
	w.(http.Flusher).Flush()
}

func TestFollowLogStreamsPastClientTimeout(t *testing.T) {
	chunks := []string{"PLAY [all] ***\n", "TASK [slow] ***\n", "\nPLAY RECAP ***\nhost : ok=1 failed=0\n"}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		for i, chunk := range chunks {
			if i > 0 {
				time.Sleep(150 * time.Millisecond)
			}
			writeAndFlush(t, w, chunk)
		}
	}))
	defer server.Close()

	// A request timeout shorter than the gaps between chunks must not cut the
	// live stream short.
	client := newTestClient(server.URL).WithTimeout(50 * time.Millisecond)
	var buf bytes.Buffer
	if err := client.FollowLog(context.Background(), testEnv, &buf, time.Millisecond); err != nil {
		t.Fatalf("FollowLog failed: %v", err)
	}

	if got, want := buf.String(), strings.Join(chunks, ""); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1 (one long-lived stream)", got)
	}
}

func TestFollowLogReconnectsWithoutDuplicateOutput(t *testing.T) {
	const first = "PLAY [all] ***\nTASK [one] ***\n"
	const full = first + "TASK [two] ***\n\nPLAY RECAP ***\nhost : ok=2 failed=0\n"
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			writeAndFlush(t, w, first)
			panic(http.ErrAbortHandler) // drop the connection mid-stream
		}
		_, _ = io.WriteString(w, full)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	var buf bytes.Buffer
	if err := client.FollowLog(context.Background(), testEnv, &buf, time.Millisecond); err != nil {
		t.Fatalf("FollowLog failed: %v", err)
	}

	if got := buf.String(); got != full {
		t.Errorf("output = %q, want each line once: %q", got, full)
	}
	if got := requests.Load(); got != 2 {
		t.Errorf("requests = %d, want 2", got)
	}
}

func TestFollowLogGivesUpAfterConsecutiveFailures(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			writeAndFlush(t, w, "PLAY [all] ***\n")
		}
		panic(http.ErrAbortHandler)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.FollowLog(context.Background(), testEnv, io.Discard, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "consecutive log stream failures") {
		t.Fatalf("expected a give-up error, got %v", err)
	}
}

func TestFollowLogFailsFastOnFirstRequestError(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	if err := client.FollowLog(context.Background(), testEnv, io.Discard, time.Millisecond); err == nil {
		t.Fatal("expected an error for a 404 on the first request")
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want 1 (no reconnects before any output)", got)
	}
}

func TestStreamLogStillStreaming(t *testing.T) {
	tests := []struct {
		name         string
		partial      string
		wantSentinel bool
	}{
		{name: "timeout after partial output", partial: "PLAY [all] ***\n", wantSentinel: true},
		{name: "timeout before any output", partial: "", wantSentinel: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.partial != "" {
					writeAndFlush(t, w, tt.partial)
				}
				<-r.Context().Done() // keep the response open like a running playbook
			}))
			defer server.Close()

			client := newTestClient(server.URL).WithTimeout(100 * time.Millisecond)
			var buf bytes.Buffer
			err := client.StreamLog(context.Background(), testEnv, &buf)

			if got := errors.Is(err, ErrLogStillStreaming); got != tt.wantSentinel {
				t.Fatalf("errors.Is(err, ErrLogStillStreaming) = %v, want %v (err: %v)", got, tt.wantSentinel, err)
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := buf.String(); got != tt.partial {
				t.Errorf("output = %q, want %q", got, tt.partial)
			}
		})
	}
}

func TestTransientLogErrorMessage(t *testing.T) {
	inner := errors.New("connection reset")
	if got := (&transientLogError{err: inner}).Error(); got != inner.Error() {
		t.Errorf("Error() = %q, want %q", got, inner.Error())
	}
}

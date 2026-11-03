package lib

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	oldRunLog = "\nPLAY [localhost] ***\nMonday 02 November 2026  09:00:00 -0500 (0:00:00.010)\n" +
		"TASK [old] ***\nok: [hub]\n\nPLAY RECAP ***\nhub : ok=1 changed=0 unreachable=0 failed=0\n"
	newRunHeader = "\nPLAY [localhost] ***\nTuesday 03 November 2026  10:04:04 -0500 (0:00:00.016)\n"
	successLog   = newRunHeader + "TASK [deploy] ***\nok: [hub]\n\nPLAY RECAP ***\n" +
		"hub       : ok=77 changed=27 unreachable=0 failed=0 skipped=55\n" +
		"localhost : ok=13 changed=1  unreachable=0 failed=0 skipped=7\n"
	failedLog = newRunHeader + "TASK [lvms_operator_install : Fail if no recommended version found] ***\n" +
		"fatal: [hub]: FAILED! => {\n    \"changed\": false\n}\n\nMSG:\n\nNo recommended version of LVMS found for the current version.\n\n" +
		"PLAY RECAP ***\nhub       : ok=77 changed=27 unreachable=0 failed=1 skipped=55\n" +
		"localhost : ok=13 changed=1  unreachable=0 failed=0 skipped=7\n"
)

func TestParsePlayRecap(t *testing.T) {
	tests := []struct {
		name    string
		log     string
		wantErr []string
	}{
		{name: "all hosts succeeded", log: successLog},
		{name: "failed host with fatal task", log: failedLog, wantErr: []string{
			"playbook failed on hub", "TASK [lvms_operator_install", "No recommended version of LVMS",
		}},
		{name: "unreachable host", log: "PLAY RECAP ***\nspoke : ok=3 changed=0 unreachable=1 failed=0\n",
			wantErr: []string{"playbook failed on spoke"}},
		{name: "several failed hosts", log: "PLAY RECAP ***\na : ok=1 failed=2\nb : ok=1 failed=0\nc : ok=1 unreachable=3\n",
			wantErr: []string{"playbook failed on a, c"}},
		{name: "uses the last recap", log: failedLog + "\n" + successLog},
		{name: "no recap", log: newRunHeader + "TASK [deploy] ***\n", wantErr: []string{"no PLAY RECAP"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParsePlayRecap([]byte(tt.log))
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
		})
	}
}

func TestLastFatalTruncatesLongOutput(t *testing.T) {
	got := lastFatal("TASK [x] ***\nfatal: [h]: FAILED! => " + strings.Repeat("y", 3*maxFatalExcerpt))
	if !strings.HasSuffix(got, "\n...") || len(got) != maxFatalExcerpt+len("\n...") {
		t.Errorf("excerpt not truncated to %d bytes: len=%d", maxFatalExcerpt, len(got))
	}
	if lastFatal("TASK [x] ***\nok: [h]\n") != "" {
		t.Error("expected no excerpt without a fatal line")
	}
}

func TestLogPrefix(t *testing.T) {
	origTimeout := logPrefixTimeout
	logPrefixTimeout = 100 * time.Millisecond
	t.Cleanup(func() { logPrefixTimeout = origTimeout })

	tests := []struct {
		name    string
		body    string
		stall   bool
		status  int
		want    string
		wantErr bool
	}{
		{name: "short finished log", body: successLog, want: successLog},
		{name: "long log is capped", body: strings.Repeat("x", 2*logPrefixSize), want: strings.Repeat("x", logPrefixSize)},
		{name: "still streaming returns the partial prefix", body: newRunHeader, stall: true, want: newRunHeader},
		{name: "server error", status: http.StatusNotFound, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status != 0 {
					w.WriteHeader(tt.status)
					return
				}
				_, _ = io.WriteString(w, tt.body)
				if tt.stall {
					w.(http.Flusher).Flush()
					<-r.Context().Done()
				}
			}))
			defer server.Close()

			got, err := newTestClient(server.URL).LogPrefix(context.Background(), testEnv)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if string(got) != tt.want {
				t.Errorf("prefix = %q, want %q", got, tt.want)
			}
		})
	}
}

// playbookServer serves the old run's log until switched, then the new run's.
func playbookServer(t *testing.T, newLog string, switchAfter int32) *httptest.Server {
	t.Helper()
	var requests atomic.Int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) <= switchAfter {
			_, _ = io.WriteString(w, oldRunLog)
			return
		}
		_, _ = io.WriteString(w, newLog)
	}))
}

func TestWaitForPlaybook(t *testing.T) {
	baseline := []byte(oldRunLog)

	t.Run("succeeds once the new run finishes", func(t *testing.T) {
		server := playbookServer(t, successLog, 2)
		defer server.Close()

		var progress bytes.Buffer
		err := newTestClient(server.URL).WaitForPlaybook(context.Background(), testEnv, baseline, time.Millisecond, &progress)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if progress.String() != successLog {
			t.Errorf("progress = %q, want the new run's log only", progress.String())
		}
	})

	t.Run("reports the failed task", func(t *testing.T) {
		server := playbookServer(t, failedLog, 1)
		defer server.Close()

		err := newTestClient(server.URL).WaitForPlaybook(context.Background(), testEnv, baseline, time.Millisecond, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "No recommended version of LVMS") {
			t.Fatalf("expected the fatal task in the error, got %v", err)
		}
	})

	t.Run("an old run's recap is not mistaken for the new run", func(t *testing.T) {
		server := playbookServer(t, successLog, 1<<30) // never switches
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		err := newTestClient(server.URL).WaitForPlaybook(ctx, testEnv, baseline, time.Millisecond, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "waiting for the playbook to start") {
			t.Fatalf("expected a timeout waiting for the new run, got %v", err)
		}
	})

	t.Run("no baseline follows the current log", func(t *testing.T) {
		server := playbookServer(t, successLog, 0)
		defer server.Close()

		if err := newTestClient(server.URL).WaitForPlaybook(context.Background(), testEnv, nil, time.Millisecond, io.Discard); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWaitForNewRunComparesEqualLengthPrefixes(t *testing.T) {
	// The new run has only written the header line shared with the old run;
	// it must not count as new until its own timestamps appear.
	shared := "\nPLAY [localhost] ***\n"
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			_, _ = io.WriteString(w, shared)
			return
		}
		_, _ = io.WriteString(w, newRunHeader)
	}))
	defer server.Close()

	err := newTestClient(server.URL).waitForNewRun(context.Background(), testEnv, []byte(oldRunLog), time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := requests.Load(); got != 2 {
		t.Errorf("requests = %d, want 2 (the shared header alone is not a new run)", got)
	}
}

func TestTailWriterKeepsLastBytes(t *testing.T) {
	tail := &tailWriter{max: 5}
	for _, chunk := range []string{"abc", "defg", "h"} {
		if n, err := tail.Write([]byte(chunk)); err != nil || n != len(chunk) {
			t.Fatalf("Write(%q) = %d, %v", chunk, n, err)
		}
	}
	if got := string(tail.buf); got != "defgh" {
		t.Errorf("tail = %q, want %q", got, "defgh")
	}
}

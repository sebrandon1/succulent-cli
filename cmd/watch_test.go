package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func resetWatchFlags(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		pollIntervalSecs = defaultPollIntervalSecs
		maxWaitMinutes = defaultMaxWaitMinutes
		_ = watchCmd.Flags().Set("poll-interval", "0")
		_ = watchCmd.Flags().Set("max-wait", fmt.Sprint(defaultMaxWaitMinutes))
	})
}

func TestWatchCommandRejectsPollIntervalBelowMinimum(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	resetWatchFlags(t)

	rootCmd.SetArgs([]string{"watch", "--env", "testenv", "--poll-interval", "1"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error for a poll interval below the minimum")
	}
	if !strings.Contains(err.Error(), "--poll-interval must be 0 (adaptive) or at least 5 seconds") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWatchCommandJSON(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/infoplan/testenv" {
			http.NotFound(w, r)
			return
		}

		_, _ = fmt.Fprint(w, testInfoHTML)
	})
	defer cleanup()

	resetWatchFlags(t)
	resetOutputFormat(t)

	stdout := captureCommandStdout(t)
	stderr := captureCommandStderr(t)

	rootCmd.SetArgs([]string{"watch", "--env", "testenv", "--output", "json", "--poll-interval", "5"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	_ = stderr()

	var result CommandResult
	if err := json.Unmarshal([]byte(stdout()), &result); err != nil {
		t.Fatalf("output is not valid CommandResult JSON: %v", err)
	}

	if result.Status != "ready" || result.Environment != "testenv" || !strings.Contains(result.Message, "192.168.1.100") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

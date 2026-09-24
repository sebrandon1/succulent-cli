package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestSNOProvisionWatchWaitsUntilReady(t *testing.T) {
	previousWatch, previousWait, previousPoll, previousControlPlane := watchAfterProvision, provisionWaitMinutes, provisionPollIntervalSecs, provisionControlPlaneOnly
	t.Cleanup(func() {
		watchAfterProvision, provisionWaitMinutes, provisionPollIntervalSecs, provisionControlPlaneOnly = previousWatch, previousWait, previousPoll, previousControlPlane
	})

	provisionRequests := 0
	infoRequests := 0
	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sno/testenv":
			provisionRequests++
			w.WriteHeader(http.StatusOK)
		case "/infoplan/testenv":
			infoRequests++
			_, _ = fmt.Fprint(w, testInfoHTML)
		default:
			http.NotFound(w, r)
		}
	})
	defer cleanup()

	stdout := captureCommandStdout(t)
	rootCmd.SetArgs([]string{
		"sno", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com",
		"--ocp-tag", "4.17", "--confirm", "--watch", "--max-wait", "1", "--poll-interval", "5", "--output", "json",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	output := stdout()

	var result CommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output %q is not valid result JSON: %v", output, err)
	}
	if provisionRequests != 1 || infoRequests != 1 {
		t.Fatalf("requests: provision = %d, info = %d; want 1 each", provisionRequests, infoRequests)
	}
	if result.Status != "ready" || result.Environment != "testenv" || !strings.Contains(result.Message, "cluster ready") || !strings.Contains(result.Message, "192.168.1.100") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSNOProvisionWatchReportsReadinessFailure(t *testing.T) {
	previousWatch, previousWait, previousPoll := watchAfterProvision, provisionWaitMinutes, provisionPollIntervalSecs
	t.Cleanup(func() {
		watchAfterProvision, provisionWaitMinutes, provisionPollIntervalSecs = previousWatch, previousWait, previousPoll
	})

	provisionRequests := 0
	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sno/testenv":
			provisionRequests++
			w.WriteHeader(http.StatusOK)
		case "/infoplan/testenv":
			_, _ = fmt.Fprint(w, `<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>testenv</td><td>client1</td><td>2026-05-27</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>testenv-installer</td><td>error</td><td>192.168.1.100</td></tr>
</table></body></html>`)
		default:
			http.NotFound(w, r)
		}
	})
	defer cleanup()

	rootCmd.SetArgs([]string{
		"sno", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com",
		"--ocp-tag", "4.17", "--confirm", "--watch", "--max-wait", "1", "--poll-interval", "5",
	})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "request submitted, but waiting for the cluster failed") {
		t.Fatalf("Execute() error = %v, want submitted-then-watch error", err)
	}
	if provisionRequests != 1 {
		t.Fatalf("provision requests = %d, want 1", provisionRequests)
	}
}

func TestValidateProvisionWatchFlags(t *testing.T) {
	previousWatch, previousWait, previousPoll := watchAfterProvision, provisionWaitMinutes, provisionPollIntervalSecs
	t.Cleanup(func() {
		watchAfterProvision, provisionWaitMinutes, provisionPollIntervalSecs = previousWatch, previousWait, previousPoll
	})

	tests := []struct {
		name       string
		watch      bool
		maxWait    int
		poll       int
		dryRun     bool
		wantErrMsg string
	}{
		{name: "disabled", watch: false, maxWait: 0, poll: 0},
		{name: "valid", watch: true, maxWait: 1, poll: minPollIntervalSecs},
		{name: "dry run", watch: true, dryRun: true, maxWait: 1, poll: minPollIntervalSecs, wantErrMsg: "--watch cannot be used with --dry-run"},
		{name: "nonpositive wait", watch: true, maxWait: 0, poll: minPollIntervalSecs, wantErrMsg: "--max-wait"},
		{name: "short poll", watch: true, maxWait: 1, poll: minPollIntervalSecs - 1, wantErrMsg: "--poll-interval"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watchAfterProvision = tt.watch
			provisionWaitMinutes = tt.maxWait
			provisionPollIntervalSecs = tt.poll
			err := validateProvisionWatchFlags(tt.dryRun)
			if tt.wantErrMsg == "" && err != nil {
				t.Fatalf("validateProvisionWatchFlags() error = %v", err)
			}
			if tt.wantErrMsg != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrMsg)) {
				t.Fatalf("validateProvisionWatchFlags() error = %v, want %q", err, tt.wantErrMsg)
			}
		})
	}
}

func captureCommandStdout(t *testing.T) func() string {
	t.Helper()
	previous := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	t.Cleanup(func() {
		os.Stdout = previous
		_ = read.Close()
	})
	return func() string {
		t.Helper()
		_ = write.Close()
		os.Stdout = previous
		data, err := io.ReadAll(read)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
}

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
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
		{name: "adaptive polling", watch: true, maxWait: 1, poll: 0},
		{name: "valid", watch: true, maxWait: 1, poll: minPollIntervalSecs},
		{name: "dry run", watch: true, dryRun: true, maxWait: 1, poll: minPollIntervalSecs, wantErrMsg: "--watch cannot be used with --dry-run"},
		{name: "nonpositive wait", watch: true, maxWait: 0, poll: minPollIntervalSecs, wantErrMsg: "--max-wait"},
		{name: "short poll", watch: true, maxWait: 1, poll: minPollIntervalSecs - 1, wantErrMsg: "--poll-interval"},
		{name: "negative poll", watch: true, maxWait: 1, poll: -1, wantErrMsg: "--poll-interval"},
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

const (
	ztpOldRunLog = "\nPLAY [localhost] ***\nMonday 02 November 2026  09:00:00 -0500\nTASK [old] ***\n\nPLAY RECAP ***\nhub : ok=1 unreachable=0 failed=0\n"
	ztpNewHeader = "\nPLAY [localhost] ***\nTuesday 03 November 2026  10:04:04 -0500\n"
)

// ztpWatchServer serves the previous run's log until the provision request
// arrives, then newLog, plus a ready info plan.
func ztpWatchServer(t *testing.T, newLog string) (cleanup func()) {
	t.Helper()
	submitted := false
	return setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/create_ztp":
			submitted = true
			w.WriteHeader(http.StatusOK)
		case "/ztp_log/testenv":
			if submitted {
				_, _ = io.WriteString(w, newLog)
			} else {
				_, _ = io.WriteString(w, ztpOldRunLog)
			}
		case "/infoplan/testenv":
			_, _ = fmt.Fprint(w, testInfoHTML)
		default:
			http.NotFound(w, r)
		}
	})
}

func runZTPProvisionWatch(t *testing.T) error {
	t.Helper()
	previousWatch, previousWait, previousPoll := watchAfterProvision, provisionWaitMinutes, provisionPollIntervalSecs
	t.Cleanup(func() {
		watchAfterProvision, provisionWaitMinutes, provisionPollIntervalSecs = previousWatch, previousWait, previousPoll
	})

	rootCmd.SetArgs([]string{
		"ztp", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com",
		"--sno-tag", "4.22", "--spoke-tag", "4.22", "--confirm", "--watch", "--max-wait", "1", "--poll-interval", "5", "--output", "json",
	})
	return rootCmd.Execute()
}

func TestZTPProvisionWatchWaitsForPlaybook(t *testing.T) {
	cleanup := ztpWatchServer(t, ztpNewHeader+"TASK [deploy] ***\nok: [hub]\n\nPLAY RECAP ***\nhub : ok=5 unreachable=0 failed=0\n")
	defer cleanup()

	stdout := captureCommandStdout(t)
	if err := runZTPProvisionWatch(t); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var result CommandResult
	if err := json.Unmarshal([]byte(stdout()), &result); err != nil {
		t.Fatalf("output is not valid result JSON: %v", err)
	}
	if result.Status != "ready" || !strings.Contains(result.Message, "playbook finished; cluster ready") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestZTPProvisionWatchReportsFailedPlaybook(t *testing.T) {
	cleanup := ztpWatchServer(t, ztpNewHeader+"TASK [lvms_operator_install : Fail if no recommended version found] ***\n"+
		"fatal: [hub]: FAILED! => {}\n\nMSG:\n\nNo recommended version of LVMS found for the current version.\n\n"+
		"PLAY RECAP ***\nhub : ok=77 unreachable=0 failed=1\n")
	defer cleanup()

	err := runZTPProvisionWatch(t)
	if err == nil {
		t.Fatal("expected an error for a failed playbook, even though the nodes are up")
	}
	for _, want := range []string{"the playbook did not succeed", "playbook failed on hub", "No recommended version of LVMS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func TestTaskProgressWriter(t *testing.T) {
	var out strings.Builder
	progress := &taskProgressWriter{w: &out}

	// Lines split across writes are reassembled; only headers are printed.
	for _, chunk := range []string{"\nPLAY [localhost] ****\nMonday 02 Nov", "ember\nTASK [deploy hub] ***\nok: [hub]\r\n", "\nPLAY RECAP ***\nhub : ok=1\n"} {
		if n, err := progress.Write([]byte(chunk)); err != nil || n != len(chunk) {
			t.Fatalf("Write(%q) = %d, %v", chunk, n, err)
		}
	}

	want := "  PLAY [localhost]\n  TASK [deploy hub]\n  PLAY RECAP\n"
	if got := out.String(); got != want {
		t.Errorf("progress = %q, want %q", got, want)
	}
}

func TestPlaybookPollInterval(t *testing.T) {
	previous := provisionPollIntervalSecs
	t.Cleanup(func() { provisionPollIntervalSecs = previous })

	provisionPollIntervalSecs = 0
	if got := playbookPollInterval(); got != defaultPlaybookPollInterval {
		t.Errorf("adaptive polling: got %v, want %v", got, defaultPlaybookPollInterval)
	}
	provisionPollIntervalSecs = 7
	if got := playbookPollInterval(); got != 7*time.Second {
		t.Errorf("fixed polling: got %v, want 7s", got)
	}
}

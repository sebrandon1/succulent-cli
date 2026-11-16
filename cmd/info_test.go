package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/sebrandon1/succulent-cli/lib"
)

// captureCommandStderr captures what a command writes to os.Stderr. Call the
// returned function once to restore os.Stderr and read the captured output.
func captureCommandStderr(t *testing.T) func() string {
	t.Helper()

	previous := os.Stderr

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stderr = write
	t.Cleanup(func() {
		os.Stderr = previous
		_ = read.Close()
	})

	return func() string {
		t.Helper()

		_ = write.Close()
		os.Stderr = previous

		data, err := io.ReadAll(read)
		if err != nil {
			t.Fatal(err)
		}

		return string(data)
	}
}

func resetOutputFormat(t *testing.T) {
	t.Helper()

	previous := outputFormat
	t.Cleanup(func() {
		outputFormat = previous
		_ = rootCmd.PersistentFlags().Set("output", previous)
	})
}

func resetInfoNoCache(t *testing.T) {
	t.Helper()

	infoNoCache = false
	_ = infoCmd.Flags().Set("no-cache", "false")

	t.Cleanup(func() {
		infoNoCache = false
		_ = infoCmd.Flags().Set("no-cache", "false")
	})
}

func TestInfoCommandCachesAndNoCacheBypasses(t *testing.T) {
	configCleanup := setupTestConfigDir(t)
	defer configCleanup()

	requests := 0

	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/infoplan/testenv" {
			http.NotFound(w, r)
			return
		}

		requests++
		_, _ = fmt.Fprint(w, testInfoHTML)
	})
	defer cleanup()

	resetOutputFormat(t)
	resetInfoNoCache(t)

	stderr := captureCommandStderr(t)
	discardStdout := captureCommandStdout(t)

	rootCmd.SetArgs([]string{"get", "info", "--env", "testenv"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first run: %v", err)
	}
	_ = discardStdout()

	if got := stderr(); strings.Contains(got, "(cached)") {
		t.Errorf("first run used the cache: stderr = %q", got)
	}
	if requests != 1 {
		t.Fatalf("requests after first run = %d, want 1", requests)
	}

	stderr = captureCommandStderr(t)
	discardStdout = captureCommandStdout(t)

	rootCmd.SetArgs([]string{"get", "info", "--env", "testenv"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second run: %v", err)
	}
	_ = discardStdout()

	if got := stderr(); !strings.Contains(got, "(cached)") {
		t.Errorf("second run did not use the cache: stderr = %q", got)
	}
	if requests != 1 {
		t.Fatalf("requests after cached run = %d, want 1", requests)
	}

	discardStdout = captureCommandStdout(t)

	rootCmd.SetArgs([]string{"get", "info", "--env", "testenv", "--no-cache"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("--no-cache run: %v", err)
	}
	_ = discardStdout()

	if requests != 2 {
		t.Fatalf("requests after --no-cache run = %d, want 2", requests)
	}
}

func TestInfoCommandJSON(t *testing.T) {
	configCleanup := setupTestConfigDir(t)
	defer configCleanup()

	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/infoplan/testenv" {
			http.NotFound(w, r)
			return
		}

		_, _ = fmt.Fprint(w, testInfoHTML)
	})
	defer cleanup()

	resetOutputFormat(t)
	resetInfoNoCache(t)

	stdout := captureCommandStdout(t)
	rootCmd.SetArgs([]string{"get", "info", "--env", "testenv", "--output", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var info lib.ClusterInfo
	if err := json.Unmarshal([]byte(stdout()), &info); err != nil {
		t.Fatalf("output is not valid ClusterInfo JSON: %v", err)
	}

	if info.Environment != "testenv" || info.InstallerIP != "192.168.1.100" || len(info.Nodes) != 2 {
		t.Fatalf("unexpected info: %+v", info)
	}
}

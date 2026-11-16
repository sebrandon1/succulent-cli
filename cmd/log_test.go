package cmd

import (
	"net/http"
	"strings"
	"testing"
)

func TestLogCommandFollowFailsOnInitialFetchError(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	defer cleanup()

	t.Cleanup(func() {
		followLog = false
		_ = logCmd.Flags().Set("follow", "false")
	})

	rootCmd.SetArgs([]string{"get", "log", "--env", "testenv", "--follow"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when the first follow fetch fails")
	}
	if !strings.Contains(err.Error(), "fetching log") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

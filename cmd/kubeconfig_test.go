package cmd

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestKubeconfigFetchRequiresEnvPlaceholderForMultipleEnvironments(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	// setupTestServer sets viper's env directly, which outranks the flag
	// binding; set the comma-separated target list through viper as well.
	viper.Set("env", "env-one,env-two")

	t.Cleanup(func() {
		selectedEnvironments = nil
		destPath = ""
		_ = fetchKubeconfigCmd.Flags().Set("dest", "")
	})

	rootCmd.SetArgs([]string{"kubeconfig", "fetch", "--env", "env-one,env-two", "--dest", "./kubeconfig.yaml"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error for a shared --dest with multiple environments")
	}
	if !strings.Contains(err.Error(), "must include {env}") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKubeconfigFetchWithoutInstallerIP(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/infoplan/testenv" {
			http.NotFound(w, r)
			return
		}

		_, _ = fmt.Fprint(w, `<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>testenv</td><td>client1</td><td>2026-05-27</td></tr>
</table></body></html>`)
	})
	defer cleanup()

	t.Cleanup(func() {
		destPath = ""
		_ = fetchKubeconfigCmd.Flags().Set("dest", "")
	})

	dest := filepath.Join(t.TempDir(), "kubeconfig")
	rootCmd.SetArgs([]string{"kubeconfig", "fetch", "--env", "testenv", "--dest", dest})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when the infoplan has no installer IP")
	}
	if !strings.Contains(err.Error(), "could not determine installer IP") {
		t.Fatalf("unexpected error: %v", err)
	}
}

package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func setupTestServer(handler http.HandlerFunc) func() {
	server := httptest.NewServer(handler)
	viper.Set("url", server.URL)
	viper.Set("env", "testenv")
	viper.Set("verify_ssl", false)

	return func() {
		server.Close()
		sharedClient = nil
		viper.Set("url", "")
		viper.Set("env", "")
	}
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

const testInfoHTML = `<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>testenv</td><td>client1</td><td>2026-05-27</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>testenv-installer</td><td>up</td><td>192.168.1.100</td></tr>
<tr><td>testenv-master-0</td><td>up</td><td>192.168.1.101</td></tr>
</table></body></html>`

const testMainPageHTML = `<html><body><table>
<tr style="background-color: #e0f7fa;"><th colspan="3">Hosts TEST</th></tr>
<tr><th>testenv</th><th>
  <button onclick="location.href='/infoplan/testenv'">Info</button>
</th></tr>
</table></body></html>`

func TestInfoCommand(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testInfoHTML))
	})
	defer cleanup()

	rootCmd.SetArgs([]string{"get", "info", "--env", "testenv"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestInfoCommandTable(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testInfoHTML))
	})
	defer cleanup()

	rootCmd.SetArgs([]string{"get", "info", "--env", "testenv", "--output", "table"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestLogCommand(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("PLAY [all] ***\nTASK [setup] ***\n"))
	})
	defer cleanup()

	rootCmd.SetArgs([]string{"get", "log", "--env", "testenv"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDeleteCommand(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	rootCmd.SetArgs([]string{"delete", "--env", "testenv", "--confirm"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDeleteCommandNoConfirm(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	confirmDelete = false
	rootCmd.SetArgs([]string{"delete", "--env", "testenv"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected error without --confirm, got nil")
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		w.Write([]byte(testMainPageHTML))
	case "/infoplan/testenv":
		w.Write([]byte(testInfoHTML))
	default:
		w.Write([]byte("<html><body><table></table></body></html>"))
	}
}

func TestListCommand(t *testing.T) {
	cleanup := setupTestServer(listHandler)
	defer cleanup()

	rootCmd.SetArgs([]string{"list", "--env", "testenv"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestListCommandNoDetail(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(testMainPageHTML))
	})
	defer cleanup()

	rootCmd.SetArgs([]string{"list", "--env", "testenv", "--no-detail"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestListCommandJSON(t *testing.T) {
	cleanup := setupTestServer(listHandler)
	defer cleanup()

	rootCmd.SetArgs([]string{"list", "--env", "testenv", "--output", "json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWatchCommand(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testInfoHTML))
	})
	defer cleanup()

	rootCmd.SetArgs([]string{"watch", "--env", "testenv", "--max-wait", "1", "--poll-interval", "5"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestReprovisionCommand(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	rootCmd.SetArgs([]string{"reprovision", "--env", "testenv", "--email", "test@example.com", "--owner", "testuser", "--ocp-tag", "4.17", "--confirm"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestReprovisionCommandNoConfirm(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	confirmReprovision = false
	rootCmd.SetArgs([]string{"reprovision", "--env", "testenv", "--email", "test@example.com", "--owner", "testuser", "--ocp-tag", "4.17"})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("Expected error without --confirm, got nil")
	}
}

func TestSNOProvisionCommand(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	rootCmd.SetArgs([]string{"sno", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--ocp-tag", "4.17", "--confirm"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSNOProvisionCommandNoConfirm(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	confirmSNO = false
	rootCmd.SetArgs([]string{"sno", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--ocp-tag", "4.17"})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("Expected error without --confirm, got nil")
	}
}

func TestSNOKubeconfigCommand(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("apiVersion: v1\nkind: Config\n"))
	})
	defer cleanup()

	dest := filepath.Join(t.TempDir(), "kubeconfig")
	rootCmd.SetArgs([]string{"sno", "kubeconfig", "--env", "testenv", "--dest", dest})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("Expected kubeconfig file at %s, got error: %v", dest, err)
	}
}

func TestHypershiftProvisionCommand(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	rootCmd.SetArgs([]string{"hypershift", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--sno-tag", "4.17", "--hcp-tag", "4.17", "--confirm"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestHypershiftProvisionCommandNoConfirm(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	confirmHS = false
	rootCmd.SetArgs([]string{"hypershift", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--sno-tag", "4.17", "--hcp-tag", "4.17"})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("Expected error without --confirm, got nil")
	}
}

func TestHypershiftKubeconfigCommand(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("apiVersion: v1\nkind: Config\n"))
	})
	defer cleanup()

	dest := filepath.Join(t.TempDir(), "kubeconfig")
	rootCmd.SetArgs([]string{"hypershift", "kubeconfig", "--env", "testenv", "--choice", "management", "--dest", dest})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestZTPProvisionCommand(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	rootCmd.SetArgs([]string{"ztp", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--sno-tag", "4.17", "--spoke-tag", "4.17", "--confirm"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestZTPProvisionCommandNoConfirm(t *testing.T) {
	cleanup := setupTestServer(okHandler)
	defer cleanup()

	confirmZTP = false
	rootCmd.SetArgs([]string{"ztp", "provision", "--env", "testenv", "--owner", "testuser", "--email", "test@example.com", "--sno-tag", "4.17", "--spoke-tag", "4.17"})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("Expected error without --confirm, got nil")
	}
}

func TestZTPKubeconfigCommand(t *testing.T) {
	cleanup := setupTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("apiVersion: v1\nkind: Config\n"))
	})
	defer cleanup()

	dest := filepath.Join(t.TempDir(), "kubeconfig")
	rootCmd.SetArgs([]string{"ztp", "kubeconfig", "--env", "testenv", "--choice", "spoke", "--dest", dest})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestVersionCommand(t *testing.T) {
	SetVersion("v0.0.1-test")
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestConfigShowCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"config", "show"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestConfigPathCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"config", "path"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestConfigInitCommand(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	defer func() { t.Setenv("HOME", origHome) }()

	rootCmd.SetArgs([]string{"config", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	configPath := filepath.Join(tmpDir, ".config", "succulent-cli", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Expected config file at %s, got error: %v", configPath, err)
	}
}

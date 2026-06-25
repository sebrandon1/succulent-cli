package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	data := struct {
		Name string `json:"name"`
	}{Name: "test"}

	if err := printJSON(data); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDefaultDestPath(t *testing.T) {
	path, err := defaultDestPath("myenv", "kubeconfig")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.HasSuffix(path, "Downloads/myenv-kubeconfig") {
		t.Errorf("Expected path ending with Downloads/myenv-kubeconfig, got %s", path)
	}
}

func TestResolveOwnerEmail(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		email     string
		wantOwner string
		wantEmail string
		wantErr   string
	}{
		{
			name:      "both provided",
			owner:     "myuser",
			email:     "user@example.com",
			wantOwner: "myuser",
			wantEmail: "user@example.com",
		},
		{
			name:    "owner missing",
			owner:   "",
			email:   "user@example.com",
			wantErr: "--owner is required",
		},
		{
			name:    "email missing",
			owner:   "myuser",
			email:   "",
			wantErr: "--email is required",
		},
		{
			name:    "both missing",
			owner:   "",
			email:   "",
			wantErr: "--owner is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			owner, email, err := resolveOwnerEmail(tc.owner, tc.email)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Expected error containing %q, got nil", tc.wantErr)
				}

				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Expected error containing %q, got %q", tc.wantErr, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if owner != tc.wantOwner {
				t.Errorf("Expected owner %q, got %q", tc.wantOwner, owner)
			}

			if email != tc.wantEmail {
				t.Errorf("Expected email %q, got %q", tc.wantEmail, email)
			}
		})
	}
}

func TestSaveKubeconfig(t *testing.T) {
	validKubeconfig := []byte("apiVersion: v1\nkind: Config\nclusters: []\ncontexts: []\nusers: []\n")

	t.Run("with explicit dest", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "subdir", "kubeconfig")

		got, err := saveKubeconfig(validKubeconfig, dest, "myenv", "suffix")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if got != dest {
			t.Errorf("Expected dest %q, got %q", dest, got)
		}

		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("Expected file to exist: %v", err)
		}

		if string(data) != string(validKubeconfig) {
			t.Errorf("Expected content %q, got %q", validKubeconfig, data)
		}
	})

	t.Run("with default dest", func(t *testing.T) {
		got, err := saveKubeconfig(validKubeconfig, "", "myenv", "test-kubeconfig")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !strings.HasSuffix(got, "Downloads/myenv-test-kubeconfig") {
			t.Errorf("Expected path ending with Downloads/myenv-test-kubeconfig, got %s", got)
		}

		_ = os.Remove(got)
	})

	t.Run("invalid kubeconfig rejected", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "bad-kubeconfig")

		_, err := saveKubeconfig([]byte("<html>error</html>"), dest, "myenv", "suffix")
		if err == nil {
			t.Fatal("Expected error for invalid kubeconfig, got nil")
		}

		if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
			t.Error("Invalid kubeconfig should not have been written to disk")
		}
	})
}

func TestConfigDir(t *testing.T) {
	dir := configDir()
	if dir == "" {
		t.Fatal("Expected non-empty config dir")
	}

	if !strings.HasSuffix(dir, ".config/succulent-cli") {
		t.Errorf("Expected path ending with .config/succulent-cli, got %s", dir)
	}
}

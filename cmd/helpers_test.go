package cmd

import (
	"net/url"
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

	if !strings.HasSuffix(path, filepath.Join("Downloads", "succulent", "myenv", "kubeconfig")) {
		t.Errorf("Expected path ending with Downloads/succulent/myenv/kubeconfig, got %s", path)
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

		if !strings.HasSuffix(got, filepath.Join("Downloads", "succulent", "myenv", "test-kubeconfig")) {
			t.Errorf("Expected path ending with Downloads/succulent/myenv/test-kubeconfig, got %s", got)
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

func TestPrintDryRun(t *testing.T) {
	t.Run("with form data", func(t *testing.T) {
		data := url.Values{
			"email": {"user@example.com"},
			"tag":   {"4.17"},
		}
		printDryRun("reprovision", "myenv", data)
	})

	t.Run("without form data", func(t *testing.T) {
		printDryRun("delete", "myenv", nil)
	})
}

func TestValidateOCPTag(t *testing.T) {
	tests := []struct {
		tag     string
		wantErr bool
	}{
		{"", false},
		{"4.17", false},
		{"5.0", false},
		{"4.17.0", false},
		{"4.17.0-0.nightly-2026-05-20-123456", false},
		{"4.15.0-rc.1", false},
		{"latest", true},
		{"nightly", true},
		{"4.17.0.0", true},
		{"abc", true},
		{"4", true},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			err := validateOCPTag(tt.tag, "--ocp-tag")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOCPTag(%q) error = %v, wantErr %v", tt.tag, err, tt.wantErr)
			}
		})
	}
}

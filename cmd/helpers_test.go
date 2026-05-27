package cmd

import (
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

func TestConfigDir(t *testing.T) {
	dir := configDir()
	if dir == "" {
		t.Fatal("Expected non-empty config dir")
	}

	if !strings.HasSuffix(dir, ".config/succulent-cli") {
		t.Errorf("Expected path ending with .config/succulent-cli, got %s", dir)
	}
}

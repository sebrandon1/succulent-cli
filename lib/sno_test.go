package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvisionSNO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/sno/cnfdc7" {
			t.Errorf("Expected path /sno/cnfdc7, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("owner") != "testuser" {
			t.Errorf("Expected owner testuser, got %s", r.FormValue("owner"))
		}

		if r.FormValue("mailto") != "test@redhat.com" {
			t.Errorf("Expected mailto test@redhat.com, got %s", r.FormValue("mailto"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &SNOProvisionRequest{
		Owner:       "testuser",
		Email:       "test@redhat.com",
		OCPTag:      "4.17",
		ReleaseType: "nightly",
	}

	if err := client.ProvisionSNO(testEnv, req); err != nil {
		t.Fatalf("ProvisionSNO failed: %v", err)
	}
}

func TestGetSNOKubeconfig(t *testing.T) {
	mockKubeconfig := "apiVersion: v1\nkind: Config\nclusters:\n- cluster:\n    server: https://api.sno-cnfdc7.example.com:6443\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sno_kubeconfig/cnfdc7" {
			t.Errorf("Expected path /sno_kubeconfig/cnfdc7, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockKubeconfig))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	data, err := client.GetSNOKubeconfig(testEnv)
	if err != nil {
		t.Fatalf("GetSNOKubeconfig failed: %v", err)
	}

	if string(data) != mockKubeconfig {
		t.Errorf("Expected kubeconfig %q, got %q", mockKubeconfig, string(data))
	}
}

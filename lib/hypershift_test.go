package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvisionHypershift(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/create_hypershift" {
			t.Errorf("Expected path /create_hypershift, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("plan") != testEnv {
			t.Errorf("Expected plan %s, got %s", testEnv, r.FormValue("plan"))
		}

		if r.FormValue("owner") != "testuser" {
			t.Errorf("Expected owner testuser, got %s", r.FormValue("owner"))
		}

		if r.FormValue("sno_tag") != "4.17" {
			t.Errorf("Expected sno_tag 4.17, got %s", r.FormValue("sno_tag"))
		}

		if r.FormValue("hcp_tag") != "4.17" {
			t.Errorf("Expected hcp_tag 4.17, got %s", r.FormValue("hcp_tag"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &HypershiftRequest{
		Owner:      "testuser",
		Email:      "test@example.com",
		SNOTag:     "4.17",
		SNORelease: "nightly",
		HCPTag:     "4.17",
		HCPRelease: "nightly",
	}

	if err := client.ProvisionHypershift(testEnv, req); err != nil {
		t.Fatalf("ProvisionHypershift failed: %v", err)
	}
}

func TestProvisionHypershiftError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &HypershiftRequest{Owner: "testuser", Email: "test@example.com"}

	if err := client.ProvisionHypershift(testEnv, req); err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

func TestGetHypershiftKubeconfig(t *testing.T) {
	mockKubeconfig := "apiVersion: v1\nkind: Config\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/hypershift_kubeconfig" {
			t.Errorf("Expected path /hypershift_kubeconfig, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("plan_name") != testEnv {
			t.Errorf("Expected plan_name %s, got %s", testEnv, r.FormValue("plan_name"))
		}

		if r.FormValue("choice") != "management" {
			t.Errorf("Expected choice management, got %s", r.FormValue("choice"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockKubeconfig))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	data, err := client.GetHypershiftKubeconfig(testEnv, "management")
	if err != nil {
		t.Fatalf("GetHypershiftKubeconfig failed: %v", err)
	}

	if string(data) != mockKubeconfig {
		t.Errorf("Expected kubeconfig %q, got %q", mockKubeconfig, string(data))
	}
}

func TestGetHypershiftKubeconfigError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	_, err := client.GetHypershiftKubeconfig(testEnv, "management")
	if err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}
}

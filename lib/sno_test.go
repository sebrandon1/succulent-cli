package lib

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvisionSNO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/sno/testenv1" {
			t.Errorf("Expected path /sno/testenv1, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("owner") != "testuser" {
			t.Errorf("Expected owner testuser, got %s", r.FormValue("owner"))
		}

		if r.FormValue("mailto") != "test@example.com" {
			t.Errorf("Expected mailto test@example.com, got %s", r.FormValue("mailto"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	req := &SNOProvisionRequest{
		Owner:       "testuser",
		Email:       "test@example.com",
		OCPTag:      "4.17",
		ReleaseType: "nightly",
	}

	if err := client.ProvisionSNO(context.Background(), testEnv, req); err != nil {
		t.Fatalf("ProvisionSNO failed: %v", err)
	}
}

func TestGetSNOKubeconfig(t *testing.T) {
	mockKubeconfig := "apiVersion: v1\nkind: Config\nclusters:\n- cluster:\n    server: https://api.sno-testenv1.example.com:6443\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sno_kubeconfig/testenv1" {
			t.Errorf("Expected path /sno_kubeconfig/testenv1, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockKubeconfig))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	data, err := client.GetSNOKubeconfig(context.Background(), testEnv)
	if err != nil {
		t.Fatalf("GetSNOKubeconfig failed: %v", err)
	}

	if string(data) != mockKubeconfig {
		t.Errorf("Expected kubeconfig %q, got %q", mockKubeconfig, string(data))
	}
}

func TestProvisionSNOError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	req := &SNOProvisionRequest{
		Owner: "testuser",
		Email: "test@example.com",
	}

	if err := client.ProvisionSNO(context.Background(), testEnv, req); err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

func TestGetSNOKubeconfigError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.GetSNOKubeconfig(context.Background(), testEnv)
	if err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}
}

func TestProvisionSNOWithFullTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("full_ocp_tag") != "4.14.0-0.nightly-2023-12-14-072431" {
			t.Errorf("Expected full_ocp_tag, got %s", r.FormValue("full_ocp_tag"))
		}

		if r.FormValue("ocp_tag") != "" {
			t.Errorf("Expected empty ocp_tag, got %s", r.FormValue("ocp_tag"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	req := &SNOProvisionRequest{
		Owner:      "testuser",
		Email:      "test@example.com",
		FullOCPTag: "4.14.0-0.nightly-2023-12-14-072431",
	}

	if err := client.ProvisionSNO(context.Background(), testEnv, req); err != nil {
		t.Fatalf("ProvisionSNO failed: %v", err)
	}
}

func TestGetSNOKubeconfigBodyTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, oversizedBody())
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.GetSNOKubeconfig(context.Background(), testEnv)
	assertTruncated(t, err)
}

package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProvisionZTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/create_ztp" {
			t.Errorf("Expected path /create_ztp, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("plan") != testEnv {
			t.Errorf("Expected plan %s, got %s", testEnv, r.FormValue("plan"))
		}

		if r.FormValue("ztp_type") != "sno" {
			t.Errorf("Expected ztp_type sno, got %s", r.FormValue("ztp_type"))
		}

		if r.FormValue("sno_tag") != "4.17" {
			t.Errorf("Expected sno_tag 4.17, got %s", r.FormValue("sno_tag"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &ZTPRequest{
		Owner:      "testuser",
		Email:      "test@example.com",
		SNOTag:     "4.17",
		SNORelease: "nightly",
		ZTPTag:     "4.17",
		ZTPRelease: "nightly",
		ZTPType:    "sno",
	}

	if err := client.ProvisionZTP(testEnv, req); err != nil {
		t.Fatalf("ProvisionZTP failed: %v", err)
	}
}

func TestProvisionZTPMNO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("ztp_type") != "mno" {
			t.Errorf("Expected ztp_type mno, got %s", r.FormValue("ztp_type"))
		}

		if r.FormValue("vm-masters-count") != "3" {
			t.Errorf("Expected vm-masters-count 3, got %s", r.FormValue("vm-masters-count"))
		}

		if r.FormValue("vm-workers-count") != "1" {
			t.Errorf("Expected vm-workers-count 1, got %s", r.FormValue("vm-workers-count"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &ZTPRequest{
		Owner:          "testuser",
		Email:          "test@example.com",
		SNOTag:         "4.17",
		SNORelease:     "nightly",
		ZTPTag:         "4.17",
		ZTPRelease:     "nightly",
		ZTPType:        "mno",
		VMMastersCount: "3",
		VMWorkersCount: "1",
	}

	if err := client.ProvisionZTP(testEnv, req); err != nil {
		t.Fatalf("ProvisionZTP failed: %v", err)
	}
}

func TestProvisionZTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &ZTPRequest{Owner: "testuser", Email: "test@example.com"}

	if err := client.ProvisionZTP(testEnv, req); err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

func TestGetZTPKubeconfig(t *testing.T) {
	mockKubeconfig := "apiVersion: v1\nkind: Config\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ztp_kubeconfig" {
			t.Errorf("Expected path /ztp_kubeconfig, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("choice") != "spoke" {
			t.Errorf("Expected choice spoke, got %s", r.FormValue("choice"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockKubeconfig))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	data, err := client.GetZTPKubeconfig(testEnv, "spoke")
	if err != nil {
		t.Fatalf("GetZTPKubeconfig failed: %v", err)
	}

	if string(data) != mockKubeconfig {
		t.Errorf("Expected kubeconfig %q, got %q", mockKubeconfig, string(data))
	}
}

func TestGetZTPKubeconfigError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	_, err := client.GetZTPKubeconfig(testEnv, "management")
	if err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}
}

func TestProvisionZTPAllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("stop_before_deployment") != "on" {
			t.Errorf("Expected stop_before_deployment on, got %s", r.FormValue("stop_before_deployment"))
		}

		if r.FormValue("bm-masters-hosts") != "host1,host2,host3" {
			t.Errorf("Expected bm-masters-hosts, got %s", r.FormValue("bm-masters-hosts"))
		}

		if r.FormValue("bm-workers-hosts") != "worker1" {
			t.Errorf("Expected bm-workers-hosts worker1, got %s", r.FormValue("bm-workers-hosts"))
		}

		if r.FormValue("sno_full_tag") != "4.17.0-0.nightly-2026-01-01-000000" {
			t.Errorf("Expected sno_full_tag, got %s", r.FormValue("sno_full_tag"))
		}

		if r.FormValue("ztp_full_tag") != "4.17.0-0.nightly-2026-01-01-000000" {
			t.Errorf("Expected ztp_full_tag, got %s", r.FormValue("ztp_full_tag"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &ZTPRequest{
		Owner:                "testuser",
		Email:                "test@example.com",
		SNOTag:               "4.17",
		SNORelease:           "nightly",
		SNOFullTag:           "4.17.0-0.nightly-2026-01-01-000000",
		ZTPTag:               "4.17",
		ZTPRelease:           "nightly",
		ZTPFullTag:           "4.17.0-0.nightly-2026-01-01-000000",
		ZTPType:              "mno",
		StopBeforeDeployment: true,
		VMMastersCount:       "3",
		BMMastersHosts:       "host1,host2,host3",
		BMWorkersHosts:       "worker1",
		VMWorkersCount:       "2",
	}

	if err := client.ProvisionZTP(testEnv, req); err != nil {
		t.Fatalf("ProvisionZTP failed: %v", err)
	}
}

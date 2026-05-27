package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReprovision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/exposeform/testenv1" {
			t.Errorf("Expected path /exposeform/testenv1, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("mailto") != "test@example.com" {
			t.Errorf("Expected mailto test@example.com, got %s", r.FormValue("mailto"))
		}

		if r.FormValue("owner") != "testuser" {
			t.Errorf("Expected owner testuser, got %s", r.FormValue("owner"))
		}

		if r.FormValue("tag") != "4.17" {
			t.Errorf("Expected tag 4.17, got %s", r.FormValue("tag"))
		}

		if r.FormValue("version") != "nightly" {
			t.Errorf("Expected version nightly, got %s", r.FormValue("version"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &ReprovisionRequest{
		Email:   "test@example.com",
		Owner:   "testuser",
		Tag:     "4.17",
		Version: "nightly",
	}

	if err := client.Reprovision(testEnv, req); err != nil {
		t.Fatalf("Reprovision failed: %v", err)
	}
}

func TestReprovisionWithOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("disk_size") != "100" {
			t.Errorf("Expected disk_size 100, got %s", r.FormValue("disk_size"))
		}

		if r.FormValue("virtual_workers") != "false" {
			t.Errorf("Expected virtual_workers false, got %s", r.FormValue("virtual_workers"))
		}

		if r.FormValue("additional_workers") != "cnfdt56,cnfdr98" {
			t.Errorf("Expected additional_workers cnfdt56,cnfdr98, got %s", r.FormValue("additional_workers"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &ReprovisionRequest{
		Email:             "test@example.com",
		Owner:             "testuser",
		Tag:               "4.17",
		Version:           "nightly",
		DiskSize:          "100",
		VirtualWorkers:    "false",
		AdditionalWorkers: "cnfdt56,cnfdr98",
	}

	if err := client.Reprovision(testEnv, req); err != nil {
		t.Fatalf("Reprovision failed: %v", err)
	}
}

func TestReprovisionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &ReprovisionRequest{
		Email:   "test@example.com",
		Owner:   "testuser",
		Tag:     "4.17",
		Version: "nightly",
	}

	if err := client.Reprovision(testEnv, req); err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

func TestReprovisionAllOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("openshift_image") != "quay.io/test/ocp:4.17" {
			t.Errorf("Expected openshift_image, got %s", r.FormValue("openshift_image"))
		}

		if r.FormValue("end_date") != "2026-12-31" {
			t.Errorf("Expected end_date, got %s", r.FormValue("end_date"))
		}

		if r.FormValue("kcli_params") != "disconnected:False" {
			t.Errorf("Expected kcli_params, got %s", r.FormValue("kcli_params"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	req := &ReprovisionRequest{
		Email:             "test@example.com",
		Owner:             "testuser",
		Tag:               "4.17",
		Version:           "nightly",
		OpenshiftImage:    "quay.io/test/ocp:4.17",
		DiskSize:          "100",
		VirtualWorkers:    "true",
		AdditionalWorkers: "worker1,worker2",
		EndDate:           "2026-12-31",
		KcliParams:        "disconnected:False",
	}

	if err := client.Reprovision(testEnv, req); err != nil {
		t.Fatalf("Reprovision failed: %v", err)
	}
}

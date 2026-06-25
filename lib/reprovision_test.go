package lib

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReprovision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/exposecreate" {
			t.Errorf("Expected path /exposecreate, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("plan") != testEnv {
			t.Errorf("Expected plan %s, got %s", testEnv, r.FormValue("plan"))
		}

		if r.FormValue("parameter_mail_to") != "test@example.com" {
			t.Errorf("Expected parameter_mail_to test@example.com, got %s", r.FormValue("parameter_mail_to"))
		}

		if r.FormValue("parameter_owner") != "testuser" {
			t.Errorf("Expected parameter_owner testuser, got %s", r.FormValue("parameter_owner"))
		}

		if r.FormValue("parameter_tag") != "4.17" {
			t.Errorf("Expected parameter_tag 4.17, got %s", r.FormValue("parameter_tag"))
		}

		if r.FormValue("parameter_version") != "nightly" {
			t.Errorf("Expected parameter_version nightly, got %s", r.FormValue("parameter_version"))
		}

		if r.FormValue("parameter_virtual_workers") != "true" {
			t.Errorf("Expected default parameter_virtual_workers true, got %s", r.FormValue("parameter_virtual_workers"))
		}

		if r.FormValue("parameter_additional_workers") != "false" {
			t.Errorf("Expected default parameter_additional_workers false, got %s", r.FormValue("parameter_additional_workers"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	req := &ReprovisionRequest{
		Email:   "test@example.com",
		Owner:   "testuser",
		Tag:     "4.17",
		Version: "nightly",
	}

	if err := client.Reprovision(context.Background(), testEnv, req); err != nil {
		t.Fatalf("Reprovision failed: %v", err)
	}
}

func TestReprovisionWithOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("parameter_disk_size") != "100" {
			t.Errorf("Expected parameter_disk_size 100, got %s", r.FormValue("parameter_disk_size"))
		}

		if r.FormValue("parameter_virtual_workers") != "false" {
			t.Errorf("Expected parameter_virtual_workers false, got %s", r.FormValue("parameter_virtual_workers"))
		}

		if r.FormValue("parameter_additional_workers") != "cnfdt56,cnfdr98" {
			t.Errorf("Expected parameter_additional_workers cnfdt56,cnfdr98, got %s", r.FormValue("parameter_additional_workers"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	req := &ReprovisionRequest{
		Email:             "test@example.com",
		Owner:             "testuser",
		Tag:               "4.17",
		Version:           "nightly",
		DiskSize:          "100",
		VirtualWorkers:    "false",
		AdditionalWorkers: "cnfdt56,cnfdr98",
	}

	if err := client.Reprovision(context.Background(), testEnv, req); err != nil {
		t.Fatalf("Reprovision failed: %v", err)
	}
}

func TestReprovisionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	req := &ReprovisionRequest{
		Email:   "test@example.com",
		Owner:   "testuser",
		Tag:     "4.17",
		Version: "nightly",
	}

	if err := client.Reprovision(context.Background(), testEnv, req); err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

func TestReprovisionAllOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("parameter_openshift_image") != "quay.io/test/ocp:4.17" {
			t.Errorf("Expected parameter_openshift_image, got %s", r.FormValue("parameter_openshift_image"))
		}

		if r.FormValue("additional_params") != "disconnected:False" {
			t.Errorf("Expected additional_params, got %s", r.FormValue("additional_params"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	req := &ReprovisionRequest{
		Email:             "test@example.com",
		Owner:             "testuser",
		Tag:               "4.17",
		Version:           "nightly",
		OpenshiftImage:    "quay.io/test/ocp:4.17",
		DiskSize:          "100",
		VirtualWorkers:    "true",
		AdditionalWorkers: "worker1,worker2",
		KcliParams:        "disconnected:False",
	}

	if err := client.Reprovision(context.Background(), testEnv, req); err != nil {
		t.Fatalf("Reprovision failed: %v", err)
	}
}

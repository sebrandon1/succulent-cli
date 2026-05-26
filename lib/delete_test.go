package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/exposedelete" {
			t.Errorf("Expected path /exposedelete, got %s", r.URL.Path)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("plan") != "testenv1" {
			t.Errorf("Expected plan testenv1, got %s", r.FormValue("plan"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	if err := client.DeleteEnvironment(testEnv); err != nil {
		t.Fatalf("DeleteEnvironment failed: %v", err)
	}
}

func TestDeleteEnvironmentError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	if err := client.DeleteEnvironment(testEnv); err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

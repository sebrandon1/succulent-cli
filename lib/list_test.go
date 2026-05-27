package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const mockMainPageHTML = `<html><body>
<table class="table table-bordered">
<tr class="table-primary"><th>Plan name</th><th>Actions</th></tr>
<tr style="background-color: #e0f7fa;"><th colspan="3">Hosts CNFD-B</th></tr>
<tr>
  <th>env1 (<b>B-0</b>)</th>
  <th>
    <button type="button" onclick="location.href='/infoplan/env1'">Info</button>
    <button type="button" onclick="location.href='/exposeform/env1'">Reprovision</button>
    <button type="button" onclick="location.href='/sno/env1'">SNO</button>
  </th>
</tr>
<tr>
  <th>env2 (<b>B-1</b>)</th>
  <th>
    <button type="button" onclick="location.href='/infoplan/env2'">Info</button>
    <button type="button" onclick="location.href='/exposeform/env2'">Reprovision</button>
  </th>
</tr>
<tr style="background-color: #e0f7fa;"><th colspan="3">Hosts CNFD-C</th></tr>
<tr>
  <th>env3 (<b>C-0</b>)</th>
  <th>
    <button type="button" onclick="location.href='/infoplan/env3'">Info</button>
  </th>
</tr>
</table></body></html>`

func TestListEnvironments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("Expected path /, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockMainPageHTML))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	envs, err := client.ListEnvironments()
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	if len(envs) != 3 {
		t.Fatalf("Expected 3 environments, got %d", len(envs))
	}

	if envs[0].Name != "env1" {
		t.Errorf("Expected first env name env1, got %s", envs[0].Name)
	}

	if envs[0].Group != "CNFD-B" {
		t.Errorf("Expected first env group CNFD-B, got %s", envs[0].Group)
	}

	if envs[2].Name != "env3" {
		t.Errorf("Expected third env name env3, got %s", envs[2].Name)
	}

	if envs[2].Group != "CNFD-C" {
		t.Errorf("Expected third env group CNFD-C, got %s", envs[2].Group)
	}
}

func TestListEnvironmentsNoDuplicates(t *testing.T) {
	html := `<html><body><table>
<tr><th>env1</th><th>
  <button onclick="location.href='/infoplan/env1'">Info</button>
  <button onclick="location.href='/infoplan/env1'">Info2</button>
</th></tr>
</table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	envs, err := client.ListEnvironments()
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	if len(envs) != 1 {
		t.Errorf("Expected 1 environment (no duplicates), got %d", len(envs))
	}
}

func TestListEnvironmentsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><p>No environments</p></body></html>"))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	envs, err := client.ListEnvironments()
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	if len(envs) != 0 {
		t.Errorf("Expected 0 environments, got %d", len(envs))
	}
}

func TestListEnvironmentsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	_, err := client.ListEnvironments()
	if err == nil {
		t.Fatal("Expected error for 500 response, got nil")
	}
}

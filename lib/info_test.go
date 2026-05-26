package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const mockInfoHTML = `<html><body>
<table class="table table-bordered">
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>testenv1</td><td>client1</td><td>05-26-2026 13:00 testuser</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>testenv1-installer</td><td>up</td><td>192.168.1.100</td></tr>
<tr><td>testenv1-master-0</td><td>up</td><td>192.168.1.101</td></tr>
<tr><td>testenv1-master-1</td><td>up</td><td>192.168.1.102</td></tr>
<tr><td>testenv1-worker-0</td><td>down</td><td></td></tr>
</table></body></html>`

func TestGetInfoPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/infoplan/testenv1" {
			t.Errorf("Expected path /infoplan/testenv1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockInfoHTML))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	info, err := client.GetInfoPlan(testEnv)
	if err != nil {
		t.Fatalf("GetInfoPlan failed: %v", err)
	}

	if info.PlanName != "testenv1" {
		t.Errorf("Expected plan name testenv1, got %s", info.PlanName)
	}

	if info.Client != "client1" {
		t.Errorf("Expected client client1, got %s", info.Client)
	}

	if info.InstallerIP != testInstallerIP {
		t.Errorf("Expected installer IP %s, got %s", testInstallerIP, info.InstallerIP)
	}

	if len(info.Nodes) != 4 {
		t.Fatalf("Expected 4 nodes, got %d", len(info.Nodes))
	}

	if info.Nodes[0].Name != "testenv1-installer" {
		t.Errorf("Expected first node testenv1-installer, got %s", info.Nodes[0].Name)
	}

	if info.Nodes[3].Status != "down" {
		t.Errorf("Expected worker-0 status down, got %s", info.Nodes[3].Status)
	}

	if info.Nodes[3].IP != "" {
		t.Errorf("Expected worker-0 IP to be empty, got %s", info.Nodes[3].IP)
	}
}

func TestGetInfoPlanNoInstaller(t *testing.T) {
	html := `<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>testenv1</td><td>client1</td><td>05-26-2026</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>testenv1-master-0</td><td>up</td><td>192.168.1.101</td></tr>
</table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := NewClient(server.URL, true)

	info, err := client.GetInfoPlan(testEnv)
	if err != nil {
		t.Fatalf("GetInfoPlan failed: %v", err)
	}

	if info.InstallerIP != "" {
		t.Errorf("Expected empty installer IP, got %s", info.InstallerIP)
	}
}

package lib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const mockInfoHTML = `<html><body>
<table class="table table-bordered">
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>cnfdc7</td><td>cnfdc1</td><td>05-26-2026 13:00 bpalm</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>cnfdc7-installer</td><td>up</td><td>10.6.105.126</td></tr>
<tr><td>cnfdc7-master-0</td><td>up</td><td>10.6.105.127</td></tr>
<tr><td>cnfdc7-master-1</td><td>up</td><td>10.6.105.128</td></tr>
<tr><td>cnfdc7-worker-0</td><td>down</td><td></td></tr>
</table></body></html>`

func TestGetInfoPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/infoplan/cnfdc7" {
			t.Errorf("Expected path /infoplan/cnfdc7, got %s", r.URL.Path)
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

	if info.PlanName != "cnfdc7" {
		t.Errorf("Expected plan name cnfdc7, got %s", info.PlanName)
	}

	if info.Client != "cnfdc1" {
		t.Errorf("Expected client cnfdc1, got %s", info.Client)
	}

	if info.InstallerIP != testInstallerIP {
		t.Errorf("Expected installer IP %s, got %s", testInstallerIP, info.InstallerIP)
	}

	if len(info.Nodes) != 4 {
		t.Fatalf("Expected 4 nodes, got %d", len(info.Nodes))
	}

	if info.Nodes[0].Name != "cnfdc7-installer" {
		t.Errorf("Expected first node cnfdc7-installer, got %s", info.Nodes[0].Name)
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
<tr><td>cnfdc7</td><td>cnfdc1</td><td>05-26-2026</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>cnfdc7-master-0</td><td>up</td><td>10.6.105.127</td></tr>
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

package lib

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid IPv4", "192.168.1.100", false},
		{"valid IPv6", "::1", false},
		{"valid IPv6 full", "2001:db8::1", false},
		{"empty string", "", true},
		{"invalid octets", "999.999.999.999", true},
		{"shell metacharacters", "192.168.1.1; rm -rf /", true},
		{"command injection", "$(whoami)", true},
		{"hostname not IP", "example.com", true},
		{"partial IP", "192.168", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIP(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIP(%q) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

func TestRemoveSSHHostKeyInvalidIP(t *testing.T) {
	err := RemoveSSHHostKey("not-an-ip")
	if err == nil {
		t.Fatal("Expected error for invalid IP")
	}

	if !strings.Contains(err.Error(), "invalid IP address") {
		t.Errorf("Expected 'invalid IP address' error, got: %v", err)
	}
}

func TestFetchKubeconfigInvalidIP(t *testing.T) {
	err := FetchKubeconfig("not-an-ip", "root", "/path", "/dest")
	if err == nil {
		t.Fatal("Expected error for invalid IP")
	}

	if !strings.Contains(err.Error(), "invalid IP address") {
		t.Errorf("Expected 'invalid IP address' error, got: %v", err)
	}
}

func TestWaitForClusterReadyAllUp(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>client1</td><td>2026-05-26</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>up</td><td>%s</td></tr>
<tr><td>%s-master-0</td><td>up</td><td>192.168.1.101</td></tr>
</table></body></html>`, testEnv, testEnv, testInstallerIP, testEnv)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	ip, err := client.WaitForClusterReady(testEnv, 1, 1, io.Discard)
	if err != nil {
		t.Fatalf("WaitForClusterReady failed: %v", err)
	}

	if ip != testInstallerIP {
		t.Errorf("Expected installer IP %s, got %s", testInstallerIP, ip)
	}
}

func TestWaitForClusterReadyEventuallyUp(t *testing.T) {
	attempt := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempt++

		var status string
		if attempt >= 2 {
			status = "up"
		} else {
			status = "down"
		}

		html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>client1</td><td>2026-05-26</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>%s</td><td>%s</td></tr>
</table></body></html>`, testEnv, testEnv, status, testInstallerIP)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	ip, err := client.WaitForClusterReady(testEnv, 1, 1, io.Discard)
	if err != nil {
		t.Fatalf("WaitForClusterReady failed: %v", err)
	}

	if ip != testInstallerIP {
		t.Errorf("Expected installer IP %s, got %s", testInstallerIP, ip)
	}

	if attempt < 2 {
		t.Errorf("Expected at least 2 attempts, got %d", attempt)
	}
}

func TestWaitForClusterReadyNoInstallerIP(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>client1</td><td>2026-05-26</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-master-0</td><td>up</td><td>192.168.1.101</td></tr>
</table></body></html>`, testEnv, testEnv)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	// Short timeout so test doesn't hang — no installer IP means it times out
	_, err := client.WaitForClusterReady(testEnv, 0, 1, io.Discard)
	if err == nil {
		t.Fatal("Expected error for timeout, got nil")
	}
}

func TestWaitForClusterReadyServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.WaitForClusterReady(testEnv, 0, 1, io.Discard)
	if err == nil {
		t.Fatal("Expected error for timeout after server errors, got nil")
	}
}

func TestWaitForClusterReadyPartialUp(t *testing.T) {
	attempt := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempt++

		var installerStatus, workerStatus string
		if attempt >= 2 {
			installerStatus = "up"
			workerStatus = "up"
		} else {
			installerStatus = "up"
			workerStatus = "down"
		}

		html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>client1</td><td>2026-05-27</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>%s</td><td>%s</td></tr>
<tr><td>%s-worker-0</td><td>%s</td><td>192.168.1.101</td></tr>
</table></body></html>`, testEnv, testEnv, installerStatus, testInstallerIP, testEnv, workerStatus)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	ip, err := client.WaitForClusterReady(testEnv, 1, 1, io.Discard)
	if err != nil {
		t.Fatalf("WaitForClusterReady failed: %v", err)
	}

	if ip != testInstallerIP {
		t.Errorf("Expected installer IP %s, got %s", testInstallerIP, ip)
	}

	if attempt < 2 {
		t.Errorf("Expected at least 2 attempts for partial-up, got %d", attempt)
	}
}

package lib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateKubeconfig(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid kubeconfig",
			data: []byte("apiVersion: v1\nkind: Config\nclusters: []\ncontexts: []\nusers: []\n"),
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name:    "HTML error page",
			data:    []byte("<html><body>404 Not Found</body></html>"),
			wantErr: true,
			errMsg:  "not valid YAML",
		},
		{
			name:    "valid YAML but wrong kind",
			data:    []byte("apiVersion: v1\nkind: Secret\nclusters: []\ncontexts: []\nusers: []\n"),
			wantErr: true,
			errMsg:  "unexpected kind",
		},
		{
			name:    "missing kind",
			data:    []byte("apiVersion: v1\nclusters: []\ncontexts: []\nusers: []\n"),
			wantErr: true,
			errMsg:  "unexpected kind",
		},
		{
			name:    "missing clusters key",
			data:    []byte("apiVersion: v1\nkind: Config\ncontexts: []\nusers: []\n"),
			wantErr: true,
			errMsg:  "missing required key: clusters",
		},
		{
			name:    "missing users key",
			data:    []byte("apiVersion: v1\nkind: Config\nclusters: []\ncontexts: []\n"),
			wantErr: true,
			errMsg:  "missing required key: users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKubeconfig(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKubeconfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid IPv4", "192.168.1.100", false},
		{"valid private 10.x", "10.0.0.1", false},
		{"valid IPv6", "2001:db8::1", false},
		{"empty string", "", true},
		{"invalid octets", "999.999.999.999", true},
		{"shell metacharacters", "192.168.1.1; rm -rf /", true},
		{"command injection", "$(whoami)", true},
		{"hostname not IP", "example.com", true},
		{"partial IP", "192.168", true},
		{"loopback IPv4", "127.0.0.1", true},
		{"loopback IPv6", "::1", true},
		{"unspecified IPv4", "0.0.0.0", true},
		{"unspecified IPv6", "::", true},
		{"link-local IPv4", "169.254.1.1", true},
		{"link-local IPv6", "fe80::1", true},
		{"multicast IPv4", "224.0.0.1", true},
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
	for _, ip := range []string{"not-an-ip", "127.0.0.1", "0.0.0.0"} {
		t.Run(ip, func(t *testing.T) {
			err := RemoveSSHHostKey(ip)
			if err == nil {
				t.Fatal("Expected error for invalid IP")
			}

			if !strings.Contains(err.Error(), "invalid IP address") {
				t.Errorf("Expected 'invalid IP address' error, got: %v", err)
			}
		})
	}
}

func TestFetchKubeconfigInvalidIP(t *testing.T) {
	ips := []string{"not-an-ip", "127.0.0.1", "169.254.1.1", "0.0.0.0"}

	for _, ip := range ips {
		for _, password := range []string{"", "secret"} {
			name := ip + "/no password"
			if password != "" {
				name = ip + "/with password"
			}

			t.Run(name, func(t *testing.T) {
				err := FetchKubeconfig(ip, "root", password, "/path", "/dest")
				if err == nil {
					t.Fatal("Expected error for invalid IP")
				}

				if !strings.Contains(err.Error(), "invalid IP address") {
					t.Errorf("Expected 'invalid IP address' error, got: %v", err)
				}
			})
		}
	}
}

func assertArgPresent(t *testing.T, args []string, want string) {
	t.Helper()

	for _, arg := range args {
		if arg == want {
			return
		}
	}

	t.Errorf("Expected %q in args: %v", want, args)
}

func assertEnvContains(t *testing.T, env []string, want string) {
	t.Helper()

	for _, e := range env {
		if e == want {
			return
		}
	}

	t.Errorf("Expected %q in env", want)
}

func TestBuildSCPCommand(t *testing.T) {
	tests := []struct {
		name        string
		ip          string
		user        string
		password    string
		remotePath  string
		destPath    string
		wantBinary  string
		wantSSHPASS bool
	}{
		{
			name:       "without password uses scp directly",
			ip:         "192.168.1.100",
			user:       "root",
			remotePath: "/root/ocp/auth/kubeconfig",
			destPath:   "/tmp/kubeconfig",
			wantBinary: "scp",
		},
		{
			name:        "with password uses sshpass",
			ip:          "192.168.1.100",
			user:        "root",
			password:    "secret123",
			remotePath:  "/root/ocp/auth/kubeconfig",
			destPath:    "/tmp/kubeconfig",
			wantBinary:  "sshpass",
			wantSSHPASS: true,
		},
		{
			name:       "custom user without password",
			ip:         "10.0.0.1",
			user:       "kni",
			remotePath: "/home/kni/kubeconfig",
			destPath:   "/tmp/kc",
			wantBinary: "scp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := buildSCPCommand(tt.ip, tt.user, tt.password, tt.remotePath, tt.destPath)

			if cmd.Args[0] != tt.wantBinary {
				t.Errorf("Expected binary %q, got %q", tt.wantBinary, cmd.Args[0])
			}

			remote := fmt.Sprintf("%s@%s:%s", tt.user, tt.ip, tt.remotePath)
			assertArgPresent(t, cmd.Args, remote)
			assertArgPresent(t, cmd.Args, "StrictHostKeyChecking=no")

			if tt.wantSSHPASS {
				assertEnvContains(t, cmd.Env, "SSHPASS="+tt.password)
				assertArgPresent(t, cmd.Args, "-e")
			}
		})
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

	ip, err := client.WaitForClusterReady(context.Background(), testEnv, 1, 1, io.Discard, false)
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

	ip, err := client.WaitForClusterReady(context.Background(), testEnv, 1, 1, io.Discard, false)
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
	_, err := client.WaitForClusterReady(context.Background(), testEnv, 0, 1, io.Discard, false)
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

	_, err := client.WaitForClusterReady(context.Background(), testEnv, 0, 1, io.Discard, false)
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

	ip, err := client.WaitForClusterReady(context.Background(), testEnv, 1, 1, io.Discard, false)
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

func TestWaitForClusterReadyControlPlaneOnly(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>client1</td><td>2026-06-03</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>up</td><td>%s</td></tr>
<tr><td>%s-master-0</td><td>up</td><td>192.168.1.101</td></tr>
<tr><td>%s-master-1</td><td>up</td><td>192.168.1.102</td></tr>
<tr><td>%s-master-2</td><td>up</td><td>192.168.1.103</td></tr>
<tr><td>%s-worker-0</td><td>down</td><td></td></tr>
</table></body></html>`, testEnv, testEnv, testInstallerIP, testEnv, testEnv, testEnv, testEnv)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	ip, err := client.WaitForClusterReady(context.Background(), testEnv, 1, 1, io.Discard, true)
	if err != nil {
		t.Fatalf("Expected ready with control-plane-only, got error: %v", err)
	}

	if ip != testInstallerIP {
		t.Errorf("Expected installer IP %s, got %s", testInstallerIP, ip)
	}
}

func TestWaitForClusterReadyErrorState(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>client1</td><td>2026-06-25</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>up</td><td>%s</td></tr>
<tr><td>%s-master-0</td><td>error</td><td>192.168.1.101</td></tr>
</table></body></html>`, testEnv, testEnv, testInstallerIP, testEnv)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.WaitForClusterReady(context.Background(), testEnv, 1, 1, io.Discard, false)
	if err == nil {
		t.Fatal("Expected error for error state, got nil")
	}

	if !strings.Contains(err.Error(), "permanent error") {
		t.Errorf("Expected 'permanent error' in message, got: %v", err)
	}

	if !strings.Contains(err.Error(), "master-0") {
		t.Errorf("Expected node name in error, got: %v", err)
	}
}

func TestWaitForClusterReadyTimeoutNoNodeStates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.WaitForClusterReady(context.Background(), testEnv, 0, 1, io.Discard, false)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "no node data received") {
		t.Errorf("Expected 'no node data received' when loop never ran, got: %v", err)
	}
}

func TestHasErrorState(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []NodeInfo
		wantError bool
	}{
		{"all up", []NodeInfo{{Name: "n1", Status: "up"}}, false},
		{"all down", []NodeInfo{{Name: "n1", Status: "down"}}, false},
		{"error state", []NodeInfo{{Name: "n1", Status: "error"}}, true},
		{"failed state", []NodeInfo{{Name: "n1", Status: "failed"}}, true},
		{"unreachable state", []NodeInfo{{Name: "n1", Status: "unreachable"}}, true},
		{"mixed with error", []NodeInfo{{Name: "n1", Status: "up"}, {Name: "n2", Status: "error"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := hasErrorState(tt.nodes)
			if got != tt.wantError {
				t.Errorf("hasErrorState() = %v, want %v", got, tt.wantError)
			}
		})
	}
}

func TestHandlePollError(t *testing.T) {
	t.Run("normal returns nil", func(t *testing.T) {
		var buf bytes.Buffer
		err := handlePollError(context.Background(), &buf, fmt.Errorf("some error"), 1*time.Millisecond)
		if err != nil {
			t.Fatalf("Expected nil error, got %v", err)
		}

		if !strings.Contains(buf.String(), "some error") {
			t.Errorf("Expected warning message in output, got: %s", buf.String())
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var buf bytes.Buffer
		err := handlePollError(ctx, &buf, fmt.Errorf("some error"), 10*time.Second)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	})
}

func TestClassifyNodeType(t *testing.T) {
	tests := []struct {
		name     string
		nodeName string
		want     string
	}{
		{"installer", "env-installer", NodeTypeInstaller},
		{"master", "env-master-0", NodeTypeMaster},
		{"bootstrap", "env-bootstrap", NodeTypeBootstrap},
		{"worker", "env-worker-0", NodeTypeWorker},
		{"unknown defaults to worker", "env-something", NodeTypeWorker},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyNodeType(tt.nodeName)
			if got != tt.want {
				t.Errorf("classifyNodeType(%q) = %q, want %q", tt.nodeName, got, tt.want)
			}
		})
	}
}

func TestParseNodeRowUnrecognizedStatus(t *testing.T) {
	node := parseNodeRow("test-node", "provisioning", []string{"test-node", "provisioning", "192.168.1.1"})
	if node != nil {
		t.Errorf("Expected nil for unrecognized status 'provisioning', got %+v", node)
	}
}

func TestParseNodeRowErrorStatus(t *testing.T) {
	for _, status := range []string{"error", "failed", "unreachable"} {
		t.Run(status, func(t *testing.T) {
			node := parseNodeRow("test-node", status, []string{"test-node", status, "192.168.1.1"})
			if node == nil {
				t.Fatalf("Expected node for status %q, got nil", status)
			}

			if node.Status != status {
				t.Errorf("Expected status %q, got %q", status, node.Status)
			}
		})
	}
}

func TestParseNodeRowInvalidIP(t *testing.T) {
	node := parseNodeRow("test-node", "up", []string{"test-node", "up", "999.999.999.999"})
	if node == nil {
		t.Fatal("Expected node for valid status, got nil")
	}

	if node.IP != "" {
		t.Errorf("Expected invalid IP to be ignored, got %q", node.IP)
	}
}

func TestWaitForClusterReadyControlPlaneOnlyMasterDown(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>client1</td><td>2026-06-03</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>up</td><td>%s</td></tr>
<tr><td>%s-master-0</td><td>up</td><td>192.168.1.101</td></tr>
<tr><td>%s-master-1</td><td>down</td><td></td></tr>
<tr><td>%s-worker-0</td><td>down</td><td></td></tr>
</table></body></html>`, testEnv, testEnv, testInstallerIP, testEnv, testEnv, testEnv)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	_, err := client.WaitForClusterReady(context.Background(), testEnv, 0, 1, io.Discard, true)
	if err == nil {
		t.Fatal("Expected timeout when master is still down in control-plane-only mode")
	}
}

func TestWaitForClusterReadyCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>client1</td><td>2026-06-25</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>down</td><td></td></tr>
</table></body></html>`, testEnv, testEnv)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.WaitForClusterReady(ctx, testEnv, 60, 1, io.Discard, false)
	if err == nil {
		t.Fatal("Expected error from canceled context, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestFormatNodeSummary(t *testing.T) {
	tests := []struct {
		name         string
		nodes        []NodeInfo
		wantContains []string
	}{
		{
			name: "mixed up and down",
			nodes: []NodeInfo{
				{Name: "env-installer", Status: "up", IP: "10.0.0.1"},
				{Name: "env-master-0", Status: "up", IP: "10.0.0.2"},
				{Name: "env-worker-0", Status: "down"},
			},
			wantContains: []string{
				"Nodes not ready (1)",
				"env-worker-0 [down]",
				"Nodes up (2)",
				"env-installer (10.0.0.1)",
				"env-master-0 (10.0.0.2)",
			},
		},
		{
			name: "all down",
			nodes: []NodeInfo{
				{Name: "env-installer", Status: "down"},
				{Name: "env-master-0", Status: "down"},
			},
			wantContains: []string{
				"Nodes not ready (2)",
				"env-installer [down]",
				"env-master-0 [down]",
			},
		},
		{
			name: "all up",
			nodes: []NodeInfo{
				{Name: "env-installer", Status: "up", IP: "10.0.0.1"},
			},
			wantContains: []string{
				"Nodes up (1)",
				"env-installer (10.0.0.1)",
			},
		},
		{
			name: "error status shown in brackets",
			nodes: []NodeInfo{
				{Name: "env-installer", Status: "up", IP: "10.0.0.1"},
				{Name: "env-master-0", Status: "error", IP: "10.0.0.2"},
			},
			wantContains: []string{
				"Nodes not ready (1)",
				"env-master-0 (10.0.0.2) [error]",
				"Nodes up (1)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNodeSummary(tt.nodes)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected %q in output:\n%s", want, result)
				}
			}
		})
	}
}

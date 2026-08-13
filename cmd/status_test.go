package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sebrandon1/succulent-cli/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func setupStatusTest(server *httptest.Server) {
	client, err := lib.NewClient(server.URL, true, "")
	if err != nil {
		panic(err)
	}
	sharedClient = client
}

func setupStatusCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "status"}
	cmd.SetContext(context.Background())
	return cmd
}

func TestStatusCommand(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>testclient</td><td>2026-08-13 user@example.com</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>up</td><td>192.168.1.100</td></tr>
<tr><td>%s-master-0</td><td>up</td><td>192.168.1.101</td></tr>
<tr><td>%s-worker-0</td><td>down</td><td></td></tr>
</table></body></html>`, "testenv", "testenv", "testenv", "testenv")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	setupStatusTest(server)

	viper.Set("env", "testenv")
	envName = "testenv"
	outputFormat = "table"

	cmd := setupStatusCmd()
	if err := statusCmd.RunE(cmd, nil); err != nil {
		t.Fatalf("status command failed: %v", err)
	}
}

func TestStatusCommandJSON(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>testclient</td><td>2026-08-13 user@example.com</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>up</td><td>192.168.1.100</td></tr>
</table></body></html>`, "testenv", "testenv")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	setupStatusTest(server)

	viper.Set("env", "testenv")
	envName = "testenv"
	outputFormat = "json"

	cmd := setupStatusCmd()
	if err := statusCmd.RunE(cmd, nil); err != nil {
		t.Fatalf("status command with JSON output failed: %v", err)
	}
}

func TestStatusCommandAllNodesUp(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>testclient</td><td>2026-08-13 user@example.com</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>up</td><td>192.168.1.100</td></tr>
<tr><td>%s-master-0</td><td>up</td><td>192.168.1.101</td></tr>
</table></body></html>`, "testenv", "testenv", "testenv")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	setupStatusTest(server)

	viper.Set("env", "testenv")
	envName = "testenv"
	outputFormat = "table"

	cmd := setupStatusCmd()
	if err := statusCmd.RunE(cmd, nil); err != nil {
		t.Fatalf("status command with all nodes up failed: %v", err)
	}
}

func TestStatusCommandNoNodes(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>testclient</td><td>2026-08-13 user@example.com</td></tr>
</table></body></html>`, "testenv")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	setupStatusTest(server)

	viper.Set("env", "testenv")
	envName = "testenv"
	outputFormat = "table"

	cmd := setupStatusCmd()
	if err := statusCmd.RunE(cmd, nil); err != nil {
		t.Fatalf("status command with no nodes failed: %v", err)
	}
}

func TestStatusCommandAllNodesDown(t *testing.T) {
	html := fmt.Sprintf(`<html><body><table>
<tr><th>Plan name</th><th>Client</th><th>Creation Date</th></tr>
<tr><td>%s</td><td>testclient</td><td>2026-08-13 user@example.com</td></tr>
<tr><th>Vm name</th><th>Status</th><th>Ip</th></tr>
<tr><td>%s-installer</td><td>down</td><td></td></tr>
<tr><td>%s-master-0</td><td>down</td><td></td></tr>
</table></body></html>`, "testenv", "testenv", "testenv")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	}))
	defer server.Close()

	setupStatusTest(server)

	viper.Set("env", "testenv")
	envName = "testenv"
	outputFormat = "table"

	cmd := setupStatusCmd()
	if err := statusCmd.RunE(cmd, nil); err != nil {
		t.Fatalf("status command with all nodes down failed: %v", err)
	}
}

func TestPrintStatus(t *testing.T) {
	info := &lib.ClusterInfo{
		Environment:  "testenv",
		PlanName:     "testplan",
		Client:       "testclient",
		CreationDate: "2026-08-13 user@example.com",
		InstallerIP:  "192.168.1.100",
		Nodes: []lib.NodeInfo{
			{Name: "testenv-installer", Status: "up", IP: "192.168.1.100", NodeType: "installer"},
			{Name: "testenv-master-0", Status: "up", IP: "192.168.1.101", NodeType: "master"},
			{Name: "testenv-worker-0", Status: "down", IP: "", NodeType: "worker"},
		},
	}

	printStatus(info)
}

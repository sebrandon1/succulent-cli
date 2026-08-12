package lib

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

func ValidateKubeconfig(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("kubeconfig data is empty")
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("kubeconfig is not valid YAML: %w", err)
	}

	kind, _ := config["kind"].(string)
	if kind != "Config" {
		return fmt.Errorf("kubeconfig has unexpected kind: %q (expected \"Config\")", kind)
	}

	for _, key := range []string{"apiVersion", "clusters", "contexts", "users"} {
		if _, ok := config[key]; !ok {
			return fmt.Errorf("kubeconfig missing required key: %s", key)
		}
	}

	return nil
}

func validateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}

	return nil
}

func RemoveSSHHostKey(ip string) error {
	if err := validateIP(ip); err != nil {
		return err
	}

	cmd := exec.Command("ssh-keygen", "-R", ip) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen -R %s failed: %w", ip, err)
	}

	return nil
}

func FetchKubeconfig(ip, user, password, remotePath, destPath string) error {
	if err := validateIP(ip); err != nil {
		return err
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	cmd := buildSCPCommand(ip, user, password, remotePath, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp failed: %w", err)
	}

	if err := os.Chmod(destPath, 0o600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		return fmt.Errorf("failed to read downloaded kubeconfig: %w", err)
	}

	if err := ValidateKubeconfig(data); err != nil {
		return fmt.Errorf("downloaded kubeconfig is invalid: %w", err)
	}

	return nil
}

func buildSCPCommand(ip, user, password, remotePath, destPath string) *exec.Cmd {
	remote := fmt.Sprintf("%s@%s:%s", user, ip, remotePath)

	scpArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		remote, destPath,
	}

	if password != "" {
		args := append([]string{"-e", "scp"}, scpArgs...)
		cmd := exec.Command("sshpass", args...) //nolint:gosec
		cmd.Env = append(os.Environ(), "SSHPASS="+password)

		return cmd
	}

	return exec.Command("scp", scpArgs...) //nolint:gosec
}

func hasErrorState(nodes []NodeInfo) (bool, string) {
	for _, node := range nodes {
		if errorStatuses[node.Status] {
			return true, fmt.Sprintf("node %s is in %s state", node.Name, node.Status)
		}
	}

	return false, ""
}

func isClusterReady(nodes []NodeInfo, controlPlaneOnly bool) bool {
	for _, node := range nodes {
		if node.Status != StatusUp {
			if !controlPlaneOnly || node.NodeType == NodeTypeInstaller || node.NodeType == NodeTypeMaster {
				return false
			}
		}
	}

	return true
}

func handlePollError(ctx context.Context, w io.Writer, err error, interval time.Duration) error {
	fmt.Fprintf(w, "  Warning: %v\n", err)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(interval):
		return nil
	}
}

func printControlPlaneNotes(nodes []NodeInfo, w io.Writer) {
	for _, node := range nodes {
		if node.Status != StatusUp {
			fmt.Fprintf(w, "  Note: %s still %s\n", node.Name, node.Status)
		}
	}
}

func printNodeStatuses(nodes []NodeInfo, pollIntervalSeconds int, w io.Writer) {
	for _, node := range nodes {
		fmt.Fprintf(w, "  %s: %s (%s)\n", node.Name, node.Status, node.IP)
	}
	fmt.Fprintf(w, "  Waiting %d seconds...\n", pollIntervalSeconds)
}

func (c *Client) WaitForClusterReady(ctx context.Context, env string, maxWaitMinutes, pollIntervalSeconds int, w io.Writer, controlPlaneOnly bool) (string, error) {
	deadline := time.Now().Add(time.Duration(maxWaitMinutes) * time.Minute)
	interval := time.Duration(pollIntervalSeconds) * time.Second
	attempt := 0

	var lastNodes []NodeInfo

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		attempt++
		fmt.Fprintf(w, "[Attempt %d] Checking node status for %s...\n", attempt, env)

		info, err := c.GetInfoPlan(ctx, env)
		if err != nil {
			if ctxErr := handlePollError(ctx, w, err, interval); ctxErr != nil {
				return "", ctxErr
			}
			continue
		}

		lastNodes = info.Nodes

		if errored, msg := hasErrorState(info.Nodes); errored {
			return "", fmt.Errorf("cluster has permanent error: %s", msg)
		}

		if info.InstallerIP != "" {
			ready := isClusterReady(info.Nodes, controlPlaneOnly)

			if ready {
				fmt.Fprintf(w, "Cluster ready. Installer IP: %s\n", info.InstallerIP)

				printControlPlaneNotes(info.Nodes, w)

				return info.InstallerIP, nil
			}
		}

		printNodeStatuses(info.Nodes, pollIntervalSeconds, w)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}
	}

	var nodeStates []string
	for _, node := range lastNodes {
		nodeStates = append(nodeStates, fmt.Sprintf("%s=%s", node.Name, node.Status))
	}

	if len(nodeStates) > 0 {
		return "", fmt.Errorf("cluster not ready after %d minutes; last state: %s", maxWaitMinutes, strings.Join(nodeStates, ", "))
	}

	return "", fmt.Errorf("cluster not ready after %d minutes", maxWaitMinutes)
}

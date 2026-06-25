package lib

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

func FetchKubeconfig(ip, user, remotePath, destPath string) error {
	if err := validateIP(ip); err != nil {
		return err
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	remote := fmt.Sprintf("%s@%s:%s", user, ip, remotePath)

	cmd := exec.Command("scp", //nolint:gosec
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		remote, destPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp failed: %w", err)
	}

	if err := os.Chmod(destPath, 0o600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
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

func (c *Client) WaitForClusterReady(env string, maxWaitMinutes, pollIntervalSeconds int, w io.Writer, controlPlaneOnly bool) (string, error) {
	deadline := time.Now().Add(time.Duration(maxWaitMinutes) * time.Minute)
	interval := time.Duration(pollIntervalSeconds) * time.Second
	attempt := 0

	var lastNodes []NodeInfo

	for time.Now().Before(deadline) {
		attempt++
		fmt.Fprintf(w, "[Attempt %d] Checking node status for %s...\n", attempt, env)

		info, err := c.GetInfoPlan(env)
		if err != nil {
			fmt.Fprintf(w, "  Warning: %v\n", err)
			time.Sleep(interval)

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

				if controlPlaneOnly {
					for _, node := range info.Nodes {
						if node.Status != StatusUp {
							fmt.Fprintf(w, "  Note: %s still %s\n", node.Name, node.Status)
						}
					}
				}

				return info.InstallerIP, nil
			}
		}

		for _, node := range info.Nodes {
			fmt.Fprintf(w, "  %s: %s (%s)\n", node.Name, node.Status, node.IP)
		}

		fmt.Fprintf(w, "  Waiting %d seconds...\n", pollIntervalSeconds)
		time.Sleep(interval)
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

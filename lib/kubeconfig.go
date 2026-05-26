package lib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func RemoveSSHHostKey(ip string) error {
	cmd := exec.Command("ssh-keygen", "-R", ip) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen -R %s failed: %w", ip, err)
	}

	return nil
}

func FetchKubeconfig(ip, user, remotePath, destPath string) error {
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

func (c *Client) WaitForClusterReady(env string, maxWaitMinutes, pollIntervalSeconds int) (string, error) {
	deadline := time.Now().Add(time.Duration(maxWaitMinutes) * time.Minute)
	interval := time.Duration(pollIntervalSeconds) * time.Second
	attempt := 0

	for time.Now().Before(deadline) {
		attempt++
		fmt.Printf("[Attempt %d] Checking node status for %s...\n", attempt, env)

		info, err := c.GetInfoPlan(env)
		if err != nil {
			fmt.Printf("  Warning: %v\n", err)
			time.Sleep(interval)

			continue
		}

		if info.InstallerIP != "" {
			allUp := true

			for _, node := range info.Nodes {
				if node.Status != "up" {
					allUp = false

					break
				}
			}

			if allUp {
				fmt.Printf("Cluster ready. Installer IP: %s\n", info.InstallerIP)

				return info.InstallerIP, nil
			}
		}

		for _, node := range info.Nodes {
			fmt.Printf("  %s: %s (%s)\n", node.Name, node.Status, node.IP)
		}

		fmt.Printf("  Waiting %d seconds...\n", pollIntervalSeconds)
		time.Sleep(interval)
	}

	return "", fmt.Errorf("cluster not ready after %d minutes", maxWaitMinutes)
}

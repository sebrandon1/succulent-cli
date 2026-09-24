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

	// Succulent's kubeconfig endpoint may omit this optional TypeMeta field.
	if rawKind, ok := config["kind"]; ok {
		kind, isString := rawKind.(string)
		if !isString || kind != "Config" {
			return fmt.Errorf("kubeconfig has unexpected kind: %v (expected \"Config\")", rawKind)
		}
	}

	for _, key := range []string{"clusters", "contexts", "users"} {
		if _, ok := config[key]; !ok {
			return fmt.Errorf("kubeconfig missing required key: %s", key)
		}
	}

	return nil
}

func validateIP(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}

	switch {
	case parsed.IsUnspecified():
		return fmt.Errorf("invalid IP address: %q (unspecified)", ip)
	case parsed.IsLoopback():
		return fmt.Errorf("invalid IP address: %q (loopback)", ip)
	case parsed.IsLinkLocalUnicast(), parsed.IsLinkLocalMulticast():
		return fmt.Errorf("invalid IP address: %q (link-local)", ip)
	case parsed.IsMulticast():
		return fmt.Errorf("invalid IP address: %q (multicast)", ip)
	}

	return nil
}

func RemoveSSHHostKey(ip string) error {
	return RemoveSSHHostKeyContext(context.Background(), ip)
}

// RemoveSSHHostKeyContext removes an IP's SSH host key until ctx is canceled.
func RemoveSSHHostKeyContext(ctx context.Context, ip string) error {
	if err := validateIP(ip); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "ssh-keygen", "-R", ip) // #nosec G204 -- ip is validated by validateIP
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen -R %s failed: %w", ip, err)
	}

	return nil
}

func FetchKubeconfig(ip, user, password, remotePath, destPath string, strictSSH bool) error {
	return FetchKubeconfigContext(context.Background(), ip, user, password, remotePath, destPath, strictSSH)
}

// FetchKubeconfigContext fetches a kubeconfig from an installer node until ctx is canceled.
func FetchKubeconfigContext(ctx context.Context, ip, user, password, remotePath, destPath string, strictSSH bool) error {
	if err := validateIP(ip); err != nil {
		return err
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	cmd := buildSCPCommand(ctx, ip, user, password, remotePath, destPath, strictSSH)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp failed: %w", err)
	}

	if err := os.Chmod(destPath, 0o600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	data, err := os.ReadFile(destPath) // #nosec G304 -- dest path is chosen by the caller after MkdirAll
	if err != nil {
		return fmt.Errorf("failed to read downloaded kubeconfig: %w", err)
	}

	if err := ValidateKubeconfig(data); err != nil {
		return fmt.Errorf("downloaded kubeconfig is invalid: %w", err)
	}

	return nil
}

func buildSCPCommand(ctx context.Context, ip, user, password, remotePath, destPath string, strictSSH bool) *exec.Cmd {
	remote := fmt.Sprintf("%s@%s:%s", user, ip, remotePath)

	scpArgs := append(sshHostKeyArgs(strictSSH), remote, destPath)

	if password != "" {
		args := append([]string{"-e", "scp"}, scpArgs...)
		cmd := exec.CommandContext(ctx, "sshpass", args...) // #nosec G204 -- ip is validated; password is passed via SSHPASS
		cmd.Env = append(os.Environ(), "SSHPASS="+password)

		return cmd
	}

	return exec.CommandContext(ctx, "scp", scpArgs...) // #nosec G204 -- ip is validated by validateIP
}

func sshHostKeyArgs(strictSSH bool) []string {
	if strictSSH {
		return []string{"-o", "StrictHostKeyChecking=yes"}
	}

	return []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}
}

func formatNodeSummary(nodes []NodeInfo) string {
	var up, down []string

	for _, node := range nodes {
		entry := node.Name
		if node.IP != "" {
			entry += " (" + node.IP + ")"
		}

		if node.Status == StatusUp {
			up = append(up, entry)
		} else {
			down = append(down, fmt.Sprintf("%s [%s]", entry, node.Status))
		}
	}

	var parts []string
	if len(down) > 0 {
		parts = append(parts, fmt.Sprintf("  Nodes not ready (%d):\n    - %s", len(down), strings.Join(down, "\n    - ")))
	}

	if len(up) > 0 {
		parts = append(parts, fmt.Sprintf("  Nodes up (%d):\n    - %s", len(up), strings.Join(up, "\n    - ")))
	}

	return strings.Join(parts, "\n")
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

func clusterReadyPollInterval(elapsed, remaining time.Duration, overrideSeconds int) time.Duration {
	if overrideSeconds > 0 {
		return time.Duration(overrideSeconds) * time.Second
	}

	switch {
	case elapsed >= time.Hour || remaining <= 10*time.Minute:
		return 15 * time.Second
	case elapsed >= 30*time.Minute:
		return 30 * time.Second
	default:
		return time.Minute
	}
}

func pollDelay(interval, remaining time.Duration) time.Duration {
	if remaining <= 0 {
		return 0
	}
	if interval > remaining {
		return remaining
	}
	return interval
}

func waitAfterPollError(ctx context.Context, w io.Writer, pollErr error, started, deadline time.Time, overrideSeconds int) (bool, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return false, ctxErr
	}
	if !time.Now().Before(deadline) {
		return false, nil
	}

	interval := clusterReadyPollInterval(time.Since(started), time.Until(deadline), overrideSeconds)
	if err := handlePollError(ctx, w, pollErr, pollDelay(interval, time.Until(deadline))); err != nil {
		return false, err
	}
	return true, nil
}

func reportClusterReady(info *ClusterInfo, w io.Writer, controlPlaneOnly bool) (string, bool) {
	if info.InstallerIP == "" || !isClusterReady(info.Nodes, controlPlaneOnly) {
		return "", false
	}

	fmt.Fprintf(w, "Cluster ready. Installer IP: %s\n", info.InstallerIP)
	printControlPlaneNotes(info.Nodes, w)
	return info.InstallerIP, true
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
	started := time.Now()
	deadline := started.Add(time.Duration(maxWaitMinutes) * time.Minute)
	requestCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
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

		info, err := c.GetInfoPlan(requestCtx, env)
		if err != nil {
			retry, waitErr := waitAfterPollError(ctx, w, err, started, deadline, pollIntervalSeconds)
			if waitErr != nil {
				return "", waitErr
			}
			if retry {
				continue
			}
			break
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		if !time.Now().Before(deadline) {
			break
		}

		lastNodes = info.Nodes

		if errored, msg := hasErrorState(info.Nodes); errored {
			return "", fmt.Errorf("cluster has permanent error: %s", msg)
		}

		if installerIP, ready := reportClusterReady(info, w, controlPlaneOnly); ready {
			return installerIP, nil
		}

		interval := clusterReadyPollInterval(time.Since(started), time.Until(deadline), pollIntervalSeconds)
		wait := pollDelay(interval, time.Until(deadline))
		waitSeconds := int(wait / time.Second)
		if wait%time.Second != 0 {
			waitSeconds++
		}
		printNodeStatuses(info.Nodes, waitSeconds, w)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(wait):
		}
	}

	if len(lastNodes) > 0 {
		return "", fmt.Errorf("cluster not ready after %d minutes:\n%s", maxWaitMinutes, formatNodeSummary(lastNodes))
	}

	return "", fmt.Errorf("cluster not ready after %d minutes (no node data received)", maxWaitMinutes)
}

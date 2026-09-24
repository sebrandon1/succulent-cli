package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (c *Client) GetServerVersion(ctx context.Context) (*ServerVersion, error) {
	resp, err := c.doRequest(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, c.endpointURL(endpointVersion), nil)
	})
	if err != nil {
		return nil, fmt.Errorf("requesting server version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server version endpoint returned HTTP %d", resp.StatusCode)
	}

	data, err := readLimited(resp.Body, MaxResponseSize)
	if err != nil {
		return nil, fmt.Errorf("reading server version: %w", err)
	}

	var version ServerVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, fmt.Errorf("decoding server version: %w", err)
	}
	if version.APIVersion == "" && version.Version == "" {
		return nil, fmt.Errorf("server version response has no version fields")
	}

	return &version, nil
}

// CheckVersionCompatibility compares major and minor versions. Patch versions
// may differ without affecting API compatibility.
func CheckVersionCompatibility(cliVersion string, server ServerVersion) (bool, string) {
	serverVersion := server.APIVersion
	if _, _, ok := parseMajorMinor(serverVersion); !ok {
		serverVersion = server.Version
	}

	cliMajor, cliMinor, cliOK := parseMajorMinor(cliVersion)
	serverMajor, serverMinor, serverOK := parseMajorMinor(serverVersion)
	if !cliOK || !serverOK || (cliMajor == serverMajor && cliMinor == serverMinor) {
		return true, ""
	}

	cliLabel := versionLabel(cliVersion)
	serverLabel := versionLabel(serverVersion)
	if cliMajor > serverMajor || (cliMajor == serverMajor && cliMinor > serverMinor) {
		return false, fmt.Sprintf("version mismatch: CLI v%s is newer than server API v%s; some features may not be available", cliLabel, serverLabel)
	}

	return false, fmt.Sprintf("version mismatch: server API v%s is newer than CLI v%s; some features may not be available", serverLabel, cliLabel)
}

func parseMajorMinor(version string) (int, int, bool) {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}

	return major, minor, true
}

func versionLabel(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

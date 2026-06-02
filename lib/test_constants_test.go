package lib

import "time"

const (
	testEnv         = "testenv1"
	testInstallerIP = "192.168.1.100"
)

func newTestClient(baseURL string) *Client {
	c, _ := NewClient(baseURL, true, "")
	c.RetryBaseDelay = time.Millisecond

	return c
}

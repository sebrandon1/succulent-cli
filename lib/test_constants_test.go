package lib

import (
	"io"
	"strings"
	"testing"
	"time"
)

const (
	testEnv         = "testenv1"
	testInstallerIP = "192.168.1.100"
)

type infiniteByte byte

func (b infiniteByte) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(b)
	}

	return len(p), nil
}

func oversizedBody() io.Reader {
	return io.LimitReader(infiniteByte('x'), MaxResponseSize+1)
}

func assertTruncated(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("Expected error for oversized response, got nil")
	}

	if !strings.Contains(err.Error(), "truncated") {
		t.Errorf("Expected truncated error, got: %v", err)
	}
}

func newTestClient(baseURL string) *Client {
	c, _ := NewClient(baseURL, true, "")
	c.RetryBaseDelay = time.Millisecond

	return c
}

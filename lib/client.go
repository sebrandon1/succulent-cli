package lib

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	DefaultSucculentURL = "https://succulent.eng.redhat.com"

	// Size limits for response body reads to prevent memory exhaustion
	maxKubeconfigSize = 10 * 1024 * 1024 // 10MB for kubeconfig/binary data

	// Error message hints for common troubleshooting
	errHintCheckSettings = "check --url and --verify-ssl settings"
	errHintServerLogs    = "check server logs for details"
	errHintTruncated     = "(truncated at %s; response may be incomplete)"
)

func friendlyHTTPError(statusCode int) string {
	switch statusCode {
	case 400:
		return "bad request: the server rejected the input; check flag values"
	case 401, 403:
		return "access denied: check credentials and permissions"
	case 404:
		return "not found: verify --env name exists (try: succulent-cli list)"
	case 405:
		return "method not allowed: CLI may be out of date (try: make install)"
	case 409:
		return "conflict: environment may already be provisioned or in use"
	case 422:
		return "invalid parameters: check flag values and required fields"
	case 500:
		return "internal server error (" + errHintServerLogs + ")"
	case 502, 503:
		return "server unavailable: try again in a few minutes"
	case 504:
		return "server timeout: operation may still be running; check status with 'get info'"
	default:
		return fmt.Sprintf("HTTP %d (%s)", statusCode, errHintCheckSettings)
	}
}

type Client struct {
	BaseURL        string
	HTTPClient     *http.Client
	MaxRetries     int
	RetryBaseDelay time.Duration
	Timeout        time.Duration
}

func NewClient(baseURL string, insecureSkipVerify bool, caCertPath string) (*Client, error) {
	return NewClientWithTimeout(baseURL, insecureSkipVerify, caCertPath, 60*time.Second)
}

func NewClientWithTimeout(baseURL string, insecureSkipVerify bool, caCertPath string, timeout time.Duration) (*Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec
	}

	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("reading CA certificate: %w", err)
		}

		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}

		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate from %s", caCertPath)
		}

		tlsConfig.RootCAs = pool
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Second,
		Timeout:        timeout,
	}, nil
}

// WithTimeout returns a shallow copy of the client with a different timeout
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	// Create new HTTP client with different timeout
	newHTTPClient := &http.Client{
		Transport: c.HTTPClient.Transport,
		Timeout:   timeout,
	}

	return &Client{
		BaseURL:        c.BaseURL,
		HTTPClient:     newHTTPClient,
		MaxRetries:     c.MaxRetries,
		RetryBaseDelay: c.RetryBaseDelay,
		Timeout:        timeout,
	}
}

func (c *Client) doWithRetry(ctx context.Context, newReq func() (*http.Request, error)) (*http.Response, error) {
	var lastErr error

	for attempt := range c.MaxRetries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		req, err := newReq()
		if err != nil {
			return nil, err
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err

			if attempt < c.MaxRetries-1 {
				c.sleepWithJitter(attempt)
			}

			continue
		}

		if resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("server error: status code %d (%s)", resp.StatusCode, errHintServerLogs)

			if attempt < c.MaxRetries-1 {
				c.sleepWithJitter(attempt)
			}

			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

func (c *Client) sleepWithJitter(attempt int) {
	delay := c.RetryBaseDelay * (1 << attempt)
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(delay/2)))
	time.Sleep(delay + time.Duration(n.Int64()))
}

func (c *Client) getRaw(ctx context.Context, requestURL string) (*http.Response, error) {
	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("%w (check --url and --verify-ssl settings)", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()

		return nil, fmt.Errorf("%s", friendlyHTTPError(resp.StatusCode))
	}

	return resp, nil
}

func (c *Client) postForm(ctx context.Context, endpoint string, data url.Values) error {
	encoded := data.Encode()

	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(encoded))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		return req, nil
	})
	if err != nil {
		return fmt.Errorf("failed to submit form: %w (check --url and --verify-ssl settings)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		return fmt.Errorf("%s", friendlyHTTPError(resp.StatusCode))
	}

	return nil
}

// postFormRaw submits a form and returns the raw response body.
// Used for endpoints that return binary data (e.g., kubeconfig files).
func (c *Client) postFormRaw(ctx context.Context, endpoint string, data url.Values) ([]byte, error) {
	encoded := data.Encode()

	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(encoded))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to submit form: %w (check --url and --verify-ssl settings)", err)
	}
	defer resp.Body.Close()

	body, err := readLimited(resp.Body, maxKubeconfigSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s", friendlyHTTPError(resp.StatusCode))
	}

	return body, nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeded %s limit %s",
			humanizeBytes(limit),
			fmt.Sprintf(errHintTruncated, humanizeBytes(limit)))
	}

	return data, nil
}

func humanizeBytes(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%dMB", b/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%dKB", b/1024)
	default:
		return fmt.Sprintf("%d bytes", b)
	}
}

package lib

import (
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

const DefaultSucculentURL = "https://succulent.eng.redhat.com"

type Client struct {
	BaseURL        string
	HTTPClient     *http.Client
	MaxRetries     int
	RetryBaseDelay time.Duration
}

func NewClient(baseURL string, insecureSkipVerify bool, caCertPath string) (*Client, error) {
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
			Timeout:   60 * time.Second,
			Transport: transport,
		},
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Second,
	}, nil
}

func (c *Client) doWithRetry(newReq func() (*http.Request, error)) (*http.Response, error) {
	var lastErr error

	for attempt := range c.MaxRetries {
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
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))

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

func (c *Client) getRaw(requestURL string) (*http.Response, error) {
	resp, err := c.doWithRetry(func() (*http.Request, error) {
		return http.NewRequest("GET", requestURL, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("%w (check --url and --verify-ssl settings)", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()

		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *Client) postForm(endpoint string, data url.Values) error {
	encoded := data.Encode()

	resp, err := c.doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("POST", endpoint, strings.NewReader(encoded))
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return nil
}

// postFormRaw submits a form and returns the raw response body.
// Used for endpoints that return binary data (e.g., kubeconfig files).
func (c *Client) postFormRaw(endpoint string, data url.Values) ([]byte, error) {
	encoded := data.Encode()

	resp, err := c.doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("POST", endpoint, strings.NewReader(encoded))
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

	// Limit response size to prevent memory exhaustion (10MB for kubeconfig data)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// For errors, limit output to 4KB
		errorBody := body
		if len(errorBody) > 4096 {
			errorBody = errorBody[:4096]
		}

		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(errorBody))
	}

	return body, nil
}

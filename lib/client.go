package lib

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
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

func NewClient(baseURL string, insecureSkipVerify bool) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify, //nolint:gosec
		},
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Second,
	}
}

func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := range c.MaxRetries {
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
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()

		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *Client) postForm(endpoint string, data url.Values) error {
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithRetry(req)
	if err != nil {
		return fmt.Errorf("failed to submit form: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return nil
}

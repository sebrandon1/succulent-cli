package lib

import (
	"fmt"
	"io"
	"net/url"
)

func (c *Client) ProvisionZTP(env string, req *ZTPRequest) error {
	endpoint := c.BaseURL + endpointZTPProvision

	data := req.FormValues()
	data.Set("plan", env)

	if err := c.postForm(endpoint, data); err != nil {
		return fmt.Errorf("failed to provision ZTP on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetZTPKubeconfig(env, choice string) ([]byte, error) {
	endpoint := c.BaseURL + endpointZTPKubeconfig

	data := url.Values{
		"plan_name": {env},
		"choice":    {choice},
	}

	resp, err := c.HTTPClient.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZTP kubeconfig for %s: %w", env, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

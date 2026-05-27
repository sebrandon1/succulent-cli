package lib

import (
	"fmt"
	"io"
	"net/url"
)

func (c *Client) ProvisionHypershift(env string, req *HypershiftRequest) error {
	endpoint := fmt.Sprintf("%s/create_hypershift", c.BaseURL)

	data := url.Values{
		"plan":          {env},
		formFieldOwner:  {req.Owner},
		formFieldMailTo: {req.Email},
	}

	if req.SNOTag != "" {
		data.Set("sno_tag", req.SNOTag)
	}

	if req.SNORelease != "" {
		data.Set("sno_release", req.SNORelease)
	}

	if req.SNOFullTag != "" {
		data.Set("sno_full_tag", req.SNOFullTag)
	}

	if req.HCPTag != "" {
		data.Set("hcp_tag", req.HCPTag)
	}

	if req.HCPRelease != "" {
		data.Set("hcp_release", req.HCPRelease)
	}

	if req.HCPFullTag != "" {
		data.Set("hcp_full_tag", req.HCPFullTag)
	}

	if req.VMWorkersCount != "" {
		data.Set("vm-workers-count", req.VMWorkersCount)
	}

	if req.ImageOverride != "" {
		data.Set("image_override", req.ImageOverride)
	}

	if err := c.postForm(endpoint, data); err != nil {
		return fmt.Errorf("failed to provision Hypershift on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetHypershiftKubeconfig(env, choice string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/hypershift_kubeconfig", c.BaseURL)

	data := url.Values{
		"plan_name": {env},
		"choice":    {choice},
	}

	resp, err := c.HTTPClient.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Hypershift kubeconfig for %s: %w", env, err)
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

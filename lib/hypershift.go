package lib

import (
	"fmt"
	"net/url"
)

func (c *Client) ProvisionHypershift(env string, req *HypershiftRequest) error {
	endpoint := c.BaseURL + endpointHypershiftProvision

	data := req.FormValues()
	data.Set("plan", env)

	if err := c.postForm(endpoint, data); err != nil {
		return fmt.Errorf("failed to provision Hypershift on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetHypershiftKubeconfig(env, choice string) ([]byte, error) {
	endpoint := c.BaseURL + endpointHypershiftKubeconfig

	data := url.Values{
		"plan_name": {env},
		"choice":    {choice},
	}

	body, err := c.postFormRaw(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Hypershift kubeconfig for %s: %w", env, err)
	}

	return body, nil
}

package lib

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ProvisionHypershift(ctx context.Context, env string, req *HypershiftRequest) error {
	data := req.FormValues()
	data.Set("plan", env)

	if err := c.postForm(ctx, c.endpointURL(endpointHypershiftProvision), data); err != nil {
		return fmt.Errorf("failed to provision Hypershift on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetHypershiftKubeconfig(ctx context.Context, env, choice string) ([]byte, error) {
	data := url.Values{
		"plan_name": {env},
		"choice":    {choice},
	}

	body, err := c.postFormRaw(ctx, c.endpointURL(endpointHypershiftKubeconfig), data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Hypershift kubeconfig for %s: %w", env, err)
	}

	return body, nil
}

package lib

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ProvisionZTP(ctx context.Context, env string, req *ZTPRequest) error {
	endpoint := c.BaseURL + endpointZTPProvision

	data := req.FormValues()
	data.Set("plan", env)

	if err := c.postForm(ctx, endpoint, data); err != nil {
		return fmt.Errorf("failed to provision ZTP on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetZTPKubeconfig(ctx context.Context, env, choice string) ([]byte, error) {
	endpoint := c.BaseURL + endpointZTPKubeconfig

	data := url.Values{
		"plan_name": {env},
		"choice":    {choice},
	}

	body, err := c.postFormRaw(ctx, endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZTP kubeconfig for %s: %w", env, err)
	}

	return body, nil
}

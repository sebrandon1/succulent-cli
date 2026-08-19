package lib

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) ProvisionZTP(ctx context.Context, env string, req *ZTPRequest) error {
	data := req.FormValues()
	data.Set("plan", env)

	if err := c.postForm(ctx, c.endpointURL(endpointZTPProvision), data); err != nil {
		return fmt.Errorf("failed to provision ZTP on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetZTPKubeconfig(ctx context.Context, env, choice string) ([]byte, error) {
	data := url.Values{
		"plan_name": {env},
		"choice":    {choice},
	}

	body, err := c.postFormRaw(ctx, c.endpointURL(endpointZTPKubeconfig), data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZTP kubeconfig for %s: %w", env, err)
	}

	return body, nil
}

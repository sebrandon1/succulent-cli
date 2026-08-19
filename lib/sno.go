package lib

import (
	"context"
	"fmt"
)

func (c *Client) ProvisionSNO(ctx context.Context, env string, req *SNOProvisionRequest) error {
	if err := c.postForm(ctx, c.endpointURL(endpointSNOProvision, env), req.FormValues()); err != nil {
		return fmt.Errorf("failed to provision SNO on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetSNOKubeconfig(ctx context.Context, env string) ([]byte, error) {
	resp, err := c.getRaw(ctx, c.endpointURL(endpointSNOKubeconfig, env))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SNO kubeconfig for %s: %w", env, err)
	}
	defer resp.Body.Close()

	data, err := readLimited(resp.Body, MaxResponseSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig response: %w", err)
	}

	return data, nil
}

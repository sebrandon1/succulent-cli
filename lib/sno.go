package lib

import (
	"fmt"
	"io"
)

func (c *Client) ProvisionSNO(env string, req *SNOProvisionRequest) error {
	endpoint := fmt.Sprintf("%s"+endpointSNOProvision, c.BaseURL, env)

	if err := c.postForm(endpoint, req.FormValues()); err != nil {
		return fmt.Errorf("failed to provision SNO on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetSNOKubeconfig(env string) ([]byte, error) {
	requestURL := fmt.Sprintf("%s"+endpointSNOKubeconfig, c.BaseURL, env)

	resp, err := c.getRaw(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SNO kubeconfig for %s: %w", env, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig response: %w", err)
	}

	return data, nil
}

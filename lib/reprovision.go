package lib

import (
	"fmt"
)

func (c *Client) Reprovision(env string, req *ReprovisionRequest) error {
	endpoint := fmt.Sprintf("%s"+endpointReprovision, c.BaseURL, env)

	if err := c.postForm(endpoint, req.FormValues()); err != nil {
		return fmt.Errorf("failed to reprovision %s: %w", env, err)
	}

	return nil
}

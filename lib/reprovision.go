package lib

import "fmt"

func (c *Client) Reprovision(env string, req *ReprovisionRequest) error {
	endpoint := c.BaseURL + endpointReprovision

	data := req.FormValues()
	data.Set("plan", env)

	if err := c.postForm(endpoint, data); err != nil {
		return fmt.Errorf("failed to reprovision %s: %w", env, err)
	}

	return nil
}

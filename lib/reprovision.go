package lib

import (
	"context"
	"fmt"
)

func (c *Client) Reprovision(ctx context.Context, env string, req *ReprovisionRequest) error {
	data := req.FormValues()
	data.Set("plan", env)

	if err := c.postForm(ctx, c.endpointURL(endpointReprovision), data); err != nil {
		return fmt.Errorf("failed to reprovision %s: %w", env, err)
	}

	return nil
}

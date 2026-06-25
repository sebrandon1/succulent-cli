package lib

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) DeleteEnvironment(ctx context.Context, env string) error {
	endpoint := c.BaseURL + endpointDelete

	data := url.Values{
		"plan": {env},
	}

	if err := c.postForm(ctx, endpoint, data); err != nil {
		return fmt.Errorf("failed to delete %s: %w", env, err)
	}

	return nil
}

package lib

import (
	"fmt"
	"net/url"
)

func (c *Client) DeleteEnvironment(env string) error {
	endpoint := c.BaseURL + endpointDelete

	data := url.Values{
		"plan": {env},
	}

	if err := c.postForm(endpoint, data); err != nil {
		return fmt.Errorf("failed to delete %s: %w", env, err)
	}

	return nil
}

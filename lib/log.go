package lib

import (
	"context"
	"fmt"
	"io"
)

func (c *Client) StreamLog(ctx context.Context, env string, w io.Writer) error {
	resp, err := c.getRaw(ctx, c.endpointURL(endpointLog, env))
	if err != nil {
		return fmt.Errorf("failed to fetch log for %s: %w", env, err)
	}
	defer resp.Body.Close()

	if err := copyLimited(w, resp.Body, MaxResponseSize); err != nil {
		return fmt.Errorf("error streaming log: %w", err)
	}

	return nil
}

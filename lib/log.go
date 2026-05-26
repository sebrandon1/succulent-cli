package lib

import (
	"fmt"
	"io"
)

func (c *Client) StreamLog(env string, w io.Writer) error {
	requestURL := fmt.Sprintf("%s/ztp_log/%s", c.BaseURL, env)

	resp, err := c.getRaw(requestURL)
	if err != nil {
		return fmt.Errorf("failed to fetch log for %s: %w", env, err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("error streaming log: %w", err)
	}

	return nil
}

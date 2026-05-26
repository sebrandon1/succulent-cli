package lib

import (
	"fmt"
	"io"
	"net/url"
)

func (c *Client) ProvisionSNO(env string, req *SNOProvisionRequest) error {
	endpoint := fmt.Sprintf("%s/sno/%s", c.BaseURL, env)

	data := url.Values{
		"owner":  {req.Owner},
		"mailto": {req.Email},
	}

	if req.OCPTag != "" {
		data.Set("ocp_tag", req.OCPTag)
	}

	if req.ReleaseType != "" {
		data.Set("ocp_release_type", req.ReleaseType)
	}

	if req.FullOCPTag != "" {
		data.Set("full_ocp_tag", req.FullOCPTag)
	}

	if req.FullImageName != "" {
		data.Set("full_image_name", req.FullImageName)
	}

	if err := c.postForm(endpoint, data); err != nil {
		return fmt.Errorf("failed to provision SNO on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetSNOKubeconfig(env string) ([]byte, error) {
	requestURL := fmt.Sprintf("%s/sno_kubeconfig/%s", c.BaseURL, env)

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

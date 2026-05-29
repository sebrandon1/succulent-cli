package lib

import (
	"fmt"
	"io"
	"net/url"
)

func (c *Client) ProvisionZTP(env string, req *ZTPRequest) error {
	endpoint := c.BaseURL + endpointZTPProvision

	data := url.Values{
		"plan":          {env},
		formFieldOwner:  {req.Owner},
		formFieldMailTo: {req.Email},
	}

	if req.SNOTag != "" {
		data.Set("sno_tag", req.SNOTag)
	}

	if req.SNORelease != "" {
		data.Set("sno_release", req.SNORelease)
	}

	if req.SNOFullTag != "" {
		data.Set("sno_full_tag", req.SNOFullTag)
	}

	if req.ZTPTag != "" {
		data.Set("ztp_tag", req.ZTPTag)
	}

	if req.ZTPRelease != "" {
		data.Set("ztp_release", req.ZTPRelease)
	}

	if req.ZTPFullTag != "" {
		data.Set("ztp_full_tag", req.ZTPFullTag)
	}

	if req.ZTPType != "" {
		data.Set("ztp_type", req.ZTPType)
	}

	if req.StopBeforeDeployment {
		data.Set("stop_before_deployment", "on")
	}

	if req.VMMastersCount != "" {
		data.Set("vm-masters-count", req.VMMastersCount)
	}

	if req.BMMastersHosts != "" {
		data.Set("bm-masters-hosts", req.BMMastersHosts)
	}

	if req.BMWorkersHosts != "" {
		data.Set("bm-workers-hosts", req.BMWorkersHosts)
	}

	if req.VMWorkersCount != "" {
		data.Set("vm-workers-count", req.VMWorkersCount)
	}

	if err := c.postForm(endpoint, data); err != nil {
		return fmt.Errorf("failed to provision ZTP on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetZTPKubeconfig(env, choice string) ([]byte, error) {
	endpoint := c.BaseURL + endpointZTPKubeconfig

	data := url.Values{
		"plan_name": {env},
		"choice":    {choice},
	}

	resp, err := c.HTTPClient.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZTP kubeconfig for %s: %w", env, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

package lib

import (
	"fmt"
	"net/url"
)

func (c *Client) Reprovision(env string, req *ReprovisionRequest) error {
	endpoint := fmt.Sprintf("%s/exposeform/%s", c.BaseURL, env)

	data := url.Values{
		"mailto":       {req.Email},
		formFieldOwner: {req.Owner},
		"tag":          {req.Tag},
		"version":      {req.Version},
	}

	if req.OpenshiftImage != "" {
		data.Set("openshift_image", req.OpenshiftImage)
	}

	if req.DiskSize != "" {
		data.Set("disk_size", req.DiskSize)
	}

	if req.VirtualWorkers != "" {
		data.Set("virtual_workers", req.VirtualWorkers)
	}

	if req.AdditionalWorkers != "" {
		data.Set("additional_workers", req.AdditionalWorkers)
	}

	if req.EndDate != "" {
		data.Set("end_date", req.EndDate)
	}

	if req.KcliParams != "" {
		data.Set("kcli_params", req.KcliParams)
	}

	if err := c.postForm(endpoint, data); err != nil {
		return fmt.Errorf("failed to reprovision %s: %w", env, err)
	}

	return nil
}

package lib

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) ProvisionZTP(ctx context.Context, env string, req *ZTPRequest) error {
	data := req.FormValues()
	data.Set("plan", env)

	if err := c.postForm(ctx, c.endpointURL(endpointZTPProvision), data); err != nil {
		return fmt.Errorf("failed to provision ZTP on %s: %w", env, err)
	}

	return nil
}

func (c *Client) GetZTPKubeconfig(ctx context.Context, env, choice string) ([]byte, error) {
	data := url.Values{
		"plan_name": {env},
		"choice":    {choice},
	}

	body, err := c.postFormRaw(ctx, c.endpointURL(endpointZTPKubeconfig), data)
	if err != nil {
		// The service answers 500 when the requested kubeconfig does not exist,
		// most often because the ZTP spoke was never deployed.
		if IsHTTPStatus(err, http.StatusInternalServerError) {
			return nil, fmt.Errorf("failed to fetch ZTP kubeconfig for %s: %w; the %s kubeconfig may not exist yet "+
				"(check 'succulent-cli get log --env %s' to see whether the ZTP deployment finished)", env, err, choice, env)
		}

		return nil, fmt.Errorf("failed to fetch ZTP kubeconfig for %s: %w", env, err)
	}

	return body, nil
}

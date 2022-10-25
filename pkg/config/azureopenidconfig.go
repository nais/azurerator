package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	WellKnownUrlTemplate = "https://login.microsoftonline.com/%s/v2.0/.well-known/openid-configuration"
)

type AzureOpenIdConfig struct {
	Issuer            string `json:"issuer"`
	TokenEndpoint     string `json:"token_endpoint"`
	JwksURI           string `json:"jwks_uri"`
	WellKnownEndpoint string `json:"well_known_endpoint,omitempty"`
}

func NewAzureOpenIdConfig(ctx context.Context, tenant AzureTenant) (*AzureOpenIdConfig, error) {
	wellKnownUrl := wellKnownUrl(tenant.Id)
	body, err := requestOpenIdConfiguration(ctx, wellKnownUrl)
	if err != nil {
		return nil, err
	}

	azureOpenIdConfig := &AzureOpenIdConfig{}
	if err := json.Unmarshal(body, &azureOpenIdConfig); err != nil {
		return nil, fmt.Errorf("unmarshalling: %w", err)
	}

	azureOpenIdConfig.WellKnownEndpoint = wellKnownUrl

	return azureOpenIdConfig, nil
}

func requestOpenIdConfiguration(ctx context.Context, url string) (body []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating client GET request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading server response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server responded with %s: %s", resp.Status, body)
	}

	return
}

func wellKnownUrl(tenant string) string {
	return fmt.Sprintf(WellKnownUrlTemplate, tenant)
}

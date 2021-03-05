package fake

import "github.com/nais/azureator/pkg/config"

func AzureOpenIdConfig() config.AzureOpenIdConfig {
	return config.AzureOpenIdConfig{
		WellKnownEndpoint: "https://azure-issuer/.well-known/openid-configuration",
		Issuer:            "https://azure-issuer/",
		TokenEndpoint:     "https://azure-issuer/token",
		JwksURI:           "https://azure-issuer/keys",
	}
}

package options

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/secrets"
)

type TransactionOptions struct {
	Finalizer FinalizerOptions
	Namespace NamespaceOptions
	Tenant    TenantOptions
	Process   ProcessOptions
}

type optionsBuilder struct {
	instance v1.AzureAdApplication
	config   config.Config
	secrets  secrets.TransactionSecrets
}

func NewOptions(instance v1.AzureAdApplication, cfg config.Config, secrets secrets.TransactionSecrets) (TransactionOptions, error) {
	builder := optionsBuilder{
		instance: instance,
		config:   cfg,
		secrets:  secrets,
	}

	process, err := builder.Process()
	if err != nil {
		return TransactionOptions{}, err
	}

	return TransactionOptions{
		Finalizer: builder.Finalizer(),
		Namespace: builder.Namespace(),
		Process:   process,
		Tenant:    builder.Tenant(),
	}, nil
}

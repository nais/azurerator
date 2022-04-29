package options

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/transaction/secrets"
)

type TransactionOptions struct {
	Finalizer FinalizerOptions
	Tenant    TenantOptions
	Process   ProcessOptions
}

type optionsBuilder struct {
	instance v1.AzureAdApplication
	config   config.Config
	secrets  secrets.Secrets
}

func NewOptions(instance v1.AzureAdApplication, cfg config.Config, secrets secrets.Secrets) (TransactionOptions, error) {
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
		Process:   process,
		Tenant:    builder.Tenant(),
	}, nil
}

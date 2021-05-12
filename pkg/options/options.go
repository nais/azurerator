package options

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/config"
)

type TransactionOptions struct {
	Finalizer FinalizerOptions
	Namespace NamespaceOptions
	Tenant    TenantOptions
	Process   ProcessOptions
}

type optionsBuilder struct {
	instance v1.AzureAdApplication
	Config   config.Config
}

func NewOptions(instance v1.AzureAdApplication, cfg config.Config) (TransactionOptions, error) {
	builder := optionsBuilder{
		instance: instance,
		Config:   cfg,
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

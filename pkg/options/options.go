package options

import (
	"github.com/nais/azureator/pkg/config"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

type TransactionOptions struct {
	Azure     AzureOptions
	Finalizer FinalizerOptions
	Namespace NamespaceOptions
	Tenant    TenantOptions
	Process   ProcessOptions
	Secret    SecretOptions
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

	azure, err := builder.Azure()
	if err != nil {
		return TransactionOptions{}, err
	}

	process, err := builder.Process()
	if err != nil {
		return TransactionOptions{}, err
	}

	return TransactionOptions{
		Azure:     azure,
		Finalizer: builder.Finalizer(),
		Namespace: builder.Namespace(),
		Process:   process,
		Tenant:    builder.Tenant(),
		Secret:    builder.Secret(),
	}, nil
}

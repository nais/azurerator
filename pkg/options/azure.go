package options

import "github.com/nais/azureator/pkg/customresources"

func (b optionsBuilder) Azure() (AzureOptions, error) {
	hashChanged, err := customresources.IsHashChanged(&b.instance)
	if err != nil {
		return AzureOptions{}, err
	}

	shouldResynchronize := customresources.ShouldResynchronize(&b.instance)

	needsSynchronization := hashChanged || shouldResynchronize

	return AzureOptions{
		Synchronize: needsSynchronization,
	}, nil
}

type AzureOptions struct {
	Synchronize bool
}

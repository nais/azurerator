package options

import "github.com/nais/azureator/pkg/customresources"

func (b optionsBuilder) Process() (ProcessOptions, error) {
	hashChanged, err := customresources.IsHashChanged(&b.instance)
	if err != nil {
		return ProcessOptions{}, err
	}

	secretNameChanged := customresources.SecretNameChanged(&b.instance)
	hasExpiredSecrets := customresources.HasExpiredSecrets(&b.instance, b.Config.SecretRotation.MaxAge)
	shouldResynchronize := customresources.ShouldResynchronize(&b.instance)

	needsSynchronization := hashChanged || secretNameChanged || hasExpiredSecrets || shouldResynchronize

	return ProcessOptions{
		Synchronize: needsSynchronization,
	}, nil
}

type ProcessOptions struct {
	Synchronize bool
}

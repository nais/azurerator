package options

import (
	"github.com/nais/azureator/pkg/annotations"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

type NamespaceOptions struct {
	HasIgnoreAnnotation bool
}

func (b optionsBuilder) Namespace() NamespaceOptions {
	hasAnnotation := HasNotInTeamNamespaceAnnotation(&b.instance)

	return NamespaceOptions{
		HasIgnoreAnnotation: hasAnnotation,
	}
}

func HasNotInTeamNamespaceAnnotation(instance *v1.AzureAdApplication) bool {
	_, found := annotations.HasAnnotation(instance, annotations.NotInTeamNamespaceKey)
	return found
}

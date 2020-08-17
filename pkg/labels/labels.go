package labels

import (
	v1 "github.com/nais/azureator/api/v1"
)

const (
	AppLabelKey    string = "app"
	TeamLabelKey   string = "team"
	TypeLabelKey   string = "type"
	TypeLabelValue string = "azurerator.nais.io"
)

func Labels(instance *v1.AzureAdApplication) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.GetName(),
		TypeLabelKey: TypeLabelValue,
	}
}

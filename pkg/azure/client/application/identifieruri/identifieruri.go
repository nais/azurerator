package identifieruri

import (
	"context"
	"fmt"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/transaction"
	"github.com/nais/azureator/pkg/azure/util"
)

type IdentifierUri interface {
	Set(tx transaction.Transaction, uris azure.IdentifierUris) error
}

type identifierUri struct {
	Application
}

type Application interface {
	Patch(ctx context.Context, id azure.ObjectId, application any) error
}

func NewIdentifierUri(application Application) IdentifierUri {
	return identifierUri{Application: application}
}

func (i identifierUri) Set(tx transaction.Transaction, uris azure.IdentifierUris) error {
	objectId := tx.Instance.GetObjectId()
	app := util.EmptyApplication().
		IdentifierUriList(uris).
		Build()
	if err := i.Application.Patch(tx.Ctx, objectId, app); err != nil {
		return fmt.Errorf("failed to add application identifier URI: %w", err)
	}

	return nil
}

func DescribeCreate(instance v1.AzureAdApplication) azure.IdentifierUris {
	return defaultUris(instance)
}

func DescribeUpdate(instance v1.AzureAdApplication, existing azure.IdentifierUris) azure.IdentifierUris {
	result := make(azure.IdentifierUris, len(existing))
	copy(result, existing)

	for _, uri := range defaultUris(instance) {
		seen := false

		for _, existingUri := range existing {
			if uri == existingUri {
				seen = true
				break
			}
		}

		if !seen {
			result = append(result, uri)
		}
	}

	return result
}

func uriClientId(id azure.ClientId) string {
	return fmt.Sprintf("api://%s", id)
}

func uriHumanReadable(spec v1.AzureAdApplication) string {
	return fmt.Sprintf("api://%s.%s.%s", spec.GetClusterName(), spec.GetNamespace(), spec.GetName())
}

func defaultUris(instance v1.AzureAdApplication) azure.IdentifierUris {
	return []string{
		uriClientId(instance.GetClientId()),
		uriHumanReadable(instance),
	}
}

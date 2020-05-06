package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/azureator/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func (c client) getClientId(ctx context.Context, app v1alpha1.AzureAdPreAuthorizedApplication) (string, error) {
	if len(app.ClientId) > 0 {
		return app.ClientId, nil
	}
	azureApp, err := c.GetByName(ctx, app.Name)
	if err != nil {
		return "", err
	}
	return *azureApp.AppID, nil
}

func (c client) mapToPreAuthorizedApplications(tx azure.Transaction, defaultAccessPermissionId uuid.UUID) []msgraph.PreAuthorizedApplication {
	var preAuthorizedApplications []msgraph.PreAuthorizedApplication
	for _, app := range tx.Resource.Spec.PreAuthorizedApplications {
		clientId, err := c.getClientId(tx.Ctx, app)
		if err != nil {
			// TODO - currently best effort. should separate between technical and functional (e.g. app doesnt exist in AAD) errors
			fmt.Printf("%v\n", err)
			continue
		}
		preAuthorizedApplication := msgraph.PreAuthorizedApplication{
			AppID: &clientId,
			DelegatedPermissionIDs: []string{
				defaultAccessPermissionId.String(),
			},
		}
		preAuthorizedApplications = append(preAuthorizedApplications, preAuthorizedApplication)
	}
	return preAuthorizedApplications
}

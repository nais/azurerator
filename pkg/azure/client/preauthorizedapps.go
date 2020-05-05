package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/apis/v1alpha1"
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

func (c client) mapToPreAuthorizedApplications(ctx context.Context, credential v1alpha1.AzureAdCredential, defaultAccessPermissionId uuid.UUID) []msgraph.PreAuthorizedApplication {
	var preAuthorizedApplications []msgraph.PreAuthorizedApplication
	for _, app := range credential.Spec.PreAuthorizedApplications {
		clientId, err := c.getClientId(ctx, app)
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

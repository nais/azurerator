package client

import (
	"context"
	"fmt"

	"github.com/nais/azureator/apis/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

// Workaround to include empty array of PreAuthorizedApplications in JSON serialization.
// The autogenerated library code uses 'omitempty' for preAuthorizedApplications.
// If all the pre-authorized applications are removed from our custom resource, the PATCH operation on the Azure
// 'Application' resource will not update the list of pre-authorized applications in Azure AD,
// which will no longer reflect our observed nor desired cluster state.
type preAuthAppApi struct {
	PreAuthorizedApplications []msgraph.PreAuthorizedApplication `json:"preAuthorizedApplications"`
}

func (c client) updatePreAuthApps(tx azure.Transaction) ([]azure.PreAuthorizedApp, error) {
	objectId := tx.Resource.Status.ObjectId
	preAuthApps, err := c.createPreAuthAppsMsGraph(tx)
	if err != nil {
		return nil, err
	}
	app := &struct {
		msgraph.DirectoryObject
		API preAuthAppApi `json:"api"`
	}{API: preAuthAppApi{PreAuthorizedApplications: preAuthApps}}
	appReq := c.graphClient.Applications().ID(objectId).Request()
	if err := appReq.JSONRequest(tx.Ctx, "PATCH", "", app, nil); err != nil {
		return nil, fmt.Errorf("failed to update pre-authorized apps in azure: %w", err)
	}
	api := &msgraph.APIApplication{PreAuthorizedApplications: preAuthApps}
	return c.mapPreAuthAppsWithNames(tx.Ctx, *util.EmptyApplication().Api(api).Build())
}

func (c client) getClientId(ctx context.Context, app v1alpha1.AzureAdPreAuthorizedApplication) (string, error) {
	if len(app.ClientId) > 0 {
		return app.ClientId, nil
	}
	azureApp, err := c.GetByName(ctx, app.Name)
	if err != nil {
		return "", fmt.Errorf("failed to fetch pre-authorized application from Azure")
	}
	return *azureApp.AppID, nil
}

func (c client) preAuthAppExists(ctx context.Context, app v1alpha1.AzureAdPreAuthorizedApplication) (bool, error) {
	if len(app.ClientId) == 0 {
		return c.applicationExistsByFilter(ctx, util.FilterByName(app.Name))
	} else {
		return c.applicationExistsByFilter(ctx, util.FilterByAppId(app.ClientId))
	}
}

func (c client) createPreAuthAppsMsGraph(tx azure.Transaction) ([]msgraph.PreAuthorizedApplication, error) {
	preAuthorizedApplications := make([]msgraph.PreAuthorizedApplication, 0)
	for _, app := range tx.Resource.Spec.PreAuthorizedApplications {
		exists, err := c.preAuthAppExists(tx.Ctx, app)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup existence of pre-authorized app (clientId '%s', name '%s'): %w", app.ClientId, app.Name, err)
		}
		if !exists {
			tx.Log.Info(fmt.Sprintf("pre-authorized app (clientId '%s', name '%s') does not exist, skipping assignment...", app.ClientId, app.Name))
			continue
		}
		clientId, err := c.getClientId(tx.Ctx, app)
		if err != nil {
			return nil, err
		}
		preAuthorizedApplication := msgraph.PreAuthorizedApplication{
			AppID:                  &clientId,
			DelegatedPermissionIDs: []string{OAuth2DefaultPermissionScopeId},
		}
		preAuthorizedApplications = append(preAuthorizedApplications, preAuthorizedApplication)
	}
	return preAuthorizedApplications, nil
}

func (c client) mapPreAuthAppsWithNames(ctx context.Context, app msgraph.Application) ([]azure.PreAuthorizedApp, error) {
	preAuthApps := make([]azure.PreAuthorizedApp, 0)
	for _, preAuthApp := range app.API.PreAuthorizedApplications {
		app, err := c.getApplicationByClientId(ctx, *preAuthApp.AppID)
		if err != nil {
			return nil, fmt.Errorf("failed to map preauthorized apps with names: %w", err)
		}
		preAuthApps = append(preAuthApps, azure.PreAuthorizedApp{
			Name:     *app.DisplayName,
			ClientId: *preAuthApp.AppID,
		})
	}
	return preAuthApps, nil
}

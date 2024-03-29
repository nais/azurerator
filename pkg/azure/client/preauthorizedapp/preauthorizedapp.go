package preauthorizedapp

import (
	"context"
	"fmt"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application"
	"github.com/nais/azureator/pkg/azure/client/serviceprincipal"
	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/resource"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/transaction"
)

// Workaround to include empty array of PreAuthorizedApplications in JSON serialization.
// The autogenerated library code uses 'omitempty' for preAuthorizedApplications.
// If all the pre-authorized applications are removed from our custom resource, the PATCH operation on the Azure
// 'Application' resource will not update the list of pre-authorized applications in Azure AD,
// which will no longer reflect our observed nor desired cluster state.
type preAuthAppPatch struct {
	PreAuthorizedApplications []msgraph.PreAuthorizedApplication `json:"preAuthorizedApplications"`
}

type appPatch struct {
	msgraph.DirectoryObject
	API preAuthAppPatch `json:"api"`
}

type PreAuthApps interface {
	Get(tx transaction.Transaction) (*result.PreAuthorizedApps, error)
	Process(tx transaction.Transaction, permissions permissions.Permissions) (*result.PreAuthorizedApps, error)
}

type Client interface {
	azure.RuntimeClient
	Application() application.Application
	AppRoleAssignments(tx transaction.Transaction, targetId azure.ObjectId) serviceprincipal.AppRoleAssignments
	ServicePrincipal() serviceprincipal.ServicePrincipal
}

type preAuthApps struct {
	Client
}

func NewPreAuthApps(client Client) PreAuthApps {
	return preAuthApps{Client: client}
}

func (p preAuthApps) Process(tx transaction.Transaction, permissions permissions.Permissions) (*result.PreAuthorizedApps, error) {
	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	preAuthorizedApps, err := p.mapDesiredPreAuthorizedApps(tx)
	if err != nil {
		return nil, fmt.Errorf("mapping preauthorizedapps to resources: %w", err)
	}

	err = p.patchPreAuthorizedApplications(tx, preAuthorizedApps.Valid, permissions)
	if err != nil {
		return nil, fmt.Errorf("patching preauthorizedapps for application: %w", err)
	}

	err = p.AppRoleAssignments(tx, servicePrincipalId).
		ProcessForServicePrincipals(preAuthorizedApps.Valid, permissions)
	if err != nil {
		return nil, fmt.Errorf("updating approle assignments for service principals: %w", err)
	}

	return preAuthorizedApps, nil
}

func (p preAuthApps) Get(tx transaction.Transaction) (*result.PreAuthorizedApps, error) {
	// lookup desired apps in Azure AD to check for existence
	desired, err := p.mapDesiredPreAuthorizedApps(tx)
	if err != nil {
		return nil, err
	}

	assigned := make([]resource.Resource, 0)
	unassigned := desired.Invalid

	// fetch currently pre-authorized applications from the Application resource
	app, err := p.Application().Get(tx)
	if err != nil {
		return nil, err
	}
	actual := List(app.API.PreAuthorizedApplications)

	// fetch current AppRole assignments from the ServicePrincipal resource
	allAssignments, err := p.AppRoleAssignments(tx, tx.Instance.GetServicePrincipalId()).
		GetAllServicePrincipals()
	if err != nil {
		return nil, err
	}

	for _, valid := range desired.Valid {
		if actual.HasResource(valid) && allAssignments.HasResource(valid) {
			assigned = append(assigned, valid)
			continue
		}

		unassigned = append(unassigned, valid)
	}

	return &result.PreAuthorizedApps{
		Valid:   assigned,
		Invalid: unassigned,
	}, nil
}

func (p preAuthApps) exists(ctx context.Context, app v1.AccessPolicyInboundRule) (*msgraph.Application, bool, error) {
	return p.Application().ExistsByFilter(ctx, util.FilterByName(customresources.GetUniqueName(app.AccessPolicyRule)))
}

func (p preAuthApps) mapDesiredPreAuthorizedApps(tx transaction.Transaction) (*result.PreAuthorizedApps, error) {
	seen := make(map[string]bool)

	validResources := make([]resource.Resource, 0)
	invalidResources := make([]resource.Resource, 0)

	for _, app := range tx.Instance.Spec.PreAuthorizedApplications {
		app = ensureFieldsAreSet(tx, app)

		res, exists, err := p.mapToResource(tx, app)
		if err != nil {
			return nil, fmt.Errorf("looking up existence of PreAuthorizedApp '%s': %w", customresources.GetUniqueName(app.AccessPolicyRule), err)
		}

		if !exists {
			invalidResources = append(invalidResources, *res)
			continue
		}

		if !seen[res.Name] {
			seen[res.Name] = true
			validResources = append(validResources, *res)
		}
	}

	// add self to preauthorizedapps
	if !seen[tx.UniformResourceName] {
		validResources = append(validResources, toResource(tx))
	}

	return &result.PreAuthorizedApps{
		Valid:   validResources,
		Invalid: invalidResources,
	}, nil
}

func (p preAuthApps) patchPreAuthorizedApplications(tx transaction.Transaction, resources []resource.Resource, permissions permissions.Permissions) error {
	added := make(map[azure.ClientId]bool)
	apps := make([]msgraph.PreAuthorizedApplication, 0)

	// add all resources which are valid apps that we've previously checked for existence
	for _, r := range resources {
		apps = append(apps, r.ToPreAuthorizedApp(permissions))
		added[r.ClientId] = true
	}

	msgraphApp, err := p.Application().Get(tx)
	if err != nil {
		return err
	}

	// we want to keep existing pre-authorized applications, but only those that aren't managed by us
	for _, existingPreAuthorizedApp := range msgraphApp.API.PreAuthorizedApplications {
		clientId := *existingPreAuthorizedApp.AppID

		// skip applications already added from resources
		if added[clientId] {
			continue
		}

		isManaged, found := application.IsManagedCache.Get(clientId)
		if found && isManaged {
			continue
		}

		// cache miss or unmanaged, check for existence first - because AAD doesn't automatically remove deleted
		// apps from this list...
		app, exists, err := p.Application().ExistsByFilter(tx.Ctx, util.FilterByAppId(clientId))
		if err != nil {
			return fmt.Errorf("checking existence for pre-authorized app '%s': %w", clientId, err)
		}

		if exists && !application.IsManaged(*app) {
			tx.Logger.Debugf("preserving unmanaged pre-authorized app '%s' (%s)", *app.DisplayName, clientId)
			apps = append(apps, existingPreAuthorizedApp)
		}
	}

	objectId := tx.Instance.GetObjectId()
	payload := appPatch{API: preAuthAppPatch{PreAuthorizedApplications: apps}}

	return p.Application().Patch(tx.Ctx, objectId, payload)
}

func (p preAuthApps) mapToResource(tx transaction.Transaction, app v1.AccessPolicyInboundRule) (*resource.Resource, bool, error) {
	a, exists, err := p.exists(tx.Ctx, app)
	if err != nil || !exists {
		return invalidResource(app), false, err
	}

	exists, servicePrincipal, err := p.ServicePrincipal().Exists(tx.Ctx, *a.AppID)
	if err != nil || !exists {
		return invalidResource(app), false, err
	}

	return &resource.Resource{
		Name:                    *a.DisplayName,
		ClientId:                *a.AppID,
		ObjectId:                *servicePrincipal.ID,
		PrincipalType:           resource.PrincipalTypeServicePrincipal,
		AccessPolicyInboundRule: app,
	}, true, nil
}

func invalidResource(app v1.AccessPolicyInboundRule) *resource.Resource {
	return &resource.Resource{
		Name:                    customresources.GetUniqueName(app.AccessPolicyRule),
		ClientId:                "",
		ObjectId:                "",
		PrincipalType:           resource.PrincipalTypeServicePrincipal,
		AccessPolicyInboundRule: app,
	}
}

func toResource(tx transaction.Transaction) resource.Resource {
	return resource.Resource{
		Name:          tx.UniformResourceName,
		ClientId:      tx.Instance.Status.ClientId,
		ObjectId:      tx.Instance.Status.ServicePrincipalId,
		PrincipalType: resource.PrincipalTypeServicePrincipal,
		AccessPolicyInboundRule: v1.AccessPolicyInboundRule{
			AccessPolicyRule: v1.AccessPolicyRule{
				Application: tx.Instance.GetName(),
				Namespace:   tx.Instance.GetNamespace(),
				Cluster:     tx.ClusterName,
			},
		},
	}
}

func ensureFieldsAreSet(tx transaction.Transaction, rule v1.AccessPolicyInboundRule) v1.AccessPolicyInboundRule {
	if len(rule.Cluster) == 0 {
		rule.Cluster = tx.ClusterName
	}

	if len(rule.Namespace) == 0 {
		rule.Namespace = tx.Instance.GetNamespace()
	}

	return rule
}

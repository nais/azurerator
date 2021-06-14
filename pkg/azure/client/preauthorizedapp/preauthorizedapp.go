package preauthorizedapp

import (
	"context"
	"fmt"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util"
	"github.com/nais/azureator/pkg/azure/util/approle"
	"github.com/nais/azureator/pkg/azure/util/permissions"
	"github.com/nais/azureator/pkg/azure/util/permissionscope"
	"github.com/nais/azureator/pkg/customresources"
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

type preAuthApps struct {
	azure.RuntimeClient
}

func NewPreAuthApps(runtimeClient azure.RuntimeClient) azure.PreAuthApps {
	return preAuthApps{RuntimeClient: runtimeClient}
}

func (p preAuthApps) Process(tx azure.Transaction, permissions permissions.Permissions) (*azure.PreAuthorizedApps, error) {
	// TODO(tronghn): assign/revoke scopes in preauthorizedapps
	//  assign/revoke approles in approleassignment

	servicePrincipalId := tx.Instance.GetServicePrincipalId()

	preAuthorizedApps, err := p.mapToResources(tx)
	if err != nil {
		return nil, fmt.Errorf("mapping preauthorizedapps to resources: %w", err)
	}

	err = p.patchApplication(tx, preAuthorizedApps.Valid, permissions)
	if err != nil {
		return nil, fmt.Errorf("updating preauthorizedapps for application: %w", err)
	}

	permission := permissions[approle.DefaultAppRoleValue]
	err = p.AppRoleAssignments(permission.ID, servicePrincipalId).
		ProcessForServicePrincipals(tx, preAuthorizedApps.Valid)
	if err != nil {
		return nil, fmt.Errorf("updating approle assignments for service principals: %w", err)
	}
	return preAuthorizedApps, nil
}

func (p preAuthApps) Get(tx azure.Transaction) (*azure.PreAuthorizedApps, error) {
	// lookup desired apps in Azure AD to check for existence
	desired, err := p.mapToResources(tx)
	if err != nil {
		return nil, err
	}

	assigned := make([]azure.Resource, 0)
	unassigned := desired.Invalid

	// fetch currently pre-authorized applications from the Application resource
	app, err := p.Application().Get(tx)
	if err != nil {
		return nil, err
	}
	actual := app.API.PreAuthorizedApplications

	// fetch current AppRole assignments from the ServicePrincipal resource
	allAssignments, err := p.AppRoleAssignmentsNoRoleId(tx.Instance.GetServicePrincipalId()).
		GetAllServicePrincipals(tx.Ctx)
	if err != nil {
		return nil, err
	}

	for _, resource := range desired.Valid {
		if !resourceInPreAuthorizedApps(resource, actual) || !resourceInAssignments(resource, allAssignments) {
			unassigned = append(unassigned, resource)
			continue
		}
		assigned = append(assigned, resource)
	}

	return &azure.PreAuthorizedApps{
		Valid:   assigned,
		Invalid: unassigned,
	}, nil
}

func (p preAuthApps) patchApplication(tx azure.Transaction, resources []azure.Resource, permissions permissions.Permissions) error {
	objectId := tx.Instance.GetObjectId()
	payload := p.mapToGraphRequest(resources, permissions)

	if err := p.Application().Patch(tx.Ctx, objectId, payload); err != nil {
		return fmt.Errorf("patching preauthorizedapps for application: %w", err)
	}

	return nil
}

func (p preAuthApps) exists(ctx context.Context, app v1.AccessPolicyRule) (*msgraph.Application, bool, error) {
	return p.Application().ExistsByFilter(ctx, util.FilterByName(customresources.GetUniqueName(app)))
}

func (p preAuthApps) mapToResources(tx azure.Transaction) (*azure.PreAuthorizedApps, error) {
	seen := make(map[string]bool)

	validResources := make([]azure.Resource, 0)
	invalidResources := make([]azure.Resource, 0)

	for _, app := range tx.Instance.Spec.PreAuthorizedApplications {
		app = ensureFieldsAreSet(tx, app)

		resource, exists, err := p.mapToResource(tx, app)
		if err != nil {
			return nil, fmt.Errorf("looking up existence of PreAuthorizedApp '%s': %w", customresources.GetUniqueName(app), err)
		}

		if !exists {
			invalidResources = append(invalidResources, *resource)
			continue
		}

		if !seen[resource.Name] {
			seen[resource.Name] = true
			validResources = append(validResources, *resource)
		}
	}

	if !seen[kubernetes.UniformResourceName(&tx.Instance)] {
		validResources = append(validResources, toResource(tx.Instance))
	}

	return &azure.PreAuthorizedApps{
		Valid:   validResources,
		Invalid: invalidResources,
	}, nil
}

func (p preAuthApps) mapToGraphRequest(resources []azure.Resource, permissions permissions.Permissions) appPatch {
	apps := make([]msgraph.PreAuthorizedApplication, 0)
	for _, resource := range resources {
		clientId := resource.ClientId
		defaultPermission := permissions[permissionscope.DefaultAccessScopeValue]

		apps = append(apps, msgraph.PreAuthorizedApplication{
			AppID: &clientId,
			DelegatedPermissionIDs: []string{
				string(defaultPermission.ID),
			},
		})
	}

	return appPatch{API: preAuthAppPatch{PreAuthorizedApplications: apps}}
}

func (p preAuthApps) mapToResource(tx azure.Transaction, app v1.AccessPolicyRule) (*azure.Resource, bool, error) {
	a, exists, err := p.exists(tx.Ctx, app)
	if err != nil || !exists {
		return invalidResource(app), false, err
	}

	exists, servicePrincipal, err := p.ServicePrincipal().Exists(tx.Ctx, *a.AppID)
	if err != nil || !exists {
		return invalidResource(app), false, err
	}

	return &azure.Resource{
		Name:             *a.DisplayName,
		ClientId:         *a.AppID,
		ObjectId:         *servicePrincipal.ID,
		PrincipalType:    azure.PrincipalTypeServicePrincipal,
		AccessPolicyRule: app,
	}, true, nil
}

func invalidResource(app v1.AccessPolicyRule) *azure.Resource {
	return &azure.Resource{
		Name:             customresources.GetUniqueName(app),
		ClientId:         "",
		ObjectId:         "",
		PrincipalType:    azure.PrincipalTypeServicePrincipal,
		AccessPolicyRule: app,
	}
}

func toResource(instance v1.AzureAdApplication) azure.Resource {
	return azure.Resource{
		Name:          kubernetes.UniformResourceName(&instance),
		ClientId:      instance.Status.ClientId,
		ObjectId:      instance.Status.ServicePrincipalId,
		PrincipalType: azure.PrincipalTypeServicePrincipal,
		AccessPolicyRule: v1.AccessPolicyRule{
			Application: instance.GetName(),
			Namespace:   instance.GetNamespace(),
			Cluster:     instance.GetClusterName(),
		},
	}
}

func ensureFieldsAreSet(tx azure.Transaction, rule v1.AccessPolicyRule) v1.AccessPolicyRule {
	if len(rule.Cluster) == 0 {
		rule.Cluster = tx.Instance.GetClusterName()
	}

	if len(rule.Namespace) == 0 {
		rule.Namespace = tx.Instance.GetNamespace()
	}

	return rule
}

func resourceInPreAuthorizedApps(resource azure.Resource, apps []msgraph.PreAuthorizedApplication) bool {
	for _, app := range apps {
		if *app.AppID == resource.ClientId {
			return true
		}
	}
	return false
}

func resourceInAssignments(resource azure.Resource, assignments []msgraph.AppRoleAssignment) bool {
	for _, a := range assignments {
		equalPrincipalID := *a.PrincipalID == msgraph.UUID(resource.ObjectId)
		equalPrincipalType := azure.PrincipalType(*a.PrincipalType) == resource.PrincipalType

		if equalPrincipalID && equalPrincipalType {
			return true
		}
	}
	return false
}

package application

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/util/permissions"
	"github.com/nais/azureator/pkg/azure/util/permissionscope"
)

type oAuth2PermissionScopes struct {
	azure.Application
}

func newOAuth2PermissionScopes(application azure.Application) azure.OAuth2PermissionScope {
	return oAuth2PermissionScopes{Application: application}
}

// DescribeCreate returns a slice describing the desired msgraph.PermissionScope to be created without actually creating them.
func (o oAuth2PermissionScopes) DescribeCreate(desired permissions.Permissions) []msgraph.PermissionScope {
	existingSet := make(permissionscope.Map)
	return existingSet.ToCreate(desired).ToSlice()
}

// DescribeUpdate returns a slice describing the desired state of both new (if any) and existing msgraph.PermissionScope, i.e:
// 1) add any non-existing, desired scopes.
// 2) disable existing, non-desired scopes.
// It does not perform any modifying operations on the remote state in Azure AD.
func (o oAuth2PermissionScopes) DescribeUpdate(desired permissions.Permissions, existing []msgraph.PermissionScope) []msgraph.PermissionScope {
	result := make([]msgraph.PermissionScope, 0)

	existingSet := permissionscope.ToMap(existing)

	toCreate := existingSet.ToCreate(desired)
	toDisable := existingSet.ToDisable(desired)
	unmodified := existingSet.Unmodified(toCreate, toDisable)

	result = append(result, unmodified.ToSlice()...)
	result = append(result, toCreate.ToSlice()...)
	result = append(result, toDisable.ToSlice()...)
	result = permissionscope.EnsureScopesRequireAdminConsent(result)
	result = permissionscope.EnsureDefaultScopeIsEnabled(result)
	return result
}

package permissionscope

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/permissions"
)

type OAuth2PermissionScope interface {
	DescribeCreate(desired permissions.Permissions) Result
	DescribeUpdate(desired permissions.Permissions, existing []msgraph.PermissionScope) Result
}

type oAuth2PermissionScopes struct{}

func NewOAuth2PermissionScopes() OAuth2PermissionScope {
	return oAuth2PermissionScopes{}
}

// DescribeCreate returns a slice describing the desired msgraph.PermissionScope to be created without actually creating them.
func (o oAuth2PermissionScopes) DescribeCreate(desired permissions.Permissions) Result {
	existingSet := make(Map)
	return NewCreateResult(existingSet.ToCreate(desired))
}

// DescribeUpdate returns a slice describing the desired state of both new (if any) and existing msgraph.PermissionScope, i.e:
// 1) add any non-existing, desired scopes.
// 2) disable existing, non-desired scopes.
// It does not perform any modifying operations on the remote state in Azure AD.
func (o oAuth2PermissionScopes) DescribeUpdate(desired permissions.Permissions, existing []msgraph.PermissionScope) Result {
	result := make([]msgraph.PermissionScope, 0)

	existingSet := ToMap(existing)

	toCreate := existingSet.ToCreate(desired)
	toDisable := existingSet.ToDisable(desired)
	unmodified := existingSet.Unmodified(toCreate, toDisable)

	result = append(result, unmodified.ToSlice()...)
	result = append(result, toCreate.ToSlice()...)
	result = append(result, toDisable.ToSlice()...)
	result = EnsureScopesRequireAdminConsent(result)
	result = EnsureDefaultScopeIsEnabled(result)
	return NewUpdateResult(toCreate, toDisable, unmodified, result)
}

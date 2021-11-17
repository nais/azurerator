package approle

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/permissions"
)

type AppRoles interface {
	DescribeCreate(desired permissions.Permissions) Result
	DescribeUpdate(desired permissions.Permissions, existing []msgraph.AppRole) Result
}

type appRoles struct{}

func NewAppRoles() AppRoles {
	return appRoles{}
}

// DescribeCreate returns a slice describing the desired msgraph.AppRole to be created without actually creating them.
func (a appRoles) DescribeCreate(desired permissions.Permissions) Result {
	existingSet := make(Map)
	return NewCreateResult(existingSet.ToCreate(desired))
}

// DescribeUpdate returns a slice describing the desired state of both new (if any) and existing msgraph.AppRole, i.e:
// 1) add any non-existing, desired roles.
// 2) disable existing, non-desired roles.
// It does not perform any modifying operations on the remote state in Azure AD.
func (a appRoles) DescribeUpdate(desired permissions.Permissions, existing []msgraph.AppRole) Result {
	result := make([]msgraph.AppRole, 0)

	existingSet := ToMap(existing)

	toCreate := existingSet.ToCreate(desired)
	toDisable := existingSet.ToDisable(desired)
	unmodified := existingSet.Unmodified(toCreate, toDisable)

	result = append(result, unmodified.ToSlice()...)
	result = append(result, toCreate.ToSlice()...)
	result = append(result, toDisable.ToSlice()...)
	result = EnsureDefaultAppRoleIsEnabled(result)
	return NewUpdateResult(toCreate, toDisable, unmodified, result)
}

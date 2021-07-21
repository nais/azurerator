package application

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application/approle"
	"github.com/nais/azureator/pkg/azure/permissions"
)

type appRoles struct {
	azure.Application
}

func NewAppRoles(application azure.Application) azure.AppRoles {
	return appRoles{Application: application}
}

// DescribeCreate returns a slice describing the desired msgraph.AppRole to be created without actually creating them.
func (a appRoles) DescribeCreate(desired permissions.Permissions) approle.Result {
	existingSet := make(approle.Map)
	return approle.NewCreateResult(existingSet.ToCreate(desired))
}

// DescribeUpdate returns a slice describing the desired state of both new (if any) and existing msgraph.AppRole, i.e:
// 1) add any non-existing, desired roles.
// 2) disable existing, non-desired roles.
// It does not perform any modifying operations on the remote state in Azure AD.
func (a appRoles) DescribeUpdate(desired permissions.Permissions, existing []msgraph.AppRole) approle.Result {
	result := make([]msgraph.AppRole, 0)

	existingSet := approle.ToMap(existing)

	toCreate := existingSet.ToCreate(desired)
	toDisable := existingSet.ToDisable(desired)
	unmodified := existingSet.Unmodified(toCreate, toDisable)

	result = append(result, unmodified.ToSlice()...)
	result = append(result, toCreate.ToSlice()...)
	result = append(result, toDisable.ToSlice()...)
	result = approle.EnsureDefaultAppRoleIsEnabled(result)
	return approle.NewUpdateResult(toCreate, toDisable, unmodified, result)
}

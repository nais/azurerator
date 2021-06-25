package approleassignment

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure/permissions"
	"github.com/nais/azureator/pkg/azure/resource"
)

type List []msgraph.AppRoleAssignment

func ToAppRoleAssignments(resources resource.Resources, target string, role permissions.Permission) List {
	result := make(List, 0)

	for _, re := range resources {
		result = append(result, re.ToAppRoleAssignment(target, role))
	}

	return result
}

func (l List) Has(assignment msgraph.AppRoleAssignment) bool {
	for _, a := range l {
		equalPrincipalID := *a.PrincipalID == *assignment.PrincipalID
		equalAppRoleID := *a.AppRoleID == *assignment.AppRoleID
		equalPrincipalType := *a.PrincipalType == *assignment.PrincipalType

		if equalPrincipalID && equalAppRoleID && equalPrincipalType {
			return true
		}
	}
	return false
}

func (l List) HasResource(in resource.Resource) bool {
	for _, a := range l {
		equalPrincipalID := *a.PrincipalID == msgraph.UUID(in.ObjectId)
		equalPrincipalType := resource.PrincipalType(*a.PrincipalType) == in.PrincipalType

		if equalPrincipalID && equalPrincipalType {
			return true
		}
	}
	return false
}

func (l List) FilterByRoleID(roleId msgraph.UUID) List {
	filtered := make(List, 0)
	for _, assignment := range l {
		if *assignment.AppRoleID == roleId {
			filtered = append(filtered, assignment)
		}
	}
	return filtered
}

func (l List) FilterByType(principalType resource.PrincipalType) List {
	filtered := make(List, 0)
	for _, assignment := range l {
		if resource.PrincipalType(*assignment.PrincipalType) == principalType {
			filtered = append(filtered, assignment)
		}
	}
	return filtered
}

func (l List) Groups() List {
	return l.FilterByType(resource.PrincipalTypeGroup)
}

func (l List) ServicePrincipals() List {
	return l.FilterByType(resource.PrincipalTypeServicePrincipal)
}

func (l List) WithoutMatchingRole(roles permissions.Permissions) List {
	nonDesired := make(List, 0)

	for _, assignment := range l {
		if !roles.HasRoleID(*assignment.AppRoleID) {
			nonDesired = append(nonDesired, assignment)
		}
	}

	return nonDesired
}

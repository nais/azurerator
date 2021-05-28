package approleassignment

import (
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
)

type appRoleAssignmentKey struct {
	AppRoleID   msgraph.UUID
	PrincipalID msgraph.UUID
	ResourceID  msgraph.UUID
}

func toAppRoleAssignmentKey(assignment msgraph.AppRoleAssignment) appRoleAssignmentKey {
	return appRoleAssignmentKey{
		AppRoleID:   *assignment.AppRoleID,
		PrincipalID: *assignment.PrincipalID,
		ResourceID:  *assignment.ResourceID,
	}
}

// Difference returns the elements in `a` that aren't in `b`.
// Shamelessly stolen and modified from https://stackoverflow.com/a/45428032/11868133
func Difference(a, b []msgraph.AppRoleAssignment) []msgraph.AppRoleAssignment {
	mb := make(map[appRoleAssignmentKey]struct{}, len(b))
	for _, x := range b {
		key := toAppRoleAssignmentKey(x)
		mb[key] = struct{}{}
	}
	diff := make([]msgraph.AppRoleAssignment, 0)
	for _, x := range a {
		key := toAppRoleAssignmentKey(x)
		if _, found := mb[key]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func AssignmentInAssignments(assignment msgraph.AppRoleAssignment, assignments []msgraph.AppRoleAssignment) bool {
	for _, a := range assignments {
		equalPrincipalID := *a.PrincipalID == *assignment.PrincipalID
		equalAppRoleID := *a.AppRoleID == *assignment.AppRoleID
		equalPrincipalType := *a.PrincipalType == *assignment.PrincipalType

		if equalPrincipalID && equalAppRoleID && equalPrincipalType {
			return true
		}
	}
	return false
}

func ResourceInAssignments(resource azure.Resource, assignments []msgraph.AppRoleAssignment) bool {
	for _, a := range assignments {
		equalPrincipalID := *a.PrincipalID == msgraph.UUID(resource.ObjectId)
		equalPrincipalType := azure.PrincipalType(*a.PrincipalType) == resource.PrincipalType

		if equalPrincipalID && equalPrincipalType {
			return true
		}
	}
	return false
}

func ToAssignment(
	roleId msgraph.UUID,
	assignee azure.ServicePrincipalId,
	target azure.ServicePrincipalId,
	principalType azure.PrincipalType,
) (*msgraph.AppRoleAssignment, error) {
	appRoleAssignment := &msgraph.AppRoleAssignment{
		AppRoleID:     &roleId,                    // The ID of the AppRole belonging to the target resource to be assigned
		PrincipalID:   (*msgraph.UUID)(&assignee), // Service Principal ID for the assignee, i.e. the principal that should be assigned to the app role
		ResourceID:    (*msgraph.UUID)(&target),   // Service Principal ID for the target resource, i.e. the application/service principal that owns the app role
		PrincipalType: (*string)(&principalType),
	}
	return appRoleAssignment, nil
}

func FilterByType(assignments []msgraph.AppRoleAssignment, principalType azure.PrincipalType) []msgraph.AppRoleAssignment {
	filtered := make([]msgraph.AppRoleAssignment, 0)
	for _, assignment := range assignments {
		if azure.PrincipalType(*assignment.PrincipalType) == principalType {
			filtered = append(filtered, assignment)
		}
	}
	return filtered
}

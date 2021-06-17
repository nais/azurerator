package approleassignment

import (
	msgraph "github.com/nais/msgraph.go/v1.0"
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

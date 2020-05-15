package util

import (
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
)

type appRoleAssignmentKey struct {
	AppRoleID   msgraphbeta.UUID
	PrincipalID msgraphbeta.UUID
	ResourceID  msgraphbeta.UUID
}

func toAppRoleAssignmentKey(assignment msgraphbeta.AppRoleAssignment) appRoleAssignmentKey {
	return appRoleAssignmentKey{
		AppRoleID:   *assignment.AppRoleID,
		PrincipalID: *assignment.PrincipalID,
		ResourceID:  *assignment.ResourceID,
	}
}

// Difference returns the elements in `a` that aren't in `b`.
// Shamelessly stolen and modified from https://stackoverflow.com/a/45428032/11868133
func Difference(a, b []msgraphbeta.AppRoleAssignment) []msgraphbeta.AppRoleAssignment {
	mb := make(map[appRoleAssignmentKey]struct{}, len(b))
	for _, x := range b {
		key := toAppRoleAssignmentKey(x)
		mb[key] = struct{}{}
	}
	diff := make([]msgraphbeta.AppRoleAssignment, 0)
	for _, x := range a {
		key := toAppRoleAssignmentKey(x)
		if _, found := mb[key]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

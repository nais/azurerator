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
func Difference(a, b List) List {
	mb := make(map[appRoleAssignmentKey]struct{}, len(b))
	for _, x := range b {
		key := toAppRoleAssignmentKey(x)
		mb[key] = struct{}{}
	}
	diff := make(List, 0)
	for _, x := range a {
		key := toAppRoleAssignmentKey(x)
		if _, found := mb[key]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

// ToAssign returns a List describing the desired assignments that do not already exist, i.e. (desired - existing).
func ToAssign(existing, desired List) List {
	return Difference(desired, existing)
}

// ToRevoke returns a List describing existing assignments that are no longer desired, i.e. (existing - desired).
func ToRevoke(existing, desired List) List {
	return Difference(existing, desired)
}

// Unmodified returns a List describing desired assignments that are not modified, i.e. (existing - (toAssign + toRevoke)).
func Unmodified(existing, toAssign, toRevoke List) List {
	return Difference(existing, append(toAssign, toRevoke...))
}

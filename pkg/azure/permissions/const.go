package permissions

const (
	// DefaultPermissionScopeValue is the OAuth2 permission scope that the web API application exposes to client applications
	DefaultPermissionScopeValue string = "defaultaccess"
	// DefaultPermissionScopeId is a unique (per application) ID for the default permission access scope.
	DefaultPermissionScopeId string = "00000000-1337-d34d-b33f-000000000000"
	// DefaultScopeType denotes the default scope type to be set for all the application's permission scopes.
	DefaultScopeType string = "Admin"

	// DefaultAppRoleValue is the default AppRole that the web API application can assign to client applications.
	DefaultAppRoleValue string = "access_as_application"
	// DefaultAppRoleId is the unique (per application) ID for the default AppRole.
	DefaultAppRoleId string = "00000001-abcd-9001-0000-000000000000"

	// DefaultGroupRoleValue is the default AppRole for Groups
	DefaultGroupRoleValue string = "defaultrole"
	// DefaultGroupRoleId is the ID that denotes that the group should be assigned to the application without any special
	// AppRole: https://docs.microsoft.com/en-us/graph/api/group-post-approleassignments?view=graph-rest-1.0&tabs=http#request-body
	DefaultGroupRoleId string = "00000000-0000-0000-0000-000000000000"
)

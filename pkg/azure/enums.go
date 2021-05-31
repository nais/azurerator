package azure

type OperationResult int

const (
	OperationResultCreated OperationResult = iota
	OperationResultUpdated
	OperationResultNotModified
)

// GroupMembershipClaim is the type of groups to emit for tokens returned to the Application from Azure AD
type GroupMembershipClaim string

const (
	// Emits _all_ security groups the user is a member of in the groups claim.
	GroupMembershipClaimSecurityGroup GroupMembershipClaim = "SecurityGroup"
	// Emits only the groups that are explicitly assigned to the application and the user is a member of.
	GroupMembershipClaimApplicationGroup GroupMembershipClaim = "ApplicationGroup"
	// No groups are returned.
	GroupMembershipClaimNone GroupMembershipClaim = "None"
)

type PrincipalType string

const (
	PrincipalTypeGroup            PrincipalType = "Group"
	PrincipalTypeServicePrincipal PrincipalType = "ServicePrincipal"
	PrincipalTypeUser             PrincipalType = "User"
)

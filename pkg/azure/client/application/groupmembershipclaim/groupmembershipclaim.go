package groupmembershipclaim

import (
	"fmt"
	"strings"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

// GroupMembershipClaim is the type of groups to emit for tokens returned to the Application from Azure AD
// See https://learn.microsoft.com/en-us/entra/identity-platform/reference-app-manifest#groupmembershipclaims-attribute.
type GroupMembershipClaim = string

const (
	// All emits all the security groups, distribution groups, and Microsoft Entra directory roles that the signed-in user is a member of
	All GroupMembershipClaim = "All"
	// DirectoryRole emits the Microsoft Entra directory roles the user is a member of)
	DirectoryRole GroupMembershipClaim = "DirectoryRole"
	// SecurityGroup emits _all_ security groups the user is a member of in the groups claim.
	SecurityGroup GroupMembershipClaim = "SecurityGroup"
	// ApplicationGroup emits only the groups that are explicitly assigned to the application and the user is a member of.
	ApplicationGroup GroupMembershipClaim = "ApplicationGroup"
	// None results in no groups emitted.
	None GroupMembershipClaim = "None"
)

func FromAzureAdApplication(app *naisiov1.AzureAdApplication) (GroupMembershipClaim, error) {
	if app.Spec.GroupMembershipClaims == nil {
		return "", nil
	}
	return Normalize(*app.Spec.GroupMembershipClaims)
}

func FromAzureAdApplicationOrDefault(app *naisiov1.AzureAdApplication, defaultValue GroupMembershipClaim) (GroupMembershipClaim, error) {
	claims := defaultValue
	if app.Spec.GroupMembershipClaims != nil {
		claims = *app.Spec.GroupMembershipClaims
	}
	return Normalize(claims)
}

func Normalize(claim string) (GroupMembershipClaim, error) {
	allowed := []GroupMembershipClaim{All, DirectoryRole, SecurityGroup, ApplicationGroup, None}
	for _, valid := range allowed {
		if strings.EqualFold(claim, valid) {
			return valid, nil
		}
	}
	return "", fmt.Errorf("invalid group membership claim: %q, must be one of %s", claim, allowed)
}

package client

import (
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

type requiredResourceAccess struct {
	client
}

func (c client) requiredResourceAccess() requiredResourceAccess {
	return requiredResourceAccess{c}
}

// Access to Microsoft Graph API
func (r requiredResourceAccess) microsoftGraph() msgraph.RequiredResourceAccess {
	userReadScopeId := msgraph.UUID("e1fe6dd8-ba31-4d61-89e7-88639da4683d")    // User.Read
	openidScopeId := msgraph.UUID("37f7f235-527c-4136-accd-4a02d197296e")      // openid
	groupMemberReadAll := msgraph.UUID("bc024368-1153-4739-b217-4326f2e966d0") // GroupMember.Read.All
	return msgraph.RequiredResourceAccess{
		ResourceAppID: ptr.String("00000003-0000-0000-c000-000000000000"),
		ResourceAccess: []msgraph.ResourceAccess{
			{
				ID:   &userReadScopeId,
				Type: ptr.String("Scope"),
			},
			{
				ID:   &openidScopeId,
				Type: ptr.String("Scope"),
			},
			{
				ID:   &groupMemberReadAll,
				Type: ptr.String("Scope"),
			},
		},
	}
}

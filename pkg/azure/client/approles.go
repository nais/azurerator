package client

import (
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

const (
	DefaultAppRole   string = "access_as_application"
	DefaultAppRoleId string = "00000001-abcd-9001-0000-000000000000"
)

func toApprole(roleName string) msgraph.AppRole {
	roleId := msgraph.UUID(DefaultAppRoleId)
	allowedmembertypes := []string{"Application"}
	return msgraph.AppRole{
		Object:             msgraph.Object{},
		AllowedMemberTypes: allowedmembertypes,
		Description:        ptr.String(roleName),
		DisplayName:        ptr.String(roleName),
		ID:                 &roleId,
		IsEnabled:          ptr.Bool(true),
		Value:              ptr.String(roleName),
	}
}

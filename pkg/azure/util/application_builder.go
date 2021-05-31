package util

import (
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
)

type ApplicationBuilder struct {
	*msgraph.Application
}

func EmptyApplication() ApplicationBuilder {
	return ApplicationBuilder{&msgraph.Application{}}
}

func Application(template *msgraph.Application) ApplicationBuilder {
	return ApplicationBuilder{template}
}

func (a ApplicationBuilder) Keys(keyCredentials []msgraph.KeyCredential) ApplicationBuilder {
	a.KeyCredentials = keyCredentials
	return a
}

func (a ApplicationBuilder) IdentifierUriList(uris azure.IdentifierUris) ApplicationBuilder {
	a.IdentifierUris = uris
	return a
}

func (a ApplicationBuilder) PreAuthorizedApps(preAuthApps []msgraph.PreAuthorizedApplication) ApplicationBuilder {
	a.API.PreAuthorizedApplications = preAuthApps
	return a
}

func (a ApplicationBuilder) ResourceAccess(access []msgraph.RequiredResourceAccess) ApplicationBuilder {
	a.RequiredResourceAccess = access
	return a
}

func (a ApplicationBuilder) GroupMembershipClaims(groupMembershipClaim azure.GroupMembershipClaim) ApplicationBuilder {
	a.Application.GroupMembershipClaims = ptr.String(string(groupMembershipClaim))
	return a
}

func (a ApplicationBuilder) AppRoles(appRoles []msgraph.AppRole) ApplicationBuilder {
	a.Application.AppRoles = appRoles
	return a
}

func (a ApplicationBuilder) RedirectUris(redirectUris []string) ApplicationBuilder {
	a.Web.RedirectUris = redirectUris
	return a
}

func (a ApplicationBuilder) Build() *msgraph.Application {
	return a.Application
}

package util

import (
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/msgraph.go/ptr"
	msgraph "github.com/nais/msgraph.go/v1.0"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client/application/groupmembershipclaim"
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

func (a ApplicationBuilder) GroupMembershipClaims(groupMembershipClaim groupmembershipclaim.GroupMembershipClaim) ApplicationBuilder {
	a.Application.GroupMembershipClaims = ptr.String(groupMembershipClaim)
	return a
}

func (a ApplicationBuilder) AppRoles(appRoles []msgraph.AppRole) ApplicationBuilder {
	a.Application.AppRoles = appRoles
	return a
}

func (a ApplicationBuilder) RedirectUris(redirectUris []string, instance *naisiov1.AzureAdApplication) ApplicationBuilder {
	if instance.Spec.SinglePageApplication != nil && *instance.Spec.SinglePageApplication {
		return a.singlePageAppRedirectUri(redirectUris)
	}
	return a.webAppRedirectUri(redirectUris)
}

func (a ApplicationBuilder) webAppRedirectUri(redirectUris []string) ApplicationBuilder {
	if a.Web == nil {
		a.Web = &msgraph.WebApplication{}
	}
	a.Web.RedirectUris = redirectUris
	return a
}

func (a ApplicationBuilder) singlePageAppRedirectUri(redirectUris []string) ApplicationBuilder {
	if a.Spa == nil {
		a.Spa = &msgraph.SpaApplication{}
	}
	a.Spa.RedirectUris = redirectUris
	return a
}

func (a ApplicationBuilder) PermissionScopes(scopes []msgraph.PermissionScope) ApplicationBuilder {
	if a.API == nil {
		a.API = &msgraph.APIApplication{}
	}
	a.API.OAuth2PermissionScopes = scopes
	return a
}

func (a ApplicationBuilder) OptionalClaims(optionalClaims *msgraph.OptionalClaims) ApplicationBuilder {
	a.Application.OptionalClaims = optionalClaims
	return a
}

func (a ApplicationBuilder) Build() *msgraph.Application {
	return a.Application
}

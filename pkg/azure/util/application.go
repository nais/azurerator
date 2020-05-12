package util

import (
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
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

func (a ApplicationBuilder) Key(keyCredential msgraph.KeyCredential) ApplicationBuilder {
	a.KeyCredentials = []msgraph.KeyCredential{keyCredential}
	return a
}

func (a ApplicationBuilder) Keys(keyCredentials []msgraph.KeyCredential) ApplicationBuilder {
	a.KeyCredentials = keyCredentials
	return a
}

func (a ApplicationBuilder) Api(apiApplication *msgraph.APIApplication) ApplicationBuilder {
	a.API = apiApplication
	return a
}

func (a ApplicationBuilder) IdentifierUri(uri string) ApplicationBuilder {
	a.IdentifierUris = []string{uri}
	return a
}

func (a ApplicationBuilder) AppRoles(appRoles []msgraph.AppRole) ApplicationBuilder {
	a.Application.AppRoles = appRoles
	return a
}

func (a ApplicationBuilder) Build() *msgraph.Application {
	return a.Application
}

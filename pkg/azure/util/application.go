package util

import (
	"github.com/nais/azureator/pkg/azure"
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

func (a ApplicationBuilder) IdentifierUri(uri azure.IdentifierUri) ApplicationBuilder {
	a.IdentifierUris = []string{uri}
	return a
}

func (a ApplicationBuilder) PreAuthorizedApps(preAuthApps []msgraph.PreAuthorizedApplication) ApplicationBuilder {
	a.API.PreAuthorizedApplications = preAuthApps
	return a
}

func (a ApplicationBuilder) Build() *msgraph.Application {
	return a.Application
}

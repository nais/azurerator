package azure

import (
	"context"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	msgraph "github.com/nais/msgraph.go/v1.0"
	log "github.com/sirupsen/logrus"
)

type Transaction struct {
	Ctx      context.Context
	Instance v1.AzureAdApplication
	Log      log.Entry
}

func (t Transaction) UpdateWithApplicationIDs(application msgraph.Application) Transaction {
	newInstance := t.Instance
	newInstance.Status.ClientId = *application.AppID
	newInstance.Status.ObjectId = *application.ID
	t.Instance = newInstance
	return t
}

func (t Transaction) UpdateWithServicePrincipalID(servicePrincipal msgraph.ServicePrincipal) Transaction {
	newInstance := t.Instance
	newInstance.Status.ServicePrincipalId = *servicePrincipal.ID
	t.Instance = newInstance
	return t
}

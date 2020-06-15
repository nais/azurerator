package azure

import (
	"github.com/google/uuid"
	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func ExternalAzureApp(instance v1.AzureAdApplication) msgraph.Application {
	objectId := getOrGenerate(instance.Status.ObjectId)
	clientId := getOrGenerate(instance.Status.ClientId)

	return msgraph.Application{
		DirectoryObject: msgraph.DirectoryObject{
			Entity: msgraph.Entity{ID: ptr.String(objectId)},
		},
		DisplayName: ptr.String(instance.GetUniqueName()),
		AppID:       ptr.String(clientId),
	}
}

func InternalAzureApp(instance v1.AzureAdApplication) azure.Application {
	jwk, err := crypto.GenerateJwkPair(instance)
	if err != nil {
		panic(err)
	}

	objectId := getOrGenerate(instance.Status.ObjectId)
	clientId := getOrGenerate(instance.Status.ClientId)
	servicePrincipalId := getOrGenerate(instance.Status.ServicePrincipalId)

	tenantId := uuid.New().String()
	lastPasswordKeyId := uuid.New().String()
	lastCertificateKeyId := uuid.New().String()

	return azure.Application{
		Certificate: azure.Certificate{
			KeyId: azure.KeyId{
				Latest:   lastCertificateKeyId,
				AllInUse: []string{lastCertificateKeyId},
			},
			Jwks: azure.Jwks{
				Public:  jwk.Public,
				Private: jwk.Private,
			},
		},
		Password: azure.Password{
			KeyId: azure.KeyId{
				Latest:   lastPasswordKeyId,
				AllInUse: []string{lastPasswordKeyId},
			},
			ClientSecret: uuid.New().String(),
		},
		ClientId:           clientId,
		ObjectId:           objectId,
		ServicePrincipalId: servicePrincipalId,
		PreAuthorizedApps:  mapToInternalPreAuthApps(instance.Spec.PreAuthorizedApplications),
		Tenant:             tenantId,
	}
}

func mapToInternalPreAuthApps(apps []v1.AzureAdPreAuthorizedApplication) []azure.PreAuthorizedApp {
	as := make([]azure.PreAuthorizedApp, 0)
	for _, app := range apps {
		as = append(as, mapToInternalPreAuthApp(app))
	}
	return as
}

func mapToInternalPreAuthApp(app v1.AzureAdPreAuthorizedApplication) azure.PreAuthorizedApp {
	clientId := uuid.New().String()
	name := getOrGenerate(app.GetUniqueName())
	return azure.PreAuthorizedApp{
		Name:     name,
		ClientId: clientId,
	}
}

func getOrGenerate(field string) string {
	if len(field) > 0 {
		return field
	} else {
		return uuid.New().String()
	}
}

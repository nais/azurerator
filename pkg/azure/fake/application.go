package fake

import (
	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/util/crypto"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
)

func ExternalAzureApp(instance v1.AzureAdApplication) msgraph.Application {
	objectId := getOrGenerate(instance.GetObjectId())
	clientId := getOrGenerate(instance.GetClientId())

	return msgraph.Application{
		DirectoryObject: msgraph.DirectoryObject{
			Entity: msgraph.Entity{ID: ptr.String(objectId)},
		},
		DisplayName: ptr.String(instance.GetUniqueName()),
		AppID:       ptr.String(clientId),
	}
}

func InternalAzureApp(instance v1.AzureAdApplication) azure.ApplicationResult {
	jwk, err := crypto.GenerateJwk(instance)
	if err != nil {
		panic(err)
	}

	objectId := getOrGenerate(instance.GetObjectId())
	clientId := getOrGenerate(instance.GetClientId())
	servicePrincipalId := getOrGenerate(instance.GetServicePrincipalId())

	tenantId := uuid.New().String()
	lastPasswordKeyId := uuid.New().String()
	lastCertificateKeyId := uuid.New().String()

	return azure.ApplicationResult{
		Certificate: azure.Certificate{
			KeyId: azure.KeyId{
				Latest:   lastCertificateKeyId,
				AllInUse: []string{lastCertificateKeyId},
			},
			Jwk: jwk,
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

func mapToInternalPreAuthApps(apps []v1.AccessPolicyRule) []azure.Resource {
	as := make([]azure.Resource, 0)
	for _, app := range apps {
		as = append(as, mapToInternalPreAuthApp(app))
	}
	return as
}

func mapToInternalPreAuthApp(app v1.AccessPolicyRule) azure.Resource {
	clientId := uuid.New().String()
	objectId := uuid.New().String()
	name := getOrGenerate(app.GetUniqueName())
	return azure.Resource{
		Name:          name,
		ClientId:      clientId,
		ObjectId:      objectId,
		PrincipalType: azure.PrincipalTypeServicePrincipal,
	}
}

func getOrGenerate(field string) string {
	if len(field) > 0 {
		return field
	} else {
		return uuid.New().String()
	}
}

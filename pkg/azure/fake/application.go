package fake

import (
	"github.com/google/uuid"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/util/crypto"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func MsGraphApplication(instance v1.AzureAdApplication) msgraph.Application {
	objectId := getOrGenerate(instance.GetObjectId())
	clientId := getOrGenerate(instance.GetClientId())

	return msgraph.Application{
		DirectoryObject: msgraph.DirectoryObject{
			Entity: msgraph.Entity{ID: ptr.String(objectId)},
		},
		DisplayName: ptr.String(kubernetes.UniformResourceName(&instance)),
		AppID:       ptr.String(clientId),
	}
}

func AzureApplicationResult(instance v1.AzureAdApplication) azure.ApplicationResult {
	objectId := getOrGenerate(instance.GetObjectId())
	clientId := getOrGenerate(instance.GetClientId())
	servicePrincipalId := getOrGenerate(instance.GetServicePrincipalId())

	tenantId := uuid.New().String()

	return azure.ApplicationResult{
		ClientId:           clientId,
		ObjectId:           objectId,
		ServicePrincipalId: servicePrincipalId,
		PreAuthorizedApps:  mapToInternalPreAuthApps(instance.Spec.PreAuthorizedApplications),
		Tenant:             tenantId,
	}
}

func AzureCredentialsSet(instance v1.AzureAdApplication) azure.CredentialsSet {
	currJwk, err := crypto.GenerateJwk(instance)
	if err != nil {
		panic(err)
	}

	nextJwk, err := crypto.GenerateJwk(instance)
	if err != nil {
		panic(err)
	}

	return azure.CredentialsSet{
		Current: azure.Credentials{
			Certificate: azure.Certificate{
				KeyId: uuid.New().String(),
				Jwk:   currJwk,
			},
			Password: azure.Password{
				KeyId:        uuid.New().String(),
				ClientSecret: uuid.New().String(),
			},
		},
		Next: azure.Credentials{
			Certificate: azure.Certificate{
				KeyId: uuid.New().String(),
				Jwk:   nextJwk,
			},
			Password: azure.Password{
				KeyId:        uuid.New().String(),
				ClientSecret: uuid.New().String(),
			},
		},
	}
}

func mapToInternalPreAuthApps(apps []v1.AccessPolicyRule) azure.PreAuthorizedApps {
	valid := make([]azure.Resource, 0)
	invalid := make([]azure.Resource, 0)

	for _, app := range apps {
		if strings.Contains(customresources.GetUniqueName(app), "invalid") {
			invalid = append(invalid, mapToInternalPreAuthApp(app))
		} else {
			valid = append(valid, mapToInternalPreAuthApp(app))
		}
	}

	return azure.PreAuthorizedApps{
		Valid:   valid,
		Invalid: invalid,
	}
}

func mapToInternalPreAuthApp(app v1.AccessPolicyRule) azure.Resource {
	clientId := uuid.New().String()
	objectId := uuid.New().String()
	name := getOrGenerate(kubernetes.UniformResourceName(&metav1.ObjectMeta{
		Name:        app.Application,
		Namespace:   app.Namespace,
		ClusterName: app.Cluster,
	}))
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

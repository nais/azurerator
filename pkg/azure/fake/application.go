package fake

import (
	"strings"

	"github.com/google/uuid"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/azureator/pkg/azure/credentials"
	"github.com/nais/azureator/pkg/azure/resource"
	"github.com/nais/azureator/pkg/azure/result"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/util/crypto"
)

func AzureApplicationResult(instance v1.AzureAdApplication, operation result.Operation) result.Application {
	objectId := GetOrGenerate(instance.GetObjectId())
	clientId := GetOrGenerate(instance.GetClientId())
	servicePrincipalId := GetOrGenerate(instance.GetServicePrincipalId())

	tenantId := uuid.New().String()

	return result.Application{
		ClientId:           clientId,
		ObjectId:           objectId,
		ServicePrincipalId: servicePrincipalId,
		PreAuthorizedApps:  mapToInternalPreAuthApps(instance.Spec.PreAuthorizedApplications),
		Tenant:             tenantId,
		Result:             operation,
	}
}

func AzureCredentialsSet(instance v1.AzureAdApplication, clusterName string) credentials.Set {
	currJwk, err := crypto.GenerateJwk(instance, clusterName)
	if err != nil {
		panic(err)
	}

	nextJwk, err := crypto.GenerateJwk(instance, clusterName)
	if err != nil {
		panic(err)
	}

	return credentials.Set{
		Current: credentials.Credentials{
			Certificate: credentials.Certificate{
				KeyId: uuid.New().String(),
				Jwk:   currJwk,
			},
			Password: credentials.Password{
				KeyId:        uuid.New().String(),
				ClientSecret: uuid.New().String(),
			},
		},
		Next: credentials.Credentials{
			Certificate: credentials.Certificate{
				KeyId: uuid.New().String(),
				Jwk:   nextJwk,
			},
			Password: credentials.Password{
				KeyId:        uuid.New().String(),
				ClientSecret: uuid.New().String(),
			},
		},
	}
}

func AzurePreAuthorizedApps(instance v1.AzureAdApplication) *result.PreAuthorizedApps {
	preAuthApps := mapToInternalPreAuthApps(instance.Spec.PreAuthorizedApplications)
	return &preAuthApps
}

func mapToInternalPreAuthApps(apps []v1.AccessPolicyInboundRule) result.PreAuthorizedApps {
	valid := make([]resource.Resource, 0)
	invalid := make([]resource.Resource, 0)

	for _, app := range apps {
		if strings.Contains(customresources.GetUniqueName(app.AccessPolicyRule), "invalid") {
			invalid = append(invalid, mapToInternalPreAuthApp(app))
		} else {
			valid = append(valid, mapToInternalPreAuthApp(app))
		}
	}

	return result.PreAuthorizedApps{
		Valid:   valid,
		Invalid: invalid,
	}
}

func mapToInternalPreAuthApp(app v1.AccessPolicyInboundRule) resource.Resource {
	clientId := uuid.New().String()
	objectId := uuid.New().String()
	name := GetOrGenerate(kubernetes.UniformResourceName(&metav1.ObjectMeta{
		Name:      app.Application,
		Namespace: app.Namespace,
	}, app.Cluster))
	return resource.Resource{
		Name:                    name,
		ClientId:                clientId,
		ObjectId:                objectId,
		PrincipalType:           resource.PrincipalTypeServicePrincipal,
		AccessPolicyInboundRule: app,
	}
}

func GetOrGenerate(field string) string {
	if len(field) > 0 {
		return field
	} else {
		return uuid.New().String()
	}
}

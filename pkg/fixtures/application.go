package fixtures

import (
	"github.com/google/uuid"
	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/util/crypto"
	"github.com/yaegashi/msgraph.go/ptr"
	msgraph "github.com/yaegashi/msgraph.go/v1.0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func MinimalK8sAzureAdApplication() *v1alpha1.AzureAdApplication {
	key := types.NamespacedName{
		Name:      "test-app",
		Namespace: "test-namespace",
	}
	spec := v1alpha1.AzureAdApplicationSpec{
		ReplyUrls:                 nil,
		PreAuthorizedApplications: nil,
		LogoutUrl:                 "test",
		SecretName:                "test",
		ConfigMapName:             "test",
	}
	status := v1alpha1.AzureAdApplicationStatus{
		PasswordKeyId:      "test",
		CertificateKeyId:   "test",
		ClientId:           "test",
		ObjectId:           "test",
		ServicePrincipalId: "test",
		ProvisionHash:      "100306fda4b3e77",
	}
	return K8sAzureAdApplication(key, spec, status)
}

func K8sAzureAdApplication(key types.NamespacedName, spec v1alpha1.AzureAdApplicationSpec, status v1alpha1.AzureAdApplicationStatus) *v1alpha1.AzureAdApplication {
	return &v1alpha1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        key.Name,
			Namespace:   key.Namespace,
			ClusterName: "test-cluster",
		},
		Spec:   spec,
		Status: status,
	}
}

func ExternalAzureApp(instance v1alpha1.AzureAdApplication) msgraph.Application {
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

func InternalAzureApp(instance v1alpha1.AzureAdApplication) azure.Application {
	jwk, err := crypto.GenerateJwkPair(instance)
	if err != nil {
		panic(err)
	}

	objectId := getOrGenerate(instance.Status.ObjectId)
	clientId := getOrGenerate(instance.Status.ClientId)
	servicePrincipalId := getOrGenerate(instance.Status.ServicePrincipalId)

	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: clientId,
				Jwk:      jwk.Public,
			},
			Private: azure.Private{
				Jwk:          jwk.Private,
				ClientSecret: uuid.New().String(),
			},
		},
		ClientId:           clientId,
		ObjectId:           objectId,
		ServicePrincipalId: servicePrincipalId,
		CertificateKeyId:   uuid.New().String(),
		PasswordKeyId:      uuid.New().String(),
		PreAuthorizedApps:  mapToInternalPreAuthApps(instance.Spec.PreAuthorizedApplications),
	}
}

func mapToInternalPreAuthApps(apps []v1alpha1.AzureAdPreAuthorizedApplication) []azure.PreAuthorizedApp {
	as := make([]azure.PreAuthorizedApp, 0)
	for _, app := range apps {
		as = append(as, mapToInternalPreAuthApp(app))
	}
	return as
}

func mapToInternalPreAuthApp(app v1alpha1.AzureAdPreAuthorizedApplication) azure.PreAuthorizedApp {
	clientId := getOrGenerate(app.ClientId)
	name := getOrGenerate(app.Name)
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

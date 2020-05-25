package fixtures

import (
	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/util/crypto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MinimalApplication() *v1alpha1.AzureAdApplication {
	return &v1alpha1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: v1alpha1.AzureAdApplicationSpec{
			ReplyUrls:                 nil,
			PreAuthorizedApplications: nil,
			LogoutUrl:                 "test",
			SecretName:                "test",
			ConfigMapName:             "test",
		},
		Status: v1alpha1.AzureAdApplicationStatus{
			PasswordKeyId:      "test",
			CertificateKeyId:   "test",
			ClientId:           "test",
			ObjectId:           "test",
			ServicePrincipalId: "test",
			ProvisionHash:      "100306fda4b3e77",
		},
	}
}

func AzureApp() azure.Application {
	jwk, err := crypto.GenerateJwkPair(*MinimalApplication())
	if err != nil {
		panic(err)
	}
	clientId := "test-id"
	return azure.Application{
		Credentials: azure.Credentials{
			Public: azure.Public{
				ClientId: "",
				Jwk:      jwk.Public,
			},
			Private: azure.Private{
				Jwk:          jwk.Private,
				ClientId:     clientId,
				ClientSecret: "test-secret",
			},
		},
		ClientId:           clientId,
		ObjectId:           "test-object",
		ServicePrincipalId: "test-serviceprincipal",
		CertificateKeyId:   "test-certificate",
		PasswordKeyId:      "test-password",
		PreAuthorizedApps: []azure.PreAuthorizedApp{
			{
				Name:     "test-preauthapp",
				ClientId: "test-preauthapp-id",
			},
		},
	}
}

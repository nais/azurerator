package fixtures

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MinimalApplication() *nais_io_v1.AzureAdApplication {
	now := metav1.Now()
	return &nais_io_v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test-namespace",
		},
		Spec: nais_io_v1.AzureAdApplicationSpec{
			ReplyUrls:                 nil,
			PreAuthorizedApplications: nil,
			LogoutUrl:                 "test",
			SecretName:                "test",
		},
		Status: nais_io_v1.AzureAdApplicationStatus{
			PasswordKeyIds:                    []string{"test"},
			CertificateKeyIds:                 []string{"test"},
			ClientId:                          "test",
			ObjectId:                          "test",
			ServicePrincipalId:                "test",
			SynchronizationHash:               "b85f1aaff45fcfc2",
			SynchronizationSecretName:         "test",
			SynchronizationSecretRotationTime: &now,
		},
	}
}

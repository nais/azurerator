package azurerator_test

import (
	"github.com/nais/azureator/pkg/util/azurerator"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestAzureAdApplication_IsUpToDate(t *testing.T) {
	t.Run("Application with unchanged spec should be synchronized", func(t *testing.T) {
		app := minimalApplication()
		actual, err := azurerator.IsUpToDate(app)
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed spec should not be synchronized", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.SecretName = "yolo"
		actual, err := azurerator.IsUpToDate(app)
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestHasExtraPolicy(t *testing.T) {
	claims := &nais_io_v1.AzureAdClaims{
		Extra: []nais_io_v1.AzureAdExtraClaim{
			"some-policy",
			"some-other-policy",
		},
	}

	t.Run("Checking for non-existent extra claim should return false", func(t *testing.T) {
		actual := azurerator.HasExtraPolicy(claims, "non-existent")
		assert.False(t, actual)
	})
	t.Run("Checking for extra claim should return true", func(t *testing.T) {
		actual := azurerator.HasExtraPolicy(claims, "some-policy")
		assert.True(t, actual)
	})
}

func minimalApplication() *nais_io_v1.AzureAdApplication {
	return &nais_io_v1.AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: nais_io_v1.AzureAdApplicationSpec{
			ReplyUrls:                 nil,
			PreAuthorizedApplications: nil,
			LogoutUrl:                 "test",
			SecretName:                "test",
		},
		Status: nais_io_v1.AzureAdApplicationStatus{
			PasswordKeyIds:            []string{"test"},
			CertificateKeyIds:         []string{"test"},
			ClientId:                  "test",
			ObjectId:                  "test",
			ServicePrincipalId:        "test",
			SynchronizationHash:       "3b810bb8df7a4bf1",
			SynchronizationSecretName: "test",
		},
	}
}

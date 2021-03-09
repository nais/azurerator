package customresources_test

import (
	"github.com/nais/azureator/pkg/customresources"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestAzureAdApplication_IsHashChanged(t *testing.T) {
	t.Run("Application with unchanged spec should be synchronized", func(t *testing.T) {
		app := minimalApplication()
		actual, err := customresources.IsHashChanged(app)
		assert.NoError(t, err)
		assert.False(t, actual)
	})
	t.Run("Application with changed spec should not be synchronized", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.LogoutUrl = "yolo"
		actual, err := customresources.IsHashChanged(app)
		assert.NoError(t, err)
		assert.True(t, actual)
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
		actual := customresources.HasExtraPolicy(claims, "non-existent")
		assert.False(t, actual)
	})
	t.Run("Checking for extra claim should return true", func(t *testing.T) {
		actual := customresources.HasExtraPolicy(claims, "some-policy")
		assert.True(t, actual)
	})
}

func TestIsSecretNameChanged(t *testing.T) {
	t.Run("Application with unchanged secret name", func(t *testing.T) {
		app := minimalApplication()
		shouldUpdate := customresources.IsSecretNameChanged(app)
		assert.False(t, shouldUpdate)
	})

	t.Run("Application with changed secret name", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.SecretName = "some-secret"
		shouldUpdate := customresources.IsSecretNameChanged(app)
		assert.True(t, shouldUpdate)
	})

	t.Run("Application with not set synchronized secret name in status", func(t *testing.T) {
		app := minimalApplication()
		app.Status.SynchronizationSecretName = ""
		shouldUpdate := customresources.IsSecretNameChanged(app)
		assert.True(t, shouldUpdate)
	})
}

func TestHasExpiredSecrets(t *testing.T) {
	t.Run("not set rotation time should return not expired", func(t *testing.T) {
		app := minimalApplication()
		app.Status.SynchronizationSecretRotationTime = nil

		shouldUpdate := customresources.HasExpiredSecrets(app, time.Minute)
		assert.False(t, shouldUpdate)
	})

	t.Run("valid secret should return not expired", func(t *testing.T) {
		app := minimalApplication()
		shouldUpdate := customresources.HasExpiredSecrets(app, time.Minute)

		assert.False(t, shouldUpdate)
	})

	t.Run("expired secret should return expired", func(t *testing.T) {
		app := minimalApplication()

		expiredTime := metav1.NewTime(metav1.Now().Add(-1 * time.Minute))
		app.Status.SynchronizationSecretRotationTime = &expiredTime

		shouldUpdate := customresources.HasExpiredSecrets(app, time.Minute)
		assert.True(t, shouldUpdate)
	})
}

func TestShouldUpdateSecrets(t *testing.T) {
	t.Run("Application with unchanged secret name", func(t *testing.T) {
		t.Run("not set rotation time should not update secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Status.SynchronizationSecretRotationTime = nil

			shouldUpdate := customresources.ShouldUpdateSecrets(app, time.Minute)
			assert.False(t, shouldUpdate)
		})

		t.Run("not set synchronizedsecretname should update secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Status.SynchronizationSecretName = ""

			shouldUpdate := customresources.ShouldUpdateSecrets(app, time.Minute)

			assert.True(t, shouldUpdate)
		})

		t.Run("valid secret should not update secrets", func(t *testing.T) {
			app := minimalApplication()
			shouldUpdate := customresources.ShouldUpdateSecrets(app, time.Minute)

			assert.False(t, shouldUpdate)
		})

		t.Run("expired secret should update secrets", func(t *testing.T) {
			app := minimalApplication()

			expiredTime := metav1.NewTime(metav1.Now().Add(-1 * time.Minute))
			app.Status.SynchronizationSecretRotationTime = &expiredTime

			shouldUpdate := customresources.ShouldUpdateSecrets(app, time.Minute)
			assert.True(t, shouldUpdate)
		})
	})

	t.Run("Application with changed secret name", func(t *testing.T) {
		t.Run("not set synchronizedsecretname should update secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Spec.SecretName = "some-secret"
			app.Status.SynchronizationSecretName = ""

			shouldUpdate := customresources.ShouldUpdateSecrets(app, time.Minute)

			assert.True(t, shouldUpdate)
		})

		t.Run("not set rotation time should update secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Spec.SecretName = "some-secret"
			app.Status.SynchronizationSecretRotationTime = nil

			shouldUpdate := customresources.ShouldUpdateSecrets(app, time.Minute)
			assert.True(t, shouldUpdate)
		})

		t.Run("valid secret should update secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Spec.SecretName = "some-secret"
			shouldUpdate := customresources.ShouldUpdateSecrets(app, time.Minute)
			assert.True(t, shouldUpdate)
		})

		t.Run("expired secret should update secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Spec.SecretName = "some-secret"

			expiredTime := metav1.NewTime(metav1.Now().Add(-1 * time.Minute))
			app.Status.SynchronizationSecretRotationTime = &expiredTime

			shouldUpdate := customresources.ShouldUpdateSecrets(app, time.Minute)
			assert.True(t, shouldUpdate)
		})
	})
}

func TestShouldRotateSecrets(t *testing.T) {
	t.Run("Application with unchanged secret name", func(t *testing.T) {
		t.Run("not set rotation time should not rotate secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Status.SynchronizationSecretRotationTime = nil

			shouldUpdate := customresources.ShouldRotateSecrets(app, time.Minute)
			assert.False(t, shouldUpdate)
		})

		t.Run("not set synchronizedsecretname should not rotate secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Status.SynchronizationSecretName = ""

			shouldUpdate := customresources.ShouldRotateSecrets(app, time.Minute)

			assert.False(t, shouldUpdate)
		})

		t.Run("valid secret should not rotate secrets", func(t *testing.T) {
			app := minimalApplication()
			shouldUpdate := customresources.ShouldRotateSecrets(app, time.Minute)

			assert.False(t, shouldUpdate)
		})

		t.Run("expired secret should not rotate secrets", func(t *testing.T) {
			app := minimalApplication()

			expiredTime := metav1.NewTime(metav1.Now().Add(-1 * time.Minute))
			app.Status.SynchronizationSecretRotationTime = &expiredTime

			shouldUpdate := customresources.ShouldRotateSecrets(app, time.Minute)
			assert.False(t, shouldUpdate)
		})
	})

	t.Run("Application with changed secret name", func(t *testing.T) {
		t.Run("not set rotation time should rotate secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Spec.SecretName = "some-secret"
			app.Status.SynchronizationSecretRotationTime = nil

			shouldUpdate := customresources.ShouldRotateSecrets(app, time.Minute)
			assert.True(t, shouldUpdate)
		})

		t.Run("not set synchronizedsecretname should not rotate secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Spec.SecretName = "some-secret"
			app.Status.SynchronizationSecretName = ""

			shouldUpdate := customresources.ShouldRotateSecrets(app, time.Minute)

			assert.False(t, shouldUpdate)
		})

		t.Run("valid secret should not rotate secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Spec.SecretName = "some-secret"
			shouldUpdate := customresources.ShouldRotateSecrets(app, time.Minute)
			assert.False(t, shouldUpdate)
		})

		t.Run("expired secret should rotate secrets", func(t *testing.T) {
			app := minimalApplication()
			app.Spec.SecretName = "some-secret"

			expiredTime := metav1.NewTime(metav1.Now().Add(-1 * time.Minute))
			app.Status.SynchronizationSecretRotationTime = &expiredTime

			shouldUpdate := customresources.ShouldRotateSecrets(app, time.Minute)
			assert.True(t, shouldUpdate)
		})
	})
}

func minimalApplication() *nais_io_v1.AzureAdApplication {
	now := metav1.Now()
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
			PasswordKeyIds:                    []string{"test"},
			CertificateKeyIds:                 []string{"test"},
			ClientId:                          "test",
			ObjectId:                          "test",
			ServicePrincipalId:                "test",
			SynchronizationHash:               "3b810bb8df7a4bf1",
			SynchronizationSecretName:         "test",
			SynchronizationSecretRotationTime: &now,
		},
	}
}

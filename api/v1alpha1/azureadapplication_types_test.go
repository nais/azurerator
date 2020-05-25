package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var finalizerName = "test-finalizer"

func TestAzureAdApplication_GetUniqueName(t *testing.T) {
	expected := "test-cluster:test-namespace:test-app"
	assert.Equal(t, expected, minimalApplication().GetUniqueName())
}

func TestAzureAdApplication_HasFinalizer(t *testing.T) {
	t.Run("Minimal Application should not have finalizer", func(t *testing.T) {
		assert.False(t, minimalApplication().HasFinalizer(finalizerName))
	})
	t.Run("Application with finalizer should have finalizer", func(t *testing.T) {
		app := minimalApplication()
		app.ObjectMeta.Finalizers = []string{finalizerName}
		assert.True(t, app.HasFinalizer(finalizerName))
	})
}

func TestAzureAdApplication_AddFinalizer(t *testing.T) {
	app := minimalApplication()
	t.Run("Minimal Application should not have finalizer", func(t *testing.T) {
		assert.False(t, app.HasFinalizer(finalizerName))
	})
	t.Run("Application should have finalizer after add", func(t *testing.T) {
		app.AddFinalizer(finalizerName)
		assert.True(t, app.HasFinalizer(finalizerName))
	})
}

func TestAzureAdApplication_RemoveFinalizer(t *testing.T) {
	app := minimalApplication()
	app.ObjectMeta.Finalizers = []string{finalizerName}
	t.Run("Minimal Application should have finalizer", func(t *testing.T) {
		assert.True(t, app.HasFinalizer(finalizerName))
	})
	t.Run("Application should not have finalizer after remove", func(t *testing.T) {
		app.RemoveFinalizer(finalizerName)
		actual := app.HasFinalizer(finalizerName)
		assert.False(t, actual)
	})
}

func TestAzureAdApplication_IsBeingDeleted(t *testing.T) {
	t.Run("Minimal Application without deletion marker should not be marked for deletion", func(t *testing.T) {
		assert.False(t, minimalApplication().IsBeingDeleted())
	})
	t.Run("Application with deletion marker should be marked for deletion", func(t *testing.T) {
		app := minimalApplication()
		now := metav1.Now()
		app.ObjectMeta.DeletionTimestamp = &now
		assert.True(t, app.IsBeingDeleted())
	})
}

func TestAzureAdApplication_Hash(t *testing.T) {
	actual, err := minimalApplication().Hash()
	assert.NoError(t, err)
	assert.Equal(t, "100306fda4b3e77", actual)
}

func TestAzureAdApplication_HashUnchanged(t *testing.T) {
	t.Run("Minimal Application should have unchanged hash value", func(t *testing.T) {
		actual, err := minimalApplication().HashUnchanged()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed value should have changed hash value", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.LogoutUrl = "changed"
		actual, err := app.HashUnchanged()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestAzureAdApplication_UpdateHash(t *testing.T) {
	app := minimalApplication()
	app.Spec.LogoutUrl = "changed"

	err := app.UpdateHash()
	assert.NoError(t, err)
	assert.Equal(t, "2bf0d1b52d35c54d", app.Status.ProvisionHash)
}

func TestAzureAdApplication_IsUpToDate(t *testing.T) {
	t.Run("Minimal Application should not be up to date", func(t *testing.T) {
		actual, err := minimalApplication().IsUpToDate()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
	t.Run("Application should not be up to date", func(t *testing.T) {
		app := minimalApplication()
		app.Status.UpToDate = false
		actual, err := app.IsUpToDate()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
	t.Run("Application should be up to date", func(t *testing.T) {
		app := minimalApplication()
		app.Status.UpToDate = true
		actual, err := app.IsUpToDate()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
}

func TestAzureAdApplication_SetStatuses(t *testing.T) {
	app := minimalApplication()

	t.Run("Set Status to New", func(t *testing.T) {
		app.SetStatusNew()
		assert.NotEmpty(t, app.Status.ProvisionStateTime)
		assert.False(t, app.Status.UpToDate)
		assert.Equal(t, New, app.Status.ProvisionState)
	})

	t.Run("Set Status to Retrying", func(t *testing.T) {
		app.SetStatusRetrying()
		assert.NotEmpty(t, app.Status.ProvisionStateTime)
		assert.False(t, app.Status.UpToDate)
		assert.Equal(t, Retrying, app.Status.ProvisionState)
	})

	t.Run("Set Status to Rotate", func(t *testing.T) {
		app.SetStatusRotate()
		assert.NotEmpty(t, app.Status.ProvisionStateTime)
		assert.False(t, app.Status.UpToDate)
		assert.Equal(t, Rotate, app.Status.ProvisionState)
	})

	t.Run("Set Status to Provisioned", func(t *testing.T) {
		app.SetStatusProvisioned()
		assert.NotEmpty(t, app.Status.ProvisionStateTime)
		assert.True(t, app.Status.UpToDate)
		assert.Equal(t, Provisioned, app.Status.ProvisionState)
	})
}

func minimalApplication() *AzureAdApplication {
	return &AzureAdApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: AzureAdApplicationSpec{
			ReplyUrls:                 nil,
			PreAuthorizedApplications: nil,
			LogoutUrl:                 "test",
			SecretName:                "test",
			ConfigMapName:             "test",
		},
		Status: AzureAdApplicationStatus{
			PasswordKeyId:      "test",
			CertificateKeyId:   "test",
			ClientId:           "test",
			ObjectId:           "test",
			ServicePrincipalId: "test",
			ProvisionHash:      "100306fda4b3e77",
		},
	}
}

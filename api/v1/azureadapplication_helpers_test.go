package v1

import (
	"testing"

	"github.com/nais/azureator/pkg/annotations"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var finalizerName = "test-finalizer"

const expectedHash = "3b810bb8df7a4bf1"

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
	assert.Equal(t, expectedHash, actual)
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
	assert.Equal(t, "9f11b10559a0c2ac", app.Status.SynchronizationHash)
}

func TestAzureAdApplication_IsUpToDate(t *testing.T) {
	t.Run("Application with unchanged spec should be synchronized", func(t *testing.T) {
		app := minimalApplication()
		actual, err := app.IsUpToDate()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed spec should not be synchronized", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.SecretName = "yolo"
		actual, err := app.IsUpToDate()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestAzureAdApplication_SetStatuses(t *testing.T) {
	app := minimalApplication()

	t.Run("Set SynchronizationState Status to true", func(t *testing.T) {
		app.SetSynchronized()
		assert.NotEmpty(t, app.Status.SynchronizationTime)
		assert.Equal(t, EventSynchronized, app.Status.SynchronizationState)
	})
}

func TestAzureAdApplication_SetSkipAnnotation(t *testing.T) {
	app := minimalApplication()

	t.Run("Minimal Application should not have skip annotation", func(t *testing.T) {
		_, exists := app.Annotations[annotations.SkipKey]
		assert.False(t, exists)
	})
	t.Run("Application should have skip annotation after add", func(t *testing.T) {
		app.SetSkipAnnotation()
		value, exists := app.Annotations[annotations.SkipKey]
		assert.True(t, exists)
		assert.Equal(t, value, annotations.SkipValue)
	})
}

func TestAzureAdApplication_ShouldUpdateSecrets(t *testing.T) {
	app := minimalApplication()

	t.Run("Minimal Application should not update secrets", func(t *testing.T) {
		assert.False(t, app.ShouldUpdateSecrets())
	})
	t.Run("Application should update secrets when SecretName has changed", func(t *testing.T) {
		app.Spec.SecretName = "changed"
		assert.True(t, app.ShouldUpdateSecrets())
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
		},
		Status: AzureAdApplicationStatus{
			PasswordKeyIds:            []string{"test"},
			CertificateKeyIds:         []string{"test"},
			ClientId:                  "test",
			ObjectId:                  "test",
			ServicePrincipalId:        "test",
			SynchronizationHash:       expectedHash,
			SynchronizationSecretName: "test",
		},
	}
}

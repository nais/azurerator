package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var finalizerName = "test-finalizer"

func TestAzureAdApplication_GetUniqueName(t *testing.T) {
	app := minimalApplication()
	actual := app.GetUniqueName()
	expected := "test-cluster:test-namespace:test-app"
	assert.Equal(t, expected, actual)
}

func TestAzureAdApplication_HasFinalizer(t *testing.T) {
	app := &AzureAdApplication{}
	actual := app.HasFinalizer(finalizerName)
	assert.False(t, actual)

	app.ObjectMeta.Finalizers = []string{finalizerName}
	actual = app.HasFinalizer(finalizerName)
	assert.True(t, actual)
}

func TestAzureAdApplication_AddFinalizer(t *testing.T) {
	app := &AzureAdApplication{}
	actual := app.HasFinalizer(finalizerName)
	assert.False(t, actual)

	app.AddFinalizer(finalizerName)
	actual = app.HasFinalizer(finalizerName)
	assert.True(t, actual)
}

func TestAzureAdApplication_RemoveFinalizer(t *testing.T) {
	app := minimalApplication()
	app.ObjectMeta.Finalizers = []string{finalizerName}
	app.RemoveFinalizer(finalizerName)

	actual := app.HasFinalizer(finalizerName)
	assert.False(t, actual)
}

func TestAzureAdApplication_IsBeingDeleted(t *testing.T) {
	app := minimalApplication()
	actual := app.IsBeingDeleted()
	assert.False(t, actual)

	now := metav1.Now()
	app.ObjectMeta.DeletionTimestamp = &now
	actual = app.IsBeingDeleted()
	assert.True(t, actual)
}

func TestAzureAdApplication_Hash(t *testing.T) {
	app := minimalApplication()
	actual, _ := app.Hash()
	assert.Equal(t, "100306fda4b3e77", actual)
}

func TestAzureAdApplication_HashUnchanged(t *testing.T) {
	app := minimalApplication()
	actual, _ := app.HashUnchanged()
	assert.True(t, actual)

	app.Spec.LogoutUrl = "changed"
	actual, _ = app.HashUnchanged()
	assert.False(t, actual)
}

func TestAzureAdApplication_UpdateHash(t *testing.T) {
	app := minimalApplication()
	app.Spec.LogoutUrl = "changed"

	_ = app.UpdateHash()
	assert.Equal(t, "2bf0d1b52d35c54d", app.Status.ProvisionHash)
}

func TestAzureAdApplication_IsUpToDate(t *testing.T) {
	app := minimalApplication()
	actual, _ := app.IsUpToDate()
	app.Status.UpToDate = false
	assert.False(t, actual)

	app.Status.UpToDate = true
	actual, _ = app.IsUpToDate()
	assert.True(t, actual)
}

func TestAzureAdApplication_SetStatusNew(t *testing.T) {
	app := &AzureAdApplication{}
	app.SetStatusNew()

	assert.NotEmpty(t, app.Status.ProvisionStateTime)
	assert.False(t, app.Status.UpToDate)
	assert.Equal(t, New, app.Status.ProvisionState)
}

func TestAzureAdApplication_SetStatusRetrying(t *testing.T) {
	app := &AzureAdApplication{}
	app.SetStatusRetrying()

	assert.NotEmpty(t, app.Status.ProvisionStateTime)
	assert.False(t, app.Status.UpToDate)
	assert.Equal(t, Retrying, app.Status.ProvisionState)
}

func TestAzureAdApplication_SetStatusRotate(t *testing.T) {
	app := &AzureAdApplication{}
	app.SetStatusRotate()

	assert.NotEmpty(t, app.Status.ProvisionStateTime)
	assert.False(t, app.Status.UpToDate)
	assert.Equal(t, Rotate, app.Status.ProvisionState)
}

func TestAzureAdApplication_SetStatusProvisioned(t *testing.T) {
	app := &AzureAdApplication{}
	app.SetStatusProvisioned()

	assert.NotEmpty(t, app.Status.ProvisionStateTime)
	assert.True(t, app.Status.UpToDate)
	assert.Equal(t, Provisioned, app.Status.ProvisionState)
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

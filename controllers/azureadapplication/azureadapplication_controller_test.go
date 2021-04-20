package azureadapplication_test

import (
	"context"
	"fmt"
	controller "github.com/nais/azureator/controllers/azureadapplication"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/finalizers"
	"github.com/nais/liberator/pkg/crd"
	"github.com/nais/liberator/pkg/finalizer"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/nais/azureator/pkg/azure/fake"
	"github.com/nais/azureator/pkg/fixtures"
	"github.com/nais/azureator/pkg/labels"
	"github.com/nais/azureator/pkg/secrets"
	"github.com/nais/azureator/pkg/util/test"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	timeout      = 10 * time.Second
	interval     = 100 * time.Millisecond
	maxSecretAge = 1 * time.Hour

	alreadyInUseSecret = "in-use-by-pod"
	unusedSecret       = "unused-secret"
	newSecret          = "new-secret"

	namespace = "default"
)

var cli client.Client
var azureClient = fake.NewFakeAzureClient()
var secretDataKeys = secrets.NewSecretDataKeys()

func TestMain(m *testing.M) {
	testEnv, err := setup()
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = testEnv.Stop()
	os.Exit(code)
}

func TestReconciler_CreateAzureAdApplication(t *testing.T) {
	cases := []struct {
		name    string
		appName string
	}{
		{
			"Application already exists in Azure AD",
			fake.ApplicationExists,
		},
		{
			"Application does not exist in Azure AD",
			fake.ApplicationNotExistsName,
		},
	}
	for _, c := range cases {
		// prefix app name to secret name for "unique" secret in this test environment
		secretName := fmt.Sprintf("%s-%s", c.appName, alreadyInUseSecret)

		// set up preconditions for cluster
		clusterFixtures := fixtures.New(cli, fixtures.Config{
			AzureAppName:     c.appName,
			SecretName:       secretName,
			UnusedSecretName: unusedSecret,
			NamespaceName:    namespace,
		}).WithMinimalConfig()

		if err := clusterFixtures.Setup(); err != nil {
			t.Fatalf("failed to set up cluster fixtures: %v", err)
		}

		t.Run(c.name, func(t *testing.T) {
			instance := assertApplicationExists(t, c.appName)
			assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

			assertSecretExists(t, secretName, instance)

			t.Run("Unused Secret should not exist", func(t *testing.T) {
				key := client.ObjectKey{
					Namespace: namespace,
					Name:      unusedSecret,
				}
				a := &corev1.Secret{}
				assert.Eventually(t, resourceDoesNotExist(key, a), timeout, interval, "Secret should not exist")
			})
		})
	}
}

func TestReconciler_CreateAzureAdApplication_ShouldNotProcessInSharedNamespace(t *testing.T) {
	appName := "should-not-process-shared-namespace"
	sharedNamespace := "shared"
	secretName := fmt.Sprintf("%s-%s", appName, alreadyInUseSecret)
	clusterFixtures := fixtures.New(cli, fixtures.Config{
		AzureAppName:     appName,
		SecretName:       secretName,
		UnusedSecretName: unusedSecret,
		NamespaceName:    sharedNamespace,
	}).WithMinimalConfig().WithSharedNamespace()

	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}
	key := client.ObjectKey{
		Name:      appName,
		Namespace: sharedNamespace,
	}
	instance := assertApplicationShouldNotProcess(t, "AzureAdApplication in shared namespace should not be processed", key)
	assert.True(t, finalizer.HasFinalizer(instance, finalizers.Name), "AzureAdApplication should contain a finalizer")
	assert.Equal(t, v1.EventNotInTeamNamespace, instance.Status.SynchronizationState, "AzureAdApplication should be skipped")
	assertAnnotationExists(t, instance, annotations.NotInTeamNamespaceKey, strconv.FormatBool(true))
}

func TestReconciler_CreateAzureAdApplication_ShouldNotProcessNonMatchingTenantAnnotation(t *testing.T) {
	appName := "should-not-process-non-matching-tenant-annotation"
	secretName := fmt.Sprintf("%s-%s", appName, alreadyInUseSecret)
	tenant := "some-tenant"
	clusterFixtures := fixtures.New(cli, fixtures.Config{
		AzureAppName:     appName,
		SecretName:       secretName,
		UnusedSecretName: unusedSecret,
		NamespaceName:    namespace,
	}).WithMinimalConfig().WithTenant(tenant)

	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}
	key := client.ObjectKey{
		Name:      appName,
		Namespace: namespace,
	}
	instance := assertApplicationShouldNotProcess(t, "AzureAdApplication with tenant should not be processed", key)
	assert.Empty(t, instance.Status.SynchronizationState, "AzureAdApplication should not be processed")
	assert.False(t, finalizer.HasFinalizer(instance, finalizers.Name), "AzureAdApplication should not contain a finalizer")
}

func TestReconciler_UpdateAzureAdApplication_InvalidPreAuthorizedApps_ShouldNotRetry(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)

	previousHash := instance.Status.SynchronizationHash
	previousSyncTime := instance.Status.SynchronizationTime
	previousPreAuthorizedApps := instance.Spec.PreAuthorizedApplications

	invalidPreAuthorizedApp := v1.AccessPolicyRule{
		Application: "invalid-app",
		Namespace:   "some-namespace",
		Cluster:     "some-cluster",
	}
	validPreAuthorizedApp := v1.AccessPolicyRule{
		Application: "valid-app",
		Namespace:   "some-namespace",
		Cluster:     "some-cluster",
	}
	instance.Spec.PreAuthorizedApplications = append(previousPreAuthorizedApps, invalidPreAuthorizedApp, validPreAuthorizedApp)

	err := cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, instance.GetName())

	assert.Eventually(t, func() bool {
		return newInstance.Status.SynchronizationTime.After(previousSyncTime.Time)
	}, timeout, interval, "Synchronization Time is updated")
	assert.Eventually(t, func() bool {
		return newInstance.Status.SynchronizationHash != previousHash
	}, timeout, interval, "Synchronization Hash is changed")

	newInstance = assertApplicationExists(t, newInstance.GetName())

	assert.NotContains(t, newInstance.Annotations, annotations.ResynchronizeKey, "AzureAdApplication should not contain resync annotation")

	// reset pre-authorized applications to only contain valid applications
	newInstance.Spec.PreAuthorizedApplications = append(previousPreAuthorizedApps, validPreAuthorizedApp)
	previousHash = newInstance.Status.SynchronizationHash
	previousSyncTime = newInstance.Status.SynchronizationTime

	// sleep to ensure synchronization time is actually updated
	time.Sleep(time.Second)

	err = cli.Update(context.Background(), newInstance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance = assertApplicationExists(t, newInstance.GetName(), v1.EventSynchronized)

	assert.Eventually(t, func() bool {
		return newInstance.Status.SynchronizationTime.After(previousSyncTime.Time)
	}, timeout, interval, "Synchronization Time is updated")
	assert.Eventually(t, func() bool {
		return newInstance.Status.SynchronizationHash != previousHash
	}, timeout, interval, "Synchronization Hash is changed")
}

func TestReconciler_UpdateAzureAdApplication_ResyncAnnotation_ShouldResyncAndNotModifySecrets(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)

	previousHash := instance.Status.SynchronizationHash
	previousSyncTime := instance.Status.SynchronizationTime
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousSecret := assertSecretExists(t, instance.Spec.SecretName, instance)

	annotations.SetAnnotation(instance, annotations.ResynchronizeKey, strconv.FormatBool(true))

	// sleep to ensure synchronization time is actually updated
	time.Sleep(time.Second)

	err := cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, instance.GetName())

	assert.Eventually(t, func() bool {
		return newInstance.Status.SynchronizationTime.After(previousSyncTime.Time)
	}, timeout, interval, "Synchronization Time is updated")
	assert.Eventually(t, func() bool {
		return newInstance.Status.SynchronizationHash == previousHash
	}, timeout, interval, "Synchronization Hash is unchanged")
	assert.Eventually(t, func() bool {
		return previousSecretRotationTime.Equal(newInstance.Status.SynchronizationSecretRotationTime)
	}, timeout, interval, "Secret Rotation Time is unchanged")

	assertApplicationExists(t, newInstance.GetName())
	assert.NotContains(t, newInstance.Annotations, annotations.ResynchronizeKey, "AzureAdApplication should not contain resync annotation")

	newSecret := assertSecretExists(t, instance.Spec.SecretName, instance)
	assert.EqualValues(t, previousSecret, newSecret, "Secrets are unchanged")
}

func TestReconciler_UpdateAzureAdApplication_NewSecretName_ShouldRotateCredentials(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousHash := instance.Status.SynchronizationHash
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime

	previousSecret := assertSecretExists(t, previousSecretName, instance)

	newSecretName := fmt.Sprintf("%s-%s-new-secret-name", instance.GetName(), newSecret)
	instance.Spec.SecretName = newSecretName

	err := cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, instance.GetName())

	assert.Eventually(t, func() bool {
		return previousHash == newInstance.Status.SynchronizationHash
	}, timeout, interval, "Synchronization Hash should be unchanged")
	assert.Eventually(t, func() bool {
		return !previousSecretRotationTime.Equal(newInstance.Status.SynchronizationSecretRotationTime)
	}, timeout, interval, "Secret rotation time is changed")
	assert.Eventually(t, func() bool {
		return (newInstance.Status.SynchronizationSecretName != previousSecretName) && (newInstance.Status.SynchronizationSecretName == newSecretName)
	}, timeout, interval, "Secret name is changed")

	assert.NotEmpty(t, newInstance.Status.SynchronizationSecretRotationTime)

	newSecret := assertSecretExists(t, newSecretName, instance)

	assertSecretsAreRotated(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_SpecChangeAndNotExpiredSecret_ShouldNotRotateCredentials(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousHash := instance.Status.SynchronizationHash
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousLogoutUrl := instance.Spec.LogoutUrl

	previousSecret := assertSecretExists(t, previousSecretName, instance)

	// only update spec, secretName is unchanged
	instance.Spec.LogoutUrl = "https://some-url/logout"
	err := cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, instance.GetName())

	assert.Eventually(t, func() bool {
		return previousLogoutUrl != newInstance.Spec.LogoutUrl
	}, timeout, interval, "Logout URL is updated")
	assert.Eventually(t, func() bool {
		return previousHash != newInstance.Status.SynchronizationHash
	}, timeout, interval, "Synchronization hash is changed")
	assert.Eventually(t, func() bool {
		return previousSecretRotationTime.Equal(newInstance.Status.SynchronizationSecretRotationTime)
	}, timeout, interval, "Secret Rotation Time is unchaned")
	assert.Eventually(t, func() bool {
		return newInstance.Status.SynchronizationSecretName == previousSecretName
	}, timeout, interval, "Synchronization Secret Name is unchanged")

	assert.NotEmpty(t, newInstance.Status.SynchronizationSecretRotationTime)

	newSecret := assertSecretExists(t, instance.Spec.SecretName, instance)
	assertSecretsAreNotRotated(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_SpecChangeAndExpiredSecret_ShouldAddNewCredentials(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousHash := instance.Status.SynchronizationHash
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime

	previousSecret := assertSecretExists(t, previousSecretName, instance)

	// set last rotation time to (previous - maxSecretAge) to trigger rotation
	expiredTime := metav1.NewTime(previousSecretRotationTime.Add(-1 * maxSecretAge))
	instance.Status.SynchronizationSecretRotationTime = &expiredTime

	err := cli.Status().Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application status subresource should not return error")

	// only update spec, secretName is unchanged
	instance.Spec.LogoutUrl = "https://some-other-url/logout"
	err = cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, instance.GetName())

	assert.Eventually(t, func() bool {
		return previousHash != newInstance.Status.SynchronizationHash
	}, timeout, interval, "Synchronization Hash is changed")
	assert.Eventually(t, func() bool {
		return previousSecretName == newInstance.Status.SynchronizationSecretName
	}, timeout, interval, "Synchronization Secret Name is unchanged")
	assert.Eventually(t, func() bool {
		return previousSecretRotationTime.Before(newInstance.Status.SynchronizationSecretRotationTime)
	}, timeout, interval, "Secret Rotation Time is updated")
	assert.Eventually(t, func() bool {
		return len(newInstance.Status.PasswordKeyIds) == 2
	}, timeout, interval, "Password Key IDs are updated")
	assert.Eventually(t, func() bool {
		return len(newInstance.Status.CertificateKeyIds) == 2
	}, timeout, interval, "Certificate Key IDs are updated")

	assert.NotEmpty(t, newInstance.Status.SynchronizationSecretRotationTime)

	newSecret := assertSecretExists(t, previousSecretName, instance)
	assertSecretsAreAdded(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_NewSecretNameAndExpired_ShouldAddNewCredentials(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousHash := instance.Status.SynchronizationHash
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime

	previousSecret := assertSecretExists(t, previousSecretName, instance)

	// set last rotation time to (previous - maxSecretAge) to trigger rotation
	expiredTime := metav1.NewTime(previousSecretRotationTime.Add(-1 * maxSecretAge))
	instance.Status.SynchronizationSecretRotationTime = &expiredTime

	err := cli.Status().Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application status subresource should not return error")

	// only update spec with new secret name, other fields are unchanged
	newSecretName := fmt.Sprintf("%s-%s-new-expired", instance.GetName(), newSecret)
	instance.Spec.SecretName = newSecretName

	err = cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, instance.GetName())

	assert.Eventually(t, func() bool {
		return previousHash == newInstance.Status.SynchronizationHash
	}, timeout, interval, "Synchronization hash is unchanged")
	assert.Eventually(t, func() bool {
		return previousSecretName != newInstance.Status.SynchronizationSecretName
	}, timeout, interval, "Synchronization Secret Name is changed")
	assert.Eventually(t, func() bool {
		return previousSecretRotationTime.Before(newInstance.Status.SynchronizationSecretRotationTime)
	}, timeout, interval, "Secret Rotation Time is updated")
	assert.Eventually(t, func() bool {
		return len(newInstance.Status.PasswordKeyIds) == 2
	}, timeout, interval, "Password Key IDs are updated")
	assert.Eventually(t, func() bool {
		return len(newInstance.Status.CertificateKeyIds) == 2
	}, timeout, interval, "Certificate Key IDs are updated")

	assert.NotEmpty(t, newInstance.Status.SynchronizationSecretRotationTime)

	newSecret := assertSecretExists(t, newSecretName, instance)

	assertSecretsAreAdded(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_MissingSecretRotationTimeAndNewSecretName_ShouldRotateCredentials(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousHash := instance.Status.SynchronizationHash
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime

	previousSecret := assertSecretExists(t, previousSecretName, instance)

	instance.Status.SynchronizationSecretRotationTime = nil

	err := cli.Status().Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application status subresource should not return error")

	newSecretName := fmt.Sprintf("%s-%s-missing-secret-rotation-time", instance.GetName(), newSecret)
	instance.Spec.SecretName = newSecretName

	err = cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, instance.GetName())

	assert.Eventually(t, func() bool {
		return previousHash == newInstance.Status.SynchronizationHash
	}, timeout, interval, "Synchronization hash is unchanged")
	assert.Eventually(t, func() bool {
		return previousSecretRotationTime.Before(newInstance.Status.SynchronizationSecretRotationTime)
	}, timeout, interval, "Secret Rotation Time is updated")
	assert.Eventually(t, func() bool {
		return (newInstance.Status.SynchronizationSecretName != previousSecretName) && (newInstance.Status.SynchronizationSecretName == newSecretName)
	}, timeout, interval, "Synchronization Secret Name is updated")

	assert.NotEmpty(t, newInstance.Status.SynchronizationSecretRotationTime)

	newSecret := assertSecretExists(t, newSecretName, instance)

	assertSecretsAreRotated(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_MissingSecretRotationTime_ShouldNotRotateCredentials(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousHash := instance.Status.SynchronizationHash
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousLogoutUrl := instance.Spec.LogoutUrl

	previousSecret := assertSecretExists(t, previousSecretName, instance)

	instance.Status.SynchronizationSecretRotationTime = nil

	err := cli.Status().Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	instance.Spec.LogoutUrl = "some-changed-value"

	err = cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, instance.GetName())
	assert.Empty(t, instance.Status.SynchronizationSecretRotationTime)

	assert.Eventually(t, func() bool {
		return previousLogoutUrl != newInstance.Spec.LogoutUrl
	}, timeout, interval, "Logout URL is changed")
	assert.Eventually(t, func() bool {
		return previousHash != newInstance.Status.SynchronizationHash
	}, timeout, interval, "Synchronization Hash is changed")
	assert.Eventually(t, func() bool {
		return !previousSecretRotationTime.Equal(newInstance.Status.SynchronizationSecretRotationTime)
	}, timeout, interval, "Secret Rotation Time is set")
	assert.Eventually(t, func() bool {
		return newInstance.Status.SynchronizationSecretName == previousSecretName
	}, timeout, interval, "Synchronization Secret Name is unchanged")

	assert.Empty(t, newInstance.Status.SynchronizationSecretRotationTime)

	newSecret := assertSecretExists(t, instance.Spec.SecretName, instance)

	assertSecretsAreNotRotated(t, previousSecret, newSecret)
}

func TestReconciler_DeleteAzureAdApplication(t *testing.T) {
	instance := assertApplicationExists(t, fake.ApplicationExists)

	t.Run("Delete existing AzureAdApplication", func(t *testing.T) {
		err := cli.Delete(context.Background(), instance)
		assert.NoError(t, err, "deleting existing AzureAdApplication should not return error")

		key := client.ObjectKey{
			Name:      fake.ApplicationExists,
			Namespace: namespace,
		}
		assert.Eventually(t, resourceDoesNotExist(key, instance), timeout, interval)
	})
}

// asserts that the application exists in the cluster and is valid
func assertApplicationExists(t *testing.T, name string, state ...string) *v1.AzureAdApplication {
	instance := &v1.AzureAdApplication{}
	key := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	assert.Eventually(t, resourceExists(key, instance), timeout, interval, "AzureAdApplication should exist")

	assert.Eventually(t, func() bool {
		err := cli.Get(context.Background(), key, instance)
		assert.NoError(t, err)

		isHashChanged, err := customresources.IsHashChanged(instance)
		assert.NoError(t, err)

		hasExpiredSecrets := customresources.HasExpiredSecrets(instance, maxSecretAge)
		secretNameChanged := customresources.SecretNameChanged(instance)
		return !isHashChanged && !hasExpiredSecrets && !secretNameChanged
	}, timeout, interval, "AzureAdApplication should be synchronized")

	assert.True(t, finalizer.HasFinalizer(instance, finalizers.Name), "AzureAdApplication should contain a finalizer")

	assert.Empty(t, instance.Annotations[annotations.NotInTeamNamespaceKey], "AzureAdApplication should not contain skip annotation")

	test.AssertAllNotEmpty(t, []interface{}{
		instance.Status.CertificateKeyIds,
		instance.GetClientId(),
		instance.Status.CorrelationId,
		instance.GetObjectId(),
		instance.Status.PasswordKeyIds,
		instance.Status.SynchronizationHash,
		instance.GetServicePrincipalId(),
		instance.Status.SynchronizationTime,
		instance.Status.SynchronizationTenant,
		instance.Status.SynchronizationSecretName,
	})

	if len(state) == 0 {
		assert.Equal(t, v1.EventSynchronized, instance.Status.SynchronizationState, "AzureAdApplication should be synchronized")
	} else {
		assert.Equal(t, state[0], instance.Status.SynchronizationState, fmt.Sprintf("AzureAdApplication should be %s", state[0]))
	}
	return instance
}

func assertApplicationShouldNotProcess(t *testing.T, testName string, key client.ObjectKey) *v1.AzureAdApplication {
	instance := &v1.AzureAdApplication{}
	t.Run(testName, func(t *testing.T) {
		assert.Eventually(t, resourceExists(key, instance), timeout, interval, "AzureAdApplication should exist")
		assert.Empty(t, instance.Status.CertificateKeyIds)
		assert.Empty(t, instance.GetClientId())
		assert.Empty(t, instance.GetObjectId())
		assert.Empty(t, instance.Status.PasswordKeyIds)
		assert.Empty(t, instance.Status.SynchronizationHash)
		assert.Empty(t, instance.GetServicePrincipalId())
		assert.Empty(t, instance.Status.SynchronizationTime)
		assert.Empty(t, instance.Status.SynchronizationTenant)
	})
	return instance
}

func assertAnnotationExists(t *testing.T, instance *v1.AzureAdApplication, annotationKey, annotationValue string) {
	assert.Eventually(t, func() bool {
		_, key := instance.Annotations[annotationKey]
		return key
	}, timeout, interval, fmt.Sprintf("Annotation '%s' should exist on resource", annotationKey))
	assert.Equal(t, instance.Annotations[annotationKey], annotationValue, fmt.Sprintf("AzureAdApplication should contain annotation %s", annotationKey))
}

func assertSecretExists(t *testing.T, name string, instance *v1.AzureAdApplication) *corev1.Secret {
	secret := &corev1.Secret{}

	t.Run(fmt.Sprintf("Secret '%s' should exist", name), func(t *testing.T) {
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}

		assert.Eventually(t, resourceExists(key, secret), timeout, interval, "Secret should exist")

		assert.True(t, containsOwnerRef(secret.GetOwnerReferences(), *instance), "Secret should contain ownerReference")

		actualLabels := secret.GetLabels()
		expectedLabels := map[string]string{
			labels.AppLabelKey:  instance.GetName(),
			labels.TypeLabelKey: labels.TypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")

		assert.Equal(t, corev1.SecretTypeOpaque, secret.Type, "Secret type should be Opaque")

		test.AssertContainsKeysWithNonEmptyValues(t, secret.Data, secretDataKeys.AllKeys())

		azureOpenIdConfig := fake.AzureOpenIdConfig()
		assert.Equal(t, azureOpenIdConfig.WellKnownEndpoint, string(secret.Data[secretDataKeys.WellKnownUrl]))
		assert.Equal(t, azureOpenIdConfig.Issuer, string(secret.Data[secretDataKeys.OpenId.Issuer]))
		assert.Equal(t, azureOpenIdConfig.JwksURI, string(secret.Data[secretDataKeys.OpenId.JwksUri]))
		assert.Equal(t, azureOpenIdConfig.TokenEndpoint, string(secret.Data[secretDataKeys.OpenId.TokenEndpoint]))

	})

	return secret
}

var relevantSecretValues = []string{
	secretDataKeys.CurrentCredentials.CertificateKeyId,
	secretDataKeys.CurrentCredentials.ClientSecret,
	secretDataKeys.CurrentCredentials.Jwks,
	secretDataKeys.CurrentCredentials.Jwk,
	secretDataKeys.CurrentCredentials.PasswordKeyId,
	secretDataKeys.NextCredentials.CertificateKeyId,
	secretDataKeys.NextCredentials.ClientSecret,
	secretDataKeys.NextCredentials.Jwk,
	secretDataKeys.NextCredentials.PasswordKeyId,
}

func assertSecretsAreAdded(t *testing.T, previous *corev1.Secret, new *corev1.Secret) {
	for _, key := range relevantSecretValues {
		assert.NotEqual(t, previous.Data[key], new.Data[key], fmt.Sprintf("%s", key))
	}
}

func assertSecretsAreRotated(t *testing.T, previous *corev1.Secret, new *corev1.Secret) {
	for _, key := range relevantSecretValues {
		assert.NotEqual(t, previous.Data[key], new.Data[key], fmt.Sprintf("%s", key))
	}
	assert.Equal(t, previous.Data[secretDataKeys.NextCredentials.CertificateKeyId], new.Data[secretDataKeys.CurrentCredentials.CertificateKeyId])
	assert.Equal(t, previous.Data[secretDataKeys.NextCredentials.ClientSecret], new.Data[secretDataKeys.CurrentCredentials.ClientSecret])
	assert.Equal(t, previous.Data[secretDataKeys.NextCredentials.Jwk], new.Data[secretDataKeys.CurrentCredentials.Jwk])
	assert.Equal(t, previous.Data[secretDataKeys.NextCredentials.PasswordKeyId], new.Data[secretDataKeys.CurrentCredentials.PasswordKeyId])
}

func assertSecretsAreNotRotated(t *testing.T, previous *corev1.Secret, new *corev1.Secret) {
	for _, key := range relevantSecretValues {
		assert.Equal(t, previous.Data[key], new.Data[key], fmt.Sprintf("%s", key))
	}
}

func resourceExists(key client.ObjectKey, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return !errors.IsNotFound(err)
	}
}

func resourceDoesNotExist(key client.ObjectKey, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return errors.IsNotFound(err)
	}
}

func containsOwnerRef(refs []metav1.OwnerReference, owner v1.AzureAdApplication) bool {
	expected := metav1.OwnerReference{
		APIVersion: owner.APIVersion,
		Kind:       owner.Kind,
		Name:       owner.Name,
		UID:        owner.UID,
	}
	for _, ref := range refs {
		sameApiVersion := ref.APIVersion == expected.APIVersion
		sameKind := ref.Kind == expected.Kind
		sameName := ref.Name == expected.Name
		sameUID := ref.UID == expected.UID
		if sameApiVersion && sameKind && sameName && sameUID {
			return true
		}
	}
	return false
}

func setup() (*envtest.Environment, error) {
	logger := zap.New(zap.UseDevMode(true))
	ctrl.SetLogger(logger)
	log.SetLevel(log.DebugLevel)

	crdPath := crd.YamlDirectory()
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{crdPath},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return nil, err
	}

	err = v1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	// +kubebuilder:scaffold:scheme

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, err
	}

	cli = mgr.GetClient()

	azureratorCfg, err := config.New()
	if err != nil {
		return nil, err
	}
	azureratorCfg.SecretRotation.MaxAge = maxSecretAge

	azureOpenIDConfig := fake.AzureOpenIdConfig()

	err = (&controller.Reconciler{
		Client:            cli,
		Reader:            mgr.GetAPIReader(),
		Scheme:            mgr.GetScheme(),
		AzureClient:       azureClient,
		Recorder:          mgr.GetEventRecorderFor("azurerator"),
		Config:            azureratorCfg,
		AzureOpenIDConfig: azureOpenIDConfig,
	}).SetupWithManager(mgr)
	if err != nil {
		return nil, err
	}

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()

	return testEnv, nil
}

package azureadapplication_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/crd"
	"github.com/nais/liberator/pkg/events"
	"github.com/nais/liberator/pkg/finalizer"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	controller "github.com/nais/azureator/controllers/azureadapplication"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/azure/fake"
	az "github.com/nais/azureator/pkg/azure/fake/client"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/customresources"
	"github.com/nais/azureator/pkg/fixtures"
	"github.com/nais/azureator/pkg/labels"
	"github.com/nais/azureator/pkg/secrets"
	"github.com/nais/azureator/pkg/synchronizer"
	"github.com/nais/azureator/pkg/transaction/options"
	"github.com/nais/azureator/pkg/util/test"
)

const (
	timeout      = 10 * time.Second
	interval     = 100 * time.Millisecond
	maxSecretAge = 1 * time.Hour

	alreadyInUseSecret = "in-use-by-pod"
	unusedSecret       = "unused-secret"
	newSecret          = "new-secret"

	namespace = "aura"
)

var (
	cli            client.Client
	azureClient    = az.NewFakeAzureClient()
	secretDataKeys = secrets.NewSecretDataKeys()
)

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
			az.ApplicationExists,
		},
		{
			"Application does not exist in Azure AD",
			az.ApplicationNotExistsName,
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
	assert.False(t, finalizer.HasFinalizer(instance, options.FinalizerName), "AzureAdApplication should not contain a finalizer")
}

func TestReconciler_UpdateAzureAdApplication_InvalidPreAuthorizedApps_ShouldNotRetry(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)

	previousPreAuthorizedApps := instance.Spec.PreAuthorizedApplications
	invalidPreAuthorizedApp := v1.AccessPolicyInboundRule{AccessPolicyRule: v1.AccessPolicyRule{
		Application: "invalid-app",
		Namespace:   "some-namespace",
		Cluster:     "some-cluster",
	}}
	validPreAuthorizedApp := v1.AccessPolicyInboundRule{AccessPolicyRule: v1.AccessPolicyRule{
		Application: "valid-app",
		Namespace:   "some-namespace",
		Cluster:     "some-cluster",
	}}
	instance.Spec.PreAuthorizedApplications = append(previousPreAuthorizedApps, invalidPreAuthorizedApp, validPreAuthorizedApp)

	updatedInstance := updateApplication(t, instance, eventuallyHashUpdated(instance))
	assert.NotContains(t, updatedInstance.Annotations, annotations.ResynchronizeKey, "AzureAdApplication should not contain resync annotation")

	// reset pre-authorized applications to only contain valid applications
	updatedInstance.Spec.PreAuthorizedApplications = append(previousPreAuthorizedApps, validPreAuthorizedApp)
	updateApplication(t, updatedInstance, eventuallyHashUpdated(updatedInstance))
}

func TestReconciler_UpdateAzureAdApplication_ResyncAnnotation_ShouldResyncAndNotModifySecrets(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)

	previousHash := instance.Status.SynchronizationHash
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousSecret := assertSecretExists(t, instance.Spec.SecretName, instance)

	annotations.SetAnnotation(instance, annotations.ResynchronizeKey, strconv.FormatBool(true))
	updatedInstance := updateApplication(t, instance, func(updated *v1.AzureAdApplication) bool {
		_, ok := updated.Annotations[annotations.ResynchronizeKey]
		return syncTimeUpdated(instance, updated) && !ok
	})
	assert.Equal(t, events.Synchronized, updatedInstance.Status.SynchronizationState, "AzureAdApplication should be synchronized")
	assert.Equal(t, updatedInstance.Status.SynchronizationHash, previousHash, "Synchronization Hash is unchanged")
	assert.Equal(t, updatedInstance.Status.SynchronizationSecretRotationTime, previousSecretRotationTime, "Secret Rotation Time is unchanged")

	newSecret := assertSecretExists(t, updatedInstance.Spec.SecretName, instance)
	assertSecretsAreNotRotated(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_RotateAnnotation_ShouldRotateSecrets(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)

	previousHash := instance.Status.SynchronizationHash
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousSecret := assertSecretExists(t, instance.Spec.SecretName, instance)

	annotations.SetAnnotation(instance, annotations.RotateKey, strconv.FormatBool(true))
	updatedInstance := updateApplication(t, instance, func(updated *v1.AzureAdApplication) bool {
		_, ok := updated.Annotations[annotations.RotateKey]
		return syncTimeUpdated(instance, updated) && !ok
	})
	assert.Equal(t, events.Synchronized, updatedInstance.Status.SynchronizationState, "AzureAdApplication should be synchronized")
	assert.Equal(t, updatedInstance.Status.SynchronizationHash, previousHash, "Synchronization Hash is unchanged")
	assert.True(t, updatedInstance.Status.SynchronizationSecretRotationTime.After(previousSecretRotationTime.Time), "Secret Rotation Time is changed")

	newSecret := assertSecretExists(t, updatedInstance.Spec.SecretName, instance)
	assertSecretsAreRotated(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_NewSecretName_ShouldRotateCredentials(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousHash := instance.Status.SynchronizationHash
	previousSecret := assertSecretExists(t, previousSecretName, instance)

	newSecretName := fmt.Sprintf("%s-%s-new-secret-name", instance.GetName(), newSecret)
	instance.Spec.SecretName = newSecretName

	updatedInstance := updateApplication(t, instance, func(updated *v1.AzureAdApplication) bool {
		return syncTimeUpdated(instance, updated)
	})
	assert.Equal(t, events.Synchronized, updatedInstance.Status.SynchronizationState, "AzureAdApplication should be synchronized")
	assert.Equal(t, updatedInstance.Status.SynchronizationHash, previousHash, "Synchronization Hash is unchanged")
	assert.NotEmpty(t, updatedInstance.Status.SynchronizationSecretRotationTime)
	assert.True(t, updatedInstance.Status.SynchronizationSecretRotationTime.After(previousSecretRotationTime.Time), "Secret rotation time is changed")
	assert.NotEqual(t, updatedInstance.Status.SynchronizationSecretName, previousSecretName, "Secret name is changed")

	newSecret := assertSecretExists(t, newSecretName, updatedInstance)
	assertSecretsAreRotated(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_SpecChangeAndNotExpiredSecret_ShouldNotRotateCredentials(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousLogoutUrl := instance.Spec.LogoutUrl
	previousSecret := assertSecretExists(t, previousSecretName, instance)

	// only update spec, secretName is unchanged
	instance.Spec.LogoutUrl = "https://some-url/logout"
	updatedInstance := updateApplication(t, instance, eventuallyHashUpdated(instance))

	assert.NotEqual(t, previousLogoutUrl, updatedInstance.Spec.LogoutUrl, "Logout URL is updated")
	assert.NotEmpty(t, updatedInstance.Status.SynchronizationSecretRotationTime)
	assert.Equal(t, previousSecretRotationTime, updatedInstance.Status.SynchronizationSecretRotationTime, "Secret Rotation Time is unchanged")
	assert.Equal(t, previousSecretName, updatedInstance.Status.SynchronizationSecretName, "Synchronization Secret Name is unchanged")

	newSecret := assertSecretExists(t, instance.Spec.SecretName, instance)
	assertSecretsAreNotRotated(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_SpecChangeAndExpiredSecret_ShouldAddNewCredentials(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousSecret := assertSecretExists(t, previousSecretName, instance)

	// set last rotation time to (previous - maxSecretAge) to trigger rotation
	expiredTime := metav1.NewTime(previousSecretRotationTime.Add(-1 * maxSecretAge))
	instance.Status.SynchronizationSecretRotationTime = &expiredTime

	err := cli.Status().Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application status subresource should not return error")

	// only update spec, secretName is unchanged
	instance.Spec.LogoutUrl = "https://some-other-url/logout"
	updatedInstance := updateApplication(t, instance, eventuallyHashUpdated(instance))
	assert.Equal(t, previousSecretName, updatedInstance.Status.SynchronizationSecretName, "Synchronization Secret Name is unchanged")
	assert.NotEmpty(t, updatedInstance.Status.SynchronizationSecretRotationTime)
	assert.True(t, updatedInstance.Status.SynchronizationSecretRotationTime.After(previousSecretRotationTime.Time), "Secret Rotation Time is updated")
	assert.Len(t, updatedInstance.Status.PasswordKeyIds, 2, "Password Key IDs are updated")
	assert.Len(t, updatedInstance.Status.CertificateKeyIds, 2, "Certificate Key IDs are updated")

	newSecret := assertSecretExists(t, previousSecretName, instance)
	assertSecretsAreAdded(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_NewSecretNameAndExpired_ShouldAddNewCredentials(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)
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
	updatedInstance := updateApplication(t, instance, func(updated *v1.AzureAdApplication) bool {
		return syncTimeUpdated(instance, updated)
	})
	assert.Equal(t, previousHash, updatedInstance.Status.SynchronizationHash, "Synchronization Hash is unchanged")
	assert.NotEmpty(t, updatedInstance.Status.SynchronizationSecretRotationTime)
	assert.NotEqual(t, previousSecretName, updatedInstance.Status.SynchronizationSecretName, "Synchronization Secret Name is changed")
	assert.True(t, updatedInstance.Status.SynchronizationSecretRotationTime.After(previousSecretRotationTime.Time), "Secret Rotation Time is updated")
	assert.Len(t, updatedInstance.Status.PasswordKeyIds, 2, "Password Key IDs are updated")
	assert.Len(t, updatedInstance.Status.CertificateKeyIds, 2, "Certificate Key IDs are updated")

	newSecret := assertSecretExists(t, newSecretName, instance)
	assertSecretsAreAdded(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_MissingSecretRotationTimeAndNewSecretName_ShouldRotateCredentials(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)
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
	updatedInstance := updateApplication(t, instance, func(updated *v1.AzureAdApplication) bool {
		return syncTimeUpdated(instance, updated)
	})
	assert.Equal(t, previousHash, updatedInstance.Status.SynchronizationHash, "Synchronization Hash is unchanged")
	assert.NotEmpty(t, updatedInstance.Status.SynchronizationSecretRotationTime)
	assert.True(t, updatedInstance.Status.SynchronizationSecretRotationTime.After(previousSecretRotationTime.Time), "Secret Rotation Time is updated")
	assert.NotEqual(t, previousSecretName, updatedInstance.Status.SynchronizationSecretName, "Synchronization Secret Name is changed")
	assert.Equal(t, newSecretName, updatedInstance.Status.SynchronizationSecretName, "Synchronization Secret Name matches new secret name")

	newSecret := assertSecretExists(t, newSecretName, instance)
	assertSecretsAreRotated(t, previousSecret, newSecret)
}

func TestReconciler_UpdateAzureAdApplication_MissingSecretRotationTime_ShouldNotRotateCredentials(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)
	assert.NotEmpty(t, instance.Status.SynchronizationSecretRotationTime)

	previousSecretName := instance.Spec.SecretName
	previousSecretRotationTime := instance.Status.SynchronizationSecretRotationTime
	previousLogoutUrl := instance.Spec.LogoutUrl

	previousSecret := assertSecretExists(t, previousSecretName, instance)

	instance.Status.SynchronizationSecretRotationTime = nil
	err := cli.Status().Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	instance.Spec.LogoutUrl = "some-changed-value"
	updatedInstance := updateApplication(t, instance, eventuallyHashUpdated(instance))
	assert.NotEqual(t, previousLogoutUrl, updatedInstance.Spec.LogoutUrl, "Logout URL is changed")
	assert.Equal(t, previousSecretName, updatedInstance.Status.SynchronizationSecretName, "Synchronization Secret Name is unchanged")
	assert.Empty(t, updatedInstance.Status.SynchronizationSecretRotationTime)
	assert.False(t, previousSecretRotationTime.Equal(updatedInstance.Status.SynchronizationSecretRotationTime), "Secret Rotation Time is not set")

	newSecret := assertSecretExists(t, instance.Spec.SecretName, instance)
	assertSecretsAreNotRotated(t, previousSecret, newSecret)
}

func TestReconciler_DeleteAzureAdApplication(t *testing.T) {
	instance := assertApplicationExists(t, az.ApplicationExists)

	t.Run("Delete existing AzureAdApplication", func(t *testing.T) {
		err := cli.Delete(context.Background(), instance)
		assert.NoError(t, err, "deleting existing AzureAdApplication should not return error")

		key := client.ObjectKey{
			Name:      az.ApplicationExists,
			Namespace: namespace,
		}
		assert.Eventually(t, resourceDoesNotExist(key, instance), timeout, interval)
	})
}

// asserts that the application exists in the cluster and is valid
func assertApplicationExists(t *testing.T, name string) *v1.AzureAdApplication {
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
		hasSynchronizeAnnotation := customresources.HasResynchronizeAnnotation(instance)
		hasRotateAnnotation := customresources.HasRotateAnnotation(instance)
		return !isHashChanged && !hasExpiredSecrets && !secretNameChanged && !hasSynchronizeAnnotation && !hasRotateAnnotation
	}, timeout, interval, "AzureAdApplication should be synchronized")

	assert.True(t, finalizer.HasFinalizer(instance, options.FinalizerName), "AzureAdApplication should contain a finalizer")

	test.AssertAllNotEmpty(t, []any{
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

	assertPreAuthorizedAppsStatusIsValid(t, instance.Spec.PreAuthorizedApplications, instance.Status.PreAuthorizedApps)

	assert.Equal(t, events.Synchronized, instance.Status.SynchronizationState, "AzureAdApplication should be synchronized")
	return instance
}

func updateApplication(t *testing.T, previous *v1.AzureAdApplication, eventually func(updated *v1.AzureAdApplication) bool) *v1.AzureAdApplication {
	// sleep to allow sync time to elapse (non-millisecond precision)
	time.Sleep(time.Second)
	err := cli.Update(t.Context(), previous)
	assert.NoError(t, err, "updating existing application should not return error")

	updated := &v1.AzureAdApplication{}
	key := client.ObjectKey{
		Name:      previous.GetName(),
		Namespace: previous.GetNamespace(),
	}
	assert.Eventually(t, func() bool {
		err := cli.Get(context.Background(), key, updated)
		assert.NoError(t, err)

		return eventually(updated)
	}, timeout, interval, "AzureAdApplication should be updated")
	return updated
}

func syncTimeUpdated(previous, updated *v1.AzureAdApplication) bool {
	syncTimeUpdated := updated.Status.SynchronizationTime.After(previous.Status.SynchronizationTime.Time)
	synchronized := updated.Status.SynchronizationState == events.Synchronized
	return syncTimeUpdated && synchronized
}

func eventuallyHashUpdated(previous *v1.AzureAdApplication) func(updated *v1.AzureAdApplication) bool {
	return func(updated *v1.AzureAdApplication) bool {
		hashUpdated := updated.Status.SynchronizationHash != previous.Status.SynchronizationHash
		return hashUpdated && syncTimeUpdated(previous, updated)
	}
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

		actualAnnotations := secret.GetAnnotations()
		expectedAnnotations := map[string]string{
			annotations.StakaterReloaderKey: "true",
		}
		assert.NotEmpty(t, actualAnnotations, "Annotations should not be empty")
		assert.Equal(t, expectedAnnotations, actualAnnotations, "Annotations should be set")

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
		assert.NotEqual(t, previous.Data[key], new.Data[key], key)
	}
}

func assertSecretsAreRotated(t *testing.T, previous *corev1.Secret, new *corev1.Secret) {
	for _, key := range relevantSecretValues {
		assert.NotEqual(t, previous.Data[key], new.Data[key], key)
	}
	assert.Equal(t, previous.Data[secretDataKeys.NextCredentials.CertificateKeyId], new.Data[secretDataKeys.CurrentCredentials.CertificateKeyId])
	assert.Equal(t, previous.Data[secretDataKeys.NextCredentials.ClientSecret], new.Data[secretDataKeys.CurrentCredentials.ClientSecret])
	assert.Equal(t, previous.Data[secretDataKeys.NextCredentials.Jwk], new.Data[secretDataKeys.CurrentCredentials.Jwk])
	assert.Equal(t, previous.Data[secretDataKeys.NextCredentials.PasswordKeyId], new.Data[secretDataKeys.CurrentCredentials.PasswordKeyId])
}

func assertSecretsAreNotRotated(t *testing.T, previous *corev1.Secret, new *corev1.Secret) {
	for _, key := range relevantSecretValues {
		assert.Equal(t, previous.Data[key], new.Data[key], key)
	}
}

func assertPreAuthorizedAppsStatusIsValid(t *testing.T, expected []v1.AccessPolicyInboundRule, actual *v1.AzureAdPreAuthorizedAppsStatus) {
	expectedInvalid := make([]v1.AccessPolicyRule, 0)
	expectedValid := make([]v1.AccessPolicyRule, 0)

	for _, a := range expected {
		if strings.HasPrefix(a.Application, "invalid") {
			expectedInvalid = append(expectedInvalid, a.AccessPolicyRule)
		} else {
			expectedValid = append(expectedValid, a.AccessPolicyRule)
		}
	}

	assert.Equal(t, len(expectedInvalid), len(actual.Unassigned))
	assert.Len(t, expectedInvalid, *actual.UnassignedCount)

	assert.Equal(t, len(expectedValid), len(actual.Assigned))
	assert.Len(t, expectedValid, *actual.AssignedCount)

	contains := func(expected v1.AccessPolicyRule, actual []v1.AzureAdPreAuthorizedApp) bool {
		for _, a := range actual {
			if *a.AccessPolicyRule == expected {
				return true
			}
		}
		return false
	}

	for _, e := range expectedInvalid {
		assert.True(t, contains(e, actual.Unassigned))
	}

	for _, e := range expectedValid {
		assert.True(t, contains(e, actual.Assigned))
	}
}

func resourceExists(key client.ObjectKey, instance client.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return !errors.IsNotFound(err)
	}
}

func resourceDoesNotExist(key client.ObjectKey, instance client.Object) func() bool {
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

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
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
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
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
	azureratorCfg.Azure.Tenant.Id = "some-id"
	azureratorCfg.ClusterName = "test-cluster"

	azureOpenIDConfig := fake.AzureOpenIdConfig()

	err = (&controller.Reconciler{
		Client:            cli,
		Reader:            mgr.GetAPIReader(),
		Scheme:            mgr.GetScheme(),
		AzureClient:       azureClient,
		Recorder:          mgr.GetEventRecorderFor("azurerator"),
		Config:            azureratorCfg,
		AzureOpenIDConfig: azureOpenIDConfig,
		Synchronizer:      synchronizer.New(azureratorCfg.ClusterName, mgr.GetClient(), mgr.GetAPIReader()),
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

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		log.WithError(err).WithField("path", basePath).Errorf("Failed to read directory; have you run 'make setup-envtest'?")
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}

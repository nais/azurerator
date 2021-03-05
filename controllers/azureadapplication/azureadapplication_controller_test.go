package azureadapplication

import (
	"context"
	"fmt"
	"github.com/nais/azureator/pkg/annotations"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/util/azurerator"
	"github.com/nais/liberator/pkg/crd"
	finalizer2 "github.com/nais/liberator/pkg/finalizer"
	"os"
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
	timeout  = time.Second * 5
	interval = time.Millisecond * 100

	alreadyInUseSecret = "in-use-by-pod"
	unusedSecret       = "unused-secret"
	newSecret          = "new-secret"

	namespace = "default"
)

var cli client.Client
var azureClient = fake.NewFakeAzureClient()

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
			instance := assertApplicationExists(t, "New AzureAdApplication should exist", c.appName)

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
	assert.True(t, finalizer2.HasFinalizer(instance, FinalizerName), "AzureAdApplication should contain a finalizer")
	assert.Equal(t, v1.EventSkipped, instance.Status.SynchronizationState, "AzureAdApplication should be skipped")
	assertAnnotationExists(t, instance, annotations.SkipKey, annotations.SkipValue)
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
	assert.False(t, finalizer2.HasFinalizer(instance, FinalizerName), "AzureAdApplication should not contain a finalizer")
}

func TestReconciler_UpdateAzureAdApplication(t *testing.T) {
	instance := assertApplicationExists(t, "Existing AzureAdApplication should exist", fake.ApplicationExists)

	// fetch secret name referenced by previous generation
	previousSecretName := instance.Spec.SecretName

	// update with new secret name
	newSecretName := fmt.Sprintf("%s-%s", instance.GetName(), newSecret)
	instance.Spec.SecretName = newSecretName
	err := cli.Update(context.Background(), instance)
	assert.NoError(t, err, "updating existing application should not return error")

	newInstance := assertApplicationExists(t, "Existing AzureAdApplication should still exist and be synchronized", instance.GetName())

	// status subresource should contain new IDs for rotated credentials
	t.Run("AzureAdApplication Status subresource should be updated", func(t *testing.T) {
		assert.Eventually(t, func() bool {
			passwordKeyIdsValid := len(newInstance.Status.PasswordKeyIds) == 2
			certificateKeyIdsValid := len(newInstance.Status.CertificateKeyIds) == 2
			return passwordKeyIdsValid && certificateKeyIdsValid
		}, timeout, interval, "should contain new IDs for rotated credentials")
	})

	// new secret should exist
	assertSecretExists(t, newSecretName, instance)

	// old secret referenced by pod should still exist
	assertSecretExists(t, previousSecretName, instance)
}

func TestReconciler_DeleteAzureAdApplication(t *testing.T) {
	instance := assertApplicationExists(t, "Existing AzureAdApplication should exist", fake.ApplicationExists)

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
func assertApplicationExists(t *testing.T, testName string, name string) *v1.AzureAdApplication {
	instance := &v1.AzureAdApplication{}
	key := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	t.Run(testName, func(t *testing.T) {
		assert.Eventually(t, resourceExists(key, instance), timeout, interval, "AzureAdApplication should exist")

		assert.Eventually(t, func() bool {
			err := cli.Get(context.Background(), key, instance)
			assert.NoError(t, err)
			b, err := azurerator.IsUpToDate(instance)
			assert.NoError(t, err)
			return b
		}, timeout, interval, "AzureAdApplication should be synchronized")

		assert.True(t, finalizer2.HasFinalizer(instance, FinalizerName), "AzureAdApplication should contain a finalizer2")

		assert.Empty(t, instance.Annotations[annotations.SkipKey], "AzureAdApplication should not contain skip annotation")

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
		})
		assert.Equal(t, v1.EventSynchronized, instance.Status.SynchronizationState, "AzureAdApplication should be synchronized")
	})
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

func assertSecretExists(t *testing.T, name string, instance *v1.AzureAdApplication) {
	t.Run(fmt.Sprintf("Secret '%s' should exist", name), func(t *testing.T) {
		key := client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		a := &corev1.Secret{}

		assert.Eventually(t, resourceExists(key, a), timeout, interval, "Secret should exist")

		assert.True(t, containsOwnerRef(a.GetOwnerReferences(), *instance), "Secret should contain ownerReference")

		actualLabels := a.GetLabels()
		expectedLabels := map[string]string{
			labels.AppLabelKey:  instance.GetName(),
			labels.TypeLabelKey: labels.TypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")

		assert.Equal(t, corev1.SecretTypeOpaque, a.Type, "Secret type should be Opaque")

		test.AssertContainsKeysWithNonEmptyValues(t, a.Data, secrets.AllKeys)

		azureOpenIdConfig := fake.AzureOpenIdConfig()
		assert.Equal(t, azureOpenIdConfig.WellKnownEndpoint, string(a.Data[secrets.WellKnownUrlKey]))
		assert.Equal(t, azureOpenIdConfig.Issuer, string(a.Data[secrets.OpenIDConfigIssuerKey]))
		assert.Equal(t, azureOpenIdConfig.JwksURI, string(a.Data[secrets.OpenIDConfigJwksUriKey]))
		assert.Equal(t, azureOpenIdConfig.TokenEndpoint, string(a.Data[secrets.OpenIDConfigTokenEndpointKey]))
	})
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

	azureOpenIDConfig := fake.AzureOpenIdConfig()

	err = (&Reconciler{
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

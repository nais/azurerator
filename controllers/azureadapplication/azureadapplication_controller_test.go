package azureadapplication

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nais/azureator/api/v1"
	"github.com/nais/azureator/pkg/annotations"
	azureFixtures "github.com/nais/azureator/pkg/fixtures/azure"
	"github.com/nais/azureator/pkg/fixtures/k8s"
	"github.com/nais/azureator/pkg/labels"
	"github.com/nais/azureator/pkg/secrets"
	"github.com/nais/azureator/pkg/util/test"
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
var azureClient = azureFixtures.NewFakeAzureClient()

func TestMain(m *testing.M) {
	testEnv, err := setup()
	if err != nil {
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
			azureFixtures.ApplicationExists,
		},
		{
			"Application does not exist in Azure AD",
			azureFixtures.ApplicationNotExistsName,
		},
	}
	for _, c := range cases {
		// prefix app name to secret name for "unique" secret in this test environment
		secretName := fmt.Sprintf("%s-%s", c.appName, alreadyInUseSecret)

		// set up preconditions for cluster
		clusterFixtures := k8s.New(cli, k8s.Config{
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
	appName := "should-not-process"
	sharedNamespace := "shared"
	secretName := fmt.Sprintf("%s-%s", appName, alreadyInUseSecret)
	clusterFixtures := k8s.New(cli, k8s.Config{
		AzureAppName:     appName,
		SecretName:       secretName,
		UnusedSecretName: unusedSecret,
		NamespaceName:    sharedNamespace,
	}).WithMinimalConfig().WithSharedNamespace()

	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}
	t.Run("AzureAdApplication in shared namespace should not be processed", func(t *testing.T) {
		instance := &v1.AzureAdApplication{}
		key := client.ObjectKey{
			Name:      appName,
			Namespace: sharedNamespace,
		}
		assert.Eventually(t, resourceExists(key, instance), timeout, interval, "AzureAdApplication should exist")

		assert.Eventually(t, func() bool {
			_, key := instance.Annotations[annotations.SkipKey]
			return key
		}, timeout, interval, fmt.Sprintf("Annotation '%s' should exist on resource", annotations.SkipKey))

		assert.True(t, instance.HasFinalizer(FinalizerName), "AzureAdApplication should contain a finalizer")
		assert.Equal(t, instance.Annotations[annotations.SkipKey], annotations.SkipValue, "AzureAdApplication should contain skip annotation")
		assert.Empty(t, instance.Status.CertificateKeyIds)
		assert.Empty(t, instance.Status.ClientId)
		assert.Empty(t, instance.Status.CorrelationId)
		assert.Empty(t, instance.Status.ObjectId)
		assert.Empty(t, instance.Status.PasswordKeyIds)
		assert.Empty(t, instance.Status.ProvisionHash)
		assert.Empty(t, instance.Status.ServicePrincipalId)
		assert.Empty(t, instance.Status.Timestamp)
		assert.False(t, instance.Status.Synchronized, "AzureAdApplication should not be synchronized")
	})
}

func TestReconciler_UpdateAzureAdApplication(t *testing.T) {
	instance := assertApplicationExists(t, "Existing AzureAdApplication should exist", azureFixtures.ApplicationExists)

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
	instance := assertApplicationExists(t, "Existing AzureAdApplication should exist", azureFixtures.ApplicationExists)

	t.Run("Delete existing AzureAdApplication", func(t *testing.T) {
		err := cli.Delete(context.Background(), instance)
		assert.NoError(t, err, "deleting existing AzureAdApplication should not return error")

		key := client.ObjectKey{
			Name:      azureFixtures.ApplicationExists,
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
			b, err := instance.IsUpToDate()
			assert.NoError(t, err)
			return b
		}, timeout, interval, "AzureAdApplication should be synchronized")

		assert.True(t, instance.HasFinalizer(FinalizerName), "AzureAdApplication should contain a finalizer")

		test.AssertAllNotEmpty(t, []interface{}{
			instance.Status.CertificateKeyIds,
			instance.Status.ClientId,
			instance.Status.CorrelationId,
			instance.Status.ObjectId,
			instance.Status.PasswordKeyIds,
			instance.Status.ProvisionHash,
			instance.Status.ServicePrincipalId,
			instance.Status.Timestamp,
		})
		assert.True(t, instance.Status.Synchronized, "AzureAdApplication should be synchronized")
	})
	return instance
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

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd")},
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

	err = (&Reconciler{
		Client:      cli,
		Reader:      mgr.GetAPIReader(),
		Scheme:      mgr.GetScheme(),
		AzureClient: azureClient,
		Recorder:    mgr.GetEventRecorderFor("azurerator"),
		ClusterName: "local",
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

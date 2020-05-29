package azureadapplication

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	azureClient "github.com/nais/azureator/pkg/azure/client"
	"github.com/nais/azureator/pkg/config"
	azureFixtures "github.com/nais/azureator/pkg/fixtures/azure"
	"github.com/nais/azureator/pkg/fixtures/k8s"
	"github.com/nais/azureator/pkg/resourcecreator"
	"github.com/nais/azureator/pkg/util/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1

	alreadyInUseSecret = "in-use-by-pod"
	unusedSecret       = "unused-secret"
	newSecret          = "new-secret"

	namespace = "default"
)

var cli client.Client

func TestReconcilerIntegration(t *testing.T) {
	// TODO
	t.Skip("TODO - skipping integration test")

	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cfg, err := config.New()
	config.Print([]string{})
	assert.NoError(t, err, "config must be set up to run integration tests")
	if err != nil {
		t.FailNow()
	}

	az, err := azureClient.New(context.Background(), &cfg.AzureAd)
	testReconciler(t, az)
}

func TestReconcilerIntegrationMock(t *testing.T) {
	az := azureFixtures.NewAzureClient()
	testReconciler(t, az)
}

func testReconciler(t *testing.T, az azure.Client) {
	testEnv := setup(t, az)

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
		t.Run(c.name, func(t *testing.T) {
			t.Run("Create new AzureAdApplication", func(t *testing.T) {
				testCreate(t, c.appName)
			})
			t.Run("Update existing AzureAdApplication", func(t *testing.T) {
				testUpdate(t, c.appName)
			})
			t.Run("Delete existing AzureAdApplication", func(t *testing.T) {
				testDelete(t, c.appName)
			})
		})
	}

	err := testEnv.Stop()
	assert.NoError(t, err)
}

func testCreate(t *testing.T, name string) {
	// prefix app name to secret name for "unique" secret in this test environment
	secretName := fmt.Sprintf("%s-%s", name, alreadyInUseSecret)

	// set up preconditions for cluster
	clusterFixtures := k8s.ClusterFixtures{
		Name:             name,
		SecretName:       secretName,
		UnusedSecretName: unusedSecret,
		Namespace:        namespace,
	}
	if err := clusterFixtures.Setup(cli); err != nil {
		t.Error(err)
		t.FailNow()
	}

	instance := assertApplicationExists(t, name)
	assertSecretExists(t, secretName, instance)
	assertSecretDoesNotExist(t, unusedSecret)
}

func testUpdate(t *testing.T, name string) {
	// existing application should exist
	instance := assertApplicationExists(t, name)

	// fetch secret name referenced by previous generation
	previousSecretName := instance.Spec.SecretName

	// update with new secret name
	newSecretName := fmt.Sprintf("%s-%s", instance.GetName(), newSecret)
	instance.Spec.SecretName = newSecretName
	err := cli.Update(context.Background(), instance)
	assert.NoError(t, err)

	// application should still exist and be synchronized
	newInstance := assertApplicationExists(t, instance.GetName())

	// status subresource should contain new IDs for rotated credentials
	t.Run("Status subresource should contain new IDs for rotated credentials", func(t *testing.T) {
		assert.Eventually(t, func() bool {
			passwordKeyIdsValid := len(newInstance.Status.PasswordKeyIds) == 2
			certificateKeyIdsValid := len(newInstance.Status.CertificateKeyIds) == 2
			return passwordKeyIdsValid && certificateKeyIdsValid
		}, timeout, interval)
	})

	// new secret should exist
	assertSecretExists(t, newSecretName, instance)

	// old secret referenced by pod should still exist
	assertSecretExists(t, previousSecretName, instance)
}

func testDelete(t *testing.T, name string) {
	// existing application should exist
	instance := assertApplicationExists(t, name)

	// delete existing application
	if err := cli.Delete(context.Background(), instance); err != nil {
		t.Error(err)
		t.FailNow()
	}

	t.Run("AzureAdApplication should be deleted", func(t *testing.T) {
		key := types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}
		assert.Eventually(t, resourceDoesNotExist(key, instance), timeout, interval)
	})
}

// asserts that the application exists in the cluster and is valid
func assertApplicationExists(t *testing.T, name string) *v1alpha1.AzureAdApplication {
	instance := &v1alpha1.AzureAdApplication{}
	t.Run("AzureAdApplication", func(t *testing.T) {
		t.Run("should exist in cluster", func(t *testing.T) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			assert.Eventually(t, resourceExists(key, instance), timeout, interval)
		})

		t.Run("should be synchronized", func(t *testing.T) {
			assert.Eventually(t, func() bool {
				b, err := instance.IsUpToDate()
				assert.NoError(t, err)
				return b
			}, timeout, interval, "AzureAdApplication should be synchronized")
		})

		t.Run("should have a finalizer", func(t *testing.T) {
			assert.True(t, instance.HasFinalizer(FinalizerName))
		})

		t.Run("should have a valid status subresource", func(t *testing.T) {
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
			assert.True(t, instance.Status.Synchronized)
		})
	})
	return instance
}

func assertSecretExists(t *testing.T, name string, instance *v1alpha1.AzureAdApplication) {
	t.Run(fmt.Sprintf("Secret '%s'", name), func(t *testing.T) {
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		a := &corev1.Secret{}
		t.Run("should exist in cluster", func(t *testing.T) {
			assert.Eventually(t, resourceExists(key, a), timeout, interval)
		})

		t.Run("should have correct OwnerReference", func(t *testing.T) {
			assert.True(t, containsOwnerRef(a.GetOwnerReferences(), *instance))
		})

		t.Run("should contain expected keys", func(t *testing.T) {
			test.AssertContainsKeysWithNonEmptyValues(t, a.Data, resourcecreator.AllKeys)
		})
	})
}

func assertSecretDoesNotExist(t *testing.T, name string) {
	t.Run("Unused Secret", func(t *testing.T) {
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		a := &corev1.Secret{}
		t.Run("should not exist in cluster", func(t *testing.T) {
			assert.Eventually(t, resourceDoesNotExist(key, a), timeout, interval)
		})
	})
}

func resourceExists(key types.NamespacedName, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return !errors.IsNotFound(err)
	}
}

func resourceDoesNotExist(key types.NamespacedName, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return errors.IsNotFound(err)
	}
}

func containsOwnerRef(refs []v1.OwnerReference, owner v1alpha1.AzureAdApplication) bool {
	expected := v1.OwnerReference{
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

func setup(t *testing.T, az azure.Client) *envtest.Environment {
	log := zap.New(zap.UseDevMode(true))
	ctrl.SetLogger(log)

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	err = v1alpha1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	// +kubebuilder:scaffold:scheme

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	assert.NoError(t, err)

	cli = mgr.GetClient()
	assert.NotNil(t, cli)

	err = (&Reconciler{
		Client:      cli,
		Log:         ctrl.Log.WithName("controllers").WithName("AzureAdApplication"),
		Scheme:      mgr.GetScheme(),
		AzureClient: az,
		Recorder:    mgr.GetEventRecorderFor("azurerator"),
		ClusterName: "local",
	}).SetupWithManager(mgr)
	assert.NoError(t, err)

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()

	return testEnv
}

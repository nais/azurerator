package azureadapplication

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/nais/azureator/api/v1alpha1"
	"github.com/nais/azureator/pkg/azure"
	azureClient "github.com/nais/azureator/pkg/azure/client"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/fixtures"
	"github.com/nais/azureator/pkg/resourcecreator"
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

	secretName    = "test-secret"
	configMapName = "test-configmap"
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
	az := fixtures.NewAzureClient()
	testReconciler(t, az)
}

func testReconciler(t *testing.T, az azure.Client) {
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

	cases := []struct {
		name          string
		keyName       string
		existsInAzure bool
	}{
		{
			"Application already exists in Azure AD",
			"test-azureadapplication",
			true,
		},
		{
			"Application does not exist in Azure AD",
			fixtures.ApplicationNotExistsName,
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Run("Create new AzureAdApplication", func(t *testing.T) {
				testCreate(t, c.keyName)
			})
			t.Run("Update existing AzureAdApplication", func(t *testing.T) {
				if c.existsInAzure {
					testUpdateExistsInAzure(t)
				} else {
					testUpdateNotExistsInAzure(t)
				}
			})
			t.Run("Delete existing AzureAdApplication", func(t *testing.T) {
				testDelete(t)
			})
		})
	}

	err = testEnv.Stop()
	assert.NoError(t, err)
}

func testCreate(t *testing.T, name string) {
	key := types.NamespacedName{
		Name:      name,
		Namespace: name,
	}
	spec := v1alpha1.AzureAdApplicationSpec{
		ReplyUrls:                 nil,
		PreAuthorizedApplications: nil,
		LogoutUrl:                 "",
		SecretName:                secretName,
		ConfigMapName:             configMapName,
	}
	instance := fixtures.K8sAzureAdApplication(key, spec, v1alpha1.AzureAdApplicationStatus{})
	azureApp := fixtures.InternalAzureApp(*instance)

	err := cli.Create(context.Background(), instance)
	assert.NoError(t, err)

	t.Run("AzureAdApplication", func(t *testing.T) {
		instance = &v1alpha1.AzureAdApplication{}
		t.Run("should exist in cluster", func(t *testing.T) {
			assert.Eventually(t, resourceExists(key, instance), timeout, interval)
		})

		t.Run("should be synchronized", func(t *testing.T) {
			b, err := instance.IsUpToDate()
			assert.NoError(t, err)
			assert.True(t, b, "AzureAdApplication should be synchronized")
		})

		t.Run("should have a finalizer", func(t *testing.T) {
			assert.True(t, instance.HasFinalizer(FinalizerName))
		})

		t.Run("should have a valid status subresource", func(t *testing.T) {
			assertAllNotEmpty(t, []interface{}{
				instance.Status.ClientId,
				instance.Status.ObjectId,
				instance.Status.ServicePrincipalId,
				instance.Status.Timestamp,
				instance.Status.ProvisionHash,
				instance.Status.CorrelationId,
				instance.Status.PasswordKeyId,
				instance.Status.CertificateKeyId,
			})
			assert.True(t, instance.Status.Synchronized)
		})
	})

	t.Run("Secret", func(t *testing.T) {
		key := expectedSecretKey(instance, azureApp)
		a := &corev1.Secret{}
		t.Run("should exist in cluster", func(t *testing.T) {
			assert.Eventually(t, resourceExists(key, a), timeout, interval)
		})

		t.Run("should have correct OwnerReference", func(t *testing.T) {
			assert.True(t, containsOwnerRef(a.GetOwnerReferences(), *instance))
		})

		t.Run("should contain expected keys", func(t *testing.T) {
			assertContainsKeysWithNonEmptyValues(t, a.Data, []string{"clientSecret", "jwk"})
		})
	})

	t.Run("ConfigMap", func(t *testing.T) {
		key := expectedConfigMapKey(instance, azureApp)
		a := &corev1.ConfigMap{}
		t.Run("should exist in cluster", func(t *testing.T) {
			assert.Eventually(t, resourceExists(key, a), timeout, interval)
		})

		t.Run("should have correct OwnerReference", func(t *testing.T) {
			assert.True(t, containsOwnerRef(a.GetOwnerReferences(), *instance))
		})

		t.Run("should contain expected keys", func(t *testing.T) {
			assertContainsKeysWithNonEmptyValues(t, a.Data, []string{"clientId", "jwk", "preAuthorizedApps"})
		})
	})
}

// TODO
func testUpdateExistsInAzure(t *testing.T) {
	t.Run("AzureAdApplication should have updated status subresource", func(t *testing.T) {

	})

	t.Run("Secret should be updated", func(t *testing.T) {

	})

	t.Run("ConfigMap should be updated", func(t *testing.T) {

	})
}

// TODO
func testUpdateNotExistsInAzure(t *testing.T) {
	t.Run("AzureAdApplication should not be updated", func(t *testing.T) {

	})

	t.Run("Secret should not be updated", func(t *testing.T) {

	})

	t.Run("ConfigMap should not be updated", func(t *testing.T) {

	})
}

// TODO
func testDelete(t *testing.T) {
	t.Run("AzureAdApplication should be deleted", func(t *testing.T) {

	})
}

func resourceExists(key types.NamespacedName, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return !errors.IsNotFound(err)
	}
}

func expectedSecretKey(instance *v1alpha1.AzureAdApplication, app azure.Application) types.NamespacedName {
	creator := resourcecreator.NewSecret(*instance, app)
	spec, _ := creator.Spec()
	secret := spec.(*corev1.Secret)
	return types.NamespacedName{
		Namespace: secret.GetNamespace(),
		Name:      secret.GetName(),
	}
}

func expectedConfigMapKey(instance *v1alpha1.AzureAdApplication, app azure.Application) types.NamespacedName {
	creator := resourcecreator.NewConfigMap(*instance, app)
	spec, _ := creator.Spec()
	secret := spec.(*corev1.ConfigMap)
	return types.NamespacedName{
		Namespace: secret.GetNamespace(),
		Name:      secret.GetName(),
	}
}

func assertContainsKeysWithNonEmptyValues(t *testing.T, a interface{}, keys []string) {
	for _, key := range keys {
		assert.Contains(t, a, key)
	}
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Map {
		for _, val := range v.MapKeys() {
			assert.NotEmpty(t, v.MapIndex(val).String())
		}
	}
}

func assertAllNotEmpty(t *testing.T, values []interface{}) {
	for _, val := range values {
		assert.NotEmpty(t, val)
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

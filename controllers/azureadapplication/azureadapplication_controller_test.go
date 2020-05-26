package azureadapplication

import (
	"context"
	"path/filepath"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	timeout  = time.Second * 30
	interval = time.Second * 1

	secretName    = "test-secret"
	configMapName = "test-configmap"
)

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

	cli := mgr.GetClient()
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

	t.Run("Create new AzureAdApplication", func(t *testing.T) {
		key := types.NamespacedName{
			Name:      "test-azureadapplication",
			Namespace: "default",
		}
		instance := &v1alpha1.AzureAdApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: v1alpha1.AzureAdApplicationSpec{
				ReplyUrls:                 nil,
				PreAuthorizedApplications: nil,
				LogoutUrl:                 "",
				SecretName:                secretName,
				ConfigMapName:             configMapName,
			},
		}

		azureApp := fixtures.InternalAzureApp(*instance)

		// Create
		err := cli.Create(context.Background(), instance)
		assert.NoError(t, err)

		t.Run("Should have created an AzureAdApplication", func(t *testing.T) {
			a := &v1alpha1.AzureAdApplication{}
			assert.Eventually(t, func() bool {
				err = cli.Get(context.Background(), key, a)
				if err != nil {
					return !errors.IsNotFound(err)
				}
				b, _ := a.IsUpToDate()
				return b
			}, timeout, interval)
		})

		t.Run("Should have created a Secret", func(t *testing.T) {
			creator := resourcecreator.NewSecret(*instance, azureApp)
			spec, _ := creator.Spec()
			secret := spec.(*corev1.Secret)
			key := types.NamespacedName{
				Namespace: secret.GetNamespace(),
				Name:      secret.GetName(),
			}
			a := &corev1.Secret{}
			assert.Eventually(t, func() bool {
				err = cli.Get(context.Background(), key, a)
				return !errors.IsNotFound(err)
			}, timeout, interval)
		})

		t.Run("Should have created a ConfigMap", func(t *testing.T) {
			creator := resourcecreator.NewConfigMap(*instance, azureApp)
			spec, _ := creator.Spec()
			secret := spec.(*corev1.ConfigMap)
			key := types.NamespacedName{
				Namespace: secret.GetNamespace(),
				Name:      secret.GetName(),
			}
			a := &corev1.ConfigMap{}
			assert.Eventually(t, func() bool {
				err = cli.Get(context.Background(), key, a)
				return !errors.IsNotFound(err)
			}, timeout, interval)
		})
	})

	// TODO
	t.Run("Update existing AzureAdApplication", func(t *testing.T) {

	})

	// TODO
	t.Run("Delete existing AzureAdApplication", func(t *testing.T) {

	})

	err = testEnv.Stop()
	assert.NoError(t, err)
}

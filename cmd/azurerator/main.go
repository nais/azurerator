package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nais/azureator/pkg/azure"
	"github.com/nais/azureator/pkg/azure/client"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/controllers/azureadapplication"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	naisiov1alpha1 "github.com/nais/azureator/pkg/apis/v1alpha1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = naisiov1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	err := run()

	if err != nil {
		setupLog.Error(err, "Run loop errored")
		os.Exit(1)
	}

	setupLog.Info("Manager shutting down")
}

func run() error {
	log := zap.New(zap.UseDevMode(true))
	ctrl.SetLogger(log)

	ctx := context.Background()

	cfg, err := config.New()
	if err != nil {
		return err
	}

	config.Print([]string{
		azure.ClientSecret,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: cfg.MetricsAddr,
		Port:               9443,
		LeaderElection:     cfg.EnableLeaderElection,
		LeaderElectionID:   "43d2b63b.nais.io",
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	azureClient, err := client.New(ctx, &cfg.AzureAd)
	if err != nil {
		return fmt.Errorf("unable to create Azure client: %w", err)
	}

	if err = (&azureadapplication.Reconciler{
		Client:      mgr.GetClient(),
		Log:         ctrl.Log.WithName("controllers").WithName("AzureAdApplication"),
		Scheme:      mgr.GetScheme(),
		AzureClient: azureClient,
		ClusterName: cfg.ClusterName,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}

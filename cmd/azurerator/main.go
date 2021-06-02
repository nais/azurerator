package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/zapr"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/nais/azureator/controllers/azureadapplication"
	"github.com/nais/azureator/pkg/azure/client"
	"github.com/nais/azureator/pkg/config"
	"github.com/nais/azureator/pkg/logger"
	azureMetrics "github.com/nais/azureator/pkg/metrics"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	metrics.Registry.MustRegister(azureMetrics.AllMetrics...)
	logger.SetupLogrus()

	_ = clientgoscheme.AddToScheme(scheme)
	_ = naisiov1.AddToScheme(scheme)
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
	ctx := context.Background()

	zapLogger, err := logger.ZapLogger()
	if err != nil {
		return err
	}
	ctrl.SetLogger(zapr.NewLogger(zapLogger))

	cfg, err := config.DefaultConfig()
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: cfg.MetricsAddr,
		LeaderElection:     false,
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	azureClient, err := client.New(ctx, &cfg.Azure)
	if err != nil {
		return fmt.Errorf("unable to create Azure client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	azureOpenIDConfig, err := config.NewAzureOpenIdConfig(ctx, cfg.Azure.Tenant)
	if err != nil {
		return fmt.Errorf("fetching Azure OpenID Configuration: %w", err)
	}

	if err = (&azureadapplication.Reconciler{
		Client:            mgr.GetClient(),
		Reader:            mgr.GetAPIReader(),
		Scheme:            mgr.GetScheme(),
		AzureClient:       azureClient,
		Config:            cfg,
		Recorder:          mgr.GetEventRecorderFor("azurerator"),
		AzureOpenIDConfig: *azureOpenIDConfig,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting metrics refresh goroutine")
	clusterMetrics := azureMetrics.New(mgr.GetAPIReader())
	go clusterMetrics.Refresh(context.Background())

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}
